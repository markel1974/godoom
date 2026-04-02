package mathematic

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

// PointSide determines which side of a line segment a point lies on.
// Restituisce -1, 0, o 1.
func PointSide(px int, py int, x0 int, y0 int, x1 int, y1 int) int {
	v := Vxs(x1-x0, y1-y0, px-x0, py-y0)
	if v == 0 {
		return 0
	}
	if v < 0 {
		return -1
	}
	return 1
}
