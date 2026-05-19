package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Compiler represents a core game engine component for managing world, game objects, player interactions, and things.
type Compiler struct {
	gScale      geometry.XYZ
	volumes     *Volumes
	player      *ThingPlayer
	lights      *Lights
	things      *Things
	calibration *Calibration
}

// NewCompiler initializes and returns a new instance of Compiler with default-nil-initialized fields.
func NewCompiler() *Compiler {
	return &Compiler{
		volumes: nil,
		player:  nil,
		things:  nil,
		lights:  nil,
	}
}

// Compile initializes and processes game data from the provided configuration, returning an error if compilation fails.
func (r *Compiler) Compile(cfg *config.Root) error {
	r.gScale = cfg.ScaleFactor
	if r.gScale.X == 0 {
		r.gScale.X = 1
	}
	if r.gScale.Y == 0 {
		r.gScale.Y = 1
	}
	if r.gScale.Z == 0 {
		r.gScale.Z = 1
	}

	full3d := cfg.Calibration.Full3d

	cfg.Scale(r.gScale)
	materials := NewMaterials(cfg.GetTextures())
	r.lights = NewLights()
	var sectors []*Sector

	if len(cfg.Sectors) > 0 {
		sectors = r.compile2d(cfg.Vertices, cfg.Sectors, materials)
		if len(sectors) == 0 {
			return fmt.Errorf("no 2D volumes compiled")
		}
		locator := NewSectors(sectors, false)
		locator.Setup()
		//player Z
		pv := locator.LocateSector(cfg.Player.Position.X, cfg.Player.Position.Y)
		if pv == nil {
			return fmt.Errorf("can't find 2d player location at X: %f Y: %f", cfg.Player.Position.X, cfg.Player.Position.Y)
		}
		cfg.Player.Position.Z = pv.GetMinZ()
		//things Z
		for idx := range cfg.Things {
			tx, ty := cfg.Things[idx].Position.X, cfg.Things[idx].Position.Y
			tv := locator.LocateSector(tx, ty)
			if tv == nil {
				fmt.Println("can't find thing location at", tx, ty)
				continue
			}
			cfg.Things[idx].Position.Z = tv.GetMinZ()
		}
		//light 2d
		r.lights.AddLights(r.compileLights2d(locator, true))
	}

	var allVolumes []*Volume
	//if cfg.Calibration.Full3d {
	if len(sectors) > 0 {
		allVolumes = append(allVolumes, r.upgrade3d(sectors)...)
	}
	if len(cfg.Volumes) > 0 {
		allVolumes = append(allVolumes, r.compile3d(cfg.Volumes, materials)...)
	}
	//} else {
	//	allVolumes = append(allVolumes, container2d...)
	//}

	r.volumes = NewVolumes(allVolumes)
	r.volumes.Setup()

	r.lights.AddLights(r.compileLights(cfg.Lights))
	r.things = NewThings(full3d, r.gScale, 10, cfg.Things, r.volumes, materials)
	r.player = NewThingPlayer(r.things, cfg.Player, r.volumes, false)
	if r.player == nil {
		return fmt.Errorf("player not found")
	}
	r.things.SetPlayer(r.player)
	r.calibration = NewCalibration(cfg.Calibration, r.volumes)
	fmt.Printf("Scan complete world: %d\n", r.volumes.Len())
	return nil
}

// GetThings returns the Things instance managed by the Compiler.
func (r *Compiler) GetThings() *Things {
	return r.things
}

// GetVolumes retrieves the Volumes instance associated with the current Compiler object.
func (r *Compiler) GetVolumes() *Volumes {
	return r.volumes
}

// GetPlayer returns the player object associated with the compiler instance.
func (r *Compiler) GetPlayer() *ThingPlayer {
	return r.player
}

// GetLights retrieves the list of Light objects managed by the Compiler.
func (r *Compiler) GetLights() *Lights {
	return r.lights
}

// GetCalibration returns the Calibration instance associated with the Compiler.
func (r *Compiler) GetCalibration() *Calibration {
	return r.calibration
}

type Slope struct {
	Nx       float64
	Ny       float64
	Gradient float64
	Start    geometry.XY
	End      geometry.XY
}

var _slopedCeiling = make(map[*Sector]*Slope)
var _slopedFloor = make(map[*Sector]*Slope)

func (r *Compiler) compile2d(vertices geometry.Polygon, css []*config.Sector, anim *Materials) []*Sector {
	const epsilon = 0.01
	modelSectorId := 0
	var container []*Sector
	var fixSegments []*Segment
	facesTree := physics.NewAABBTree(1024, epsilon)
	emptyAnim := anim.GetMaterial(nil)

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

		// 1. PRE-CALCOLO DELLE PENDENZE PER L'INTERO SETTORE
		var sectorSlopeFloor *Slope
		var sectorSlopeCeiling *Slope

		// Cerca i segmenti pivot PRIMA di iterare sui triangoli
		for _, cn := range cs.Segments {
			if cn.SlopedFloorRef {
				if nX, nY, err := cn.ComputeNormal(cs.IsCCW()); err == nil {
					sectorSlopeFloor = &Slope{Nx: nX, Ny: nY, Gradient: cs.SlopedFloorGradient, Start: cn.Start, End: cn.End}
				}
			}
			if cn.SlopedCeilingRef {
				if nX, nY, err := cn.ComputeNormal(cs.IsCCW()); err == nil {
					sectorSlopeCeiling = &Slope{Nx: nX, Ny: nY, Gradient: cs.SlopedCeilingGradient, Start: cn.Start, End: cn.End}
				}
			}
		}

		for _, triangles := range triContainer {
			for _, tri := range triangles {
				if len(tri) != 3 {
					fmt.Println("wrong tri", tri)
					continue
				}
				volumeMaterials := []*textures.Material{anim.GetMaterial(cs.Floor), anim.GetMaterial(cs.Ceil)}
				sector := NewSector(modelSectorId, cs.Id, cs.FloorY, cs.CeilY, volumeMaterials, cs.Tag)

				if sectorSlopeFloor != nil {
					_slopedFloor[sector] = sectorSlopeFloor
				}
				if sectorSlopeCeiling != nil {
					_slopedCeiling[sector] = sectorSlopeCeiling
				}

				modelSectorId++
				// Maintains consistent Winding Order for ContainsPoint
				if mathematic.PointInLineDirectionF(tri[2].X, tri[2].Y, tri[0].X, tri[0].Y, tri[1].X, tri[1].Y) < 0 {
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

					// EXACT topological match
					for _, cn := range cs.Segments {
						found := false
						if p1 == cn.Start && p2 == cn.End {
							found = true
						} else if p1 == cn.End && p2 == cn.Start {
							found = true
						} else if geometry.IsSegmentSubset(p1, p2, cn.Start, cn.End) {
							found = true
						} else if geometry.IsSegmentSubset(p2, p1, cn.End, cn.Start) {
							found = true
						}
						if found {
							isWall = cn.Kind == config.SegmentWall
							tag = cn.Tag
							upper = anim.GetMaterial(cn.Upper)
							middle = anim.GetMaterial(cn.Middle)
							lower = anim.GetMaterial(cn.Lower)
							break
						}
					}

					segMaterials := []*textures.Material{upper, middle, lower}
					seg := NewSegment(nil, start, end, tag, segMaterials)
					sector.AddSegment(seg)
					sector.AddTag(tag)
					if !isWall {
						fixSegments = append(fixSegments, seg)
					}
				}
				light := NewLight()
				sector.SetLight(light)
				if cs.Light != nil {
					centroid := sector.GetCentroid2d()
					lightPos := geometry.XYZ{X: centroid.X, Y: centroid.Y, Z: cs.FloorY + cs.CeilY}
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
	}

	// Adjacency resolution
	for _, segment := range fixSegments {
		if segment.GetNeighbor() != nil { // already linked
			continue
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborSegment *Segment
		facesTree.QueryOverlaps(segment, func(object physics.IAABB) bool {
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
			distSq := (dx1 * dx1) + (dy1 * dy1) + (dx2 * dx2) + (dy2 * dy2)
			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborSegment = overlapFace
			}
			return false
		})
		if bestNeighborSegment != nil {
			// Bidirectional link (O(N/2))
			bestNeighborSegment.SetNeighbor(segment.GetParent())
			segment.SetNeighbor(bestNeighborSegment.GetParent())
		} else {
			segment.SetNeighbor(nil)
		}
	}
	return container
}

// upgrade3d converts a slice of 2D volumes into 3D volumes by extruding geometry and resolving slopes and adjacency.
func (r *Compiler) upgrade3d(sectors []*Sector) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*Sector]*Volume)

	// resolveZ calcola la Z per pavimento e soffitto nel punto (X,Y)
	resolveZ := func(segment *Sector, p geometry.XYZ, baseF, baseC float64, clamp bool) (float64, float64) {
		zF := baseF
		if slopeF, ok := _slopedFloor[segment]; ok {
			slopeX, slopeY := slopeF.Nx*slopeF.Gradient, slopeF.Ny*slopeF.Gradient
			slopeZ := baseF - (slopeX * slopeF.Start.X) - (slopeY * slopeF.Start.Y)
			zF = slopeZ + (slopeX * p.X) + (slopeY * p.Y)
		}
		zC := baseC
		if slopeC, ok := _slopedCeiling[segment]; ok {
			slopeX, slopeY := slopeC.Nx*slopeC.Gradient, slopeC.Ny*slopeC.Gradient
			slopeZ := baseC - (slopeX * slopeC.Start.X) - (slopeY * slopeC.Start.Y)
			zC = slopeZ + (slopeX * p.X) + (slopeY * p.Y)
		}
		if clamp {
			// Clamping per evitare che il pavimento superi il soffitto
			if zF > zC {
				mid := (zF + zC) * 0.5
				return mid, mid
			}
		}
		return zF, zC
	}

	// buildQuad ora accetta s (Start) ed e (End) come XY per permettere lo split parametrico
	buildQuad := func(vol3d *Volume, neighbor *Sector, s, e geometry.XY, zBS, zBE, zTS, zTE float64, tag string, material *textures.Material) {
		v0 := geometry.XYZ{X: s.X, Y: s.Y, Z: zBS} // Bottom-Start
		v1 := geometry.XYZ{X: e.X, Y: e.Y, Z: zBE} // Bottom-End
		v2 := geometry.XYZ{X: e.X, Y: e.Y, Z: zTE} // Top-End
		v3 := geometry.XYZ{X: s.X, Y: s.Y, Z: zTS} // Top-Start

		faceT1 := NewFace([3]geometry.XYZ{v0, v1, v2}, tag, material)
		faceT2 := NewFace([3]geometry.XYZ{v0, v2, v3}, tag, material)
		vol3d.AddFace(faceT1)
		vol3d.AddFace(faceT2)
	}

	for _, sector := range sectors {
		id := fmt.Sprintf("%s_3d", sector.GetId())
		vol3d := NewVolumeStatic(sector.GetModelId(), id, sector.GetTag())
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

		p0, p1, p2 := segments[0].GetStart(), segments[1].GetStart(), segments[2].GetStart()

		zF0, zC0 := resolveZ(sector, p0, curFloorY, curCeilY, false)
		zF1, zC1 := resolveZ(sector, p1, curFloorY, curCeilY, false)
		zF2, zC2 := resolveZ(sector, p2, curFloorY, curCeilY, false)

		ceilP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zC0}, {X: p1.X, Y: p1.Y, Z: zC1}, {X: p2.X, Y: p2.Y, Z: zC2}}
		vol3d.AddFace(NewFace(ceilP, sector.GetTag()+"_ceil", sector.GetMaterialIndex(1)))

		floorP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: zF0}, {X: p2.X, Y: p2.Y, Z: zF2}, {X: p1.X, Y: p1.Y, Z: zF1}}
		vol3d.AddFace(NewFace(floorP, sector.GetTag()+"_floor", sector.GetMaterialIndex(0)))

		for x := 0; x < segmentCount; x++ {
			seg := segments[x]
			s, e := seg.GetStart(), seg.GetEnd()

			curFS, curCS := resolveZ(sector, s, curFloorY, curCeilY, false)
			curFE, curCE := resolveZ(sector, e, curFloorY, curCeilY, false)
			neighbor := seg.GetNeighbor()

			if neighbor == nil {
				s2 := geometry.XY{X: s.X, Y: s.Y}
				e2 := geometry.XY{X: e.X, Y: e.Y}
				buildQuad(vol3d, nil, s2, e2, curFS, curFE, curCS, curCE, seg.GetTag(), seg.GetMaterialIndex(1))
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

			// ==========================================
			// LOWER WALL: Scontro tra pavimenti inclinati
			// ==========================================
			diffFS := neiFS - curFS
			diffFE := neiFE - curFE

			// Rilevamento Crossover (i piani si incrociano lungo il segmento)
			if (diffFS > 0 && diffFE < 0) || (diffFS < 0 && diffFE > 0) {
				t := diffFS / (diffFS - diffFE)
				midXY := geometry.XY{X: s.X + t*(e.X-s.X), Y: s.Y + t*(e.Y-s.Y)}
				midZ := curFS + t*(curFE-curFS) // Al punto mid, curZ == neiZ

				// Segmento 1: Da Start a Mid
				zLowS1, zHighS1 := curFS, math.Max(curFS, neiFS)
				zLowE1, zHighE1 := midZ, midZ
				if zHighS1 > zLowS1 || zHighE1 > zLowE1 {
					s2 := geometry.XY{X: s.X, Y: s.Y}
					buildQuad(vol3d, seg.GetNeighbor(), s2, midXY, zLowS1, zLowE1, zHighS1, zHighE1, tagLower, matLower)
				}
				// Segmento 2: Da Mid a End
				zLowS2, zHighS2 := midZ, midZ
				zLowE2, zHighE2 := curFE, math.Max(curFE, neiFE)
				if zHighS2 > zLowS2 || zHighE2 > zLowE2 {
					e2 := geometry.XY{X: e.X, Y: e.Y}
					buildQuad(vol3d, seg.GetNeighbor(), midXY, e2, zLowS2, zLowE2, zHighS2, zHighE2, tagLower, matLower)
				}
			} else {
				// Muro lineare standard
				zLowS, zHighS := curFS, math.Max(curFS, neiFS)
				zLowE, zHighE := curFE, math.Max(curFE, neiFE)
				if zHighS > zLowS || zHighE > zLowE {
					s2 := geometry.XY{X: s.X, Y: s.Y}
					e2 := geometry.XY{X: e.X, Y: e.Y}
					buildQuad(vol3d, seg.GetNeighbor(), s2, e2, zLowS, zLowE, zHighS, zHighE, tagLower, matLower)
				}
			}

			// ==========================================
			// UPPER WALL: Scontro tra soffitti inclinati
			// ==========================================
			diffCS := neiCS - curCS
			diffCE := neiCE - curCE

			// Rilevamento Crossover (i piani si incrociano lungo il segmento)
			if (diffCS > 0 && diffCE < 0) || (diffCS < 0 && diffCE > 0) {
				t := diffCS / (diffCS - diffCE)
				midXY := geometry.XY{X: s.X + t*(e.X-s.X), Y: s.Y + t*(e.Y-s.Y)}
				midZ := curCS + t*(curCE-curCS)
				// Segmento 1: Da Start a Mid
				zTopS1, zBotS1 := curCS, math.Min(curCS, neiCS)
				zTopE1, zBotE1 := midZ, midZ
				if zBotS1 < zTopS1 || zBotE1 < zTopE1 {
					s2 := geometry.XY{X: s.X, Y: s.Y}
					buildQuad(vol3d, seg.GetNeighbor(), s2, midXY, zBotS1, zBotE1, zTopS1, zTopE1, tagUpper, matUpper)
				}
				// Segmento 2: Da Mid a End
				zTopS2, zBotS2 := midZ, midZ
				zTopE2, zBotE2 := curCE, math.Min(curCE, neiCE)
				if zBotS2 < zTopS2 || zBotE2 < zTopE2 {
					e2 := geometry.XY{X: e.X, Y: e.Y}
					buildQuad(vol3d, seg.GetNeighbor(), midXY, e2, zBotS2, zBotE2, zTopS2, zTopE2, tagUpper, matUpper)
				}
			} else {
				// Muro lineare standard
				zTopS, zBotS := curCS, math.Min(curCS, neiCS)
				zTopE, zBotE := curCE, math.Min(curCE, neiCE)
				if zBotS < zTopS || zBotE < zTopE {
					s2 := geometry.XY{X: s.X, Y: s.Y}
					e2 := geometry.XY{X: e.X, Y: e.Y}
					buildQuad(vol3d, seg.GetNeighbor(), s2, e2, zBotS, zBotE, zTopS, zTopE, tagUpper, matUpper)
				}
			}
		}

		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		vol3d.SetSector(sector)
		volMap[sector] = vol3d
	}

	/*
		for _, vol := range volumes3d {
			faces, faceCount := vol.GetFaces()
			for x := 0; x < faceCount; x++ {
				face := faces[x]
				if neighbor := face.GetNeighbor(); neighbor != nil {
					face.SetNeighbor(volMap[neighbor])
				}
			}
		}

	*/
	return volumes3d
}

// compile3d constructs 3D volumes from configurations and materials, linking geometry and calculating adjacency portals.
func (r *Compiler) compile3d(volumes []*config.Volume, anim *Materials) []*Volume {
	totalFaces := 0
	var container []*Volume
	var fixFaces []*Face
	modelSectorId := 0
	facesTree := physics.NewAABBTree(1024, 0.001)
	for _, cv := range volumes {
		// cv.Id and cv.Tag come from the BSP parser
		volume := NewVolumeStatic(modelSectorId, cv.Id, cv.Tag)
		modelSectorId++
		for _, cf := range cv.Faces {
			pts := cf.Points
			pLen := len(pts)
			if pLen < 3 {
				fmt.Println("wrong points configuration", cf.Points)
				continue
			}
			material := anim.GetMaterial(cf.Material)
			// Robust polygon decomposition (Supports concave N-Gons)
			triangles := geometry.Triangulate3d(pts)
			for _, t := range triangles {
				if len(t) != 3 {
					fmt.Println("wrong tri", t)
					continue
				}
				tri := [3]geometry.XYZ{t[0], t[1], t[2]}
				face := NewFace(tri, cf.Tag, material)
				volume.AddFace(face)
				fixFaces = append(fixFaces, face)
				facesTree.InsertObject(face)
				totalFaces++
			}
		}
		// Initialize default light (will be calculated later in compileVolumesLights)
		volume.light = NewLight()
		volume.Rebuild()
		container = append(container, volume)
	}
	return container
	/*
		// Adjacency Resolution (3D Portals)
		for _, face := range fixFaces {
			if face.GetNeighbor() != nil {
				continue
			}
			bestDistSq := math.MaxFloat64
			var bestNeighborFace *Face
			facesTree.QueryOverlaps(face, func(object physics.IAABB) bool {
				overlapFace, ok := object.(*Face)
				// Ignore already linked
				if !ok || overlapFace.GetParent() == face.GetParent() || overlapFace.GetNeighbor() != nil {
					return false
				}
				// To find 3D portals, we compare the proximity of tri centroids
				pts1 := face.GetPoints()
				pts2 := overlapFace.GetPoints()
				// Calculate current face centroid (3 points)
				cx1 := (pts1[0].X + pts1[1].X + pts1[2].X) / 3.0
				cy1 := (pts1[0].Y + pts1[1].Y + pts1[2].Y) / 3.0
				cz1 := (pts1[0].Z + pts1[1].Z + pts1[2].Z) / 3.0
				// Calculate candidate face centroid
				cx2 := (pts2[0].X + pts2[1].X + pts2[2].X) / 3.0
				cy2 := (pts2[0].Y + pts2[1].Y + pts2[2].Y) / 3.0
				cz2 := (pts2[0].Z + pts2[1].Z + pts2[2].Z) / 3.0
				dx := cx1 - cx2
				dy := cy1 - cy2
				dz := cz1 - cz2
				distSq := (dx * dx) + (dy * dy) + (dz * dz)
				// Find the most perfectly matching face in space
				if distSq < bestDistSq {
					bestDistSq = distSq
					bestNeighborFace = overlapFace
				}
				return false
			})
			if bestNeighborFace != nil {
				if bestDistSq < 0.001 {
					bestNeighborFace.SetNeighbor(face.GetParent())
					face.SetNeighbor(bestNeighborFace.GetParent())
				}
			}
		}

		//fmt.Printf("Total faces: %d, not found faces: %d\n", totalFaces, totalFaces-foundFaces)
		return container

	*/
}

// compileLights processes a list of configuration light positions and returns a slice of initialized Light objects.
func (r *Compiler) compileLights(cLights []*config.Light) []*Light {
	var out []*Light
	for _, cl := range cLights {
		if cl == nil {
			continue
		}
		light := NewLight()
		light.Setup(cl, cl.Pos)
		out = append(out, light)
	}
	return out
}

// compileLights processes and merges adjacent volumes with similar properties into unified lighting areas.
func (r *Compiler) compileLights2d(locator *Sectors, computeCenter bool) []*Light {
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
					addLight(s, s.GetCentroid2d(), s.light.intensity, s.light.falloff, s.light.kind, s.light.r, s.light.g, s.light.b, s.light.style)
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
				cVolume := locator.LocateSector(first.light.pos.X, first.light.pos.Y)
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
			addLight(first, first.GetCentroid2d(), light.intensity, light.falloff, light.kind, light.r, light.g, light.b, light.style)
		}
	}
	return out
}
