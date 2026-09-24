package core

import (
	"fmt"
)

// Rect represents a 2D rectangular area defined by two vectors: Min (bottom-left) and Max (top-right).
type Rect struct {
	Min XY
	Max XY
}

// ZR is the zero rectangle, defined as a rectangle with both Min and Max set to ZV.
var ZR = Rect{Min: ZV, Max: ZV}

// R creates a Rect with specified minimum and maximum X and Y coordinates.
func R(minX, minY, maxX, maxY float64) Rect {
	return Rect{
		Min: XY{minX, minY},
		Max: XY{maxX, maxY},
	}
}

// String returns the string representation of the Rect in the format "Rect(Min.X, Min.Y, Max.X, Max.Y)".
func (r Rect) String() string {
	return fmt.Sprintf("Rect(%v, %v, %v, %v)", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

// Norm returns a normalized rectangle with Min and Max values adjusted to ensure that Min <= Max on both axes.
func (r Rect) Norm() Rect {
	return Rect{
		Min: XY{
			min(r.Min.X, r.Max.X),
			min(r.Min.Y, r.Max.Y),
		},
		Max: XY{
			max(r.Min.X, r.Max.X),
			max(r.Min.Y, r.Max.Y),
		},
	}
}

// W returns the width of the rectangle, calculated as the difference between Max.X and Min.X.
func (r Rect) W() float64 {
	return r.Max.X - r.Min.X
}

// H calculates the height of the rectangle by subtracting the minimum Y coordinate from the maximum Y coordinate.
func (r Rect) H() float64 {
	return r.Max.Y - r.Min.Y
}

// Size returns a vector representing the width and height of the rectangle.
func (r Rect) Size() XY {
	return MakeVec(r.W(), r.H())
}

// Area returns the area of the rectangle by multiplying its width and height.
func (r Rect) Area() float64 {
	return r.W() * r.H()
}

// Edges returns the four edges of the rectangle as an array of Line segments, starting from the bottom-left corner.
func (r Rect) Edges() [4]Line {
	corners := r.Vertices()

	return [4]Line{
		{A: corners[0], B: corners[1]},
		{A: corners[1], B: corners[2]},
		{A: corners[2], B: corners[3]},
		{A: corners[3], B: corners[0]},
	}
}

// Vertices returns the four corner points of the rectangle in clockwise order, starting from the bottom-left corner.
func (r Rect) Vertices() [4]XY {
	return [4]XY{
		r.Min,
		MakeVec(r.Min.X, r.Max.Y),
		r.Max,
		MakeVec(r.Max.X, r.Min.Y),
	}
}

// Contains checks whether the vector u lies within or on the boundaries of the rectangle r.
func (r Rect) Contains(u XY) bool {
	return r.Min.X <= u.X && u.X <= r.Max.X && r.Min.Y <= u.Y && u.Y <= r.Max.Y
}

// Union returns the smallest rectangle that contains both the receiver and the provided rectangle.
func (r Rect) Union(s Rect) Rect {
	return R(
		min(r.Min.X, s.Min.X),
		min(r.Min.Y, s.Min.Y),
		max(r.Max.X, s.Max.X),
		max(r.Max.Y, s.Max.Y),
	)
}

// Intersect calculates and returns the intersection of two rectangles. If they don't overlap, it returns an empty rectangle.
func (r Rect) Intersect(s Rect) Rect {
	t := R(
		max(r.Min.X, s.Min.X),
		max(r.Min.Y, s.Min.Y),
		min(r.Max.X, s.Max.X),
		min(r.Max.Y, s.Max.Y),
	)
	if t.Min.X >= t.Max.X || t.Min.Y >= t.Max.Y {
		return ZR
	}
	return t
}

// Intersects returns true if the rectangle r intersects with the rectangle s; otherwise, it returns false.
func (r Rect) Intersects(s Rect) bool {
	return !(s.Max.X <= r.Min.X ||
		s.Min.X >= r.Max.X ||
		s.Max.Y <= r.Min.Y ||
		s.Min.Y >= r.Max.Y)
}

// Resized adjusts the size of the rectangle relative to a given anchor and new size without altering its proportions.
func (r Rect) Resized(anchor, size XY) Rect {
	if r.W()*r.H() == 0 {
		fmt.Println(fmt.Errorf("(%T).Resize: zero area", r))
		return ZR
	}
	fraction := XY{size.X / r.W(), size.Y / r.H()}
	return Rect{
		Min: anchor.Add(r.Min.Sub(anchor).ScaledXY(fraction)),
		Max: anchor.Add(r.Max.Sub(anchor).ScaledXY(fraction)),
	}
}

// ResizedMin returns a new Rect with the same minimum corner as the original and a maximum corner shifted by size.
func (r Rect) ResizedMin(size XY) Rect {
	return Rect{
		Min: r.Min,
		Max: r.Min.Add(size),
	}
}

// Center returns the position of the center of the Rect.
// `rect.Center()` is equivalent to `rect.Anchor(pixel.Anchor.Center)`
func (r Rect) Center() XY {
	return Lerp(r.Min, r.Max, 0.5)
}

// Moved returns the Rect moved (both Min and Max) by the given vector delta.
func (r Rect) Moved(delta XY) Rect {
	return Rect{
		Min: r.Min.Add(delta),
		Max: r.Max.Add(delta),
	}
}

// IntersectLine will return the shortest XY such that if the Rect is moved by the XY returned, the Line and Rect no
// longer intersect.
func (r Rect) IntersectLine(l Line) XY {
	return l.IntersectRect(r).Scaled(-1)
}

// IntersectionPoints returns all the points where the Rect intersects with the line provided.  This can be zero, one or
// two points, depending on the location of the shapes.  The points of intersection will be returned in order of
// closest-to-l.A to closest-to-l.B.
func (r Rect) IntersectionPoints(l Line) []XY {
	pointMap := make(map[XY]struct{})
	for _, edge := range r.Edges() {
		if intersect, ok := l.Intersect(edge); ok {
			pointMap[intersect] = struct{}{}
		}
	}
	points := make([]XY, 0, len(pointMap))
	for point := range pointMap {
		points = append(points, point)
	}
	if len(points) == 2 {
		if points[1].To(l.A).Len() < points[0].To(l.A).Len() {
			return []XY{points[1], points[0]}
		}
	}
	return points
}
