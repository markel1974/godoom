package model

import (
	"encoding/json"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D space defined by its boundaries, texture animations, lighting, and associated metadata.
type Sector struct {
	ModelId       uint16
	Id            string
	Segments      []*Segment
	Tag           string
	FloorY        float64
	CeilY         float64
	Ceil          *textures.Animation
	Floor         *textures.Animation
	Light         *Light
	usage         int
	compileId     uint64
	references    map[uint64]bool
	VisibleSpans  [][2]float64
	aabb          *physics.AABB
	LastCompileId uint64
}

// NewSector initializes and returns a new Sector instance with specified parameters including model ID, ID, segments, floor, and ceiling.
func NewSector(modelId uint16, id string, segments []*Segment, floor *textures.Animation, ceil *textures.Animation) *Sector {
	s := &Sector{
		ModelId:    modelId,
		Id:         id,
		CeilY:      0,
		FloorY:     0,
		Segments:   segments,
		usage:      0,
		compileId:  0,
		references: make(map[uint64]bool),
		Ceil:       ceil,
		Floor:      floor,
	}
	return s
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
		if seg.Start.X < minX {
			minX = seg.Start.X
		}
		if seg.Start.X > maxX {
			maxX = seg.Start.X
		}
		if seg.Start.Y < minY {
			minY = seg.Start.Y
		}
		if seg.Start.Y > maxY {
			maxY = seg.Start.Y
		}

		if seg.End.X < minX {
			minX = seg.End.X
		}
		if seg.End.X > maxX {
			maxX = seg.End.X
		}
		if seg.End.Y < minY {
			minY = seg.End.Y
		}
		if seg.End.Y > maxY {
			maxY = seg.End.Y
		}
	}
	s.aabb = physics.NewAABB(minX, minY, s.FloorY, maxX, maxY, s.CeilY)
}

// Reference updates the sector's compile ID if it differs or increments its usage count if it matches.
func (s *Sector) Reference(compileId uint64) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
		s.references = make(map[uint64]bool)
	} else {
		s.usage++
	}
}

// GetCompileId retrieves the unique compile ID associated with the Sector instance.
func (s *Sector) GetCompileId() uint64 {
	return s.compileId
}

// GetUsage retrieves the current usage count for the Sector instance.
func (s *Sector) GetUsage() int {
	return s.usage
}

// Add registers the given ID in the sector's references map by setting its value to true.
func (s *Sector) Add(id uint64) {
	s.references[id] = true
}

// Has checks if the given `id` exists in the `references` map and returns true if it does, otherwise false.
func (s *Sector) Has(id uint64) bool {
	_, ok := s.references[id]
	return ok
}

// IsVisible determines if a range [x1, x2] is not occluded and visible based on the provided identifier id.
func (s *Sector) IsVisible(x1 float64, x2 float64, id uint64) bool {
	if s.LastCompileId != id {
		s.VisibleSpans = s.VisibleSpans[:0]
		s.LastCompileId = id
	}
	for _, span := range s.VisibleSpans {
		// If the span to test is entirely contained within a merged span, it is occluded.
		if x1 >= span[0] && x2 <= span[1] {
			return false
		}
	}
	return true
}

// AddSpan merges a new span defined by x1 and x2 into the VisibleSpans of the Sector, ensuring proper ordering and overlap handling.
func (s *Sector) AddSpan(x1 float64, x2 float64) {
	var merged [][2]float64
	inserted := false

	for _, span := range s.VisibleSpans {
		if inserted {
			merged = append(merged, span)
			continue
		}

		if x2 < span[0] {
			// Insertion on the left (maintains ordering)
			merged = append(merged, [2]float64{x1, x2})
			merged = append(merged, span)
			inserted = true
		} else if x1 > span[1] {
			// No overlap
			merged = append(merged, span)
		} else {
			// Overlap: merge the bounds
			if span[0] < x1 {
				x1 = span[0]
			}
			if span[1] > x2 {
				x2 = span[1]
			}
		}
	}
	if !inserted {
		merged = append(merged, [2]float64{x1, x2})
	}
	s.VisibleSpans = merged
}

// LocatePoint identifies the Sector containing the point (px, py) by traversing convex polygons linked via Segments.
func (s *Sector) LocatePoint(px, py float64) *Sector {
	curr := s
	const maxSteps = 16 // Safeguard for infinite loops caused by floating-point approximations
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, seg := range curr.Segments {
			// Assuming that < 0 indicates the "external" half-space of the edge
			if mathematic.PointSideF(px, py, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y) < 0 {
				if seg.Sector == nil {
					// Hit external boundary of the mesh
					return nil
				}
				// Transition: the point is beyond this segment, jump to the neighbor
				curr = seg.Sector
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
		if mathematic.PointSideF(px, py, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y) < 0 {
			return false
		}
	}
	return true
}

// CheckSegmentsClearance determines if a line segment intersects with any sector boundary and verifies clearance within head and knee positions.
func (s *Sector) CheckSegmentsClearance(viewX, viewY, pX, pY, top float64, bottom float64, radius float64) *Segment {
	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var closestSeg *Segment = nil

	for _, seg := range s.Segments {
		if seg.Kind == config.DefinitionJoin {
			continue
		}
		dx := seg.End.X - seg.Start.X
		dy := seg.End.Y - seg.Start.Y
		den := moveX*dy - moveY*dx

		if den == 0 {
			continue
		}

		t := ((seg.Start.X-viewX)*dy - (seg.Start.Y-viewY)*dx) / den
		u := ((seg.Start.X-viewX)*moveY - (seg.Start.Y-viewY)*moveX) / den

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
			if seg.Sector != nil {
				holeLow = mathematic.MaxF(s.FloorY, seg.Sector.FloorY)
				holeHigh = mathematic.MinF(s.CeilY, seg.Sector.CeilY)
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
		x0, y0 := s.Segments[i].Start.X, s.Segments[i].Start.Y
		x1, y1 := s.Segments[i].End.X, s.Segments[i].End.Y

		// Prodotto vettoriale 2D (determinante)
		a := (x0 * y1) - (x1 * y0)

		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}

	signedArea *= 0.5

	if signedArea == 0 {
		// Fallback di sicurezza per topologia degenere (es. area nulla)
		return geometry.XY{X: s.Segments[0].Start.X, Y: s.Segments[0].Start.Y}
	}

	return geometry.XY{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
	}
}

// Print serializes the Sector into a JSON string, optionally indented, including its segments, floor, and ceiling data.
func (s *Sector) Print(indent bool) string {
	type printerSegment struct {
		Start geometry.XY
		End   geometry.XY
		Ref   string
		Kind  int
		Tag   string
	}
	type printerSector struct {
		ModelId  uint16
		Id       string
		Floor    float64
		Ceil     float64
		Segments []*printerSegment
	}

	p := printerSector{ModelId: s.ModelId, Id: s.Id, Floor: s.FloorY, Ceil: s.CeilY}
	for _, z := range s.Segments {
		ps := &printerSegment{Start: z.Start, End: z.End, Ref: z.Ref, Kind: z.Kind, Tag: z.Tag}
		p.Segments = append(p.Segments, ps)
	}
	if indent {
		d, _ := json.MarshalIndent(p, "", "  ")
		return string(d)
	}
	d, _ := json.Marshal(p)
	return string(d)
}
