package mathematic

import (
	"math"
	rnd "math/rand"
	"time"
)

// init initializes the package by seeding the random number generator with the current Unix timestamp in nanoseconds.
func init() {
	rnd.Seed(time.Now().UnixNano())
}

// Random generates a random integer between the specified min (inclusive) and max (inclusive) values.
func Random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

// RandomF returns a random float64 value between the given min and max values (inclusive of min, exclusive of max).
func RandomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

// Abs returns the absolute value of the given integer x. If x is negative, it returns -x; otherwise, it returns x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Min returns the smaller of two integer values a and b.
func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the greater of two integers, a and b.
func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp constrains an integer value `a` to lie between `mi` and `ma`.
func Clamp(a int, mi int, ma int) int {
	return Min(Max(a, mi), ma)
}

// Vxs calculates the 2D cross product of vectors (x0, y0) and (x1, y1), returning the signed area of the parallelogram.
func Vxs(x0 int, y0 int, x1 int, y1 int) int {
	return (x0)*(y1) - (x1)*(y0)
}

// FindMinAndMax returns the minimum and maximum values from the provided slice of integers.
func FindMinAndMax(a []int) (int, int) {
	min := a[0]
	max := a[0]
	for _, value := range a {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return min, max
}

// Swap exchanges the values of two integers and returns them in reversed order.
func Swap(a int, b int) (int, int) {
	return b, a
}

// Overlap checks if two ranges [a0, a1] and [b0, b1] overlap by comparing their minimum and maximum boundaries.
func Overlap(a0 int, a1 int, b0 int, b1 int) bool {
	return Min(a0, a1) <= Max(b0, b1) && Min(b0, b1) <= Max(a0, a1)
}

// IntersectBox determines if two rectangular boxes overlap based on their edge coordinates. Returns true if overlapping.
func IntersectBox(x0 int, y0 int, x1 int, y1 int, x2 int, y2 int, x3 int, y3 int) bool {
	return Overlap(x0, x1, x2, x3) && Overlap(y0, y1, y2, y3)
}

// PointSide determines which side of a line segment a point lies on using vector cross product.
func PointSide(px int, py int, x0 int, y0 int, x1 int, y1 int) int {
	return Vxs(x1-x0, y1-y0, px-x0, py-y0)
}

// MinF returns the smaller of two float64 values, a and b.
func MinF(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MaxF returns the greater of two float64 values, a and b. If a is greater than b, it returns a; otherwise, it returns b.
func MaxF(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ClampF restricts a float64 value to a specified range [mi, ma].
func ClampF(a float64, mi float64, ma float64) float64 {
	return MinF(MaxF(a, mi), ma)
}

// VxsF calculates the 2D cross product of two vectors defined by their components (x0, y0) and (x1, y1).
func VxsF(x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return (x0)*(y1) - (x1)*(y0)
}

// OverlapF returns true if the intervals [a0, a1] and [b0, b1] overlap, otherwise it returns false.
func OverlapF(a0 float64, a1 float64, b0 float64, b1 float64) bool {
	return MinF(a0, a1) <= MaxF(b0, b1) && MinF(b0, b1) <= MaxF(a0, a1)
}

// IntersectBoxF determines if two axis-aligned bounding boxes overlap based on their corner coordinates.
func IntersectBoxF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return OverlapF(x0, x1, x2, x3) && OverlapF(y0, y1, y2, y3)
}

// PointSideF returns a value indicating the direction of a point (px, py) relative to a line defined by (x0, y0) and (x1, y1).
func PointSideF(px float64, py float64, x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return VxsF(x1-x0, y1-y0, px-x0, py-y0)
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
	pre := (x1 * y2) - (y1 * x2)
	post := (x3 * y4) - (y3 * x4)
	x := (pre*(x3-x4) - (x1-x2)*post) / d
	y := (pre*(y3-y4) - (y1-y2)*post) / d
	//if x < minF(x1, x2) || x > maxF(x1, x2) || x < minF(x3, x4) || x > maxF(x3, x4) { return 0, 0, false }
	//if y < minF(y1, y2) || y > maxF(y1, y2) || y < minF(y3, y4) || y > maxF(y3, y4) { return 0, 0, false }
	return x, y, true
}

// IntersectLineSegmentsF determines if two 2D line segments intersect using their endpoint coordinates.
func IntersectLineSegmentsF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return IntersectBoxF(x0, y0, x1, y1, x2, y2, x3, y3) &&
		math.Abs(PointSideF(x2, y2, x0, y0, x1, y1)+PointSideF(x3, y3, x0, y0, x1, y1)) != 2 &&
		math.Abs(PointSideF(x0, y0, x2, y2, x3, y3)+PointSideF(x1, y1, x2, y2, x3, y3)) != 2
}

// FindMinAndMaxF finds and returns the minimum and maximum values in a slice of float64.
func FindMinAndMaxF(a []float64) (float64, float64) {
	min := a[0]
	max := a[0]
	for _, value := range a {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return min, max
}

// SwapF swaps the values of two float64 variables and returns the swapped values.
func SwapF(a float64, b float64) (float64, float64) {
	return b, a
}
