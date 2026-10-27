package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/model/sectors"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// CreateFaces generates two triangular faces representing a wall surface based on the SegmentUncovered's geometry and properties.
func CreateFaces(uw *sectors.SegmentUncovered) []*Face {
	v0, v1, v2, v3 := uw.CreateQuad()
	face1 := NewFace([3]geometry.XYZ{v0, v1, v2}, uw.GetTag(), uw.GetMat())
	face2 := NewFace([3]geometry.XYZ{v0, v2, v3}, uw.GetTag(), uw.GetMat())
	face1.ApplyWallUV(uw.GetStart(), uw.GetEnd(), uw.GetAlign(), 0.0, 1.0, uw.GetZBS(), uw.GetZBE(), uw.GetZTS(), uw.GetZTE(), true)
	face2.ApplyWallUV(uw.GetStart(), uw.GetEnd(), uw.GetAlign(), 0.0, 1.0, uw.GetZBS(), uw.GetZBE(), uw.GetZTS(), uw.GetZTE(), false)
	return []*Face{face1, face2}
}

// Compiler2d is responsible for compiling 2D sector data, vertices, and materials into structured volume representations.
type Compiler2d struct {
	cfg       *config.Root
	materials *Materials
}

// NewCompiler2d creates and returns a new instance of Compiler2d, used for compiling 2D game sectors and related data.
func NewCompiler2d(cfg *config.Root, materials *Materials) *Compiler2d {
	return &Compiler2d{
		cfg:       cfg,
		materials: materials,
	}
}

// Compile constructs 3D volumes and lights from 2D sector data, resolving player and thing positions using sector geometry.
func (r *Compiler2d) Compile() ([]*Volume, []*lights.Light, error) {
	var allVolumes []*Volume
	if len(r.cfg.Sectors) == 0 {
		return nil, nil, nil
	}

	sectors2d, segmentsUncovered := r.compile2d()
	if len(sectors2d) == 0 {
		return nil, nil, fmt.Errorf("no 2D volumes compiled")
	}
	locator := sectors.NewSectors(sectors2d, false)
	locator.Setup()
	//player Z
	pv := locator.QueryPoint(r.cfg.Player.Position.X, r.cfg.Player.Position.Y)
	if pv == nil {
		return nil, nil, fmt.Errorf("can't find 2d player location at X: %f Y: %f", r.cfg.Player.Position.X, r.cfg.Player.Position.Y)
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
	allLights := r.compileLights2d(locator, true)

	allVolumes = append(allVolumes, r.upgrade3d(sectors2d)...)
	allVolumes = append(allVolumes, r.upgrade3dSegmentsUncovered(segmentsUncovered)...)

	return allVolumes, allLights, nil
}

// compile2d processes 2D geometry and sector configurations to construct 2D sectors and identifies uncovered walls.
func (r *Compiler2d) compile2d() ([]*sectors.Sector, []*sectors.SegmentUncovered) {
	vertices, css, anim := r.cfg.Vertices, r.cfg.Sectors, r.materials
	const epsilon = 0.01
	modelSectorId := 0
	var container []*sectors.Sector
	var segmentsUncovered []*sectors.SegmentUncovered
	var fixSegments []*sectors.Segment
	facesTree := aabb.NewTree(1024, epsilon)
	emptyAnim := anim.GetEmptyMaterial()

	css = r.updateAdjDependencies(css)

	ve := sectors.NewSectorsEdges(epsilon)
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

		var sSlopeFloor *sectors.Slope = nil
		var sSlopeCeiling *sectors.Slope = nil
		if slSeg := cs.GetSlopeFloor(); slSeg != nil {
			sl := cs.SlopeFloor
			sSlopeFloor = sectors.NewSlope(sl.G, sl.Gradient, slSeg.Start, slSeg.End)
		}
		if slSeg := cs.GetSlopeCeiling(); slSeg != nil {
			sl := cs.SlopeCeiling
			sSlopeCeiling = sectors.NewSlope(sl.G, sl.Gradient, slSeg.Start, slSeg.End)
		}

		segmentsCovered := make([]bool, len(cs.Segments))
		for _, triangles := range triContainer {
			for _, tri := range triangles {
				if len(tri) != 3 {
					fmt.Println("wrong tri", tri)
					continue
				}

				volumeMaterials := []*textures.Material{anim.GetMaterial(cs.Floor), anim.GetMaterial(cs.Ceil)}

				sector := sectors.NewSector(modelSectorId, cs.Id, cs.FloorY, cs.CeilY, volumeMaterials, cs.Tag)
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
							segmentsCovered[cnIdx] = true
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
					seg := sectors.NewSegment(nil, start, end, tag, segMaterials)
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
				sector.CreateLight(cs.Light, cs.FloorY+cs.CeilY)
				sector.Rebuild()
				segments, segmentCount := sector.GetSegments()
				for x := 0; x < segmentCount; x++ {
					facesTree.InsertObject(segments[x])
				}
				container = append(container, sector)
			}
		}

		//Uncovered Segments Handling
		for cnIdx, cn := range cs.Segments {
			if cn == nil || segmentsCovered[cnIdx] {
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
			if mat == nil || mat.HeadFrame() == nil {
				continue
			}
			curFS, curFE := cs.FloorY, cs.FloorY
			curCS, curCE := cs.CeilY, cs.CeilY
			if sSlopeFloor != nil {
				curFS = sSlopeFloor.ComputeZ(cs.FloorY, cn.Start.X, cn.Start.Y)
				curFE = sSlopeFloor.ComputeZ(cs.FloorY, cn.End.X, cn.End.Y)
			}
			if sSlopeCeiling != nil {
				curCS = sSlopeCeiling.ComputeZ(cs.CeilY, cn.Start.X, cn.Start.Y)
				curCE = sSlopeCeiling.ComputeZ(cs.CeilY, cn.End.X, cn.End.Y)
			}
			uw := sectors.NewSegmentUncovered(cn.Start, cn.End, curFS, curFE, curCS, curCE, cn.Tag, mat, cs.Id, cs.Tag, cs.Light, modelSectorId, cn.UVAlign)
			segmentsUncovered = append(segmentsUncovered, uw)
			modelSectorId++
		}
	}

	// Risoluzione delle adiacenze.
	for _, segment := range fixSegments {
		if segment.GetNeighbor() != nil {
			continue
		}

		bestDistSq := math.MaxFloat64
		var bestNeighborSegment *sectors.Segment

		facesTree.QueryOverlaps(segment, func(object aabb.IObject) bool {
			overlapFace, ok := object.(*sectors.Segment)
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

	return container, segmentsUncovered
}

// upgrade3d converts a collection of 2D sectors into corresponding 3D volumes by resolving their geometric features.
// It applies floor and ceiling heights, handles segment neighbors, and builds walls based on slope and material properties.
func (r *Compiler2d) upgrade3d(sectors2d []*sectors.Sector) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*sectors.Sector]*Volume)

	for _, sector := range sectors2d {
		id := fmt.Sprintf("%s_3d", sector.GetId())
		vol3d := NewVolumeConcrete(sector.GetModelId(), id, sector.GetTag())

		if cLight := sector.GetLight(); cLight != nil {
			light := lights.NewLight()
			light.Setup(cLight, cLight.Pos)
			vol3d.SetLight(light)
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

		zF0, zC0 := sector.ResolveZ(p0, curFloorY, curCeilY, false)
		zF1, zC1 := sector.ResolveZ(p1, curFloorY, curCeilY, false)
		zF2, zC2 := sector.ResolveZ(p2, curFloorY, curCeilY, false)

		ceilP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zC0}, {X: p1.X, Y: p1.Y, Z: zC1}, {X: p2.X, Y: p2.Y, Z: zC2}}
		ceilFace := NewFace(ceilP, sector.GetTag()+"_ceil", sector.GetMaterialIndex(1))
		ceilFace.ApplyUVAlign(sector.GetCeilAlign(), ceilP)
		vol3d.AddFace(ceilFace)

		floorP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zF0}, {X: p2.X, Y: p2.Y, Z: zF2}, {X: p1.X, Y: p1.Y, Z: zF1}}
		floorFace := NewFace(floorP, sector.GetTag()+"_floor", sector.GetMaterialIndex(0))
		floorFace.ApplyUVAlign(sector.GetFloorAlign(), floorP)
		vol3d.AddFace(floorFace)

		for x := 0; x < segmentCount; x++ {
			seg := segments[x]
			s, e := seg.GetStart(), seg.GetEnd()
			align, t1, t2 := seg.GetUVAlign()

			curFS, curCS := sector.ResolveZ(s, curFloorY, curCeilY, false)
			curFE, curCE := sector.ResolveZ(e, curFloorY, curCeilY, false)

			neighbor := seg.GetNeighbor()
			if neighbor == nil {
				s2 := geometry.XY{X: s.X, Y: s.Y}
				e2 := geometry.XY{X: e.X, Y: e.Y}
				vol3d.BuildQuad(s2, e2, curFS, curFE, curCS, curCE, seg.GetTag(), seg.GetMaterialIndex(1), align, t1, t2)
				continue
			}

			neiFloorY := neighbor.GetMinZ()
			neiCeilY := neighbor.GetMaxZ()

			neiFS, neiCS := neighbor.ResolveZ(s, neiFloorY, neiCeilY, false)
			neiFE, neiCE := neighbor.ResolveZ(e, neiFloorY, neiCeilY, false)

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
					vol3d.BuildQuad(geometry.XY{X: s.X, Y: s.Y}, midXY, zLowS1, zLowE1, zHighS1, zHighE1, tagLower, matLower, align, subT1, subT2)
				}

				zLowS2, zHighS2 := midZ, midZ
				zLowE2, zHighE2 := curFE, max(curFE, neiFE)
				if zHighS2 > zLowS2 || zHighE2 > zLowE2 {
					vol3d.BuildQuad(midXY, geometry.XY{X: e.X, Y: e.Y}, zLowS2, zLowE2, zHighS2, zHighE2, tagLower, matLower, align, subT2, t2)
				}
			} else {
				zLowS, zHighS := curFS, max(curFS, neiFS)
				zLowE, zHighE := curFE, max(curFE, neiFE)
				if zHighS > zLowS || zHighE > zLowE {
					vol3d.BuildQuad(geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, zLowS, zLowE, zHighS, zHighE, tagLower, matLower, align, t1, t2)
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
					vol3d.BuildQuad(geometry.XY{X: s.X, Y: s.Y}, midXY, zBotS1, zBotE1, zTopS1, zTopE1, tagUpper, matUpper, align, subT1, subT2)
				}

				zTopS2, zBotS2 := midZ, midZ
				zTopE2, zBotE2 := curCE, min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					limMid := limS + t*(limE-limS)
					zTopS2 = min(zTopS2, limMid)
					zTopE2 = min(zTopE2, limE)
				}
				if zBotS2 < zTopS2 || zBotE2 < zTopE2 {
					vol3d.BuildQuad(midXY, geometry.XY{X: e.X, Y: e.Y}, zBotS2, zBotE2, zTopS2, zTopE2, tagUpper, matUpper, align, subT2, t2)
				}
			} else {
				zTopS, zBotS := curCS, min(curCS, neiCS)
				zTopE, zBotE := curCE, min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					zTopS = min(zTopS, limS)
					zTopE = min(zTopE, limE)
				}
				if zBotS < zTopS || zBotE < zTopE {
					vol3d.BuildQuad(geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, zBotS, zBotE, zTopS, zTopE, tagUpper, matUpper, align, t1, t2)
				}
			}

			// Middle wall for two-sided portal segments with middle material (e.g. fences, grates, masked/transparent walls)
			matMiddle := seg.GetMaterialIndex(1)
			if matMiddle != nil && matMiddle.HeadFrame() != nil {
				midBotS := max(curFS, neiFS)
				midBotE := max(curFE, neiFE)
				midTopS := min(curCS, neiCS)
				midTopE := min(curCE, neiCE)
				if limS, limE, ok := seg.GetUpperLimitZ(); ok {
					midTopS = min(midTopS, limS)
					midTopE = min(midTopE, limE)
				}
				if midTopS > midBotS || midTopE > midBotE {
					vol3d.BuildQuad(geometry.XY{X: s.X, Y: s.Y}, geometry.XY{X: e.X, Y: e.Y}, midBotS, midBotE, midTopS, midTopE, seg.GetTag()+"_middle", matMiddle, align, t1, t2)
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

// upgrade3dSegmentUncovered converts uncovered wall segments into 3D volumes by generating faces and applying lighting.
func (r *Compiler2d) upgrade3dSegmentsUncovered(segments []*sectors.SegmentUncovered) []*Volume {
	var volumes []*Volume
	for _, su := range segments {
		if su.GetMat() == nil || su.GetMat().HeadFrame() == nil {
			continue
		}
		vol3d := NewVolumeConcrete(su.GetModelId(), fmt.Sprintf("%s_wall_extra_%d", su.GetCsId(), su.GetModelId()), su.GetCsTag())
		if su.GetCsLight() != nil {
			light := lights.NewLight()
			light.Setup(su.GetCsLight(), su.GetLightPos())
			vol3d.SetLight(light)
		}
		faces := CreateFaces(su)
		for _, face := range faces {
			vol3d.AddFace(face)
		}
		vol3d.Rebuild()
		volumes = append(volumes, vol3d)
	}
	return volumes
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
func (r *Compiler2d) compileLights2d(locator *sectors.Sectors, computeCenter bool) []*lights.Light {
	// Unifies adjacent volume that belong to the same macroscopic sector.
	visited := make(map[string]bool)
	var out []*lights.Light

	addLight := func(z *sectors.Sector, pos geometry.XYZ, intensity float64, falloff float64, kind config.LightKind, r, g, b float64, style []float64) {
		lightPos := geometry.XYZ{X: pos.X, Y: pos.Y, Z: z.GetMinZ() + z.GetMaxZ()}
		cl := config.NewConfigLight(pos, intensity, kind, falloff)
		cl.R = r
		cl.G = g
		cl.B = b
		cl.Style = style
		light := lights.NewLight()
		light.Setup(cl, lightPos)
		out = append(out, light)
	}

	for idx, sect := range locator.GetSectors() {
		if visited[sect.GetId()] {
			continue
		}
		// We use a Flood Fill algorithm to find all connected sectors
		var areaSectors []*sectors.Sector
		queue := []*sectors.Sector{sect}
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
						if n.GetMaxZ() == curr.GetMaxZ() && n.GetMinZ() == curr.GetMinZ() && n.GetLight().Intensity == curr.GetLight().Intensity {
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
					cLight := s.GetLight()
					addLight(s, s.GetCentroid(), cLight.Intensity, cLight.Falloff, cLight.Kind, cLight.R, cLight.G, cLight.B, cLight.Style)
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
					cLight := s.GetLight()
					sumX += cLight.Pos.X * area
					sumY += cLight.Pos.Y * area
					totalArea += area
					intensity += cLight.Intensity
					falloff += cLight.Falloff
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
					cLight := s.GetLight()
					cLight.Pos.X = gc.X
					cLight.Pos.Y = gc.Y
				}
				first := areaSectors[0]
				fLight := first.GetLight()
				cVolume := locator.QueryPoint(fLight.Pos.X, fLight.Pos.Y)
				if cVolume == nil {
					cVolume = first
					fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f, z:%f)\n", idx, fLight.Pos.X, fLight.Pos.Y, fLight.Pos.Z)
				}
				light := cVolume.GetLight()
				addLight(cVolume, gc, intensity, falloff, light.Kind, light.R, light.G, light.B, light.Style)
			}
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			fLight := first.GetLight()
			addLight(first, first.GetCentroid(), fLight.Intensity, fLight.Falloff, fLight.Kind, fLight.R, fLight.G, fLight.B, fLight.Style)
		}
	}
	return out
}
