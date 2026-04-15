package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
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
	hasFixedZ bool
	facesTree *physics.AABBTree
}

// NewVolume2d creates a new 2.5D Volume instance with the specified attributes, mimicking legacy extruded volumes.
func NewVolume2d(modelId int, id string, minZ float64, floor *textures.Animation, maxZ float64, ceil *textures.Animation, tag string) *Volume {
	v := &Volume{
		modelId:   modelId,
		id:        id,
		tag:       tag,
		minZ:      minZ,
		maxZ:      maxZ,
		hasFixedZ: true,
		aabb:      physics.NewAABB(),
		facesTree: physics.NewAABBTree(64, 0.0),
	}
	v.materials[0] = floor
	v.materials[1] = ceil
	return v
}

// NewVolume3d creates and returns a new true 3D Volume instance (convex polyhedron) with the specified model ID, ID, and tag.
func NewVolume3d(modelId int, id string, tag string) *Volume {
	v := &Volume{
		modelId:   modelId,
		id:        id,
		tag:       tag,
		minZ:      0,
		maxZ:      0,
		hasFixedZ: false,
		aabb:      physics.NewAABB(),
		facesTree: physics.NewAABBTree(64, 0.0),
	}
	v.materials[0] = nil
	v.materials[1] = nil
	return v
}

// Rebuild recalculates the axis-aligned bounding box (AABB) for the volume based on its faces and dimensions.
func (v *Volume) Rebuild() bool {
	minX, minY, calcMinZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, calcMaxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for _, face := range v.faces {
		if v.hasFixedZ {
			face.SetZ(v.minZ, v.maxZ)
		}
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
			if p.Z < calcMinZ {
				calcMinZ = p.Z
			}
			if p.Z > calcMaxZ {
				calcMaxZ = p.Z
			}
		}
	}
	if v.hasFixedZ {
		calcMinZ = v.minZ
		calcMaxZ = v.maxZ
	} else {
		v.minZ = calcMinZ
		v.maxZ = calcMaxZ
	}
	v.aabb.Rebuild(minX, minY, calcMinZ, maxX, maxY, calcMaxZ)

	v.facesTree.Clear()
	for _, face := range v.faces {
		v.facesTree.InsertObject(face)
	}
	return true
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the volume, representing its 3D bounds.
func (v *Volume) GetAABB() *physics.AABB {
	return v.aabb
}

// SetZ sets the minimum and maximum Z coordinates for the volume, marks it as having custom Z bounds, and rebuilds its AABB.
func (v *Volume) SetZ(minZ, maxZ float64) {
	v.minZ = minZ
	v.maxZ = maxZ
	v.hasFixedZ = true
	v.Rebuild()
}

// ClearZ resets the Z-coordinate bounds of the volume, marks it as lacking custom Z bounds, and triggers a rebuild.
func (v *Volume) ClearZ() {
	v.minZ = 0
	v.maxZ = 0
	v.hasFixedZ = false
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

// Neighbor returns the neighboring volume that contains the specified point (px, py, pz), or nil if no such volume exists.
func (v *Volume) Neighbor(px, py, pz float64) *Volume {
	if v.hasFixedZ {
		if v.PointInLineSide(px, py) {
			return v
		}
		for _, face := range v.GetFaces() {
			if neighbor := face.GetNeighbor(); neighbor != nil {
				if neighbor.PointInLineSide(px, py) {
					return neighbor
				}
			}
		}
		return nil
	}
	if v.PointInVolume(px, py, pz) {
		return v
	}
	for _, face := range v.GetFaces() {
		if neighbor := face.GetNeighbor(); neighbor != nil {
			if neighbor.PointInVolume(px, py, pz) {
				return neighbor
			}
		}
	}
	return nil
}

// PointInVolume determines if the given point (px, py, pz) is inside the volume. Works for both 2D and 3D volumes.
func (v *Volume) PointInVolume(px, py, pz float64) bool {
	const epsilon = 0.01
	if v.hasFixedZ {
		if pz < (v.minZ-epsilon) || pz > (v.maxZ+epsilon) {
			return false
		}
		return v.PointInLineSide(px, py)
	}

	for _, face := range v.faces {
		pointInVolume, ok := face.PointInVolume(px, py, pz)
		if !ok {
			continue
		}
		if pointInVolume > epsilon {
			return false
		}
	}
	return true
}

// PointInLineSide checks if the point (px, py) lies on the inner side of all faces' lines within the volume.
func (v *Volume) PointInLineSide(px, py float64) bool {
	for _, face := range v.faces {
		if !face.PointInLineSide(px, py) {
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

/*
// KCC Sweep Target AABB
targetAABB := physics.NewAABB(minX, minY, minZ, maxX, maxY, maxZ)

// 1. Interroga l'albero Globale per trovare in quali volumi stiamo entrando
globalTree.QueryOverlaps(targetAABB, func(volObj physics.IAABB) bool {
    volume := volObj.(*Volume)

    // 2. Interroga l'albero Locale del volume trovato per estrarre SOLO i triangoli vicini
    volume.facesTree.QueryOverlaps(targetAABB, func(faceObj physics.IAABB) bool {
        face := faceObj.(*Face)

        // 3. Narrow-Phase e Sweep
        // Aggiungi la faccia alla lista dei candidati per il test raggio/sfera-triangolo
        // e calcolo del V_slide
        candidates = append(candidates, face)
        return false // Continua la ricerca nell'albero locale
    })
    return false // Continua la ricerca nell'albero globale
})
*/
