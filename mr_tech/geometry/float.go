package geometry

import (
	"math"
)

// ClampF restricts a float64 value to a specified range [mi, ma].
func ClampF(a float64, mi float64, ma float64) float64 {
	return min(max(a, mi), ma)
}

// VxsF calculates the 2D cross product of two vectors defined by their components (x0, y0) and (x1, y1).
func VxsF(x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return (x0)*(y1) - (x1)*(y0)
}

// OverlapF returns true if the intervals [a0, a1] and [b0, b1] overlap, otherwise it returns false.
func OverlapF(a0 float64, a1 float64, b0 float64, b1 float64) bool {
	return min(a0, a1) <= max(b0, b1) && min(b0, b1) <= max(a0, a1)
}

// IntersectBoxF determines if two axis-aligned bounding boxes overlap based on their corner coordinates.
func IntersectBoxF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return OverlapF(x0, x1, x2, x3) && OverlapF(y0, y1, y2, y3)
}

// PointInLineDirectionF determines the relative position of point (px, py) to a directed line segment (x0, y0) -> (x1, y1).
// Returns 0 if the point lies on the line, -1 if it's to the left, and 1 if it's to the right.
func PointInLineDirectionF(px, py, x0, y0, x1, y1 float64) float64 {
	o := Orient2D(XY{X: x0, Y: y0}, XY{X: x1, Y: y1}, XY{X: px, Y: py})
	if o == 0 {
		return 0
	}
	if o < 0 {
		return -1
	}
	return 1
}

// IntersectF calculates the intersection point of two lines defined by (x1, y1) to (x2, y2) and (x3, y3) to (x4, y4).
func IntersectF(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64) (float64, float64) {
	x := VxsF(VxsF(x1, y1, x2, y2), (x1)-(x2), VxsF(x3, y3, x4, y4), (x3)-(x4)) / VxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	y := VxsF(VxsF(x1, y1, x2, y2), (y1)-(y2), VxsF(x3, y3, x4, y4), (y3)-(y4)) / VxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	return x, y
}

// IntersectFn calculates the intersection point of two line segments defined by their endpoints.
// Returns the x and y coordinates of the intersection and a boolean indicating if an intersection exists.
func IntersectFn(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64) (float64, float64, bool) {
	d := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if d == 0 {
		return 0, 0, false
	}

	// Use parametric equations to find intersection points
	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / d
	u := ((x1-x3)*(y1-y2) - (y1-y3)*(x1-x2)) / d

	// Check if the intersection lies within both line segments (t and u must be between 0 and 1)
	if t >= 0.0 && t <= 1.0 && u >= 0.0 && u <= 1.0 {
		x := x1 + t*(x2-x1)
		y := y1 + t*(y2-y1)
		return x, y, true
	}

	return 0, 0, false
}

// IntersectLineSegmentsF determines if two 2D line segments intersect using their endpoint coordinates.
func IntersectLineSegmentsF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return IntersectBoxF(x0, y0, x1, y1, x2, y2, x3, y3) &&
		math.Abs(PointInLineDirectionF(x2, y2, x0, y0, x1, y1)+PointInLineDirectionF(x3, y3, x0, y0, x1, y1)) != 2 &&
		math.Abs(PointInLineDirectionF(x0, y0, x2, y2, x3, y3)+PointInLineDirectionF(x1, y1, x2, y2, x3, y3)) != 2
}

// FindMinAndMaxF finds and returns the minimum and maximum values in a slice of float64.
func FindMinAndMaxF(a []float64) (float64, float64) {
	minA := a[0]
	maxA := a[0]
	for _, value := range a {
		if value < minA {
			minA = value
		}
		if value > maxA {
			maxA = value
		}
	}
	return minA, maxA
}

// SwapF swaps the values of two float64 variables and returns the swapped values.
func SwapF(a float64, b float64) (float64, float64) {
	return b, a
}
