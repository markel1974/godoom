package core

import (
	"fmt"
	"math"
)

// Line represents a straight line segment connecting two points in 2D space, defined by its start (A) and end (B) points.
type Line struct {
	A XY
	B XY
}

// MakeLine creates a Line from two XY points, representing the start point (from) and the end point (to).
func MakeLine(from, to XY) Line {
	return Line{
		A: from,
		B: to,
	}
}

// Bounds calculates and returns the smallest rectangle that contains the line segment, normalized to ensure Min ≤ Max.
func (l Line) Bounds() Rect {
	return R(l.A.X, l.A.Y, l.B.X, l.B.Y).Norm()
}

// Center returns the midpoint of the line segment as a XY.
func (l Line) Center() XY {
	return l.A.Add(l.A.To(l.B).Scaled(0.5))
}

// Closest computes the closest point on the line segment to the provided vector `v` and returns it as a vector.
func (l Line) Closest(v XY) XY {
	// between is a helper function which determines whether x is greater than min(a, b) and less than max(a, b)
	between := func(a, b, x float64) bool {
		minV := min(a, b)
		maxV := max(a, b)
		return minV < x && x < maxV
	}

	// Closest point will be on a line which perpendicular to this line.
	// If and only if the infinite perpendicular line intersects the segment.
	m, b := l.Formula()

	// Account for horizontal lines
	if m == 0 {
		x := v.X
		y := l.A.Y

		// check if the X coordinate of v is on the line
		if between(l.A.X, l.B.X, v.X) {
			return MakeVec(x, y)
		}

		// Otherwise get the closest endpoint
		if l.A.To(v).Len() < l.B.To(v).Len() {
			return l.A
		}
		return l.B
	}

	// Account for vertical lines
	if math.IsInf(math.Abs(m), 1) {
		x := l.A.X
		y := v.Y

		// check if the Y coordinate of v is on the line
		if between(l.A.Y, l.B.Y, v.Y) {
			return MakeVec(x, y)
		}

		// Otherwise get the closest endpoint
		if l.A.To(v).Len() < l.B.To(v).Len() {
			return l.A
		}
		return l.B
	}

	perpendicularM := -1 / m
	perpendicularB := v.Y - (perpendicularM * v.X)

	// Coordinates of intersect (of infinite lines)
	x := (perpendicularB - b) / (m - perpendicularM)
	y := m*x + b

	// Check if the point lies between the x and y bounds of the segment
	if !between(l.A.X, l.B.X, x) && !between(l.A.Y, l.B.Y, y) {
		// Not within bounding box
		toStart := v.To(l.A)
		toEnd := v.To(l.B)

		if toStart.Len() < toEnd.Len() {
			return l.A
		}
		return l.B
	}

	return MakeVec(x, y)
}

// Contains checks if the given vector v lies on the line segment defined by the endpoints of the line.
func (l Line) Contains(v XY) bool {
	return l.Closest(v).Eq(v)
}

// Formula returns the slope (m) and y-intercept (b) of the line equation y = mx + b for the current Line.
func (l Line) Formula() (m, b float64) {
	// Account for horizontal lines
	if l.B.Y == l.A.Y {
		return 0, l.A.Y
	}

	m = (l.B.Y - l.A.Y) / (l.B.X - l.A.X)
	b = l.A.Y - (m * l.A.X)

	return m, b
}

// Intersect determines if two line segments intersect and returns the intersection point and a boolean flag.
func (l Line) Intersect(k Line) (XY, bool) {
	// Check if the lines are parallel
	lDir := l.A.To(l.B)
	kDir := k.A.To(k.B)
	if lDir.X == kDir.X && lDir.Y == kDir.Y {
		return ZV, false
	}

	// The lines intersect - but potentially not within the line segments.
	// Get the intersection point for the lines if they were infinitely long, check if the point exists on both of the
	// segments
	lm, lb := l.Formula()
	km, kb := k.Formula()

	// Account for vertical lines
	if math.IsInf(math.Abs(lm), 1) && math.IsInf(math.Abs(km), 1) {
		// Both vertical, therefore parallel
		return ZV, false
	}

	var x, y float64

	if math.IsInf(math.Abs(lm), 1) || math.IsInf(math.Abs(km), 1) {
		// One line is vertical
		intersectM := lm
		intersectB := lb
		verticalLine := k

		if math.IsInf(math.Abs(lm), 1) {
			intersectM = km
			intersectB = kb
			verticalLine = l
		}

		y = intersectM*verticalLine.A.X + intersectB
		x = verticalLine.A.X
	} else {
		// Coordinates of intersect
		x = (kb - lb) / (lm - km)
		y = lm*x + lb
	}

	if l.Contains(MakeVec(x, y)) && k.Contains(MakeVec(x, y)) {
		// The intersect point is on both line segments, they intersect.
		return MakeVec(x, y), true
	}

	return ZV, false
}

/*
// IntersectCircle will return the shortest Vec such that moving the Line by that Vec will cause the Line and Circle
// to no longer intesect.  If they do not intersect at all, this function will return a zero-vector.
func (l Line) IntersectCircle(c Circle) Vec {
	// Get the point on the line closest to the center of the circle.
	closest := l.Closest(c.Center)
	cirToClosest := c.Center.To(closest)

	if cirToClosest.Len() >= c.Radius {
		return ZV
	}

	return cirToClosest.Scaled(cirToClosest.Len() - c.Radius)
}

*/

// IntersectRect calculates the intersection of a line segment and a rectangle and returns the intersection point as a vector.
func (l Line) IntersectRect(r Rect) XY {
	// Check if either end of the line segment are within the rectangle
	if r.Contains(l.A) || r.Contains(l.B) {
		// Use the Rect.Intersect to get minimal return value
		rIntersect := l.Bounds().Intersect(r)
		if rIntersect.H() > rIntersect.W() {
			// Go vertical
			return MakeVec(0, rIntersect.H())
		}
		return MakeVec(rIntersect.W(), 0)
	}

	// Check if any of the rectangles' edges intersect with this line.
	for _, edge := range r.Edges() {
		if _, ok := l.Intersect(edge); ok {
			// Get the closest points on the line to each corner, where:
			//  - the point is contained by the rectangle
			//  - the point is not the corner itself
			corners := r.Vertices()
			var closest *XY
			closestCorner := corners[0]
			for _, c := range corners {
				cc := l.Closest(c)
				if closest == nil || (closest.Len() > cc.Len() && r.Contains(cc)) {
					closest = &cc
					closestCorner = c
				}
			}
			if closest == nil {
				return ZV
			}
			return closest.To(closestCorner)
		}
	}

	// No intersect
	return ZV
}

// Len computes and returns the length of the line segment defined by points A and B.
func (l Line) Len() float64 {
	return l.A.To(l.B).Len()
}

// Moved returns a new Line with both endpoints translated by the given delta vector.
func (l Line) Moved(delta XY) Line {
	return Line{
		A: l.A.Add(delta),
		B: l.B.Add(delta),
	}
}

// Rotated returns a new Line instance by rotating it around a given point by a specified angle in radians.
func (l Line) Rotated(around XY, angle float64) Line {
	// Move the line so we can use `Vec.Rotated`
	lineShifted := l.Moved(around.Scaled(-1))

	lineRotated := Line{
		A: lineShifted.A.Rotated(angle),
		B: lineShifted.B.Rotated(angle),
	}

	return lineRotated.Moved(around)
}

// Scaled returns a new Line scaled proportionally by the given factor, relative to its center.
func (l Line) Scaled(scale float64) Line {
	return l.ScaledXY(l.Center(), scale)
}

// ScaledXY scales the line relative to a given point by the specified factor and returns the resulting line.
func (l Line) ScaledXY(around XY, scale float64) Line {
	toA := around.To(l.A).Scaled(scale)
	toB := around.To(l.B).Scaled(scale)

	return Line{
		A: around.Add(toA),
		B: around.Add(toB),
	}
}

// String returns a string representation of the Line in the format "Line(A, B)".
func (l Line) String() string {
	return fmt.Sprintf("Line(%v, %v)", l.A, l.B)
}
