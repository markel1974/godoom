package main

import (
	"math"
	rnd "math/rand"
	"time"
)

func init() {
	rnd.Seed(time.Now().UnixNano())
}

func random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

func randomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(a int, mi int, ma int) int {
	return min(max(a, mi), ma)
}

func vxs(x0 int, y0 int, x1 int, y1 int) int {
	return (x0)*(y1) - (x1)*(y0)
}

func findMinAndMax(a []int) (int, int) {
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

func swap(a int, b int) (int, int) {
	return b, a
}

func overlap(a0 int, a1 int, b0 int, b1 int) bool {
	return min(a0, a1) <= max(b0, b1) && min(b0, b1) <= max(a0, a1)
}

func intersectBox(x0 int, y0 int, x1 int, y1 int, x2 int, y2 int, x3 int, y3 int) bool {
	return overlap(x0, x1, x2, x3) && overlap(y0, y1, y2, y3)
}

func pointSide(px int, py int, x0 int, y0 int, x1 int, y1 int) int {
	return vxs(x1-x0, y1-y0, px-x0, py-y0)
}

func minF(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clampF(a float64, mi float64, ma float64) float64 {
	return minF(maxF(a, mi), ma)
}

func vxsF(x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return (x0)*(y1) - (x1)*(y0)
}

func overlapF(a0 float64, a1 float64, b0 float64, b1 float64) bool {
	return minF(a0, a1) <= maxF(b0, b1) && minF(b0, b1) <= maxF(a0, a1)
}

func intersectBoxF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return overlapF(x0, x1, x2, x3) && overlapF(y0, y1, y2, y3)
}

func pointSideF(px float64, py float64, x0 float64, y0 float64, x1 float64, y1 float64) float64 {
	return vxsF(x1-x0, y1-y0, px-x0, py-y0)
}

func intersectF(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64) (float64, float64) {
	x := vxsF(vxsF(x1, y1, x2, y2), (x1)-(x2), vxsF(x3, y3, x4, y4), (x3)-(x4)) / vxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	y := vxsF(vxsF(x1, y1, x2, y2), (y1)-(y2), vxsF(x3, y3, x4, y4), (y3)-(y4)) / vxsF((x1)-(x2), (y1)-(y2), (x3)-(x4), (y3)-(y4))
	return x, y
}

func intersectFn(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64) (float64, float64, bool) {
	d := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if d == 0 {
		return 0, 0, false
	}
	pre := x1*y2 - y1*x2
	post := x3*y4 - y3*x4
	x := (pre*(x3-x4) - (x1-x2)*post) / d
	y := (pre*(y3-y4) - (y1-y2)*post) / d
	// Check if the x and y coordinates are within both lines
	//if x < minF(x1, x2) || x > maxF(x1, x2) || x < minF(x3, x4) || x > maxF(x3, x4) { return 0, 0, false }
	//if y < minF(y1, y2) || y > maxF(y1, y2) || y < minF(y3, y4) || y > maxF(y3, y4) { return 0, 0, false }
	return x, y, true
}

func intersectLineSegmentsF(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64) bool {
	return intersectBoxF(x0, y0, x1, y1, x2, y2, x3, y3) &&
		math.Abs(pointSideF(x2, y2, x0, y0, x1, y1)+pointSideF(x3, y3, x0, y0, x1, y1)) != 2 &&
		math.Abs(pointSideF(x0, y0, x2, y2, x3, y3)+pointSideF(x1, y1, x2, y2, x3, y3)) != 2
}

func findMinAndMaxF(a []float64) (float64, float64) {
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

func swapF(a float64, b float64) (float64, float64) {
	return b, a
}

func pointInPolygonF(px float64, py float64, points []XY) bool {
	nVert := len(points)
	j := nVert - 1
	c := false
	for i := 0; i < nVert; i++ {
		if ((points[i].Y >= py) != (points[j].Y >= py)) && (px <= (points[j].X-points[i].X)*(py-points[i].Y)/(points[j].Y-points[i].Y)+points[i].X) {
			c = !c
		}
		j = i
	}
	return c
}
