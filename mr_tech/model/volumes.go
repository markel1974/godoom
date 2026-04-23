package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// Volumes represents a collection of 3D or 2D world, utilizing a hierarchical spatial structure and caching for efficiency.
type Volumes struct {
	container []*Volume
	tree      *physics.AABBTree
	cache     map[string]*Volume
	fullZ     bool
}

// NewVolumes initializes a Volumes structure with a container of Volume instances and a cache for quick access by ID.
func NewVolumes(container []*Volume, fullZ bool) *Volumes {
	cache := make(map[string]*Volume)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	vs := &Volumes{
		container: container,
		cache:     cache,
		tree:      physics.NewAABBTree(uint(len(container)), 4.0),
		fullZ:     fullZ,
	}

	return vs

}

// Setup constructs a new AABB tree based on the current container and populates it with rebuilt location objects.
func (s *Volumes) Setup() {
	for _, volume := range s.container {
		if volume.Rebuild() {
			s.tree.InsertObject(volume)
		}
	}
}

// GetVolume retrieves a Volume from the cache using the provided unique identifier.
func (s *Volumes) GetVolume(id string) *Volume {
	return s.cache[id]
}

// GetVolumes returns all Volume objects managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the number of Volume objects contained within the Volumes container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves a list of world that overlap with the specified axis-aligned bounding box (AABB).
func (s *Volumes) Query(aabb physics.IAABB) []*Volume {
	var target []*Volume
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		sector, ok := object.(*Volume)
		if !ok {
			return false
		}
		target = append(target, sector)
		return false
	})
	return target
}

func (s *Volumes) QueryClosestFace(z physics.IAABB, viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64, float64) {
	var closestFace *Face = nil
	minT := 1.0
	var colNx, colNy, colNz float64

	s.QueryAABB(z, func(vol *Volume) {
		vol.facesTree.QueryOverlaps(z, func(object physics.IAABB) bool {
			face, ok := object.(*Face)
			if !ok {
				return false
			}

			// Filtro portali
			if neighbor := face.GetNeighbor(); neighbor != nil {
				holeLow := math.Max(vol.GetMinZ(), neighbor.GetMinZ())
				holeHigh := math.Min(vol.GetMaxZ(), neighbor.GetMaxZ())
				if top <= holeHigh && bottom >= holeLow {
					return false
				}
			}

			n := face.GetNormal()
			pts := face.GetPoints()
			p0 := pts[0]

			// Distanza con segno dal piano
			distStart := (viewX-p0.X)*n.X + (viewY-p0.Y)*n.Y + (viewZ-p0.Z)*n.Z
			distEnd := (pX-p0.X)*n.X + (pY-p0.Y)*n.Y + (pZ-p0.Z)*n.Z

			// --- FIX: DETERMINAZIONE DEL LATO (Double-Sided) ---
			side := 1.0
			if distStart < 0 {
				side = -1.0
			}

			// Normalizziamo le distanze rispetto al lato di approccio
			sDistStart := distStart * side
			sDistEnd := distEnd * side

			hit := false
			var hitT float64
			var cNx, cNy, cNz float64

			// Fase A: Sweep Test (ora usa sDist)
			hitPlane := false
			var hX, hY, hZ float64
			if sDistStart >= -0.01 && sDistEnd < radius {
				dotVel := sDistEnd - sDistStart
				if dotVel < 0 {
					timeHit := (radius - sDistStart) / dotVel
					if timeHit < 0 {
						timeHit = 0
					}
					if timeHit <= 1.0 {
						hX = viewX + velX*timeHit
						hY = viewY + velY*timeHit
						hZ = viewZ + velZ*timeHit
						hitPlane = true
					}
				}
			}

			pLen := len(pts)
			for i := 0; i < pLen; i++ {
				start, end := pts[i], pts[(i+1)%pLen]
				edgeX, edgeY, edgeZ := end.X-start.X, end.Y-start.Y, end.Z-start.Z
				edgeLenSq := (edgeX * edgeX) + (edgeY * edgeY) + (edgeZ * edgeZ)

				if hitPlane && !hit {
					vX, vY, vZ := hX-start.X, hY-start.Y, hZ-start.Z
					dotEdge := (vX * edgeX) + (vY * edgeY) + (vZ * edgeZ)
					if dotEdge >= -0.1 && dotEdge <= edgeLenSq+0.1 {
						hit = true
						dotVel := sDistEnd - sDistStart
						timeHit := (radius - sDistStart) / dotVel
						hitT = math.Max(0, timeHit)
						// INVERTIAMO LA NORMALE se colpiamo dal retro (side = -1)
						cNx, cNy, cNz = n.X*side, n.Y*side, n.Z*side
					}
				}

				// Fase B: Spigoli/Vertici (Double-Sided)
				if !hit {
					vX, vY, vZ := pX-start.X, pY-start.Y, pZ-start.Z
					tProj := 0.0
					if edgeLenSq > 0 {
						tProj = math.Max(0.0, math.Min(1.0, (vX*edgeX+vY*edgeY+vZ*edgeZ)/edgeLenSq))
					}
					diffX := pX - (start.X + tProj*edgeX)
					diffY := pY - (start.Y + tProj*edgeY)
					diffZ := pZ - (start.Z + tProj*edgeZ)

					distSq := (diffX * diffX) + (diffY * diffY) + (diffZ * diffZ)
					if distSq < radius*radius {
						hit = true
						hitT = 0.0
						cDist := math.Sqrt(distSq)
						if tProj > 0.0 && tProj < 1.0 {
							// Anche qui, normale relativa al lato di approccio
							cNx, cNy, cNz = n.X*side, n.Y*side, n.Z*side
						} else {
							if cDist > 0.0001 {
								cNx, cNy, cNz = diffX/cDist, diffY/cDist, diffZ/cDist
							} else {
								cNx, cNy, cNz = n.X*side, n.Y*side, n.Z*side
							}
						}
					}
				}
			}

			if hit && hitT <= minT {
				minT = hitT
				closestFace = face
				colNx, colNy, colNz = cNx, cNy, cNz
			}
			return false
		})
	})

	return closestFace, colNx, colNy, colNz, minT
}

// QueryFrustum performs a spatial query using a frustum, invoking the callback for each intersected object in the tree.
func (s *Volumes) QueryFrustum(frustum *physics.Frustum, callback func(object physics.IAABB) bool) {
	s.tree.QueryFrustum(frustum, callback)
}

// QueryMultiFrustum performs a spatial query using two frustums, invoking the callback for each overlapping object.
func (s *Volumes) QueryMultiFrustum(front, rear *physics.Frustum, callback func(object physics.IAABB) bool) {
	s.tree.QueryMultiFrustum(front, rear, callback)
}

// QueryRay performs a raycasting query starting from origin (oX, oY, oZ) in direction (dirX, dirY, dirZ) up to maxDistance.
// It invokes the callback for each intersected object, passing the object and intersection distance as arguments.
func (s *Volumes) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object physics.IAABB, distance float64) (float64, bool)) {
	s.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// QueryPoint2d returns the Volume containing the 2D point (px, py, pz), or nil if no such Volume exists.
func (s *Volumes) QueryPoint2d(px, py, pz float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			if vol.PointInside2d(px, py, pz) {
				target = vol
				return true
			}
		}
		return false
	})
	return target
}

// LocateVolume finds and returns the volume containing the point (px, py, pz). It uses 3D or 2D lookup based on the fullZ flag.
func (s *Volumes) LocateVolume(px, py, pz float64) *Volume {
	if s.fullZ {
		v, _ := s.locateVolume3d(px, py, pz)
		return v
	}
	return s.LocateVolume2d(px, py)
}

// LocateVolume2d searches for a 2D point (px, py) within the managed world and returns the corresponding Volume, or nil if not found.
func (s *Volumes) LocateVolume2d(px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
		if volume, ok := object.(*Volume); ok {
			if volume.PointInLineSide(px, py) {
				target = volume
				return true
			}
		}
		return false
	})
	return target
}

// LocateVolume3d identifies the 3D location and specific face at the given point (px, py, pz) in world coordinates.
func (s *Volumes) locateVolume3d(px, py, pz float64) (*Volume, *Face) {
	var bestVol *Volume
	var bestFace *Face
	var minZDist = math.MaxFloat64 // Per trovare il pavimento più vicino sotto ai piedi
	// Broad-Phase Globale: troviamo i volumi il cui AABB 3D contiene il punto
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		volume, volumeOk := object.(*Volume)
		if !volumeOk {
			return false
		}
		// Broad-Phase Locale
		volume.facesTree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
			face, faceOk := object.(*Face)
			if !faceOk {
				return false
			}
			norm := face.GetNormal()
			// Filtro Topologico: Selezioniamo solo i pavimenti (Normal Z negativa)
			if norm.Z >= -0.001 {
				return false
			}
			// Proiezione 2D
			if face.PointInside2d(px, py) {
				// CALCOLO Z ESATTO SUL TRIANGOLO (Plane Equation)
				// Z = V.z - (Nx*(Px - V.x) + Ny*(Py - V.y)) / Nz
				// Assumiamo che tu possa recuperare un vertice della faccia, es. face.GetVertex(0)
				v0 := face.tri[0] // <--- Adattalo al tuo metodo reale per prendere un Vector3 del triangolo
				floorZ := v0.Z - (norm.X*(px-v0.X)+norm.Y*(py-v0.Y))/norm.Z
				// Distanza verticale dal player al pavimento
				zDist := pz - floorZ
				// Se il pavimento è SOTTO il player (o entro un piccolo margine di compenetrazione/step)
				// E se è il più vicino che abbiamo trovato finora
				if zDist >= -0.5 && zDist < minZDist {
					minZDist = zDist
					bestVol = volume
					bestFace = face
				}
				// Ritorniamo false per far finire il ciclo locale e testare altri eventuali pavimenti sovrapposti
				return false
			}
			return false
		})

		// Continuiamo sempre la query globale per coprire il caso di AABB di volumi sovrapposti
		return false
	})

	return bestVol, bestFace
}

/*
// LocateVolume3d trova il location 3D che contiene il punto (px, py, pz) e
// restituisce sia il Volume che la Faccia di riferimento (es. il pavimento sotto al punto).
func (s *Volumes) LocateVolume3d(px, py, pz float64) (*Volume, *Face) {
	var targetVol *Volume
	var targetFace *Face

	// 1. Broad-Phase Globale: troviamo il location 3D
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		location, volumeOk := object.(*Volume)
		if !volumeOk {
			return false
		}
		location.facesTree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
			face, faceOk := object.(*Face)
			if !faceOk {
				return false
			}
			// Filtro Topologico: Selezioniamo solo le facce che fungono da pavimento.
			// Nei poliedri convessi con normali rivolte verso l'esterno,
			// il pavimento ha la normale Z rivolta verso il basso (negativa).
			if face.GetNormal().Z >= -0.001 {
				return false // Scarta muri (Z≈0) e soffitti (Z>0)
			}
			// Verifica Esatta: il punto cade verticalmente dentro questo specifico triangolo?
			if face.PointInTriangle3d(px, py, pz) {
				targetVol = location
				targetFace = face
				return true
			}
			return false
		})
		if targetFace != nil {
			return true
		}
		return false
	})
	return targetVol, targetFace
}
*/

// QueryAABB performs a spatial query, invoking the callback for each Volume that overlaps with the specified AABB.
func (s *Volumes) QueryAABB(aabb physics.IAABB, callback func(vol *Volume)) {
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			callback(vol)
		}
		return false
	})
}
