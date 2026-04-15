package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Calibration represents calibration parameters for rendering and spatial configuration.
type Calibration struct {
	OrthoSize  float32
	MapCenterX float32
	MapCenterZ float32
	LightCamY  float32
	ZNearRoom  float32
	ZFarRoom   float32
}

// Volumes represents a collection of 3D or 2D volumes, utilizing a hierarchical spatial structure and caching for efficiency.
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

// Setup constructs a new AABB tree based on the current container and populates it with rebuilt volume objects.
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

// GetCalibration computes calibration details based on the spatial properties of the root node in the AABB tree.
func (s *Volumes) GetCalibration() *Calibration {
	root, ok := s.tree.GetRoot()
	if !ok {
		return nil
	}
	c := &Calibration{}
	// 2. OrthoSize è esattamente la metà dell'asse maggiore
	width := root.GetWidth()
	depth := root.GetDepth()
	if width > depth {
		c.OrthoSize = float32(width / 2.0)
	} else {
		c.OrthoSize = float32(depth / 2.0)
	}
	c.MapCenterX = float32(root.GetMinX() + (width / 2.0))
	c.MapCenterZ = float32(root.GetMinZ() + (depth / 2.0))
	// La telecamera si posiziona appena sopra il punto più alto della mappa
	c.LightCamY = float32(root.GetMaxY()) //+ 2.0
	// Distanze di proiezione relative dalla telecamera
	c.ZNearRoom = 1.0
	c.ZFarRoom = float32(root.GetMaxY() - root.GetMinY())
	return c
}

// GetVolumes returns all Volume objects managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the number of Volume objects contained within the Volumes container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves a list of volumes that overlap with the specified axis-aligned bounding box (AABB).
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

// QueryPoint3d searches for a volume containing the specified 3D point (px, py, pz) and returns the matched Volume, or nil if not found.
func (s *Volumes) QueryPoint3d(px, py, pz float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			if vol.PointInside3d(px, py, pz) {
				target = vol
				return true
			}
		}
		return false
	})
	return target
}

func (s *Volumes) LocateVolume(px, py, pz float64) *Volume {
	if s.fullZ {
		v, _ := s.locateVolume3d(px, py, pz)
		return v
	}
	return s.locateVolume2d(px, py)
}

// LocateVolume2d searches for a 2D point (px, py) within the managed volumes and returns the corresponding Volume, or nil if not found.
func (s *Volumes) locateVolume2d(px, py float64) *Volume {
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

	/*
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

	*/
}

// LocateVolume3d identifies the 3D volume and specific face at the given point (px, py, pz) in world coordinates.
func (s *Volumes) locateVolume3d(px, py, pz float64) (*Volume, *Face) {
	var targetVol *Volume
	var targetFace *Face

	// Broad-Phase Globale: troviamo l'AABB del volume 3D
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		volume, volumeOk := object.(*Volume)
		if !volumeOk {
			return false
		}
		if !volume.PointInside3d(px, py, pz) {
			return false
		}
		// Broad-Phase Locale: cerchiamo la faccia sotto ai piedi
		volume.facesTree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
			face, faceOk := object.(*Face)
			if !faceOk {
				return false
			}
			// Filtro Topologico: Selezioniamo solo le facce che fungono da pavimento (Normal Z negativa)
			if face.GetNormal().Z >= -0.001 {
				return false // Scarta muri e soffitti
			}
			// CRITICO: Usiamo la proiezione 2D!
			// Vogliamo sapere se la coordinata XY del player è sovrapposta al triangolo,
			// ignorando quanto in alto (Z) si trovi la testa del player rispetto al pavimento.
			if face.PointInside2d(px, py) {
				targetVol = volume
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

/*
// LocateVolume3d trova il volume 3D che contiene il punto (px, py, pz) e
// restituisce sia il Volume che la Faccia di riferimento (es. il pavimento sotto al punto).
func (s *Volumes) LocateVolume3d(px, py, pz float64) (*Volume, *Face) {
	var targetVol *Volume
	var targetFace *Face

	// 1. Broad-Phase Globale: troviamo il volume 3D
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		volume, volumeOk := object.(*Volume)
		if !volumeOk {
			return false
		}
		volume.facesTree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
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
				targetVol = volume
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
