package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Compiler represents a core game engine component for managing volumes, game objects, player interactions, and things.
type Compiler struct {
	volumesN *Volumes
	player   *ThingPlayer
	lights   *Lights
	things   *Things
}

// NewCompiler initializes and returns a new instance of Compiler with default nil-initialized fields.
func NewCompiler() *Compiler {
	return &Compiler{
		volumesN: nil,
		player:   nil,
		things:   nil,
		lights:   nil,
	}
}

// Compile initializes and processes game data from the provided configuration, returning an error if compilation fails.
func (r *Compiler) Compile(cfg *config.ConfigRoot) error {
	animations := NewAnimations(cfg.GetTextures())
	if cfg.Full3d {
		volumes2d := r.compile2d(cfg.Vertices, cfg.Sectors, animations)
		volumes3d := r.compile3d(cfg.Volumes, animations)
		volumes3d = append(volumes3d, r.upgrade3d(volumes2d)...)
		r.volumesN = NewVolumes(volumes3d)
	} else {
		volumes2d := r.compile2d(cfg.Vertices, cfg.Sectors, animations)
		r.volumesN = NewVolumes(volumes2d)
	}
	scale := cfg.ScaleFactor
	if scale == 0 {
		scale = 1
	}
	if scale != 1 {
		for _, volume := range r.volumesN.GetVolumes() {
			//legacy lights scale
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

	r.volumesN.CreateTree()

	vLights2d, err := r.compileVolumesLights(r.volumesN, true)
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
	r.things = NewThings(uint(1+len(cfg.Things)), cfg.Things, r.volumesN, animations)
	if r.player, err = NewThingPlayer(r.things, cfg.Player, r.volumesN, false); err != nil {
		return err
	}

	fmt.Printf("Scan complete volumes: %d\n", r.volumesN.Len())

	return nil
}

// GetThings returns the Things instance managed by the Compiler.
func (r *Compiler) GetThings() *Things {
	return r.things
}

// GetVolumesN retrieves the Volumes instance associated with the current Compiler object.
func (r *Compiler) GetVolumesN() *Volumes {
	return r.volumesN
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
func (r *Compiler) compile2d(vertices geometry.Polygon, css []*config.ConfigSector, anim *Animations) []*Volume {
	modelSectorId := 0
	var container []*Volume
	totalPolygons := 0
	var fixFaces []*Face
	facesTree := physics.NewAABBTree(1024)
	emptyAnim := anim.GetAnimation(nil)
	ve := NewVertexEdges(0.001)
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
				volume := NewVolume2d(modelSectorId, cs.Id, cs.FloorY, anim.GetAnimation(cs.Floor), cs.CeilY, anim.GetAnimation(cs.Ceil), cs.Tag)
				modelSectorId++
				container = append(container, volume)
				// Mantiene il Winding Order consistente per ContainsPoint
				if mathematic.PointSideF(tri[2].X, tri[2].Y, tri[0].X, tri[0].Y, tri[1].X, tri[1].Y) < 0 {
					tri[1], tri[2] = tri[2], tri[1]
				}
				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]
					start := geometry.XYZ{X: p1.X, Y: p1.Y, Z: 0}
					end := geometry.XYZ{X: p2.X, Y: p2.Y, Z: 0}
					isWall := false
					upper, middle, lower := emptyAnim, emptyAnim, emptyAnim
					tag := "unknown"
					// Match topologico ESATTO
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
					face := NewFace2d(nil, start, end, tag, upper, middle, lower)
					volume.AddFace(face)
					volume.AddTag(tag)
					facesTree.InsertObject(face)
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
				totalPolygons++
			}
		}
	}

	// Risoluzione adiacenze
	for _, face := range fixFaces {
		if face.GetNeighbor() != nil { // already linked
			continue
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborFace *Face
		facesTree.QueryOverlaps(face, func(object physics.IAABB) bool {
			overlapFace, ok := object.(*Face)
			if !ok || overlapFace.GetParent() == face.GetParent() {
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
			// Link reciproco (O(N/2)
			bestNeighborFace.SetNeighbor(face.GetParent())
			face.SetNeighbor(bestNeighborFace.GetParent())
		} else {
			face.SetNeighbor(nil)
		}
	}
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
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
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
		// Utilizziamo un algoritmo di Flood Fill per trovare tutti i settori connessi
		var areaSectors []*Volume
		queue := []*Volume{sect}
		visited[sect.GetId()] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			areaSectors = append(areaSectors, curr)
			// Controlla i vicini di questo settore
			for _, seg := range curr.GetFaces() {
				if n := seg.GetNeighbor(); n != nil {
					if !visited[n.GetId()] {
						// Condizione di "Stessa Area": adiacenti e con stesse quote/luci
						if n.GetMaxZ() == curr.GetMaxZ() && n.GetMinZ() == curr.GetMinZ() && n.Light.intensity == curr.Light.intensity {
							visited[n.GetId()] = true
							queue = append(queue, n)
						}
					}
				}
			}
		}

		// Se l'area è composta da più poligoni, calcoliamo un baricentro globale
		if len(areaSectors) > 1 {
			if !computeCenter {
				for _, s := range areaSectors {
					addLight(s, s.Light.intensity, s.Light.falloff, s.Light.kind, s.GetCentroid2d())
				}
			} else {
				var sumX, sumY, totalArea float64
				var intensity, falloff float64
				for _, s := range areaSectors {
					// Calcola l'area del triangolo (prodotto vettoriale)
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
				// Legacy assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
				for _, s := range areaSectors {
					s.Light.pos.X = gc.X
					s.Light.pos.Y = gc.Y
				}
				first := areaSectors[0]
				cVolume := r.volumesN.QueryPoint2d(first.Light.pos.X, first.Light.pos.Y)
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

// compile3d constructs 3D volumes from configurations and animations, linking geometry and calculating adjacency portals.
func (r *Compiler) compile3d(volumes []*config.ConfigVolume, anim *Animations) []*Volume {
	var container []*Volume
	var fixFaces []*Face
	modelSectorId := 0
	facesTree := physics.NewAABBTree(1024)
	for _, cv := range volumes {
		// cv.Id e cv.Tag provengono dal parser BSP
		volume := NewVolume3d(modelSectorId, cv.Id, cv.Tag)
		modelSectorId++
		for _, cf := range cv.Faces {
			pts := cf.Points
			pLen := len(pts)
			if pLen < 3 {
				continue
			}
			material := anim.GetAnimation(cf.Material)
			// Scomposizione poligonale robusta (Supporta N-Gon concavi)
			triangles := geometry.Triangulate3d(pts)
			for _, tri := range triangles {
				face := NewFace(nil, tri, cf.Tag, material)
				volume.AddFace(face)
				fixFaces = append(fixFaces, face)
				facesTree.InsertObject(face)
			}
		}
		// Inizializza luce di default (verrà poi calcolata in compileVolumesLights)
		volume.Light = NewLight()
		volume.Rebuild()
		container = append(container, volume)
	}

	// Risoluzione Adiacenze (Portali 3D)
	for _, face := range fixFaces {
		if face.GetNeighbor() != nil {
			continue // Già linkato
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborFace *Face
		facesTree.QueryOverlaps(face, func(object physics.IAABB) bool {
			overlapFace, ok := object.(*Face)
			// Ignoriamo facce dello stesso volume o già linkate
			if !ok || overlapFace.GetParent() == face.GetParent() || overlapFace.GetNeighbor() != nil {
				return false
			}
			// Per trovare i portali 3D, confrontiamo la vicinanza dei baricentri dei triangoli
			pts1 := face.GetPoints()
			pts2 := overlapFace.GetPoints()
			// Calcolo baricentro faccia corrente (3 punti)
			cx1 := (pts1[0].X + pts1[1].X + pts1[2].X) / 3.0
			cy1 := (pts1[0].Y + pts1[1].Y + pts1[2].Y) / 3.0
			cz1 := (pts1[0].Z + pts1[1].Z + pts1[2].Z) / 3.0
			// Calcolo baricentro faccia candidata
			cx2 := (pts2[0].X + pts2[1].X + pts2[2].X) / 3.0
			cy2 := (pts2[0].Y + pts2[1].Y + pts2[2].Y) / 3.0
			cz2 := (pts2[0].Z + pts2[1].Z + pts2[2].Z) / 3.0
			dx := cx1 - cx2
			dy := cy1 - cy2
			dz := cz1 - cz2
			distSq := (dx * dx) + (dy * dy) + (dz * dz)
			// Troviamo la faccia più perfettamente combaciante nello spazio
			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborFace = overlapFace
			}
			return false
		})
		// Tolleranza per la saldatura dei portali (Epsilon)
		if bestNeighborFace != nil && bestDistSq < 0.001 {
			bestNeighborFace.SetNeighbor(face.GetParent())
			face.SetNeighbor(bestNeighborFace.GetParent())
		}
	}
	return container
}

// upgrade3d converts a collection of 2D volumes into their corresponding 3D representations with updated topology and adjacency links.
func (r *Compiler) upgrade3d(vols2d []*Volume) []*Volume {
	var volumes3d []*Volume

	// Mappa di traduzione: *Volume 2D -> *Volume 3D
	volMap := make(map[*Volume]*Volume)

	// 1. Prima Passata: Generazione della topologia solida
	for _, vol2d := range vols2d {
		id := fmt.Sprintf("%s_3d", vol2d.GetId())
		vol3d := NewVolume3d(vol2d.GetModelId(), id, vol2d.GetTag())
		if vol2d.Light != nil {
			vol3d.Light = vol2d.Light
		}
		faces2d := vol2d.GetFaces()
		if len(faces2d) != 3 {
			continue
		}
		p0 := faces2d[0].GetStart()
		p1 := faces2d[1].GetStart()
		p2 := faces2d[2].GetStart()
		floorY := vol2d.GetMinZ()
		ceilY := vol2d.GetMaxZ()
		// [Indice 0] Soffitto (Ceil)
		ceilP := []geometry.XYZ{{X: p0.X, Y: ceilY, Z: p0.Y}, {X: p1.X, Y: ceilY, Z: p1.Y}, {X: p2.X, Y: ceilY, Z: p2.Y}}
		vol3d.AddFace(NewFace(nil, ceilP, vol2d.GetTag()+"_ceil", vol2d.GetMaterialCeil()))
		// [Indice 1] Pavimento (Floor)
		floorP := []geometry.XYZ{{X: p0.X, Y: floorY, Z: p0.Y}, {X: p2.X, Y: floorY, Z: p2.Y}, {X: p1.X, Y: floorY, Z: p1.Y}}
		vol3d.AddFace(NewFace(nil, floorP, vol2d.GetTag()+"_floor", vol2d.GetMaterialFloor()))
		// [Indici 2, 3, 4] Facce Laterali
		for _, f2d := range faces2d {
			start := f2d.GetStart()
			end := f2d.GetEnd()
			wallPts := []geometry.XYZ{
				{X: start.X, Y: floorY, Z: start.Y},
				{X: end.X, Y: floorY, Z: end.Y},
				{X: end.X, Y: ceilY, Z: end.Y},
				{X: start.X, Y: ceilY, Z: start.Y},
			}
			vol3d.AddFace(NewFace(nil, wallPts, f2d.GetTag(), f2d.GetMaterialMiddle()))
		}
		vol3d.Rebuild()
		volumes3d = append(volumes3d, vol3d)
		volMap[vol2d] = vol3d
	}

	// 2. Seconda Passata: Link delle adiacenze (Portali)
	for idx, vol2d := range vols2d {
		vol3d := volumes3d[idx]
		faces2d := vol2d.GetFaces()
		faces3d := vol3d.GetFaces()
		for i, f2d := range faces2d {
			if neighbor2d := f2d.GetNeighbor(); neighbor2d != nil {
				// Recupera il puntatore al nuovo volume 3D associato al vicino 2D
				if neighbor3d, ok := volMap[neighbor2d]; ok {
					// Mappa 1:1 traslata di 2 (saltando ceil e floor)
					faces3d[i+2].SetNeighbor(neighbor3d)
				}
			}
		}
	}
	return volumes3d
}
