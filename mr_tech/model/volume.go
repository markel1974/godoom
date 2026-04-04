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
	is3d      bool
	modelId   int
	id        string
	faces     []*Face
	tag       string
	floorY    float64
	ceilY     float64
	materials []*textures.Animation
	Light     *Light
	aabb      *physics.AABB
}

// NewVolume creates a new 2.5D Volume instance with the specified attributes, mimicking legacy extruded sectors.
func NewVolume(modelId int, id string, floorY float64, floor *textures.Animation, ceilY float64, ceil *textures.Animation, tag string) *Volume {
	v := &Volume{
		is3d:      false,
		modelId:   modelId,
		id:        id,
		floorY:    floorY,
		ceilY:     ceilY,
		materials: make([]*textures.Animation, 2),
		tag:       tag,
	}
	v.materials[0] = floor
	v.materials[1] = ceil
	return v
}

// NewVolume3d creates and returns a new true 3D Volume instance (convex polyhedron) with the specified model ID, ID, and tag.
func NewVolume3d(modelId int, id string, tag string) *Volume {
	v := &Volume{
		is3d:      true,
		modelId:   modelId,
		id:        id,
		materials: make([]*textures.Animation, 2),
		tag:       tag,
	}
	return v
}

// Is3d returns true if the volume represents a true 3D space, otherwise false (2.5D extruded).
func (v *Volume) Is3d() bool {
	return v.is3d
}

// GetModelId retrieves the model ID associated with the Volume instance.
func (v *Volume) GetModelId() int {
	return v.modelId
}

// GetId retrieves the unique identifier of the volume.
func (v *Volume) GetId() string {
	return v.id
}

// GetFloorY returns the Y-coordinate of the floor. In 3D mode, it retrieves the minimum Z from the AABB if available.
func (v *Volume) GetFloorY() float64 {
	if v.is3d && v.aabb != nil {
		return v.aabb.GetMinZ()
	}
	return v.floorY
}

// GetCeilY returns the ceiling Y-coordinate of the volume. For 3D volumes with an AABB, it returns the maximum Z value.
func (v *Volume) GetCeilY() float64 {
	if v.is3d && v.aabb != nil {
		return v.aabb.GetMaxZ()
	}
	return v.ceilY
}

// GetFloorMaterial returns the material used for the floor of the volume, based on 3D state and face normals.
func (v *Volume) GetFloorMaterial() *textures.Animation {
	if v.is3d {
		for _, face := range v.faces {
			if face.GetNormal().Z > 0.9 {
				return face.GetMaterial()
			}
		}
		return nil
	}
	return v.materials[0]
}

// GetCeilMaterial returns the material used for the ceiling of the volume. Prioritizes 3D faces if the volume is 3D.
func (v *Volume) GetCeilMaterial() *textures.Animation {
	if v.is3d {
		for _, face := range v.faces {
			if face.GetNormal().Z < -0.9 {
				return face.GetMaterial()
			}
		}
		return nil
	}
	return v.materials[1]
}

// AddFace adds a new face to the volume and sets the volume as the parent of the face.
func (v *Volume) AddFace(face *Face) {
	// Attenzione: Face dovrà avere GetNeighbor che ritorna un *Volume anziché *Sector
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
	if !v.is3d {
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
		if len(v.faces) == 0 {
			minX, minY = 0, 0
		} else {
			for _, face := range v.faces {
				start := face.GetStart()
				end := face.GetEnd()
				if start.X < minX {
					minX = start.X
				}
				if start.X > maxX {
					maxX = start.X
				}
				if start.Y < minY {
					minY = start.Y
				}
				if start.Y > maxY {
					maxY = start.Y
				}
				if end.X < minX {
					minX = end.X
				}
				if end.X > maxX {
					maxX = end.X
				}
				if end.Y < minY {
					minY = end.Y
				}
				if end.Y > maxY {
					maxY = end.Y
				}
			}
		}
		v.aabb = physics.NewAABB(minX, minY, v.floorY, maxX, maxY, v.ceilY)
		return
	}
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	if len(v.faces) == 0 {
		minX, minY, minZ = 0, 0, 0
	} else {
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

// LocatePoint determines the volume containing the given 3D point (px, py, pz) using BSP traversal in a 3D convex space.
func (v *Volume) LocatePoint(px, py, pz float64) *Volume {
	if !v.is3d {
		return v.LocatePoint2D(px, py)
	}
	curr := v
	const maxSteps = 16
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, face := range curr.faces {
			pts := face.GetPoints()
			if len(pts) == 0 {
				continue
			}
			n := face.GetNormal()

			if ((px-pts[0].X)*n.X + (py-pts[0].Y)*n.Y + (pz-pts[0].Z)*n.Z) > 0.001 {
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

// LocatePoint2D attempts to locate the 2D point (px, py) within the mesh and returns the containing Volume or nil.
func (v *Volume) LocatePoint2D(px, py float64) *Volume {
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

// ContainsPoint determines if the given point (px, py, pz) is inside the volume. Works for both 2D and 3D volumes.
func (v *Volume) ContainsPoint(px, py, pz float64) bool {
	if !v.is3d {
		return v.ContainsPoint2D(px, py)
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

// ContainsPoint2D determines if a 2D point (px, py) lies within the bounds of the Volume.
func (v *Volume) ContainsPoint2D(px, py float64) bool {
	for _, face := range v.faces {
		start := face.GetStart()
		end := face.GetEnd()
		if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
			return false
		}
	}
	return true
}

// CheckFacesClearance checks if a movement intersects any face in the volume and returns the closest obstructing face.
func (v *Volume) CheckFacesClearance(viewX, viewY, pX, pY, top float64, bottom float64, radius float64) *Face {
	if v.is3d {
		return nil
	}

	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var closestFace *Face = nil

	for _, face := range v.faces {
		start := face.GetStart()
		end := face.GetEnd()
		dx := end.X - start.X
		dy := end.Y - start.Y
		den := moveX*dy - moveY*dx

		if den == 0 {
			continue
		}

		t := ((start.X-viewX)*dy - (start.Y-viewY)*dx) / den
		u := ((start.X-viewX)*moveY - (start.Y-viewY)*moveX) / den

		uPadding := 0.0
		if radius > 0 {
			faceLenSq := dx*dx + dy*dy
			if faceLenSq > 0 {
				uPadding = radius / math.Sqrt(faceLenSq)
			}
		}

		if t >= 0 && t <= minT && u >= -uPadding && u <= 1+uPadding {
			holeLow := 9e9
			holeHigh := -9e9
			neighbor := face.GetNeighbor()

			if neighbor != nil {
				holeLow = mathematic.MaxF(v.floorY, neighbor.GetFloorY())
				holeHigh = mathematic.MinF(v.ceilY, neighbor.GetCeilY())
			}

			if holeHigh < top || holeLow > bottom {
				minT = t
				closestFace = face
			}
		}
	}
	return closestFace
}

// GetCentroid calculates and returns the geometric centroid of the volume based on its faces and 3D mode.
func (v *Volume) GetCentroid() geometry.XYZ {
	if v.is3d {
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

	signedArea *= 0.5
	if signedArea == 0 {
		start := v.faces[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: v.floorY}
	}

	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: v.floorY,
	}
}
