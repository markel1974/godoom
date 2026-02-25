package geometry

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/engine/polygons/geometry/errorstree"
)

// Point is store of point coordinates
type Point struct {
	X, Y float64
}

// String is implementation of Stringer implementation for formating output
func (p Point) String() string {
	return fmt.Sprintf("[%.5e,%.5e]", p.X, p.Y)
}

// Check - check input data
func Check(pps ...Point) error {
	et := errorstree.New("Check points")
	for i := range pps {
		if x, y := pps[i].X, pps[i].Y; math.IsNaN(x) || math.IsInf(x, 0) ||
			math.IsNaN(y) || math.IsInf(y, 0) {
			_ = et.Add(fmt.Errorf("Not valid point #%d: (%.5e,%.5e)", i, x, y))
		}
	}
	if et.IsError() {
		return et
	}
	return nil
}

// PointPoint return states between two points.
func PointPoint(pt0 Point, pt1 Point) (pi []Point, stA State, stB State) {
	stA |= ZeroLengthSegment | VerticalSegment | HorizontalSegment
	if SamePoints(pt0, pt1) {
		stA |= OnPoint0Segment | OnPoint1Segment
	}
	stB = stA
	return
}

// PointLine return states between point and line.
func PointLine(pt Point, pb0 Point, pb1 Point) (pi []Point, stA State, stB State) {
	// Point - Point
	if SamePoints(pb0, pb1) {
		return PointPoint(pt, pb0)
	}
	// Point - Line

	stA |= ZeroLengthSegment | VerticalSegment | HorizontalSegment

	for _, c := range [...]struct {
		isTrue bool
		tiA    State
		tiB    State
	}{
		{isTrue: SamePoints(pt, pb0), tiA: OnPoint0Segment | OnPoint1Segment, tiB: OnPoint0Segment},
		{isTrue: SamePoints(pt, pb1), tiA: OnPoint0Segment | OnPoint1Segment, tiB: OnPoint1Segment},
		{isTrue: math.Abs(pb0.X-pb1.X) < Eps, tiB: VerticalSegment},
		{isTrue: math.Abs(pb0.Y-pb1.Y) < Eps, tiB: HorizontalSegment},
	} {
		if c.isTrue {
			stA |= c.tiA
			stB |= c.tiB
		}
	}

	if stB.Has(OnPoint0Segment) || stB.Has(OnPoint1Segment) {
		return
	}

	if stA.Has(OnPoint0Segment) || stA.Has(OnPoint1Segment) {
		return
	}

	if orient := Orientation(pt, pb0, pb1); orient != CollinearPoints {
		// points is not on line
		return
	}

	// is point on line
	if math.Min(pb0.X, pb1.X) <= pt.X && pt.X <= math.Max(pb0.X, pb1.X) &&
		math.Min(pb0.Y, pb1.Y) <= pt.Y && pt.Y <= math.Max(pb0.Y, pb1.Y) {
		stA |= OnPoint0Segment | OnPoint1Segment
		stB |= OnSegment
		pi = []Point{pt}
		return
	}

	return
}
