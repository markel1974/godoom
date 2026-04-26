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

// QueryCollisionCage evaluates 3D collision data within a given cage and applies spatial filters, assigning results into buckets.
func (s *Volumes) QueryCollisionCage(cage *CollisionCage, maxCliff float64) {
	margin := cage.GetMargin()
	self := cage.GetAABB()
	cX, cY, cZ := cage.GetC()
	tX, tY, tZ := cage.GetT()
	eRadX, eRadY, eRadZ := cage.GetRad()
	minX, minY, minZ := self.GetMinX(), self.GetMinY(), self.GetMinZ()
	maxX, maxY, maxZ := self.GetMaxX(), self.GetMaxY(), self.GetMaxZ()
	baseCliff := cZ - eRadZ

	s.QueryAABB(cage, func(vol *Volume) {
		vol.facesTree.QueryOverlaps(cage, func(otherEnt physics.IAABB) bool {
			face, ok := otherEnt.(*Face)
			if !ok {
				return false
			}
			absX, absY, absZ := face.normalAbs.X, face.normalAbs.Y, face.normalAbs.Z
			fAABB := otherEnt.GetAABB()
			fMaxZ := fAABB.GetMaxZ()

			// CLIFF CULLING
			wallWE := absX > absY && absX > absZ
			wallNS := absY > absZ
			isWall := wallWE || wallNS

			if isWall && fMaxZ <= baseCliff+maxCliff {
				//fmt.Println("FILTRO WALL ATTIVO, RETURNING", fMaxZ, baseCliff+maxCliff)
				return false
			}
			p0x, p0y, p0z := face.tri[0].X, face.tri[0].Y, face.tri[0].Z
			nX, nY, nZ := face.normal.X, face.normal.Y, face.normal.Z

			// ==========================================
			// ORIENTAMENTO E ASSEGNAZIONE BUCKET SIMULTANEA
			// ==========================================
			distStart := (cX-p0x)*nX + (cY-p0y)*nY + (cZ-p0z)*nZ
			var bucket BucketType

			if isWall {
				// MURI
				//height := fAABB.GetMaxZ() - fAABB.GetMinZ()
				//fmt.Println("CURRENT HEIGHT", height)

				if distStart < 0 {
					nX, nY, nZ = -nX, -nY, -nZ
					distStart = -distStart
				}
				// Assegnazione bucket per i muri in base alla normale finale
				if wallWE {
					if nX < 0 {
						bucket = BucketWallWest
					} else {
						bucket = BucketWallEast
					}
				} else {
					if nY < 0 {
						bucket = BucketWallNorth
					} else {
						bucket = BucketWallSouth
					}
				}
			} else {
				// PIANI ORIZZONTALI
				planeZ := p0z
				if math.Abs(nZ) > 1e-5 {
					planeZ = p0z - (nX*(cX-p0x)+nY*(cY-p0y))/nZ
				}
				if cZ >= planeZ-maxCliff {
					bucket = BucketFloor // È matematicamente un Pavimento
					if nZ < 0 {
						nX, nY, nZ = -nX, -nY, -nZ
						distStart = -distStart
					}
				} else {
					bucket = BucketCeiling // È matematicamente un Soffitto
					if nZ > 0 {
						nX, nY, nZ = -nX, -nY, -nZ
						distStart = -distStart
					}
				}
			}

			rEff := math.Sqrt((nX*eRadX)*(nX*eRadX) + (nY*eRadY)*(nY*eRadY) + (nZ*eRadZ)*(nZ*eRadZ))
			distTarget := (tX-p0x)*nX + (tY-p0y)*nY + (tZ-p0z)*nZ
			distSurfTarget := distTarget - rEff

			if distSurfTarget > margin {
				//fmt.Println("FILTRO MARGIN ATTIVO, RETURNING", distSurfTarget, margin)
				return false
			}

			fMinX, fMinY, fMinZ := fAABB.GetMinX(), fAABB.GetMinY(), fAABB.GetMinZ()
			fMaxX, fMaxY := fAABB.GetMaxX(), fAABB.GetMaxY()

			if maxX >= fMinX-margin && minX <= fMaxX+margin &&
				maxY >= fMinY-margin && minY <= fMaxY+margin &&
				maxZ >= fMinZ-margin && minZ <= fMaxZ+margin {
				//fmt.Println("###########################")
				//fmt.Printf("OUR %v\n", cage.GetAABB())
				//fmt.Printf("OTHER ID %v\n", face.GetTag())
				//fmt.Printf("OTHER %v\n", otherEnt.GetAABB())
				//fmt.Printf("OTHER triangle %v\n", face.tri)
				//fmt.Printf("%v distSurfTarget %f Eff %f\n", bucket, distSurfTarget, rEff)
				cage.AddFace(bucket, face, distSurfTarget, rEff, nX, nY, nZ)
			} else {
				//fmt.Println("FILTRO OUTSIDE ATTIVO RETURNING", margin)
			}
			return false
		})
	})
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

// QueryAABB performs a spatial query, invoking the callback for each Volume that overlaps with the specified AABB.
func (s *Volumes) QueryAABB(aabb physics.IAABB, callback func(vol *Volume)) {
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			callback(vol)
		}
		return false
	})
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

/*
// QueryClosestFace identifies the closest intersecting face during a swept volume test within the given AABB.
// Returns the closest face, normal vector (colNx, colNy, colNz), and intersection distance (minT).
func (s *Volumes) QueryClosestFace(z physics.IAABB, viewX, viewY, viewZ, velX, velY, velZ, eRadX, eRadY, eRadZ float64) (*Face, float64, float64, float64, float64) {
	var closestFace *Face = nil
	minT := 1.0
	var colNx, colNy, colNz float64
	s.QueryAABB(z, func(vol *Volume) {
		vol.facesTree.QueryOverlaps(z, func(object physics.IAABB) bool {
			face, ok := object.(*Face)
			if !ok {
				return false
			}
			hitT, cNx, cNy, cNz, hit := face.SweepTest(viewX, viewY, viewZ, velX, velY, velZ, eRadX, eRadY, eRadZ)
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

*/
