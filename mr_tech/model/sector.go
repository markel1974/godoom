package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D navigable space, defined by its geometry, boundaries, materials, lighting, and spatial limits.
type Sector struct {
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

// NewSector creates and initializes a Sector object with specified parameters and materials for floor and ceiling.
func NewSector(modelId int, id string, floorY float64, floor *textures.Animation, ceilY float64, ceil *textures.Animation, tag string) *Sector {
	s := &Sector{
		is3d:      false,
		modelId:   modelId,
		id:        id,
		floorY:    floorY,
		ceilY:     ceilY,
		materials: make([]*textures.Animation, 2),
		tag:       tag,
	}
	s.materials[0] = floor
	s.materials[1] = ceil
	return s
}

func NewSector3d(modelId int, id string, tag string) *Sector {
	s := &Sector{
		is3d:      true,
		modelId:   modelId,
		id:        id,
		floorY:    0,
		ceilY:     0,
		materials: make([]*textures.Animation, 2),
		tag:       tag,
	}
	s.materials[0] = nil
	s.materials[1] = nil
	return nil
}

// Is3d checks whether the Sector represents a 3D navigable space and returns true if it does.
func (s *Sector) Is3d() bool {
	return s.is3d
}

// GetModelId returns the model ID associated with the Sector instance.
func (s *Sector) GetModelId() int {
	return s.modelId
}

// GetId returns the unique identifier of the Sector as a string.
func (s *Sector) GetId() string {
	return s.id
}

// GetFloorY returns the Y-coordinate of the sector's floor.
func (s *Sector) GetFloorY() float64 {
	return s.floorY
}

// GetCeilY returns the ceiling height of the sector.
func (s *Sector) GetCeilY() float64 {
	return s.ceilY
}

// GetFloorMaterial retrieves the animation used for the sector's floor material, typically found at index 0 in materials.
func (s *Sector) GetFloorMaterial() *textures.Animation {
	return s.materials[0]
}

// GetCeilMaterial retrieves the material used for the ceiling of the sector, represented as an animated texture.
func (s *Sector) GetCeilMaterial() *textures.Animation {
	return s.materials[1]
}

// AddFace adds a new Face to the Sector and sets the Sector as its parent.
func (s *Sector) AddFace(face *Face) {
	face.SetParent(s)
	s.faces = append(s.faces, face)
}

// GetFaces retrieves the list of faces associated with the sector.
func (s *Sector) GetFaces() []*Face {
	return s.faces
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the sector, representing its spatial boundaries.
func (s *Sector) GetAABB() *physics.AABB {
	return s.aabb
}

// ComputeAABB calculates the axis-aligned bounding box (AABB) for the sector based on its faces and vertical bounds.
func (s *Sector) ComputeAABB() {
	if len(s.faces) == 0 {
		s.aabb = physics.NewAABB(0, 0, 0, 0, 0, 0)
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, face := range s.faces {
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
	s.aabb = physics.NewAABB(minX, minY, s.floorY, maxX, maxY, s.ceilY)
}

// AddTag appends a new tag to the sector's existing tags, separated by a semicolon if the input is non-empty.
func (s *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		s.tag += ";" + tags
	}
}

// GetTag returns the tag associated with the Sector as a string.
func (s *Sector) GetTag() string {
	return s.tag
}

// LocatePoint2D traverses neighboring sectors to locate the sector containing the given 2D point (px, py), or returns nil if outside.
func (s *Sector) LocatePoint2D(px, py float64) *Sector {
	curr := s
	const maxSteps = 16 // Safeguard for infinite loops caused by floating-point approximations
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, face := range curr.faces {
			start := face.GetStart()
			end := face.GetEnd()
			// Assuming that < 0 indicates the "external" half-space of the edge
			if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
				neighbor := face.GetNeighbor()
				if neighbor == nil {
					// Hit external boundary of the mesh
					return nil
				}
				// Transition: the point is beyond this face, jump to the neighbor
				curr = neighbor
				inside = false
				break
			}
		}
		// If the point was not outside any face,
		// by definition it is inside the current convex polygon.
		if inside {
			return curr
		}
	}
	// Walk limit exceeded (possible ping-pong between sectors due to FP edge-cases)
	return nil
}

// ContainsPoint2D checks if the given 2D point (px, py) lies within the sector by evaluating all its face boundaries.
func (s *Sector) ContainsPoint2D(px, py float64) bool {
	for _, face := range s.faces {
		start := face.GetStart()
		end := face.GetEnd()
		if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
			return false
		}
	}
	return true
}

// CheckFacesClearance determines the closest colliding face within a sector given a movement vector and an entity radius.
func (s *Sector) CheckFacesClearance(viewX, viewY, pX, pY, top float64, bottom float64, radius float64) *Face {
	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var closestFace *Face = nil

	for _, face := range s.faces {
		//todo verificare neighbor!!!
		neighbor := face.GetNeighbor()
		if neighbor != nil {
			continue
		}
		//if neighbor != nil {
		//	if top > s.GetCeilY() || bottom < s.GetFloorY() {
		//		continue
		//	}
		//}
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

		// Compute padding based on entity radius
		// This virtually extends the face to close gaps at vertices
		uPadding := 0.0
		if radius > 0 {
			faceLenSq := dx*dx + dy*dy
			if faceLenSq > 0 {
				uPadding = radius / math.Sqrt(faceLenSq)
			}
		}
		// Test with uPadding extension
		if t >= 0 && t <= minT && u >= -uPadding && u <= 1+uPadding {
			holeLow := 9e9
			holeHigh := -9e9
			//todo verificare neighbor!!!
			if neighbor != nil {
				holeLow = mathematic.MaxF(s.floorY, neighbor.floorY)
				holeHigh = mathematic.MinF(s.ceilY, neighbor.ceilY)
			}
			if holeHigh < top || holeLow > bottom {
				minT = t
				closestFace = face
			}
		}
	}
	return closestFace
}

// GetCentroid computes and returns the centroid of the sector as a geometry.XYZ object.
func (s *Sector) GetCentroid() geometry.XYZ {
	var signedArea, cx, cy float64

	for i := range s.faces {
		start := s.faces[i].GetStart()
		end := s.faces[i].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		// Prodotto vettoriale 2D (determinante)
		a := (x0 * y1) - (x1 * y0)

		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}

	signedArea *= 0.5

	if signedArea == 0 {
		start := s.faces[0].GetStart()
		// Fallback di sicurezza per topologia degenere (es. area nulla)
		return geometry.XYZ{X: start.X, Y: start.Y, Z: 0}
	}

	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: 0,
	}
}
