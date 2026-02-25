package mathematic

import (
	"math"
	rnd "math/rand"
	"time"
)

func init() {
	rnd.Seed(time.Now().UnixNano())
}

func Random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

func RandomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func Clamp(a int, mi int, ma int) int {
	return Min(Max(a, mi), ma)
}

func Vxs(x0 int, y0 int, x1 int, y1 int) int {
	return (x0)*(y1) - (x1)*(y0)
}

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

func Swap(a int, b int) (int, int) {
	return b, a
}

func Overlap(a0 int, a1 int, b0 int, b1 int) bool {
	return Min(a0, a1) <= Max(b0, b1) && Min(b0, b1) <= Max(a0, a1)
}

func IntersectBox(x0 int, y0 int, x1 int, y1 int, x2 int, y2 int, x3 int, y3 int) bool {
	return Overlap(x0, x1, x2, x3) && Overlap(y0, y1, y2, y3)
}

func PointSide(px int, py int, x0 int, y0 int, x1 int, y1 int) int {
	return Vxs(x1-x0, y1-y0, px-x0, py-y0)
}

func MinF(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func MaxF(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ClampF(a float64, mi float64, ma float64) float64 {
	return MinF(MaxF(a, mi), ma)
}

func VxsF(x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return (x0)*(y1) - (x1)*(y0)
}

func OverlapF(a0 float64, a1 float64, b0 float64, b1 float64) bool {
	return MinF(a0, a1) <= MaxF(b0, b1) && MinF(b0, b1) <= MaxF(a0, a1)
}

func IntersectBoxF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return OverlapF(x0, x1, x2, x3) && OverlapF(y0, y1, y2, y3)
}

func PointSideF(px float64, py float64, x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return VxsF(x1-x0, y1-y0, px-x0, py-y0)
}

func IntersectF(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64) (float64, float64) {
	x := VxsF(VxsF(x1, y1, x2, y2), (x1)-(x2), VxsF(x3, y3, x4, y4), (x3)-(x4)) / VxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	y := VxsF(VxsF(x1, y1, x2, y2), (y1)-(y2), VxsF(x3, y3, x4, y4), (y3)-(y4)) / VxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	return x, y
}

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

func IntersectLineSegmentsF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return IntersectBoxF(x0, y0, x1, y1, x2, y2, x3, y3) &&
		math.Abs(PointSideF(x2, y2, x0, y0, x1, y1)+PointSideF(x3, y3, x0, y0, x1, y1)) != 2 &&
		math.Abs(PointSideF(x0, y0, x2, y2, x3, y3)+PointSideF(x1, y1, x2, y2, x3, y3)) != 2
}

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

func SwapF(a float64, b float64) (float64, float64) {
	return b, a
}
