package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D region or area defined by geometric faces, materials, and associated properties.
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

// NewSector creates a new Sector instance with the specified attributes, including geometry and material properties.
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

// NewSector3d creates and returns a new 3D Sector instance with the specified model ID, ID, and tag.
func NewSector3d(modelId int, id string, tag string) *Sector {
	s := &Sector{
		is3d:      true,
		modelId:   modelId,
		id:        id,
		materials: make([]*textures.Animation, 2),
		tag:       tag,
	}
	return s
}

// Is3d returns true if the sector is a 3D sector, otherwise false.
func (s *Sector) Is3d() bool {
	return s.is3d
}

// GetModelId retrieves the model ID associated with the Sector instance.
func (s *Sector) GetModelId() int {
	return s.modelId
}

// GetId retrieves the unique identifier of the sector.
func (s *Sector) GetId() string {
	return s.id
}

// GetFloorY returns the Y-coordinate of the floor. In 3D mode, it retrieves the minimum Z from the AABB if available.
func (s *Sector) GetFloorY() float64 {
	if s.is3d && s.aabb != nil {
		return s.aabb.GetMinZ()
	}
	return s.floorY
}

// GetCeilY returns the ceiling Y-coordinate of the sector. For 3D sectors with an AABB, it returns the maximum Z value.
func (s *Sector) GetCeilY() float64 {
	if s.is3d && s.aabb != nil {
		return s.aabb.GetMaxZ()
	}
	return s.ceilY
}

// GetFloorMaterial returns the material used for the floor of the sector, based on 3D state and face normals.
func (s *Sector) GetFloorMaterial() *textures.Animation {
	if s.is3d {
		for _, face := range s.faces {
			if face.GetNormal().Z > 0.9 {
				return face.GetMaterial()
			}
		}
		return nil
	}
	return s.materials[0]
}

// GetCeilMaterial returns the material used for the ceiling of the sector. Prioritizes 3D faces if the sector is 3D.
func (s *Sector) GetCeilMaterial() *textures.Animation {
	if s.is3d {
		for _, face := range s.faces {
			if face.GetNormal().Z < -0.9 {
				return face.GetMaterial()
			}
		}
		return nil
	}
	return s.materials[1]
}

// AddFace adds a new face to the sector and sets the sector as the parent of the face.
func (s *Sector) AddFace(face *Face) {
	face.SetParent(s)
	s.faces = append(s.faces, face)
}

// GetFaces retrieves the list of face objects associated with the sector.
func (s *Sector) GetFaces() []*Face {
	return s.faces
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the sector, representing its 3D bounds.
func (s *Sector) GetAABB() *physics.AABB {
	return s.aabb
}

// Rebuild recalculates the axis-aligned bounding box (AABB) for the sector based on its faces and dimensions.
func (s *Sector) Rebuild() {
	if !s.is3d {
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
		if len(s.faces) == 0 {
			minX, minY = 0, 0
		} else {
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
		}
		s.aabb = physics.NewAABB(minX, minY, s.floorY, maxX, maxY, s.ceilY)
		return
	}
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	if len(s.faces) == 0 {
		minX, minY, minZ = 0, 0, 0
	} else {
		for _, face := range s.faces {
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
	s.aabb = physics.NewAABB(minX, minY, minZ, maxX, maxY, maxZ)
}

// AddTag appends the specified tags to the sector's existing tags, separated by a semicolon, if the input is non-empty.
func (s *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		s.tag += ";" + tags
	}
}

// GetTag retrieves the tag string associated with the Sector instance.
func (s *Sector) GetTag() string {
	return s.tag
}

// LocatePoint determines the sector containing the given 3D point (px, py, pz) using BSP traversal in a 3D convex space.
func (s *Sector) LocatePoint(px, py, pz float64) *Sector {
	if !s.is3d {
		return s.LocatePoint2D(px, py)
	}
	// Half-Space BSP traversal per i volumi convessi 3D
	curr := s
	const maxSteps = 16
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, face := range curr.faces {
			pts := face.GetPoints()
			if len(pts) == 0 {
				continue
			}
			n := face.GetNormal()

			// Se (P - P0) dot N > eps, il punto è "davanti" alla faccia (esterno al volume convesso)
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

// LocatePoint2D attempts to locate the 2D point (px, py) within the sector mesh and returns the containing Sector or nil if none.
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

// ContainsPoint determines if the given point (px, py, pz) is inside the sector. Works for both 2D and 3D sectors.
func (s *Sector) ContainsPoint(px, py, pz float64) bool {
	if !s.is3d {
		return s.ContainsPoint2D(px, py)
	}
	for _, face := range s.faces {
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

// ContainsPoint2D determines if a 2D point (px, py) lies within the convex 2D bounds of the Sector.
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

// CheckFacesClearance checks if a movement intersects any face in the sector and returns the closest obstructing face.
func (s *Sector) CheckFacesClearance(viewX, viewY, pX, pY, top float64, bottom float64, radius float64) *Face {
	if s.is3d {
		// Nel vero 3D, la clearance è gestita con l'ellissoide di collisione contro i piani (AABB Sweeping).
		return nil
	}

	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var closestFace *Face = nil

	for _, face := range s.faces {
		neighbor := face.GetNeighbor()
		if neighbor != nil {
			continue
		}

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
			if neighbor != nil {
				holeLow = mathematic.MaxF(s.floorY, neighbor.GetFloorY())
				holeHigh = mathematic.MinF(s.ceilY, neighbor.GetCeilY())
			}
			if holeHigh < top || holeLow > bottom {
				minT = t
				closestFace = face
			}
		}
	}
	return closestFace
}

// GetCentroid calculates and returns the geometric centroid of the sector based on its faces and 3D mode.
func (s *Sector) GetCentroid() geometry.XYZ {
	if s.is3d {
		var cx, cy, cz, count float64
		for _, face := range s.faces {
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
	for i := range s.faces {
		start := s.faces[i].GetStart()
		end := s.faces[i].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		a := (x0 * y1) - (x1 * y0)
		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}

	signedArea *= 0.5
	if signedArea == 0 {
		start := s.faces[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: s.floorY}
	}

	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: s.floorY,
	}
}
