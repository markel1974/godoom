package model

import (
	"fmt"

	geometry2 "github.com/markel1974/godoom/engine/polygons/geometry"

	"sort"
)

// segmentData represents detailed information about a line segment in the system, including its coordinates and properties.
type segmentData struct {
	id       string
	point    XY
	kind     int
	neighbor string
	upper    string
	middle   string
	lower    string
	distance float64
	high     bool
}

// ConfigSegment represents a segment of input data with spatial coordinates, type, and associated metadata.
type ConfigSegment struct {
	Parent   string `json:"parent"`
	Id       string `json:"id"`
	Start    XY     `json:"start"`
	End      XY     `json:"end"`
	Kind     int    `json:"Kind"`
	Neighbor string `json:"neighbor"`
	Tag      string `json:"tag"`
	Upper    string `json:"upper"`
	Middle   string `json:"middle"`
	Lower    string `json:"lower"`

	builder map[float64][]*segmentData
}

// NewConfigSegment creates a new ConfigSegment instance with the specified parent, Kind, start, and end coordinates.
func NewConfigSegment(parent string, kind int, s XY, e XY) *ConfigSegment {
	is := &ConfigSegment{
		Parent:   parent,
		Id:       NextUUId(),
		Start:    s,
		End:      e,
		Kind:     kind,
		Neighbor: "",
		Tag:      "",
		Upper:    "",
		Middle:   "",
		Lower:    "",
	}
	return is
}

// Clone creates a new instance of ConfigSegment with the same properties as the original and returns it.
func (is *ConfigSegment) Clone() *ConfigSegment {
	out := NewConfigSegment(is.Parent, is.Kind, is.Start, is.End)
	out.Neighbor = is.Neighbor
	out.Tag = is.Tag
	out.Upper = is.Upper
	out.Middle = is.Middle
	out.Lower = is.Lower

	return out
}

// EqualCoords checks if the start and end coordinates of the current segment match those of the given segment exactly.
func (is *ConfigSegment) EqualCoords(tst *ConfigSegment) bool {
	return is.Start.X == tst.Start.X && is.Start.Y == tst.Start.Y && is.End.X == tst.End.X && is.End.Y == tst.End.Y
}

// SameCoords checks if the given ConfigSegment has the same start and end coordinates as the current segment, regardless of order.
func (is *ConfigSegment) SameCoords(tst *ConfigSegment) bool {
	a := is.Start.X == tst.Start.X && is.Start.Y == tst.Start.Y && is.End.X == tst.End.X && is.End.Y == tst.End.Y
	b := is.Start.X == tst.End.X && is.Start.Y == tst.End.Y && is.End.X == tst.Start.X && is.End.Y == tst.Start.Y
	return a || b
}

// AnyCoords checks if any coordinates of the current segment match those of the provided segment, excluding same segments.
func (is *ConfigSegment) AnyCoords(tst *ConfigSegment) bool {
	if is.SameCoords(tst) {
		return false
	}
	return is.Start == tst.Start || is.End == tst.End || is.Start == tst.End || is.End == tst.Start
}

// SegmentDataWall represents the wall segment data identifier.
// SegmentDataTexture represents the texture segment data identifier.
// SegmentDataNeighbor represents the neighboring segment data identifier.
const (
	SegmentDataWall     = 0
	SegmentDataTexture  = iota
	SegmentDataNeighbor = iota
)

// Prepare initializes the builder map to group segment data by their respective distances.
func (is *ConfigSegment) Prepare() {
	is.builder = map[float64][]*segmentData{}
}

// AddNeighbor adds two Points with the specified neighbor identifier to the segment's builder as neighbor data.
func (is *ConfigSegment) AddNeighbor(p0 XY, p1 XY, neighbor string) {
	id := NextUUId()
	is.createPoint(id, p0, SegmentDataNeighbor, neighbor, "", "", "")
	is.createPoint(id, p1, SegmentDataNeighbor, neighbor, "", "", "")
}

// AddProperty adds a property between two Points with specified wall status, upper, middle, and lower textures.
func (is *ConfigSegment) AddProperty(p0 XY, p1 XY, wall bool, upper string, middle string, lower string) {
	var kind int
	if wall {
		kind = SegmentDataWall
	} else {
		kind = SegmentDataTexture
	}
	id := NextUUId()
	is.createPoint(id, p0, kind, "", upper, middle, lower)
	is.createPoint(id, p1, kind, "", upper, middle, lower)
}

// createPoint adds a new point to the builder map with specified metadata and computes its distance from the start point.
func (is *ConfigSegment) createPoint(id string, p0 XY, kind int, neighbor string, upper string, middle string, lower string) {
	pb := geometry2.Point{X: is.Start.X, Y: is.Start.Y}

	sd0 := &segmentData{id: id, point: p0, kind: kind, neighbor: neighbor, upper: upper, middle: middle, lower: lower}
	sd0.distance = geometry2.Distance(pb, geometry2.Point{X: p0.X, Y: p0.Y})
	if c, ok := is.builder[sd0.distance]; ok {
		c = append(c, sd0)
		is.builder[sd0.distance] = c
	} else {
		is.builder[sd0.distance] = []*segmentData{sd0}
	}
}

// Build processes and constructs a list of ConfigSegment objects from the current ConfigSegment instance.
func (is *ConfigSegment) Build() []*ConfigSegment {
	x := is.Start.X - is.End.X
	y := is.Start.Y - is.End.Y
	builderNegative := false
	if x < 0 || y < 0 {
		builderNegative = true
	}

	type sorter struct {
		distance float64
		data     []*segmentData
	}
	var builder []*sorter

	for k, v := range is.builder {
		builder = append(builder, &sorter{distance: k, data: v})
	}

	sort.SliceStable(builder, func(i, j int) bool {
		if !builderNegative {
			return builder[i].distance < builder[j].distance
		} else {
			return builder[i].distance > builder[j].distance
		}
	})

	var out []*ConfigSegment
	var wall *ConfigSegment = nil
	var texture *ConfigSegment = nil
	var neighbor *ConfigSegment = nil

	closeSegment := func(is *ConfigSegment, kind int, data []*segmentData) ([]*segmentData, *segmentData) {
		for i, d := range data {
			if d.kind == kind {
				if is.Id == d.id {
					is.End = d.point
					data[i] = data[len(data)-1]
					data = data[:len(data)-1]
					return data, d
				}
			}
		}
		return nil, nil
	}

	createSegment := func(kind int, data []*segmentData) ([]*segmentData, *segmentData) {
		for i, d := range data {
			if d.kind == kind {
				data[i] = data[len(data)-1]
				data = data[:len(data)-1]
				return data, d
			}
		}
		return nil, nil
	}

	for _, b := range builder {
		if wall != nil {
			if data, d := closeSegment(wall, SegmentDataWall, b.data); d != nil {
				out = append(out, wall)
				if neighbor != nil {
					neighbor.Start = wall.End
				}
				wall = nil
				b.data = data
			}
		}

		if wall == nil {
			if data, d := createSegment(SegmentDataWall, b.data); d != nil {
				wall = NewConfigSegment(is.Parent, DefinitionWall, d.point, XY{})
				wall.Id = d.id
				b.data = data
			}
		}

		if texture != nil {
			if data, d := closeSegment(texture, SegmentDataTexture, b.data); d != nil {
				texture = nil
				b.data = data
			}
		}

		if texture == nil {
			if data, d := createSegment(SegmentDataTexture, b.data); d != nil {
				texture = NewConfigSegment(is.Parent, SegmentDataTexture, d.point, XY{})
				texture.Id = d.id
				texture.Upper = d.upper
				texture.Middle = d.middle
				texture.Lower = d.lower
				b.data = data
			}
		}

		if neighbor != nil {
			if data, d := closeSegment(neighbor, SegmentDataNeighbor, b.data); d != nil {
				if wall == nil {
					out = append(out, neighbor)
				}
				neighbor = nil
				b.data = data
			}
		}

		if neighbor == nil {
			if data, d := createSegment(SegmentDataNeighbor, b.data); d != nil {
				neighbor = NewConfigSegment(is.Parent, DefinitionJoin, d.point, XY{})
				neighbor.Id = d.id
				neighbor.Neighbor = d.neighbor

				if texture != nil {
					neighbor.Upper = texture.Upper
					neighbor.Middle = texture.Middle
					neighbor.Lower = texture.Lower
				}
				b.data = data
			}
		}
	}

	if len(out) == 0 {
		out = append(out, is.Clone())
	}

	for _, r := range out {
		neighborP := r.Neighbor
		if r.Kind == DefinitionWall {
			neighborP = "wall"
		}
		fmt.Println(neighborP, r.Start, r.End, r.Upper, r.Middle, r.Lower)
	}

	return out
}
