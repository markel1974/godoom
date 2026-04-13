package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Volume represents a 3D navigable space (a region, brush, or room), defined by geometric faces, materials, and associated properties.
type Volume struct {
	modelId   int
	id        string
	faces     []*Face
	tag       string
	materials [2]*textures.Animation
	Light     *Light
	aabb      *physics.AABB
	minZ      float64
	maxZ      float64
	hasZ      bool
}

// NewVolume2d creates a new 2.5D Volume instance with the specified attributes, mimicking legacy extruded volumes.
func NewVolume2d(modelId int, id string, minZ float64, floor *textures.Animation, maxZ float64, ceil *textures.Animation, tag string) *Volume {
	v := &Volume{
		modelId: modelId,
		id:      id,
		tag:     tag,
		minZ:    minZ,
		maxZ:    maxZ,
		hasZ:    true,
	}
	v.materials[0] = floor
	v.materials[1] = ceil
	v.Rebuild()
	return v
}

// NewVolume3d creates and returns a new true 3D Volume instance (convex polyhedron) with the specified model ID, ID, and tag.
func NewVolume3d(modelId int, id string, tag string) *Volume {
	v := &Volume{
		modelId: modelId,
		id:      id,
		tag:     tag,
		minZ:    0,
		maxZ:    0,
		hasZ:    false,
	}
	v.materials[0] = nil
	v.materials[1] = nil
	v.Rebuild()
	return v
}

// SetZ sets the minimum and maximum Z coordinates for the volume, marks it as having custom Z bounds, and rebuilds its AABB.
func (v *Volume) SetZ(minZ, maxZ float64) {
	v.minZ = minZ
	v.maxZ = maxZ
	v.hasZ = true
	v.Rebuild()
}

// ClearZ resets the Z-coordinate bounds of the volume, marks it as lacking custom Z bounds, and triggers a rebuild.
func (v *Volume) ClearZ() {
	v.minZ = 0
	v.maxZ = 0
	v.hasZ = false
	v.Rebuild()
}

// GetModelId retrieves the model ID associated with the Volume instance.
func (v *Volume) GetModelId() int {
	return v.modelId
}

// GetId retrieves the unique identifier of the volume.
func (v *Volume) GetId() string {
	return v.id
}

// GetMinZ retrieves the minimum Z-coordinate of the volume's axis-aligned bounding box (AABB).
func (v *Volume) GetMinZ() float64 {
	return v.aabb.GetMinZ()
}

// GetMaxZ retrieves the maximum Z-coordinate of the Volume's axis-aligned bounding box (AABB).
func (v *Volume) GetMaxZ() float64 {
	return v.aabb.GetMaxZ()
}

// GetMaterialFloor returns the material used for the floor of the volume, based on 3D state and face normals.
func (v *Volume) GetMaterialFloor() *textures.Animation {
	return v.materials[0]
}

// GetMaterialCeil returns the material used for the ceiling of the volume. Prioritizes 3D faces if the volume is 3D.
func (v *Volume) GetMaterialCeil() *textures.Animation {
	return v.materials[1]
}

// AddFace adds a new face to the volume and sets the volume as the parent of the face.
func (v *Volume) AddFace(face *Face) {
	face.SetParent(v)
	v.faces = append(v.faces, face)
}

// GetFaces retrieves the list of face objects associated with the volume.
func (v *Volume) GetFaces() []*Face {
	return v.faces
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the volume, representing its 3D bounds.
func (v *Volume) GetAABB() *physics.AABB {
	return v.aabb
}

// Rebuild recalculates the axis-aligned bounding box (AABB) for the volume based on its faces and dimensions.
func (v *Volume) Rebuild() {
	if v.hasZ {
		for _, face := range v.faces {
			face.SetZ(v.minZ, v.maxZ)
		}
	}
	var minX, minY, minZ float64
	var maxX, maxY, maxZ float64
	if len(v.faces) > 0 {
		minX, minY, minZ = math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ = -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

		for _, face := range v.faces {
			for _, p := range face.GetPoints() {
				if p.X < minX {
					minX = p.X
				}
				if p.X > maxX {
					maxX = p.X
				}
				if p.Y < minY {
					minY = p.Y
				}
				if p.Y > maxY {
					maxY = p.Y
				}
				if p.Z < minZ {
					minZ = p.Z
				}
				if p.Z > maxZ {
					maxZ = p.Z
				}
			}
		}
	}
	if v.hasZ {
		minZ = v.minZ
		maxZ = v.maxZ
	}
	v.aabb = physics.NewAABB(minX, minY, minZ, maxX, maxY, maxZ)
}

// AddTag appends the specified tags to the volume's existing tags, separated by a semicolon.
func (v *Volume) AddTag(tags string) {
	if len(tags) > 0 {
		v.tag += ";" + tags
	}
}

// GetTag retrieves the tag string associated with the Volume instance.
func (v *Volume) GetTag() string {
	return v.tag
}

// LocatePoint3d determines the volume containing the given 3D point (px, py, pz) using BSP traversal in a 3D convex space.
func (v *Volume) LocatePoint3d(px, py, baseZ, topZ, maxStep float64) *Volume {
	if v.ContainsPoint2d(px, py) {
		if v.IsValidZ(baseZ, topZ, maxStep) {
			return v
		}
		return nil
	}
	for _, face := range v.GetFaces() {
		neighbor := face.GetNeighbor()
		if neighbor != nil {
			if neighbor.ContainsPoint2d(px, py) {
				if neighbor.IsValidZ(baseZ, topZ, maxStep) {
					return neighbor
				}
			}
		}
	}
	return nil
}

// IsValidZ verifica che la quota Z sia compatibile con il settore, gestendo i soffitti a cielo aperto.
func (v *Volume) IsValidZ(baseZ, topZ, maxStep float64) bool {
	minZ := v.GetMinZ()
	maxZ := v.GetMaxZ()
	// 1. Gestione soffitti a cielo aperto
	if maxZ <= minZ {
		maxZ = math.MaxFloat64
	}
	// 2. Controllo Pavimento (L'entità può scavalcare questo dislivello?)
	// Se baseZ è maggiore di floor (es. stiamo cadendo o saltando), la condizione è ampiamente soddisfatta.
	if baseZ+maxStep < minZ {
		return false
	}
	// 3. Controllo Soffitto (C'è spazio sufficiente per l'altezza totale?)
	// Calcoliamo la quota base attesa (il massimo tra la nostra Z e il pavimento del nuovo settore)
	expectedBase := math.Max(baseZ, minZ)
	entityHeight := topZ - baseZ
	if expectedBase+entityHeight > maxZ {
		return false
	}
	return true
}

// LocatePoint2d attempts to locate the 2D point (px, py) within the mesh and returns the containing Volume or nil.
func (v *Volume) LocatePoint2d(px, py float64) *Volume {
	curr := v
	const maxSteps = 16
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, face := range curr.faces {
			start := face.GetStart()
			end := face.GetEnd()
			if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
				neighbor := face.GetNeighbor()
				if neighbor == nil {
					return nil
				}
				curr = neighbor
				inside = false
				break
			}
		}
		if inside {
			return curr
		}
	}
	return nil
}

// ContainsPoint3d determines if the given point (px, py, pz) is inside the volume. Works for both 2D and 3D volumes.
func (v *Volume) ContainsPoint3d(px, py, pz float64) bool {
	// Validazione dei limiti Z per volumi estrusi (2.5D)
	if v.hasZ && (pz < v.minZ || pz > v.maxZ) {
		return false
	}

	for _, face := range v.faces {
		pts := face.GetPoints()
		if len(pts) == 0 {
			continue
		}
		n := face.GetNormal()
		if ((px-pts[0].X)*n.X + (py-pts[0].Y)*n.Y + (pz-pts[0].Z)*n.Z) > 0.001 {
			return false
		}
	}
	return true
}

// ContainsPoint2d determines if a 2D point (px, py) lies within the bounds of the Volume.
func (v *Volume) ContainsPoint2d(px, py float64) bool {
	for _, face := range v.faces {
		start := face.GetStart()
		end := face.GetEnd()
		if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
			return false
		}
	}
	return true
}

// GetCentroid3d calculates and returns the geometric centroid of the volume based on its faces and 3D mode.
func (v *Volume) GetCentroid3d() geometry.XYZ {
	var cx, cy, cz, count float64
	for _, face := range v.faces {
		for _, p := range face.GetPoints() {
			cx += p.X
			cy += p.Y
			cz += p.Z
			count++
		}
	}
	if count > 0 {
		return geometry.XYZ{X: cx / count, Y: cy / count, Z: cz / count}
	}
	return geometry.XYZ{}
}

// GetCentroid2d calculates and returns the 2D centroid of the volume projected onto the XY plane.
func (v *Volume) GetCentroid2d() geometry.XYZ {
	var signedArea, cx, cy float64
	for i := range v.faces {
		start := v.faces[i].GetStart()
		end := v.faces[i].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		a := (x0 * y1) - (x1 * y0)
		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}
	floorY := v.GetMinZ()
	signedArea *= 0.5
	if signedArea == 0 {
		start := v.faces[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: floorY}
	}
	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: floorY,
	}
}
