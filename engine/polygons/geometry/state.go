package geometry

import "fmt"

// State is result of intersection
type State int64

const (
	empty State = 1 << iota

	// VerticalSegment return if segment is vertical
	VerticalSegment

	// HorizontalSegment return if segment is horizontal
	HorizontalSegment

	// ZeroLengthSegment return for zero length segment
	ZeroLengthSegment

	// Segment A and segment B are parallel.
	// Intersection point data is not valid.
	Parallel

	// Collinear return if:
	// Segment A and segment B are collinear.
	// Intersection point data is not valid.
	Collinear

	// OnSegment is intersection point on segment
	OnSegment

	// OnPoint0Segment intersection point on point 0 segment
	OnPoint0Segment

	// OnPoint1Segment intersection point on point 1 segment
	OnPoint1Segment

	// ArcIsLine return only if wrong arc is line
	ArcIsLine

	// ArcIsPoint return only if wrong arc is point
	ArcIsPoint

	// last unused type
	endType
)

var stateList = [...]string{
	"empty",
	"VerticalSegment",
	"HorizontalSegment",
	"ZeroLengthSegment",
	"Parallel",
	"Collinear",
	"OnSegment",
	"OnPoint0Segment",
	"OnPoint1Segment",
	"ArcIsLine",
	"ArcIsPoint",
	"endtype",
}

// Has is mean s-State has si-State
func (s State) Has(si State) bool {
	return s&si != 0
}

// Not mean s-State have not si-State
func (s State) Not(si State) bool {
	return s&si == 0
}

// String is implementation of Stringer implementation for formating output
func (s State) String() string {
	var out string
	var size int
	for i := 0; i < 64; i++ {
		if endType == 1<<i {
			size = i
			break
		}
	}
	for i := 1; i < size; i++ {
		si := State(1 << i)
		out += fmt.Sprintf("%2d\t%30s\t", i, stateList[i])
		if s.Has(si) {
			out += "found"
		} else {
			out += "not found"
		}
		out += "\n"
	}
	return out
}
