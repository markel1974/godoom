package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D space defined by its boundaries, texture animations, lighting, and associated metadata.
type Sector struct {
	ModelId  uint16
	Id       string
	Segments []*Face
	Tag      string
	floorY   float64
	ceilY    float64
	Ceil     *textures.Animation
	Floor    *textures.Animation
	Light    *Light
	aabb     *physics.AABB
}

// NewSector initializes and returns a new Sector instance with specified parameters including model ID, ID, segments, floor, and ceiling.
func NewSector(modelId uint16, id string, floorY float64, floor *textures.Animation, ceilY float64, ceil *textures.Animation, tag string) *Sector {
	s := &Sector{
		ModelId: modelId,
		Id:      id,
		floorY:  floorY,
		ceilY:   ceilY,
		Ceil:    ceil,
		Floor:   floor,
		Tag:     tag,
	}
	return s
}

// GetFloorY returns the Y-coordinate of the floor for the Sector instance as a float64.
func (s *Sector) GetFloorY() float64 {
	return s.floorY
}

// GetCeilY returns the Y-coordinate of the ceiling for the sector.
func (s *Sector) GetCeilY() float64 {
	return s.ceilY
}

// AddSegment appends a Face to the Sector and assigns the Sector as the Face's Parent.
func (s *Sector) AddSegment(seg *Face) {
	seg.SetParent(s)
	s.Segments = append(s.Segments, seg)
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the Sector instance.
func (s *Sector) GetAABB() *physics.AABB {
	return s.aabb
}

// ComputeAABB calculates and updates the axis-aligned bounding box (AABB) for the sector based on its segments.
func (s *Sector) ComputeAABB() {
	if len(s.Segments) == 0 {
		s.aabb = physics.NewAABB(0, 0, 0, 0, 0, 0)
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, seg := range s.Segments {
		start := seg.GetStart()
		end := seg.GetEnd()
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

func (s *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		s.Tag += ";" + tags
	}
}

// LocatePoint identifies the Sector containing the point (px, py) by traversing convex polygons linked via Segments.
func (s *Sector) LocatePoint(px, py float64) *Sector {
	curr := s
	const maxSteps = 16 // Safeguard for infinite loops caused by floating-point approximations
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, seg := range curr.Segments {
			start := seg.GetStart()
			end := seg.GetEnd()
			// Assuming that < 0 indicates the "external" half-space of the edge
			if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
				neighbor := seg.GetNeighbor()
				if neighbor == nil {
					// Hit external boundary of the mesh
					return nil
				}
				// Transition: the point is beyond this segment, jump to the neighbor
				curr = neighbor
				inside = false
				break
			}
		}
		// If the point was not outside any segment,
		// by definition it is inside the current convex polygon.
		if inside {
			return curr
		}
	}
	// Walk limit exceeded (possible ping-pong between sectors due to FP edge-cases)
	return nil
}

// ContainsPoint performs a rigorous Point-in-Polygon test for convex polygons.
func (s *Sector) ContainsPoint(px, py float64) bool {
	for _, seg := range s.Segments {
		start := seg.GetStart()
		end := seg.GetEnd()
		if mathematic.PointSideF(px, py, start.X, start.Y, end.X, end.Y) < 0 {
			return false
		}
	}
	return true
}

// CheckSegmentsClearance determines if a line segment intersects with any sector boundary and verifies clearance within head and knee positions.
func (s *Sector) CheckSegmentsClearance(viewX, viewY, pX, pY, top float64, bottom float64, radius float64) *Face {
	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var closestSeg *Face = nil

	for _, seg := range s.Segments {
		//todo verificare neighbor!!!
		neighbor := seg.GetNeighbor()
		if neighbor != nil {
			continue
		}
		//if neighbor != nil {
		//	if top > s.GetCeilY() || bottom < s.GetFloorY() {
		//		continue
		//	}
		//}
		start := seg.GetStart()
		end := seg.GetEnd()
		dx := end.X - start.X
		dy := end.Y - start.Y
		den := moveX*dy - moveY*dx
		if den == 0 {
			continue
		}
		t := ((start.X-viewX)*dy - (start.Y-viewY)*dx) / den
		u := ((start.X-viewX)*moveY - (start.Y-viewY)*moveX) / den

		// Compute padding based on entity radius
		// This virtually extends the segment to close gaps at vertices
		uPadding := 0.0
		if radius > 0 {
			segLenSq := dx*dx + dy*dy
			if segLenSq > 0 {
				uPadding = radius / math.Sqrt(segLenSq)
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
				closestSeg = seg
			}
		}
	}
	return closestSeg
}

// GetCentroid calculates the centroid of the polygon formed by the sector's segments based on their vertex coordinates.
func (s *Sector) GetCentroid() geometry.XY {
	var signedArea, cx, cy float64

	for i := range s.Segments {
		start := s.Segments[i].GetStart()
		end := s.Segments[i].GetEnd()
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
		start := s.Segments[0].GetStart()
		// Fallback di sicurezza per topologia degenere (es. area nulla)
		return geometry.XY{X: start.X, Y: start.Y}
	}

	return geometry.XY{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
	}
}
