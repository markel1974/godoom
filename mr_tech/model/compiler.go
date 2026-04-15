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

// Compiler represents a core game engine component for managing volumes, game objects, player interactions, and things.
type Compiler struct {
	volumes *Volumes
	player  *ThingPlayer
	lights  *Lights
	things  *Things
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
	animations := NewAnimations(cfg.GetTextures())
	if cfg.Full3d {
		volumes2d := r.compile2d(cfg.Vertices, cfg.Sectors, animations)
		volumes3d := r.compile3d(cfg.Volumes, animations)
		upgraded3d := r.upgrade3d(volumes2d)
		volumes3d = append(volumes3d, upgraded3d...)
		r.volumes = NewVolumes(volumes3d, cfg.Full3d)
	} else {
		volumes2d := r.compile2d(cfg.Vertices, cfg.Sectors, animations)
		r.volumes = NewVolumes(volumes2d, cfg.Full3d)
	}
	scale := cfg.ScaleFactor
	if scale == 0 {
		scale = 1
	}
	if scale != 1 {
		for _, volume := range r.volumes.GetVolumes() {
			// legacy lights scale
			volume.Light.pos.Scale(scale)
			for _, face := range volume.GetFaces() {
				face.Scale2d(scale)
			}
		}
		cfg.Player.Position.Scale(scale)
		for _, t := range cfg.Things {
			t.Position.Scale(scale)
		}
		for _, t := range cfg.Lights {
			t.Pos.Scale(scale)
		}
	}

	r.volumes.Setup()

	vLights2d, err := r.compileVolumesLights(r.volumes, true)
	if err != nil {
		return err
	}
	lights, err := r.compileLights(cfg.Lights)
	if err != nil {
		return err
	}
	r.lights = NewLights()
	r.lights.AddLights(vLights2d)
	r.lights.AddLights(lights)
	r.things = NewThings(uint(1+len(cfg.Things)), cfg.Things, r.volumes, animations)
	if r.player, err = NewThingPlayer(r.things, cfg.Player, r.volumes, false); err != nil {
		return err
	}

	fmt.Printf("Scan complete volumes: %d\n", r.volumes.Len())

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

// compile2d constructs and processes game volumes based on configuration data, animations, and geometry relationships.
func (r *Compiler) compile2d(vertices geometry.Polygon, css []*config.Sector, anim *Animations) []*Volume {
	const epsilon = 0.01
	modelSectorId := 0
	var container []*Volume
	var fixFaces []*Face
	facesTree := physics.NewAABBTree(1024, epsilon)
	emptyAnim := anim.GetAnimation(nil)

	ve := NewVertexEdges(epsilon)
	ve.Construct(vertices, css)

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
					fmt.Println("wrong triangle", tri)
					continue
				}
				materials := []*textures.Animation{anim.GetAnimation(cs.Floor), anim.GetAnimation(cs.Ceil)}
				volume := NewVolume2d(modelSectorId, cs.Id, cs.FloorY, cs.CeilY, materials, cs.Tag)
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
					tag := "unknown"
					// EXACT topological match
					for _, cn := range cs.Segments {
						if (p1 == cn.Start && p2 == cn.End) || (p1 == cn.End && p2 == cn.Start) {
							isWall = cn.Kind == config.SegmentWall
							tag = cn.Tag
							upper = anim.GetAnimation(cn.Upper)
							middle = anim.GetAnimation(cn.Middle)
							lower = anim.GetAnimation(cn.Lower)
							break
						}
					}
					materials := []*textures.Animation{upper, middle, lower}
					face := NewFace2d(nil, start, end, tag, materials)
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
					volume.Light.Setup(nil, cs.Light.Intensity, cs.Light.Falloff, cs.Light.Kind, lightPos)
				}
				volume.Rebuild()
				for _, face := range volume.GetFaces() {
					facesTree.InsertObject(face)
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
func (r *Compiler) upgrade3d(vols2d []*Volume) []*Volume {
	var volumes3d []*Volume
	volMap := make(map[*Volume]*Volume)

	buildQuad := func(vol3d *Volume, f2d *Face, zBottom, zTop float64, tag string, material *textures.Animation) {
		start := f2d.GetStart()
		end := f2d.GetEnd()
		v0 := geometry.XYZ{X: start.X, Y: start.Y, Z: zBottom} // Bottom-Start
		v1 := geometry.XYZ{X: end.X, Y: end.Y, Z: zBottom}     // Bottom-End
		v2 := geometry.XYZ{X: end.X, Y: end.Y, Z: zTop}        // Top-End
		v3 := geometry.XYZ{X: start.X, Y: start.Y, Z: zTop}    // Top-Start
		materials := []*textures.Animation{material}
		faceT1 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v1, v2}, tag, materials)
		faceT2 := NewFace(f2d.GetNeighbor(), [3]geometry.XYZ{v0, v2, v3}, tag, materials)
		vol3d.AddFace(faceT1)
		vol3d.AddFace(faceT2)
	}

	// 1. First Pass: Solid topology generation
	for _, vol2d := range vols2d {
		id := fmt.Sprintf("%s_3d", vol2d.GetId())
		vol3d := NewVolume3d(vol2d.GetModelId(), id, vol2d.GetTag())
		if vol2d.Light != nil {
			vol3d.Light = vol2d.Light
		}
		faces2d := vol2d.GetFaces()
		if len(faces2d) != 3 {
			fmt.Println("wrong face count", len(faces2d))
			continue
		}
		p0 := faces2d[0].GetStart()
		p1 := faces2d[1].GetStart()
		p2 := faces2d[2].GetStart()

		ceilZ := vol2d.GetMaxZ()
		ceilP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: ceilZ}, {X: p1.X, Y: p1.Y, Z: ceilZ}, {X: p2.X, Y: p2.Y, Z: ceilZ}}
		ceilMaterial := []*textures.Animation{vol2d.GetMaterial(1)}
		ceilFace := NewFace(nil, ceilP, vol2d.GetTag()+"_ceil", ceilMaterial)
		vol3d.AddFace(ceilFace)

		floorZ := vol2d.GetMinZ()
		floorP := [3]geometry.XYZ{{X: p0.X, Y: p0.Y, Z: floorZ}, {X: p2.X, Y: p2.Y, Z: floorZ}, {X: p1.X, Y: p1.Y, Z: floorZ}}
		floorMaterial := []*textures.Animation{vol2d.GetMaterial(0)}
		floorFace := NewFace(nil, floorP, vol2d.GetTag()+"_floor", floorMaterial)
		vol3d.AddFace(floorFace)

		for _, f2d := range faces2d {
			neighbor := f2d.GetNeighbor()
			if neighbor == nil {
				buildQuad(vol3d, f2d, floorZ, ceilZ, f2d.GetTag(), f2d.GetMaterial(1))
				continue
			}
			// Adjacent sector exists: we need to calculate height differentials
			nFloorZ := neighbor.GetMinZ()
			nCeilZ := neighbor.GetMaxZ()
			// 1. Lower Wall (bottom wall: neighbor's floor is higher than ours)
			if nFloorZ > floorZ {
				buildQuad(vol3d, f2d, floorZ, nFloorZ, f2d.GetTag()+"_lower", f2d.GetMaterial(2))
			}
			// 2. Upper Wall (top wall: neighbor's ceiling drops below ours)
			if nCeilZ < ceilZ {
				buildQuad(vol3d, f2d, nCeilZ, ceilZ, f2d.GetTag()+"_upper", f2d.GetMaterial(0))
			}
			// 3. Middle Portal (the opening through which the player can navigate and look)
			portalBottom := floorZ
			if nFloorZ > floorZ {
				portalBottom = nFloorZ
			}
			portalTop := ceilZ
			if nCeilZ < ceilZ {
				portalTop = nCeilZ
			}
			// If the opening physically exists (avoids glitches if two sectors are completely misaligned)
			if portalTop > portalBottom {
				buildQuad(vol3d, f2d, portalBottom, portalTop, f2d.GetTag()+"_portal", f2d.GetMaterial(1))
			}
		}
		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		volMap[vol2d] = vol3d
	}

	for _, vol := range volumes3d {
		for _, face := range vol.GetFaces() {
			if neighbor := face.GetNeighbor(); neighbor != nil {
				newNeighbor := volMap[neighbor]
				face.SetNeighbor(newNeighbor)
			}
		}
	}

	return volumes3d
}

// compile3d constructs 3D volumes from configurations and animations, linking geometry and calculating adjacency portals.
func (r *Compiler) compile3d(volumes []*config.Volume, anim *Animations) []*Volume {
	totalFaces := 0
	var container []*Volume
	var fixFaces []*Face
	modelSectorId := 0
	facesTree := physics.NewAABBTree(1024, 4.0)
	for _, cv := range volumes {
		// cv.Id and cv.Tag come from the BSP parser
		volume := NewVolume3d(modelSectorId, cv.Id, cv.Tag)
		modelSectorId++
		for _, cf := range cv.Faces {
			pts := cf.Points
			pLen := len(pts)
			if pLen < 3 {
				continue
			}
			material := []*textures.Animation{anim.GetAnimation(cf.Material)}
			// Robust polygon decomposition (Supports concave N-Gons)
			triangles := geometry.Triangulate3d(pts)
			for _, t := range triangles {
				if len(t) != 3 {
					fmt.Println("wrong triangle", t)
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

	foundFaces := 0
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
			// To find 3D portals, we compare the proximity of triangle centroids
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
				foundFaces++
			}
		}
	}
	//fmt.Printf("Total faces: %d, not found faces: %d\n", totalFaces, totalFaces-foundFaces)
	return container
}

// compileLights processes a list of configuration light positions and returns a slice of initialized Light objects.
func (r *Compiler) compileLights(cLights []*config.ConfigLight) ([]*Light, error) {
	var out []*Light
	for _, cl := range cLights {
		if cl == nil {
			continue
		}
		light := NewLight()
		light.Setup(nil, cl.Intensity, cl.Falloff, cl.Kind, cl.Pos)
	}
	return out, nil
}

// compileLights processes and merges adjacent volumes with similar properties into unified lighting areas.
func (r *Compiler) compileVolumesLights(volumes *Volumes, computeCenter bool) ([]*Light, error) {
	// Unifies adjacent triangles that belong to the same macroscopic sector.
	visited := make(map[string]bool)
	var out []*Light

	addLight := func(z *Volume, intensity float64, falloff float64, kind config.LightKind, pos geometry.XYZ) {
		lightPos := geometry.XYZ{X: pos.X, Y: pos.Y, Z: z.GetMinZ() + z.GetMaxZ()}
		light := NewLight()
		light.Setup(z, intensity, falloff, kind, lightPos)
		out = append(out, light)
	}

	for idx, sect := range volumes.GetVolumes() {
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
			for _, seg := range curr.GetFaces() {
				if n := seg.GetNeighbor(); n != nil {
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
					addLight(s, s.Light.intensity, s.Light.falloff, s.Light.kind, s.GetCentroid2d())
				}
			} else {
				var sumX, sumY, totalArea float64
				var intensity, falloff float64
				for _, s := range areaSectors {
					// Calculate triangle area (cross product)
					area := 0.0
					for _, face := range s.GetFaces() {
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
				cVolume := r.volumes.LocateVolume2d(first.Light.pos.X, first.Light.pos.Y)
				if cVolume == nil {
					cVolume = first
					fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f)\n", idx, first.Light.pos.X, first.Light.pos.Y)
				}
				addLight(cVolume, intensity, falloff, cVolume.Light.kind, gc)
			}
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			addLight(first, first.Light.intensity, first.Light.falloff, first.Light.kind, first.GetCentroid2d())
		}
	}
	return out, nil
}
