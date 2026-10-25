package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type uncoveredWall struct {
	start   geometry.XY
	end     geometry.XY
	zBS     float64
	zBE     float64
	zTS     float64
	zTE     float64
	tag     string
	mat     *textures.Material
	csId    string
	csTag   string
	csLight *config.Light
	modelId int
	align   *config.UVAlign
}

// Compiler2d is responsible for compiling 2D sector data, vertices, and materials into structured volume representations.
type Compiler2d struct {
	cfg       *config.Root
	materials *Materials
	lights    *Lights
}

// NewCompiler2d creates and returns a new instance of Compiler2d, used for compiling 2D game sectors and related data.
func NewCompiler2d(cfg *config.Root, materials *Materials, lights *Lights) *Compiler2d {
	return &Compiler2d{
		cfg:       cfg,
		materials: materials,
		lights:    lights,
	}
}

// Compile generates 2D and 3D volumes based on the configuration, materials, and lights, returning the compiled volumes.
func (r *Compiler2d) Compile() ([]*Volume, error) {
	var allVolumes []*Volume
	if len(r.cfg.Sectors) == 0 {
		return nil, nil
	}

	sectors, uncoveredWalls := r.compile2d()
	if len(sectors) == 0 {
		return nil, fmt.Errorf("no 2D volumes compiled")
	}
	locator := NewSectors(sectors, false)
	locator.Setup()
	//player Z
	pv := locator.QueryPoint(r.cfg.Player.Position.X, r.cfg.Player.Position.Y)
	if pv == nil {
		return nil, fmt.Errorf("can't find 2d player location at X: %f Y: %f", r.cfg.Player.Position.X, r.cfg.Player.Position.Y)
	}
	if r.cfg.Player.Position.Z == 0 {
		r.cfg.Player.Position.Z = pv.GetMinZ()
	}
	//things Z
	for idx := range r.cfg.Things {
		thing := r.cfg.Things[idx]
		if thing == nil {
			continue
		}
		tx, ty := thing.Position.X, thing.Position.Y
		tv := locator.QueryPoint(tx, ty)
		if tv == nil {
			fmt.Println("can't find thing location at", tx, ty)
			continue
		}
		if thing.Position.Z == 0 {
			thing.Position.Z = tv.GetMinZ()
		}
	}
	//light 2d
	r.lights.AddLights(r.compileLights2d(locator, true))

	allVolumes = append(allVolumes, r.upgrade3d(sectors)...)
	allVolumes = append(allVolumes, r.buildUncoveredWalls(uncoveredWalls)...)

	return allVolumes, nil
}

// compile2d processes 2D geometry and sector configurations to construct 2D sectors and identifies uncovered walls.
func (r *Compiler2d) compile2d() ([]*Sector, []*uncoveredWall) {
	vertices, css, anim := r.cfg.Vertices, r.cfg.Sectors, r.materials
	const epsilon = 0.01
	modelSectorId := 0
	var container []*Sector
	var uncoveredWalls []*uncoveredWall
	var fixSegments []*Segment
	facesTree := aabb.NewTree(1024, epsilon)
	emptyAnim := anim.GetEmptyMaterial()

	css = r.updateAdjDependencies(css)

	ve := NewSectorsEdges(epsilon)
	ve.Construct(vertices, css)

	unknownCounter := 0
	for csIdx, cs := range css {
		if len(cs.Segments) == 0 {
			continue
		}

		triContainer, _, triErr := ve.GetTriangles(csIdx)
		if triErr != nil {
			fmt.Println("Error retrieving polygons for sector", csIdx, ":", triErr.Error())
			continue
		}

		var sSlopeFloor *Slope = nil
		var sSlopeCeiling *Slope = nil
		if slSeg := cs.GetSlopeFloor(); slSeg != nil {
			sl := cs.SlopeFloor

			sSlopeFloor = &Slope{G: sl.G, Gradient: sl.Gradient, Start: slSeg.Start, End: slSeg.End}
		}
		if slSeg := cs.GetSlopeCeiling(); slSeg != nil {
			sl := cs.SlopeCeiling
			sSlopeCeiling = &Slope{G: sl.G, Gradient: sl.Gradient, Start: slSeg.Start, End: slSeg.End}
		}

		coveredSegments := make([]bool, len(cs.Segments))
		for _, triangles := range triContainer {
			for _, tri := range triangles {
				if len(tri) != 3 {
					fmt.Println("wrong tri", tri)
					continue
				}

				volumeMaterials := []*textures.Material{anim.GetMaterial(cs.Floor), anim.GetMaterial(cs.Ceil)}

				sector := NewSector(modelSectorId, cs.Id, cs.FloorY, cs.CeilY, volumeMaterials, cs.Tag)
				sector.SetSlopes(sSlopeFloor, sSlopeCeiling)
				sector.SetAlign(cs.FloorAlign, cs.CeilAlign)

				modelSectorId++

				// Maintains consistent winding order for ContainsPoint.
				if geometry.PointInLineDirectionF(tri[2].X, tri[2].Y, tri[0].X, tri[0].Y, tri[1].X, tri[1].Y) < 0 {
					tri[1], tri[2] = tri[2], tri[1]
				}

				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]

					start := geometry.XY{X: p1.X, Y: p1.Y}
					end := geometry.XY{X: p2.X, Y: p2.Y}

					isWall := false
					upper, middle, lower := emptyAnim, emptyAnim, emptyAnim
					tag := fmt.Sprintf("unknown_%d", unknownCounter)
					unknownCounter++
					var upperLimitZS, upperLimitZE float64
					hasUpperLimitZ := false

					var segUVAlign *config.UVAlign
					segT1, segT2 := 0.0, 1.0

					for cnIdx, cn := range cs.Segments {
						found := false
						t1, t2 := 0.0, 1.0

						if p1 == cn.Start && p2 == cn.End {
							found = true
							t1, t2 = 0.0, 1.0
						} else if p1 == cn.End && p2 == cn.Start {
							found = true
							t1, t2 = 1.0, 0.0
						} else if geometry.IsSegmentSubset(p1, p2, cn.Start, cn.End) {
							found = true
							dx := cn.End.X - cn.Start.X
							dy := cn.End.Y - cn.Start.Y
							lenSq := dx*dx + dy*dy
							if lenSq > 0 {
								t1 = ((p1.X-cn.Start.X)*dx + (p1.Y-cn.Start.Y)*dy) / lenSq
								t2 = ((p2.X-cn.Start.X)*dx + (p2.Y-cn.Start.Y)*dy) / lenSq
							}
						} else if geometry.IsSegmentSubset(p2, p1, cn.End, cn.Start) {
							found = true
							dx := cn.Start.X - cn.End.X
							dy := cn.Start.Y - cn.End.Y
							lenSq := dx*dx + dy*dy
							if lenSq > 0 {
								t1 = ((p1.X-cn.End.X)*dx + (p1.Y-cn.End.Y)*dy) / lenSq
								t2 = ((p2.X-cn.End.X)*dx + (p2.Y-cn.End.Y)*dy) / lenSq
							}
						}

						if found {
							coveredSegments[cnIdx] = true
							isWall = cn.Kind == config.SegmentWall
							tag = cn.Tag
							segUVAlign = cn.UVAlign
							segT1, segT2 = t1, t2

							if cn.Upper != nil {
								upper = anim.GetMaterial(cn.Upper)
							}
							if cn.Middle != nil {
								middle = anim.GetMaterial(cn.Middle)
							}
							if cn.Lower != nil {
								lower = anim.GetMaterial(cn.Lower)
							}
							if cn.HasUpperLimitZ {
								hasUpperLimitZ = true
								upperLimitZS = cn.UpperLimitZS + t1*(cn.UpperLimitZE-cn.UpperLimitZS)
								upperLimitZE = cn.UpperLimitZS + t2*(cn.UpperLimitZE-cn.UpperLimitZS)
							}
							break
						}
					}

					segMaterials := []*textures.Material{upper, middle, lower}
					seg := NewSegment(nil, start, end, tag, segMaterials)
					if hasUpperLimitZ {
						seg.SetUpperLimitZ(upperLimitZS, upperLimitZE)
					}
					if segUVAlign != nil {
						seg.SetUVAlign(segUVAlign, segT1, segT2)
					}
					sector.AddSegment(seg)
					sector.AddTag(tag)

					if !isWall {
						fixSegments = append(fixSegments, seg)
					}
				}

				light := NewLight()
				sector.SetLight(light)

				if cs.Light != nil {
					centroid := sector.GetCentroid()
					lightPos := geometry.XYZ{
						X: centroid.X,
						Y: centroid.Y,
						Z: cs.FloorY + cs.CeilY,
					}
					light.Setup(cs.Light, lightPos)
				}

				sector.Rebuild()
				segments, segmentCount := sector.GetSegments()
				for x := 0; x < segmentCount; x++ {
					facesTree.InsertObject(segments[x])
				}

				container = append(container, sector)
			}
		}

		for cnIdx, cn := range cs.Segments {
			if cn == nil || coveredSegments[cnIdx] {
				continue
			}
			matCfg := cn.Middle
			if matCfg == nil {
				matCfg = cn.Upper
			}
			if matCfg == nil {
				matCfg = cn.Lower
			}
			if matCfg == nil || cn.Kind != config.SegmentWall {
				continue
			}
			mat := anim.GetMaterial(matCfg)
			if mat == nil || mat.CurrentFrame() == nil {
				continue
			}

			curFS, curFE := cs.FloorY, cs.FloorY
			curCS, curCE := cs.CeilY, cs.CeilY
			if sSlopeFloor != nil {
				curFS = cs.FloorY + sSlopeFloor.G.X*(cn.Start.X-sSlopeFloor.Start.X) + sSlopeFloor.G.Y*(cn.Start.Y-sSlopeFloor.Start.Y)
				curFE = cs.FloorY + sSlopeFloor.G.X*(cn.End.X-sSlopeFloor.Start.X) + sSlopeFloor.G.Y*(cn.End.Y-sSlopeFloor.Start.Y)
			}
			if sSlopeCeiling != nil {
				curCS = cs.CeilY + sSlopeCeiling.G.X*(cn.Start.X-sSlopeCeiling.Start.X) + sSlopeCeiling.G.Y*(cn.Start.Y-sSlopeCeiling.Start.Y)
				curCE = cs.CeilY + sSlopeCeiling.G.X*(cn.End.X-sSlopeCeiling.Start.X) + sSlopeCeiling.G.Y*(cn.End.Y-sSlopeCeiling.Start.Y)
			}

			uncoveredWalls = append(uncoveredWalls, &uncoveredWall{
				start:   cn.Start,
				end:     cn.End,
				zBS:     curFS,
				zBE:     curFE,
				zTS:     curCS,
				zTE:     curCE,
				tag:     cn.Tag,
				mat:     mat,
				csId:    cs.Id,
				csTag:   cs.Tag,
				csLight: cs.Light,
				modelId: modelSectorId,
				align:   cn.UVAlign,
			})
			modelSectorId++
		}
	}

	// Risoluzione delle adiacenze.
	for _, segment := range fixSegments {
		if segment.GetNeighbor() != nil {
			continue
		}

		bestDistSq := math.MaxFloat64
		var bestNeighborSegment *Segment

		facesTree.QueryOverlaps(segment, func(object aabb.IObject) bool {
			overlapFace, ok := object.(*Segment)
			if !ok {
				return false
			}
			if overlapFace.GetParent() == segment.GetParent() {
				return false
			}

			start := segment.GetStart()
			end := segment.GetEnd()
			overlapStart := overlapFace.GetStart()
			overlapEnd := overlapFace.GetEnd()

			dx1 := start.X - overlapEnd.X
			dy1 := start.Y - overlapEnd.Y
			dx2 := end.X - overlapStart.X
			dy2 := end.Y - overlapStart.Y
			distSq := dx1*dx1 + dy1*dy1 + dx2*dx2 + dy2*dy2

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborSegment = overlapFace
			}
			return false
		})

		if bestNeighborSegment != nil {
			bestNeighborSegment.SetNeighbor(segment.GetParent())
			segment.SetNeighbor(bestNeighborSegment.GetParent())
		} else {
			segment.SetNeighbor(nil)
		}
	}

	return container, uncoveredWalls
}

// upgrade3d converts a collection of 2D sectors into corresponding 3D volumes by resolving their geometric features.
// It applies floor and ceiling heights, handles segment neighbors, and builds walls based on slope and material properties.
func (r *Compiler2d) upgrade3d(sectors []*Sector) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*Sector]*Volume)

	resolveZ := func(sector *Sector, p geometry.XYZ, baseF, baseC float64, clamp bool) (float64, float64) {
		zF := baseF
		slopeF, slopeC := sector.GetSlopes()
		if slopeF != nil {
			zF = baseF + slopeF.G.X*(p.X-slopeF.Start.X) + slopeF.G.Y*(p.Y-slopeF.Start.Y)
		}
		zC := baseC
		if slopeC != nil {
			zC = baseC + slopeC.G.X*(p.X-slopeC.Start.X) + slopeC.G.Y*(p.Y-slopeC.Start.Y)
		}
		if clamp && zF > zC {
			mid := (zF + zC) * 0.5
			return mid, mid
		}
		return zF, zC
	}

	buildQuad := func(vol3d *Volume, neighbor *Sector, s, e geometry.XY, zBS, zBE, zTS, zTE float64, tag string, material *textures.Material, align *config.UVAlign, t1, t2 float64) {
		v0 := geometry.XYZ{X: s.X, Y: s.Y, Z: zBS}
		v1 := geometry.XYZ{X: e.X, Y: e.Y, Z: zBE}
		v2 := geometry.XYZ{X: e.X, Y: e.Y, Z: zTE}
		v3 := geometry.XYZ{X: s.X, Y: s.Y, Z: zTS}

		face1 := NewFace([3]geometry.XYZ{v0, v1, v2}, tag, material)
		face2 := NewFace([3]geometry.XYZ{v0, v2, v3}, tag, material)
		if align != nil && align.FixUV {
			r.applyWallUV(face1, face2, s, e, align, t1, t2, zBS, zBE, zTS, zTE)
		}
		vol3d.AddFace(face1)
		vol3d.AddFace(face2)
	}

	for _, sector := range sectors {
		id := fmt.Sprintf("%s_3d", sector.GetId())
		vol3d := NewVolumeConcrete(sector.GetModelId(), id, sector.GetTag())

		if sector.light != nil {
			vol3d.SetLight(sector.light)
		}

		segments, segmentCount := sector.GetSegments()
		if segmentCount != 3 {
			fmt.Println("only tringle are supported")
			continue
		}

		curFloorY := sector.GetMinZ()
		curCeilY := sector.GetMaxZ()

		p0 := segments[0].GetStart()
		p1 := segments[1].GetStart()
		p2 := segments[2].GetStart()

		zF0, zC0 := resolveZ(sector, p0, curFloorY, curCeilY, false)
		zF1, zC1 := resolveZ(sector, p1, curFloorY, curCeilY, false)
		zF2, zC2 := resolveZ(sector, p2, curFloorY, curCeilY, false)

		ceilP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zC0}, {X: p1.X, Y: p1.Y, Z: zC1}, {X: p2.X, Y: p2.Y, Z: zC2}}
		ceilFace := NewFace(ceilP, sector.GetTag()+"_ceil", sector.GetMaterialIndex(1))
		r.applyUVAlign(ceilFace, sector.GetCeilAlign(), ceilP)
		vol3d.AddFace(ceilFace)

		floorP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zF0}, {X: p2.X, Y: p2.Y, Z: zF2}, {X: p1.X, Y: p1.Y, Z: zF1}}
		floorFace := NewFace(floorP, sector.GetTag()+"_floor", sector.GetMaterialIndex(0))
		r.applyUVAlign(floorFace, sector.GetFloorAlign(), floorP)
		vol3d.AddFace(floorFace)

		for x := 0; x < segmentCount; x++ {
			seg := segments[x]
			s, e := seg.GetStart(), seg.GetEnd()
			align, t1, t2 := seg.GetUVAlign()

			curFS, curCS := resolveZ(sector, s, curFloorY, curCeilY, false)
			curFE, curCE := resolveZ(sector, e, curFloorY, curCeilY, false)

			neighbor := seg.GetNeighbor()
			if neighbor == nil {
				s2 := geometry.XY{X: s.X, Y: s.Y}
				e2 := geometry.XY{X: e.X, Y: e.Y}
				buildQuad(vol3d, nil, s2, e2, curFS, curFE, curCS, curCE, seg.GetTag(), seg.GetMaterialIndex(1), align, t1, t2)
				continue
			}

			neiFloorY := neighbor.GetMinZ()
			neiCeilY := neighbor.GetMaxZ()

			neiFS, neiCS := resolveZ(neighbor, s, neiFloorY, neiCeilY, false)
			neiFE, neiCE := resolveZ(neighbor, e, neiFloorY, neiCeilY, false)

			tagLower := seg.GetTag() + "_lower"
			tagUpper := seg.GetTag() + "_upper"
			matLower := seg.GetMaterialIndex(2)
			matUpper := seg.GetMaterialIndex(0)

			// Lower wall: intersezione dei pavimenti inclinati.
			diffFS := neiFS - curFS
			diffFE := neiFE - curFE

			if (diffFS > 0 && diffFE < 0) || (diffFS < 0 && diffFE > 0) {
				t := diffFS / (diffFS - diffFE)
				midXY := geometry.XY{X: s.X + t*(e.X-s.X), Y: s.Y + t*(e.Y-s.Y)}
				midZ := curFS + t*(curFE-curFS)

				zLowS1, zHighS1 := curFS, max(curFS, neiFS)
				zLowE1, zHighE1 := midZ, midZ
				subT1 := t1
				subT2 := t1 + t*(t2-t1)
				if zHighS1 > zLowS1 || zHighE1 > zLowE1 {
					buildQuad(vol3d, seg.GetNeighbor(), geometry.XY{X: s.X, Y: s.Y}, midXY, zLowS1, zLowE1, zHighS1, zHighE1, tagLower, matLower, align, subT1, subT2)
				}

				zLowS2, zHighS2 := midZ, midZ
				zLowE2, zHighE2 := curFE, max(curFE, neiFE)
				if zHighS2 > zLowS2 || zHighE2 > zLowE2 {
					buildQuad(vol3d, seg.GetNeighbor(), midXY, geometry.XY{X: e.X, Y: e.Y}, zLowS2, zLowE2, zHighS2, zHighE2, tagLower, matLower, align, subT2, t2)
				}
			} else {
				zLowS, zHighS := curFS, max(curFS, neiFS)
				zLowE, zHighE := curFE, max(curFE, neiFE)
				if zHighS > zLowS || zHighE > zLowE {
					buildQuad(vol3d, seg.GetNeighbor(), geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, zLowS, zLowE, zHighS, zHighE, tagLower, matLower, align, t1, t2)
				}
			}

			// Upper wall: intersezione dei soffitti inclinati.
			diffCS := neiCS - curCS
			diffCE := neiCE - curCE

			if (diffCS > 0 && diffCE < 0) || (diffCS < 0 && diffCE > 0) {
				t := diffCS / (diffCS - diffCE)
				midXY := geometry.XY{X: s.X + t*(e.X-s.X), Y: s.Y + t*(e.Y-s.Y)}
				midZ := curCS + t*(curCE-curCS)

				zTopS1, zBotS1 := curCS, min(curCS, neiCS)
				zTopE1, zBotE1 := midZ, midZ
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					limMid := limS + t*(limE-limS)
					zTopS1 = min(zTopS1, limS)
					zTopE1 = min(zTopE1, limMid)
				}
				subT1 := t1
				subT2 := t1 + t*(t2-t1)
				if zBotS1 < zTopS1 || zBotE1 < zTopE1 {
					buildQuad(vol3d, seg.GetNeighbor(), geometry.XY{X: s.X, Y: s.Y}, midXY, zBotS1, zBotE1, zTopS1, zTopE1, tagUpper, matUpper, align, subT1, subT2)
				}

				zTopS2, zBotS2 := midZ, midZ
				zTopE2, zBotE2 := curCE, min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					limMid := limS + t*(limE-limS)
					zTopS2 = min(zTopS2, limMid)
					zTopE2 = min(zTopE2, limE)
				}
				if zBotS2 < zTopS2 || zBotE2 < zTopE2 {
					buildQuad(vol3d, seg.GetNeighbor(), midXY, geometry.XY{X: e.X, Y: e.Y}, zBotS2, zBotE2, zTopS2, zTopE2, tagUpper, matUpper, align, subT2, t2)
				}
			} else {
				zTopS, zBotS := curCS, min(curCS, neiCS)
				zTopE, zBotE := curCE, min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					zTopS = min(zTopS, limS)
					zTopE = min(zTopE, limE)
				}
				if zBotS < zTopS || zBotE < zTopE {
					buildQuad(vol3d, seg.GetNeighbor(), geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, zBotS, zBotE, zTopS, zTopE, tagUpper, matUpper, align, t1, t2)
				}
			}

			// Middle wall for two-sided portal segments with middle material (e.g. fences, grates, masked/transparent walls)
			matMiddle := seg.GetMaterialIndex(1)
			if matMiddle != nil && matMiddle.CurrentFrame() != nil {
				midBotS := max(curFS, neiFS)
				midBotE := max(curFE, neiFE)
				midTopS := min(curCS, neiCS)
				midTopE := min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					midTopS = min(midTopS, limS)
					midTopE = min(midTopE, limE)
				}
				if midTopS > midBotS || midTopE > midBotE {
					buildQuad(vol3d, seg.GetNeighbor(), geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, midBotS, midBotE, midTopS, midTopE, seg.GetTag()+"_middle", matMiddle, align, t1, t2)
				}
			}
		}

		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		vol3d.SetSector(sector)
		volMap[sector] = vol3d
	}

	return volumes3d
}

// buildUncoveredWalls processes a list of uncoveredWall instances and generates corresponding 3D volumes with optional lighting.
func (r *Compiler2d) buildUncoveredWalls(walls []*uncoveredWall) []*Volume {
	var volumes []*Volume
	for _, uw := range walls {
		if uw.mat == nil || uw.mat.CurrentFrame() == nil {
			continue
		}
		vol3d := NewVolumeConcrete(uw.modelId, fmt.Sprintf("%s_wall_extra_%d", uw.csId, uw.modelId), uw.csTag)
		if uw.csLight != nil {
			light := NewLight()
			centroid := geometry.XY{X: (uw.start.X + uw.end.X) * 0.5, Y: (uw.start.Y + uw.end.Y) * 0.5}
			lightPos := geometry.XYZ{X: centroid.X, Y: centroid.Y, Z: (uw.zBS + uw.zBE + uw.zTS + uw.zTE) * 0.25}
			light.Setup(uw.csLight, lightPos)
			vol3d.SetLight(light)
		}
		v0 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zBS}
		v1 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zBE}
		v2 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zTE}
		v3 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zTS}
		face1 := NewFace([3]geometry.XYZ{v0, v1, v2}, uw.tag, uw.mat)
		face2 := NewFace([3]geometry.XYZ{v0, v2, v3}, uw.tag, uw.mat)
		if uw.align != nil && uw.align.FixUV {
			r.applyWallUV(face1, face2, uw.start, uw.end, uw.align, 0.0, 1.0, uw.zBS, uw.zBE, uw.zTS, uw.zTE)
		}
		vol3d.AddFace(face1)
		vol3d.AddFace(face2)
		vol3d.Rebuild()
		volumes = append(volumes, vol3d)
	}
	return volumes
}

// applyWallUV modifies the UV mapping of two quad faces based on wall alignment settings and quad coordinates.
func (r *Compiler2d) applyWallUV(face1, face2 *Face, s, e geometry.XY, align *config.UVAlign, t1, t2 float64, zBS, zBE, zTS, zTE float64) {
	if align == nil || !align.FixUV {
		return
	}
	mat := face1.GetMaterial()
	if mat == nil {
		return
	}
	tex := mat.CurrentFrame()
	if tex == nil {
		return
	}
	w, h := tex.GetSizeScaled()
	if w <= 0 || h <= 0 {
		return
	}

	wallLen := align.Length
	if wallLen <= 0 {
		wallLen = math.Hypot(e.X-s.X, e.Y-s.Y)
	}

	d1 := t1 * wallLen
	d2 := t2 * wallLen
	minD := math.Min(d1, d2)
	maxD := math.Max(d1, d2)

	uStart := (minD + align.Panning.X) / w
	uEnd := (maxD + align.Panning.X) / w
	if align.FlipX {
		uStart, uEnd = uEnd, uStart
	}

	vBS := (zBS + align.Panning.Y) / h
	vBE := (zBE + align.Panning.Y) / h
	vTS := (zTS + align.Panning.Y) / h
	vTE := (zTE + align.Panning.Y) / h
	if align.FlipY {
		vBS, vTS = vTS, vBS
		vBE, vTE = vTE, vBE
	}

	face1.SetUV(uStart, vBS, uEnd, vBE, uEnd, vTE)
	face1.LockUV(true)

	face2.SetUV(uStart, vBS, uEnd, vTE, uStart, vTS)
	face2.LockUV(true)
}

// applyUVAlign modifies the UV mapping of the provided face based on alignment settings and vertex positions of the triangle.
func (r *Compiler2d) applyUVAlign(face *Face, align *config.UVAlign, tri [3]geometry.XYZ) {
	if align == nil || !align.FixUV {
		return
	}
	mat := face.GetMaterial()
	if mat == nil {
		return
	}
	tex := mat.CurrentFrame()
	if tex == nil {
		return
	}
	w, h := tex.GetSizeScaled()
	if w <= 0 || h <= 0 {
		return
	}

	var tx, ty, nx, ny float64
	if align.AlignWall {
		dx := align.End.X - align.Start.X
		dy := align.End.Y - align.Start.Y
		length := math.Hypot(dx, dy)
		if length < 1e-6 {
			dx, dy, length = 1, 0, 1
		}
		tx = dx / length
		ty = dy / length
		nx = -ty
		ny = tx
	}

	var uvs [3][2]float64
	for i := 0; i < 3; i++ {
		var u, v float64
		if align.AlignWall {
			relX := tri[i].X - align.Start.X
			relY := tri[i].Y - align.Start.Y
			u = relX*tx + relY*ty
			v = relX*nx + relY*ny
		} else {
			u = tri[i].X
			v = tri[i].Y
		}
		if align.SwapXY {
			u, v = v, u
		}
		if align.FlipX {
			u = -u
		}
		if align.FlipY {
			v = -v
		}
		if align.Panning.X != 0 {
			u += align.Panning.X
		}
		if align.Panning.Y != 0 {
			v += align.Panning.Y
		}
		uvs[i][0] = u / w
		uvs[i][1] = v / h
	}
	face.SetUV(uvs[0][0], uvs[0][1], uvs[1][0], uvs[1][1], uvs[2][0], uvs[2][1])
	face.LockUV(true)
}

// updateAdjDependencies resolves dependency order among sectors and reorders them for adjacency processing.
func (r *Compiler2d) updateAdjDependencies(configSectors []*config.Sector) []*config.Sector {
	n := len(configSectors)
	if n == 0 {
		return configSectors
	}
	idToOrigIdx := make(map[string]int, n)
	for i, cs := range configSectors {
		idToOrigIdx[cs.Id] = i
	}
	adj := make(map[int][]int)
	inDegree := make([]int, n)
	for lowIdx, cs := range configSectors {
		for _, upId := range cs.DependsOn {
			if upIdx, ok := idToOrigIdx[upId]; ok && upIdx != lowIdx {
				exists := false
				for _, existing := range adj[upIdx] {
					if existing == lowIdx {
						exists = true
						break
					}
				}
				if !exists {
					adj[upIdx] = append(adj[upIdx], lowIdx)
					inDegree[lowIdx]++
				}
			}
		}
	}
	result := make([]*config.Sector, 0, n)
	ready := make([]bool, n)
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			ready[i] = true
		}
	}
	for len(result) < n {
		nextIdx := -1
		for i := 0; i < n; i++ {
			if ready[i] {
				nextIdx = i
				break
			}
		}
		if nextIdx == -1 {
			// In case of a cycle, append any remaining sectors preserving their relative order.
			for i := 0; i < n; i++ {
				if inDegree[i] > 0 {
					result = append(result, configSectors[i])
				}
			}
			break
		}
		ready[nextIdx] = false
		inDegree[nextIdx] = -1
		result = append(result, configSectors[nextIdx])
		for _, child := range adj[nextIdx] {
			inDegree[child]--
			if inDegree[child] == 0 {
				ready[child] = true
			}
		}
	}
	return result
}

// compileLights2d processes 2D sector data to generate and configure light objects based on sector connectivity and attributes.
func (r *Compiler2d) compileLights2d(locator *Sectors, computeCenter bool) []*Light {
	// Unifies adjacent volume that belong to the same macroscopic sector.
	visited := make(map[string]bool)
	var out []*Light

	addLight := func(z *Sector, pos geometry.XYZ, intensity float64, falloff float64, kind config.LightKind, r, g, b float64, style []float64) {
		lightPos := geometry.XYZ{X: pos.X, Y: pos.Y, Z: z.GetMinZ() + z.GetMaxZ()}
		cl := config.NewConfigLight(pos, intensity, kind, falloff)
		cl.R = r
		cl.G = g
		cl.B = b
		cl.Style = style
		light := NewLight()
		light.Setup(cl, lightPos)
		out = append(out, light)
	}

	for idx, sect := range locator.GetSectors() {
		if visited[sect.GetId()] {
			continue
		}
		// We use a Flood Fill algorithm to find all connected sectors
		var areaSectors []*Sector
		queue := []*Sector{sect}
		visited[sect.GetId()] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			areaSectors = append(areaSectors, curr)
			// Check neighbors of this sector
			faces, faceCount := curr.GetSegments()
			for x := 0; x < faceCount; x++ {
				face := faces[x]
				if n := face.GetNeighbor(); n != nil {
					if !visited[n.GetId()] {
						// "Same Area" condition: adjacent and with same heights/lights
						if n.GetMaxZ() == curr.GetMaxZ() && n.GetMinZ() == curr.GetMinZ() && n.light.intensity == curr.light.intensity {
							visited[n.GetId()] = true
							queue = append(queue, n)
						}
					}
				}
			}
		}

		// If the area is composed of multiple polygons, we calculate a global centroid
		if len(areaSectors) > 1 {
			if !computeCenter {
				for _, s := range areaSectors {
					addLight(s, s.GetCentroid(), s.light.intensity, s.light.falloff, s.light.kind, s.light.r, s.light.g, s.light.b, s.light.style)
				}
			} else {
				var sumX, sumY, totalArea float64
				var intensity, falloff float64
				for _, s := range areaSectors {
					// Calculate tri area (cross product)
					area := 0.0
					segments, segmentCount := s.GetSegments()
					for x := 0; x < segmentCount; x++ {
						seg := segments[x]
						start := seg.GetStart()
						end := seg.GetEnd()
						x0, y0 := start.X, start.Y
						x1, y1 := end.X, end.Y
						area += (x0 * y1) - (x1 * y0)
					}
					area = math.Abs(area * 0.5)
					sumX += s.light.pos.X * area
					sumY += s.light.pos.Y * area
					totalArea += area
					intensity += s.light.intensity
					falloff += s.light.falloff
				}
				if totalArea == 0 {
					fmt.Println("total area is zero")
					continue
				}
				intensity /= float64(len(areaSectors))
				falloff /= float64(len(areaSectors))
				gc := geometry.XYZ{X: sumX / totalArea, Y: sumY / totalArea, Z: 0.0}
				// Legacy: assign the new global light center to all area fragments
				for _, s := range areaSectors {
					s.light.pos.X = gc.X
					s.light.pos.Y = gc.Y
				}
				first := areaSectors[0]
				cVolume := locator.QueryPoint(first.light.pos.X, first.light.pos.Y)
				if cVolume == nil {
					cVolume = first
					fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f, z:%f)\n", idx, first.light.pos.X, first.light.pos.Y, first.light.pos.Z)
				}
				light := cVolume.light
				addLight(cVolume, gc, intensity, falloff, light.kind, light.r, light.g, light.b, light.style)
			}
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			light := first.light
			addLight(first, first.GetCentroid(), light.intensity, light.falloff, light.kind, light.r, light.g, light.b, light.style)
		}
	}
	return out
}
