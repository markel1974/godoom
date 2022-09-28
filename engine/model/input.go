package model

import (
	"fmt"
	"github.com/markel1974/godoom/engine/geometry"
	"sort"
)


type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}


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

type InputSegment struct {
	Parent   string `json:"id"`
	Id       string `json:"id"`
	Start    XY     `json:"start"`
	End      XY     `json:"end"`
	Kind     int    `json:"kind"`
	Neighbor string `json:"neighbor"`
	Tag      string `json:"tag"`
	Upper    string `json:"upper"`
	Middle   string `json:"middle"`
	Lower    string `json:"lower"`

	builder         map[float64][]*segmentData
}



func NewInputSegment(parent string, kind int, s XY, e XY) * InputSegment {
	is := &InputSegment{
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


func (is * InputSegment) Clone() * InputSegment {
	out := NewInputSegment(is.Parent, is.Kind, is.Start, is.End)
	out.Neighbor = is.Neighbor
	out.Tag = is.Tag
	out.Upper = is.Upper
	out.Middle = is.Middle
	out.Lower = is.Lower

	return out
}

func (is * InputSegment) EqualCoords(tst * InputSegment) bool {
	ret := is.Start.X == tst.Start.X && is.Start.Y == tst.Start.Y && is.End.X == tst.End.X && is.End.Y == tst.End.Y
	return ret
}


const (
	SegmentDataWall = 0
	SegmentDataTexture = iota
	SegmentDataNeighbor = iota
)


func (is * InputSegment) Prepare() {
	is.builder = map[float64][]*segmentData{}
}

func (is * InputSegment) AddNeighbor(p0 XY, p1 XY, neighbor string) {
	id := NextUUId()
	is.createPoint(id, p0, SegmentDataNeighbor, neighbor, "", "", "")
	is.createPoint(id, p1, SegmentDataNeighbor, neighbor, "", "", "")
}

func (is * InputSegment) AddProperty(p0 XY, p1 XY, wall bool, upper string, middle string, lower string) {
	var kind int; if wall { kind = SegmentDataWall } else { kind = SegmentDataTexture }
	id := NextUUId()
	is.createPoint(id, p0, kind, "", upper, middle, lower)
	is.createPoint(id, p1, kind, "", upper, middle, lower)
}

func (is * InputSegment) createPoint(id string, p0 XY, kind int, neighbor string,  upper string, middle string, lower string) {
	pb := geometry.Point{X: is.Start.X, Y: is.Start.Y}

	sd0 := &segmentData{id: id, point: p0, kind: kind, neighbor: neighbor, upper: upper, middle: middle, lower: lower}
	sd0.distance = geometry.Distance(pb, geometry.Point{X: p0.X, Y: p0.Y})
	if c, ok := is.builder[sd0.distance]; ok {
		c = append(c, sd0)
		is.builder[sd0.distance] = c
	} else {
		is.builder[sd0.distance] = []*segmentData{sd0}
	}
}


func (is * InputSegment) Build() []*InputSegment {
	x := is.Start.X - is.End.X
	y := is.Start.Y - is.End.Y
	builderNegative := false
	if x < 0 || y < 0 { builderNegative = true }

	type sorter struct { distance float64; data []*segmentData }
	var builder[]*sorter

	for k, v := range is.builder {
		builder = append(builder, &sorter{ distance: k, data: v })
	}

	sort.SliceStable(builder, func(i, j int) bool {
		if !builderNegative {
			return builder[i].distance < builder[j].distance
		} else {
			return builder[i].distance > builder[j].distance
		}
	})

	var out []*InputSegment
	var wall *InputSegment = nil
	var texture *InputSegment = nil
	var neighbor *InputSegment = nil

	closeSegment := func(is *InputSegment, kind int, data []*segmentData) ([]*segmentData, *segmentData) {
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
				wall = NewInputSegment(is.Parent, DefinitionWall, d.point, XY{})
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
				texture = NewInputSegment(is.Parent, SegmentDataTexture, d.point, XY{})
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
				neighbor = NewInputSegment(is.Parent, DefinitionValid, d.point, XY{})
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
		neighbor := r.Neighbor; if r.Kind == DefinitionWall { neighbor = "wall" }
		fmt.Println(neighbor, r.Start, r.End, r.Upper, r.Middle, r.Lower)
	}

	return out
}




type InputSector struct {
	Id           string           `json:"id"`
	Ceil         float64          `json:"ceil"`
	Floor        float64          `json:"floor"`
	Textures     bool             `json:"textures"`
	FloorTexture string           `json:"floorTexture"`
	CeilTexture  string           `json:"ceilTexture"`
	UpperTexture string           `json:"upperTexture"`
	LowerTexture string           `json:"lowerTexture"`
	WallTexture  string           `json:"wallTexture"`
	Segments     []*InputSegment  `json:"segments"`
	Tag          string           `json:"tag"`
}


func NewInputSector(id string) *InputSector {
	return &InputSector{ Id: id }
}





type InputLight struct {
	Where  XYZ    `json:"where"`
	Light  XYZ    `json:"light"`
	Sector string `json:"sector"`
}

type InputPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"sector"`
}

type Input struct {
	Sectors           []*InputSector `json:"sectors"`
	Lights            []*InputLight  `json:"lights"`
	Player            *InputPlayer   `json:"player"`
	ScaleFactor       float64        `json:"scaleFactor"`
	DisableLoop       bool           `json:"disableLoop"`
}
