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
	var container2d []*Volume

	if len(cfg.Sectors) > 0 {
		container2d = r.compile2d(cfg.Vertices, cfg.Sectors, materials)
		if len(container2d) == 0 {
			return fmt.Errorf("no 2D volumes compiled")
		}
		volumes2d := NewVolumes(container2d, false)
		volumes2d.Setup()
		//player Z
		pv := volumes2d.LocateVolume2d(cfg.Player.Position.X, cfg.Player.Position.Y)
		if pv == nil {
			return fmt.Errorf("can't find 2d player location at X: %f Y: %f", cfg.Player.Position.X, cfg.Player.Position.Y)
		}
		cfg.Player.Position.Z = pv.GetMinZ()
		//things Z
		for idx := range cfg.Things {
			tx, ty := cfg.Things[idx].Position.X, cfg.Things[idx].Position.Y
			tv := volumes2d.LocateVolume2d(tx, ty)
			if tv == nil {
				fmt.Println("can't find thing location at", tx, ty)
				continue
			}
			cfg.Things[idx].Position.Z = tv.GetMinZ()
		}
		//light 2d
		r.lights.AddLights(r.compileLights2d(volumes2d, true))
	}

	var allVolumes []*Volume
	if cfg.Calibration.Full3d {
		if len(container2d) > 0 {
			allVolumes = append(allVolumes, r.upgrade3d(container2d)...)
		}
		if len(cfg.Volumes) > 0 {
			allVolumes = append(allVolumes, r.compile3d(cfg.Volumes, materials)...)
		}
	} else {
		allVolumes = append(allVolumes, container2d...)
	}

	r.volumes = NewVolumes(allVolumes, full3d)
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

// compile2d constructs and processes game volumes based on configuration data, materials, and geometry relationships.
func (r *Compiler) compile2d(vertices geometry.Polygon, css []*config.Sector, anim *Materials) []*Volume {
	const epsilon = 0.01
	modelSectorId := 0
	var container []*Volume
	var fixFaces []*Face
	facesTree := physics.NewAABBTree(1024, epsilon)
	emptyAnim := anim.GetMaterial(nil)

	ve := NewSectorsEdges(epsilon)
	ve.Construct(vertices, css)

	unknownCounter := 0
	for csIdx, cs := range css {
		if len(cs.Segments) == 0 {
			continue
		}
		triContainer, _, err := ve.GetTriangles(csIdx)
		if err != nil {
			fmt.Println("Error retrieving polygons for sector", csIdx, ":", err.Error())
			continue
		}
		for _, triangles := range triContainer {
			for _, tri := range triangles {
				if len(tri) != 3 {
					fmt.Println("wrong tri", tri)
					continue
				}
				volumeMaterials := []*textures.Material{anim.GetMaterial(cs.Floor), anim.GetMaterial(cs.Ceil)}
				volume := NewVolume2d(modelSectorId, cs.Id, cs.FloorY, cs.CeilY, volumeMaterials, cs.Tag)
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
						if (p1 == cn.Start && p2 == cn.End) || (p1 == cn.End && p2 == cn.Start) {
							isWall = cn.Kind == config.SegmentWall
							tag = cn.Tag
							upper = anim.GetMaterial(cn.Upper)
							middle = anim.GetMaterial(cn.Middle)
							lower = anim.GetMaterial(cn.Lower)
							break
						}
					}
					faceMaterials := []*textures.Material{upper, middle, lower}
					face := NewFace2d(nil, start, end, tag, faceMaterials)
					volume.AddFace(face)
					volume.AddTag(tag)
					if !isWall {
						fixFaces = append(fixFaces, face)
					}
				}
				volume.Light = NewLight()
				if cs.Light != nil {
					centroid := volume.GetCentroid2d()
					lightPos := geometry.XYZ{X: centroid.X, Y: centroid.Y, Z: cs.FloorY + cs.CeilY}
					volume.Light.Setup(nil, cs.Light, lightPos)
				}
				volume.Rebuild()
				faces, faceCount := volume.GetFaces()
				for x := 0; x < faceCount; x++ {
					facesTree.InsertObject(faces[x])
				}
				container = append(container, volume)
			}
		}
	}

	// Adjacency resolution
	for _, face := range fixFaces {
		if face.GetNeighbor() != nil { // already linked
			continue
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborFace *Face
		facesTree.QueryOverlaps(face, func(object physics.IAABB) bool {
			overlapFace, ok := object.(*Face)
			if !ok {
				return false
			}
			if overlapFace.GetParent() == face.GetParent() {
				return false
			}
			start := face.GetStart()
			end := face.GetEnd()
			overlapStart := overlapFace.GetStart()
			overlapEnd := overlapFace.GetEnd()
			dx1 := start.X - overlapEnd.X
			dy1 := start.Y - overlapEnd.Y
			dx2 := end.X - overlapStart.X
			dy2 := end.Y - overlapStart.Y
			distSq := (dx1 * dx1) + (dy1 * dy1) + (dx2 * dx2) + (dy2 * dy2)
			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborFace = overlapFace
			}
			return false
		})
		if bestNeighborFace != nil {
			// Bidirectional link (O(N/2))
			bestNeighborFace.SetNeighbor(face.GetParent())
			face.SetNeighbor(bestNeighborFace.GetParent())
		} else {
			face.SetNeighbor(nil)
		}
	}
	return container
}

// upgrade3d converts a collection of 2D volumes into their corresponding 3D representations with updated topology and adjacency links.
func (r *Compiler) upgrade3d(volumes2d []*Volume) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*Volume]*Volume)

	buildQuad := func(vol3d *Volume, f2d *Face, zBottom, zTop float64, tag string, material *textures.Material) {
		start := f2d.GetStart()
		end := f2d.GetEnd()
		v0 := geometry.XYZ{X: start.X, Y: start.Y, Z: zBottom} // Bottom-Start
		v1 := geometry.XYZ{X: end.X, Y: end.Y, Z: zBottom}     // Bottom-End
		v2 := geometry.XYZ{X: end.X, Y: end.Y, Z: zTop}        // Top-End
		v3 := geometry.XYZ{X: start.X, Y: start.Y, Z: zTop}    // Top-Start
		faceT1 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v1, v2}, tag, material)
		faceT2 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v2, v3}, tag, material)
		vol3d.AddFace(faceT1)
		vol3d.AddFace(faceT2)
	}

	// 1. First Pass: Solid topology generation
	for _, vol2d := range volumes2d {
		id := fmt.Sprintf("%s_3d", vol2d.GetId())
		vol3d := NewVolume3d(vol2d.GetModelId(), id, vol2d.GetTag())
		if vol2d.Light != nil {
			vol3d.Light = vol2d.Light
		}
		faces2d, face2dCount := vol2d.GetFaces()
		if face2dCount != 3 {
			fmt.Println("wrong face count", face2dCount)
			continue
		}
		p0 := faces2d[0].GetStart()
		p1 := faces2d[1].GetStart()
		p2 := faces2d[2].GetStart()

		ceilZ := vol2d.GetMaxZ()
		ceilP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: ceilZ}, {X: p1.X, Y: p1.Y, Z: ceilZ}, {X: p2.X, Y: p2.Y, Z: ceilZ}}
		ceilFace := NewFace(nil, ceilP, vol2d.GetTag()+"_ceil", vol2d.GetMaterialIndex(1))
		vol3d.AddFace(ceilFace)

		floorZ := vol2d.GetMinZ()
		floorP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: floorZ}, {X: p2.X, Y: p2.Y, Z: floorZ}, {X: p1.X, Y: p1.Y, Z: floorZ}}
		floorFace := NewFace(nil, floorP, vol2d.GetTag()+"_floor", vol2d.GetMaterialIndex(0))
		vol3d.AddFace(floorFace)

		for x := 0; x < face2dCount; x++ {
			f2d := faces2d[x]
			neighbor := f2d.GetNeighbor()
			if neighbor == nil {
				buildQuad(vol3d, f2d, floorZ, ceilZ, f2d.GetTag(), f2d.GetMaterialIndex(1))
				continue
			}
			// Adjacent sector exists: we need to calculate height differentials
			nFloorZ := neighbor.GetMinZ()
			nCeilZ := neighbor.GetMaxZ()
			// 1. Lower Wall (bottom wall: neighbor's floor is higher than ours)
			if nFloorZ > floorZ {
				buildQuad(vol3d, f2d, floorZ, nFloorZ, f2d.GetTag()+"_lower", f2d.GetMaterialIndex(2))
			}
			// 2. Upper Wall (top wall: neighbor's ceiling drops below ours)
			if nCeilZ < ceilZ {
				buildQuad(vol3d, f2d, nCeilZ, ceilZ, f2d.GetTag()+"_upper", f2d.GetMaterialIndex(0))
			}
			// 3. Middle Portal (the opening through which the player can navigate and look)
			//portalBottom := floorZ
			//if nFloorZ > floorZ {
			//	portalBottom = nFloorZ
			//}
			//portalTop := ceilZ
			//if nCeilZ < ceilZ {
			//	portalTop = nCeilZ
			//}
			// If the opening physically exists (avoids glitches if two sectors are completely misaligned)
			//if portalTop > portalBottom {
			//buildQuad(vol3d, f2d, portalBottom, portalTop, f2d.GetTag()+"_portal", f2d.GetMaterialIndex(1))
			//}
		}
		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		volMap[vol2d] = vol3d
	}

	for _, vol := range volumes3d {
		faces, faceCount := vol.GetFaces()
		for x := 0; x < faceCount; x++ {
			face := faces[x]
			if neighbor := face.GetNeighbor(); neighbor != nil {
				newNeighbor := volMap[neighbor]
				face.SetNeighbor(newNeighbor)
			}
		}
	}

	return volumes3d
}

/*
// upgrade3d converte i volumi 2D in 3D applicando le pendenze (slopes) e rigenerando la topologia.
func (r *Compiler) upgrade3d(volumes2d []*Volume) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*Volume]*Volume)

	// Funzione helper per calcolare la Z in un punto (x, y) dato il vettore di slope.
	// Z = Slope.Z + (Slope.X * x) + (Slope.Y * y)
	getZ := func(p geometry.XYZ, slope geometry.XYZ) float64 {
		return slope.Z + (slope.X * p.X) + (slope.Y * p.Y)
	}

	// buildQuad ora accetta quote differenziate per l'inizio e la fine del segmento (trapezio).
	buildQuad := func(vol3d *Volume, f2d *Face, zBS, zBE, zTS, zTE float64, tag string, material *textures.Material) {
		start := f2d.GetStart()
		end := f2d.GetEnd()

		// Costruiamo i 4 vertici del trapezio verticale nello spazio 3D
		v0 := geometry.XYZ{X: start.X, Y: start.Y, Z: zBS} // Bottom-Start
		v1 := geometry.XYZ{X: end.X, Y: end.Y, Z: zBE}     // Bottom-End
		v2 := geometry.XYZ{X: end.X, Y: end.Y, Z: zTE}     // Top-End
		v3 := geometry.XYZ{X: start.X, Y: start.Y, Z: zTS} // Top-Start

		faceT1 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v1, v2}, tag, material)
		faceT2 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v2, v3}, tag, material)
		vol3d.AddFace(faceT1)
		vol3d.AddFace(faceT2)
	}

	for _, vol2d := range volumes2d {
		id := fmt.Sprintf("%s_3d", vol2d.GetId())
		vol3d := NewVolume3d(vol2d.GetModelId(), id, vol2d.GetTag())
		if vol2d.Light != nil {
			vol3d.Light = vol2d.Light
		}

		faces2d, face2dCount := vol2d.GetFaces()
		if face2dCount != 3 {
			continue // Supportiamo solo triangoli per la compilazione 2D
		}

		// Recuperiamo i vertici della mesh planare
		p0, p1, p2 := faces2d[0].GetStart(), faces2d[1].GetStart(), faces2d[2].GetStart()

		// 1. Orientamento Mesh Soffitto (Ceiling)
		zC0 := getZ(p0, vol2d.slopedCeil)
		zC1 := getZ(p1, vol2d.slopedCeil)
		zC2 := getZ(p2, vol2d.slopedCeil)
		ceilP := [3]geometry.XYZ{{p0.X, p0.Y, zC0}, {p1.X, p1.Y, zC1}, {p2.X, p2.Y, zC2}}
		vol3d.AddFace(NewFace(nil, ceilP, vol2d.GetTag()+"_ceil", vol2d.GetMaterialIndex(1)))

		// 2. Orientamento Mesh Pavimento (Floor)
		zF0 := getZ(p0, vol2d.slopedFloor)
		zF1 := getZ(p1, vol2d.slopedFloor)
		zF2 := getZ(p2, vol2d.slopedFloor)
		floorP := [3]geometry.XYZ{{p0.X, p0.Y, zF0}, {p2.X, p2.Y, zF2}, {p1.X, p1.Y, zF1}}
		vol3d.AddFace(NewFace(nil, floorP, vol2d.GetTag()+"_floor", vol2d.GetMaterialIndex(0)))

		// 3. Generazione Muri Perimetrali e Adiacenze
		for x := 0; x < face2dCount; x++ {
			f2d := faces2d[x]
			s, e := f2d.GetStart(), f2d.GetEnd()

			// Altezze locali per il settore corrente
			curF_S, curF_E := getZ(s, vol2d.slopedFloor), getZ(e, vol2d.slopedFloor)
			curC_S, curC_E := getZ(s, vol2d.slopedCeil), getZ(e, vol2d.slopedCeil)

			neighbor := f2d.GetNeighbor()
			if neighbor == nil {
				// Muro solido (da pavimento a soffitto)
				buildQuad(vol3d, f2d, curF_S, curF_E, curC_S, curC_E, f2d.GetTag(), f2d.GetMaterialIndex(1))
				continue
			}

			// Se c'è un vicino, calcoliamo le pendenze del vicino lungo il confine
			neiF_S, neiF_E := getZ(s, neighbor.slopedFloor), getZ(e, neighbor.slopedFloor)
			neiC_S, neiC_E := getZ(s, neighbor.slopedCeil), getZ(e, neighbor.slopedCeil)

			// Lower Wall: se il pavimento del vicino è più alto del nostro in qualsiasi punto del segmento
			if neiF_S > curF_S || neiF_E > curF_E {
				buildQuad(vol3d, f2d, curF_S, curF_E, neiF_S, neiF_E, f2d.GetTag()+"_lower", f2d.GetMaterialIndex(2))
			}

			// Upper Wall: se il soffitto del vicino è più basso del nostro
			if neiC_S < curC_S || neiC_E < curC_E {
				buildQuad(vol3d, f2d, neiC_S, neiC_E, curC_S, curC_E, f2d.GetTag()+"_upper", f2d.GetMaterialIndex(0))
			}
		}

		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		volMap[vol2d] = vol3d
	}

	// Risoluzione dei link bidirezionali tra i volumi 3D
	for _, vol := range volumes3d {
		faces, faceCount := vol.GetFaces()
		for x := 0; x < faceCount; x++ {
			face := faces[x]
			if neighbor := face.GetNeighbor(); neighbor != nil {
				face.SetNeighbor(volMap[neighbor])
			}
		}
	}
	return volumes3d
}

*/

// compile3d constructs 3D volumes from configurations and materials, linking geometry and calculating adjacency portals.
func (r *Compiler) compile3d(volumes []*config.Volume, anim *Materials) []*Volume {
	totalFaces := 0
	var container []*Volume
	var fixFaces []*Face
	modelSectorId := 0
	facesTree := physics.NewAABBTree(1024, 0.001)
	for _, cv := range volumes {
		// cv.Id and cv.Tag come from the BSP parser
		volume := NewVolume3d(modelSectorId, cv.Id, cv.Tag)
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
				face := NewFace(nil, tri, cf.Tag, material)
				volume.AddFace(face)
				fixFaces = append(fixFaces, face)
				facesTree.InsertObject(face)
				totalFaces++
			}
		}
		// Initialize default light (will be calculated later in compileVolumesLights)
		volume.Light = NewLight()
		volume.Rebuild()
		container = append(container, volume)
	}

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
}

// compileLights processes a list of configuration light positions and returns a slice of initialized Light objects.
func (r *Compiler) compileLights(cLights []*config.Light) []*Light {
	var out []*Light
	for _, cl := range cLights {
		if cl == nil {
			continue
		}
		light := NewLight()
		light.Setup(nil, cl, cl.Pos)
		out = append(out, light)
	}
	return out
}

// compileLights processes and merges adjacent volumes with similar properties into unified lighting areas.
func (r *Compiler) compileLights2d(volumes2d *Volumes, computeCenter bool) []*Light {
	// Unifies adjacent volume that belong to the same macroscopic sector.
	visited := make(map[string]bool)
	var out []*Light

	addLight := func(z *Volume, pos geometry.XYZ, intensity float64, falloff float64, kind config.LightKind, r, g, b float64, style []float64) {
		lightPos := geometry.XYZ{X: pos.X, Y: pos.Y, Z: z.GetMinZ() + z.GetMaxZ()}
		cl := config.NewConfigLight(pos, intensity, kind, falloff)
		cl.R = r
		cl.G = g
		cl.B = b
		cl.Style = style
		light := NewLight()
		light.Setup(z, cl, lightPos)
		out = append(out, light)
	}

	for idx, sect := range volumes2d.GetVolumes() {
		if visited[sect.GetId()] {
			continue
		}
		// We use a Flood Fill algorithm to find all connected sectors
		var areaSectors []*Volume
		queue := []*Volume{sect}
		visited[sect.GetId()] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			areaSectors = append(areaSectors, curr)
			// Check neighbors of this sector
			faces, faceCount := curr.GetFaces()
			for x := 0; x < faceCount; x++ {
				face := faces[x]
				if n := face.GetNeighbor(); n != nil {
					if !visited[n.GetId()] {
						// "Same Area" condition: adjacent and with same heights/lights
						if n.GetMaxZ() == curr.GetMaxZ() && n.GetMinZ() == curr.GetMinZ() && n.Light.intensity == curr.Light.intensity {
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
					addLight(s, s.GetCentroid2d(), s.Light.intensity, s.Light.falloff, s.Light.kind, s.Light.r, s.Light.g, s.Light.b, s.Light.style)
				}
			} else {
				var sumX, sumY, totalArea float64
				var intensity, falloff float64
				for _, s := range areaSectors {
					// Calculate tri area (cross product)
					area := 0.0
					faces, faceCount := s.GetFaces()
					for x := 0; x < faceCount; x++ {
						face := faces[x]
						start := face.GetStart()
						end := face.GetEnd()
						x0, y0 := start.X, start.Y
						x1, y1 := end.X, end.Y
						area += (x0 * y1) - (x1 * y0)
					}
					area = math.Abs(area * 0.5)
					sumX += s.Light.pos.X * area
					sumY += s.Light.pos.Y * area
					totalArea += area
					intensity += s.Light.intensity
					falloff += s.Light.falloff
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
					s.Light.pos.X = gc.X
					s.Light.pos.Y = gc.Y
				}
				first := areaSectors[0]
				cVolume := volumes2d.LocateVolume2d(first.Light.pos.X, first.Light.pos.Y)
				if cVolume == nil {
					cVolume = first
					fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f, z:%f)\n", idx, first.Light.pos.X, first.Light.pos.Y, first.Light.pos.Z)
				}
				light := cVolume.Light
				addLight(cVolume, gc, intensity, falloff, light.kind, light.r, light.g, light.b, light.style)
			}
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			light := first.Light
			addLight(first, first.GetCentroid2d(), light.intensity, light.falloff, light.kind, light.r, light.g, light.b, light.style)
		}
	}
	return out
}
