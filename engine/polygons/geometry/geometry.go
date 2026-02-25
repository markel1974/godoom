package geometry

import (
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/markel1974/godoom/engine/polygons/geometry/errorstree"
)

var (
	// Eps is epsilon - precision of intersection
	Eps = 1e-10
)

// LineLine return analysis of two segments
//
// Design of segments:
//
//	                                            //
//	<-- rb00 -- pb0*==========*pb1 -- rb11 -->  // Segment B
//	                                            //
//	<-- ra00 -- pa0*==========*pa1 -- ra11 -->  // Segment A
//	{   ray   }{      segment     }{   ray   }  //
//	                                            //
//
// Input data:
//
//	ipa0, ipa1 - point indexes of segment A
//	ipb0, ipb1 - point indexes of segment B
//	pps      - pointer of point slice
//
// Output data:
//
//	pi - intersection point
//	st - states of analysis
//
// Reference:
//
//	[1]  https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
func LineLine(pa0 Point, pa1 Point, pb0 Point, pb1 Point) (pi []Point, stA State, stB State) {
	// Point - Point
	if SamePoints(pa0, pa1) && SamePoints(pb0, pb1) {
		return PointPoint(pa0, pb0)
	}
	// Point - Line
	if SamePoints(pa0, pa1) {
		return PointLine(pa0, pb0, pb1)
	}
	if SamePoints(pb0, pb1) {
		pi, stA, stB = PointLine(pb0, pa0, pa1)
		stA, stB = stB, stA
		return
	}
	// Line - Line

	for _, c := range [...]struct {
		isTrue bool
		tiA    State
		tiB    State
	}{
		{isTrue: SamePoints(pa0, pb0), tiA: OnPoint0Segment, tiB: OnPoint0Segment},
		{isTrue: SamePoints(pa0, pb1), tiA: OnPoint0Segment, tiB: OnPoint1Segment},
		{isTrue: SamePoints(pa1, pb0), tiA: OnPoint1Segment, tiB: OnPoint0Segment},
		{isTrue: SamePoints(pa1, pb1), tiA: OnPoint1Segment, tiB: OnPoint1Segment},
		{isTrue: math.Abs(pa0.X-pa1.X) < Eps, tiA: VerticalSegment},
		{isTrue: math.Abs(pa0.Y-pa1.Y) < Eps, tiA: HorizontalSegment},
		{isTrue: math.Abs(pb0.X-pb1.X) < Eps, tiB: VerticalSegment},
		{isTrue: math.Abs(pb0.Y-pb1.Y) < Eps, tiB: HorizontalSegment},
	} {
		if c.isTrue {
			stA |= c.tiA
			stB |= c.tiB
		}
	}

	// collinear lines
	if Orientation(pa0, pa1, pb0) == CollinearPoints && Orientation(pa0, pa1, pb1) == CollinearPoints {
		stA |= Collinear
		stB |= Collinear
		return
	}
	// parallel lines
	// if math.Abs((pa1.Y-pa0.Y)*(pb1.X-pb0.X)-(pb1.Y-pb0.Y)*(pa1.X-pa0.X)) < Eps {
	if math.Abs(math.FMA(pa1.Y-pa0.Y, pb1.X-pb0.X, -(pb1.Y-pb0.Y)*(pa1.X-pa0.X))) < Eps {
		stA |= Parallel
		stB |= Parallel
		return
	}

	// intersection point
	Aa, Ba, Ca := Line(pa0, pa1)
	Ab, Bb, Cb := Line(pb0, pb1)
	x, y, err := Linear(Aa, Ba, -Ca, Ab, Bb, -Cb)
	if err != nil {
		panic(err)
	}
	// only for orthogonal cases
	if pa0.X == pa1.X {
		x = pa0.X
	}
	if pa0.Y == pa1.Y {
		y = pa0.Y
	}
	if pb0.X == pb1.X {
		x = pb0.X
	}
	if pb0.Y == pb1.Y {
		y = pb0.Y
	}
	root := Point{X: x, Y: y}

	{
		_, _, stBa := PointLine(root, pa0, pa1)
		_, _, stBb := PointLine(root, pb0, pb1)
		if stBa.Has(OnSegment) &&
			(stBb.Has(OnSegment) || stBb.Has(OnPoint0Segment) || stBb.Has(OnPoint1Segment)) {
			stA |= OnSegment
		}
		if stBb.Has(OnSegment) &&
			(stBa.Has(OnSegment) || stBa.Has(OnPoint0Segment) || stBa.Has(OnPoint1Segment)) {
			stB |= OnSegment
		}
	}

	if stA.Has(OnSegment) || stB.Has(OnSegment) {
		pi = []Point{root}
	}

	for _, c := range [...]struct {
		isTrue bool
		tiA    State
		tiB    State
	}{
		{isTrue: SamePoints(pa0, root), tiA: OnPoint0Segment},
		{isTrue: SamePoints(pa1, root), tiA: OnPoint1Segment},
		{isTrue: SamePoints(pb0, root), tiB: OnPoint0Segment},
		{isTrue: SamePoints(pb1, root), tiB: OnPoint1Segment},
	} {
		if c.isTrue {
			stA |= c.tiA
			stB |= c.tiB
		}
	}

	return
}

// MiddlePoint calculate middle point precisely.
func MiddlePoint(p0 Point, p1 Point) Point {
	// Simple float64 algorithm:
	x, y := p0.X, p0.Y
	if p0.X != p1.X {
		// x = p0.X*0.5 + p1.X*0.5
		x = math.FMA(p0.X, 0.5, p1.X*0.5)
	}
	if p0.Y != p1.Y {
		// y = p0.Y*0.5 + p1.Y*0.5
		y = math.FMA(p0.Y, 0.5, p1.Y*0.5)
	}
	return Point{X: x, Y: y}
}

// PointLineDistance return distance between line and point.
//
// Equation of line:
//
//	(y2-y1)*(x-x1) = (x2-x1)(y-y1)
//	dy*(x-x1) = dx*(y-y1)
//	dy*x-dy*x1-dx*y+dx*y1 = 0
//	Ax+By+C = 0
//	A = dy
//	B = -dx
//	C = -dy*x1+dx*y1
//
// Distance from point (xm, ym) to line:
//
//	d = |(A*xm+B*ym+C)/sqrt(A^2+B^2)|
func PointLineDistance(pc Point, p0 Point, p1 Point) (distance float64) {
	A, B, C := Line(p0, p1)

	// coordinates of point
	xm := pc.X
	ym := pc.Y

	// distance = math.Abs((A*xm + B*ym + C) / math.Sqrt(pow.E2(A)+pow.E2(B)))
	distance = math.Abs(math.FMA(A, xm, math.FMA(B, ym, C)) / math.Sqrt(PowE2(A)+PowE2(B)))
	return
}

// Line parameters by formula:	Ax+By+C = 0
func Line(p0 Point, p1 Point) (A float64, B float64, C float64) {
	dy := p1.Y - p0.Y
	dx := p1.X - p0.X

	// parameters of line
	A = dy
	B = -dx
	// algorithm for float64
	// C = -dy*p0.X + dx*p0.Y
	// return

	// algorithm for FMA
	C = math.FMA(-dy, p0.X, dx*p0.Y)
	return

	// algorithm for float 128
	// const prec = 128
	// var (
	// 	pdy   = new(big.Float).SetPrec(prec).SetFloat64(-dy)
	// 	pdx   = new(big.Float).SetPrec(prec).SetFloat64(dx)
	// 	px    = new(big.Float).SetPrec(prec).SetFloat64(p0.X)
	// 	py    = new(big.Float).SetPrec(prec).SetFloat64(p0.Y)
	// 	left  = new(big.Float).SetPrec(prec).Mul(pdy, px)
	// 	right = new(big.Float).SetPrec(prec).Mul(pdx, py)
	// 	summ  = new(big.Float).SetPrec(prec).Add(left, right)
	// )
	// C, _ = summ.Float64()
	// return
}

// Distance128 is distance between 2 points with 128-bit precisions
func Distance128(p0, p1 Point) float64 {
	if p0.X == p1.X && p0.Y == p1.Y {
		return 0
	}
	const prec = 128

	var (
		x0   = new(big.Float).SetPrec(prec).SetFloat64(p0.X)
		x1   = new(big.Float).SetPrec(prec).SetFloat64(p1.X)
		y0   = new(big.Float).SetPrec(prec).SetFloat64(p0.Y)
		y1   = new(big.Float).SetPrec(prec).SetFloat64(p1.Y)
		x    = new(big.Float).SetPrec(prec).Sub(x0, x1)
		y    = new(big.Float).SetPrec(prec).Sub(y0, y1)
		xx   = new(big.Float).SetPrec(prec).Mul(x, x)
		yy   = new(big.Float).SetPrec(prec).Mul(y, y)
		summ = new(big.Float).SetPrec(prec).Add(xx, yy)
		s    = new(big.Float).SetPrec(prec).Sqrt(summ)
	)

	sf, _ := s.Float64()
	return sf
}

// Distance between two points
func Distance(p0 Point, p1 Point) float64 {
	v := math.Hypot(p0.X-p1.X, p0.Y-p1.Y)
	if v < 100*Eps {
		return Distance128(p0, p1)
	}
	return v
}

// Rotate point about (xc,yc) on angle
func Rotate(xc, yc, angle float64, point Point) (p Point) {
	p.X = math.Cos(angle)*(point.X-xc) - math.Sin(angle)*(point.Y-yc) + xc
	p.Y = math.Sin(angle)*(point.X-xc) + math.Cos(angle)*(point.Y-yc) + yc
	return
}

// MirrorLine return intersection point and second mirrored point from mirror
// line (mp0-mp1) and ray (sp0-sp1)
func MirrorLine(sp0, sp1 Point, mp0, mp1 Point) (ml0 Point, ml1 Point, err error) {
	if SamePoints(mp0, mp1) {
		panic("mirror line is point")
	}

	A, B, C := Line(mp0, mp1)

	mir := func(x1, y1 float64) Point {
		temp := -2 * (A*x1 + B*y1 + C) / (A*A + B*B)
		return Point{X: temp*A + x1, Y: temp*B + y1}
	}

	ml0 = mir(sp0.X, sp0.Y)
	ml1 = mir(sp1.X, sp1.Y)
	return
}

// OrientationPoints is orientation points state
type OrientationPoints int8

const (
	CollinearPoints        OrientationPoints = -1
	ClockwisePoints                          = 0
	CounterClockwisePoints                   = 1
)

func Orientation(p1 Point, p2 Point, p3 Point) OrientationPoints {
	// middle point with collinear points
	if mid := MiddlePoint(p1, p2); p3.X == mid.X && p3.Y == mid.Y {
		return CollinearPoints
	}
	if mid := MiddlePoint(p2, p3); p1.X == mid.X && p1.Y == mid.Y {
		return CollinearPoints
	}
	if mid := MiddlePoint(p1, p3); p2.X == mid.X && p2.Y == mid.Y {
		return CollinearPoints
	}
	// vertical or horizontal collinear points
	if p1.X == p2.X && p2.X == p3.X {
		return CollinearPoints
	}
	if p1.Y == p2.Y && p2.Y == p3.Y {
		return CollinearPoints
	}

	// check other orientations
	// algorithm float64
	// v := (p2.Y-p1.Y)*(p3.X-p2.X) - (p2.X-p1.X)*(p3.Y-p2.Y)

	// algorithm FMA
	v := math.FMA(p2.Y-p1.Y, p3.X-p2.X, -(p2.X-p1.X)*(p3.Y-p2.Y))

	if math.Abs(v) < 100*Eps {
		return Orientation128(p1, p2, p3)
	}
	switch {
	case math.Abs(v) < Eps:
		return CollinearPoints
	case 0 < v:
		return ClockwisePoints
	}
	return CounterClockwisePoints
}

func Orientation128(p1 Point, p2 Point, p3 Point) OrientationPoints {
	const prec = 128

	var (
		x1 = new(big.Float).SetPrec(prec).SetFloat64(p1.X)
		x2 = new(big.Float).SetPrec(prec).SetFloat64(p2.X)
		x3 = new(big.Float).SetPrec(prec).SetFloat64(p3.X)

		y1 = new(big.Float).SetPrec(prec).SetFloat64(p1.Y)
		y2 = new(big.Float).SetPrec(prec).SetFloat64(p2.Y)
		y3 = new(big.Float).SetPrec(prec).SetFloat64(p3.Y)

		y21 = new(big.Float).SetPrec(prec).Sub(y2, y1)
		y32 = new(big.Float).SetPrec(prec).Sub(y3, y2)

		x21 = new(big.Float).SetPrec(prec).Sub(x2, x1)
		x32 = new(big.Float).SetPrec(prec).Sub(x3, x2)

		left  = new(big.Float).SetPrec(prec).Mul(y21, x32)
		right = new(big.Float).SetPrec(prec).Mul(x21, y32)

		s = new(big.Float).SetPrec(prec).Sub(left, right)
	)

	v, _ := s.Float64()

	switch {
	case math.Abs(v) < Eps:
		return CollinearPoints
	case 0 < v:
		return ClockwisePoints
	}
	return CounterClockwisePoints
}

// PointArc return state and intersections points between point and arc
func PointArc(pt Point, Arc0 Point, Arc1 Point, Arc2 Point) (pi []Point, stA, stB State) {
	// Point - Point
	if SamePoints(Arc0, Arc1) && SamePoints(Arc1, Arc2) {
		pi, stA, stB = PointPoint(pt, Arc0)
		stB |= ArcIsPoint
		return
	}
	// Point - Line
	{
		if Orientation(Arc0, Arc1, Arc2) == CollinearPoints {
			pi, stA, stB = PointLine(pt, Arc0, Arc2)
			stB |= ArcIsLine
			return
		}
		if SamePoints(Arc0, Arc1) {
			pi, stA, stB = PointLine(pt, Arc0, Arc2)
			stB |= ArcIsLine
			return
		}
		if SamePoints(Arc1, Arc2) {
			pi, stA, stB = PointLine(pt, Arc0, Arc2)
			stB |= ArcIsLine
			return
		}
	}
	// Point - Arc

	stA |= ZeroLengthSegment | VerticalSegment | HorizontalSegment

	xc, yc, r := Arc(Arc0, Arc1, Arc2)
	radius := Distance(Point{X: xc, Y: yc}, pt)
	if radius < r-Eps || r+Eps < radius {
		// point is outside of arc
		return
	}
	// point is on arc corner ?
	if SamePoints(pt, Arc0) {
		stB |= OnPoint0Segment
	}
	if SamePoints(pt, Arc2) {
		stB |= OnPoint1Segment
	}

	// point is on arc ?
	if stB.Has(OnPoint0Segment) || stB.Has(OnPoint1Segment) {
		return
	}
	if AngleBetween(Point{X: xc, Y: yc}, Arc0, Arc1, Arc2, pt) {
		stB |= OnSegment
	}

	return
}

// LineArc return state and intersections points between line and arc
func LineArc(Line0 Point, Line1 Point, Arc0 Point, Arc1 Point, Arc2 Point) (pi []Point, stA State, stB State) {
	// Point - Arc
	if SamePoints(Line0, Line1) {
		return PointArc(Line0, Arc0, Arc1, Arc2)
	}
	// Line - Point
	if SamePoints(Arc0, Arc1) && SamePoints(Arc1, Arc2) {
		pi, stA, stB = PointLine(Arc0, Line0, Line1)
		stA, stB = stB, stA
		stB |= ArcIsPoint
		return
	}
	// Line - Line
	if SamePoints(Arc0, Arc1) {
		pi, stA, stB = LineLine(Line0, Line1, Arc0, Arc2)
		stB |= ArcIsLine
		return
	}
	if SamePoints(Arc1, Arc2) {
		pi, stA, stB = LineLine(Line0, Line1, Arc0, Arc2)
		stB |= ArcIsLine
		return
	}
	{
		A01, B01, C01 := Line(Arc0, Arc1)
		A12, B12, C12 := Line(Arc1, Arc2)
		if math.Abs(A01-A12) < Eps &&
			math.Abs(B01-B12) < Eps &&
			math.Abs(C01-C12) < Eps {
			pi, stA, stB = LineLine(Line0, Line1, Arc0, Arc2)
			stB |= ArcIsLine
			return
		}
	}
	// Line - Arc

	for _, c := range [...]struct {
		isTrue bool
		tiA    State
		tiB    State
	}{
		{isTrue: math.Abs(Line0.X-Line1.X) < Eps, tiA: VerticalSegment},
		{isTrue: math.Abs(Line0.Y-Line1.Y) < Eps, tiA: HorizontalSegment},
		{isTrue: SamePoints(Line0, Arc0), tiA: OnPoint0Segment, tiB: OnPoint0Segment},
		{isTrue: SamePoints(Line0, Arc2), tiA: OnPoint0Segment, tiB: OnPoint1Segment},
		{isTrue: SamePoints(Line1, Arc0), tiA: OnPoint1Segment, tiB: OnPoint0Segment},
		{isTrue: SamePoints(Line1, Arc2), tiA: OnPoint1Segment, tiB: OnPoint1Segment},
	} {
		if c.isTrue {
			stA |= c.tiA
			stB |= c.tiB
		}
	}

	// circle function
	//	(x-xc)^2 + (y-yc)^2 = R^2
	// solve circle parameters
	//	(xi-xc)^2+(yi-yc)^2 = R^2
	//	xi^2-2*xi*xc+xc^2 + yi^2-2*yi*yc+yc^2 = R^2
	//	xi^2+yi^2 -2*xi*xc-2*yi*yc = R^2-xc^2-yc^2
	// between points 1 and 2:
	//	(x1^2-x2^2) +(y1^2-y2^2) -2*xc*(x1-x2)-2*yc*(y1-y2) = 0
	//	(x1^2-x3^2) +(y1^2-y3^2) -2*xc*(x1-x3)-2*yc*(y1-y3) = 0
	//
	//	2*xc*(x1-x2)+2*yc*(y1-y2) = (x1^2-x2^2) + (y1^2-y2^2)
	//	2*xc*(x1-x3)+2*yc*(y1-y3) = (x1^2-x3^2) + (y1^2-y3^2)
	//
	//	2*(x1-x2)*xc + 2*(y1-y2)*yc = (x1^2-x2^2)+(y1^2-y2^2)
	//	2*(x1-x3)*xc + 2*(y1-y3)*yc = (x1^2-x3^2)+(y1^2-y3^2)
	//
	// solve linear equations:
	//	a11*xc + a12*yc = b1
	//	a21*xc + a22*yc = b2
	// solve:
	//	xc = (b1 - a12*yc)*1/a11
	//	a21*(b1-a12*yc)*1/a11 + a22*yc = b2
	//	yc*(a22-a21/a11*a12) = b2 - a21/a11*b1
	xc, yc, r := Arc(Arc0, Arc1, Arc2)

	// line may be horizontal, vertical, other
	A, B, C := Line(Line0, Line1)

	// Find intersections
	//	A*x+B*y+C = 0               :   line equations
	//	(x-xc)^2 + (y-yc)^2 = r^2   : circle equations
	var roots []Point
	switch {
	case stA.Has(HorizontalSegment):
		// line is horizontal
		//	A = 0
		//
		//	B*y+C = 0
		//	(x-xc)^2 + (y-yc)^2 = r^2
		//
		//	y = -C/B
		//	(x-xc)^2 = r^2 - (-C/B-yc)^2
		//
		//	x = +/- sqrt(r^2 - (-C/B-yc)^2) + xc
		D := PowE2(r) - PowE2(-C/B-yc)
		switch {
		case D < -Eps:
			// no intersection
		case D < Eps:
			// D == 0
			// have one root
			roots = append(roots, Point{X: +xc, Y: Line0.Y})
		default:
			// 0 < D
			roots = append(roots,
				Point{X: +math.Sqrt(D) + xc, Y: Line0.Y},
				Point{X: -math.Sqrt(D) + xc, Y: Line0.Y},
			)
		}

	case stA.Has(VerticalSegment):
		// line is vertical
		// B = 0
		//
		//	A*x+C = 0
		//	(x-xc)^2 + (y-yc)^2 = r^2
		//
		//	x=-C/A
		//	(y-yc)^2 = r^2 - (-C/A-xc)^2
		//
		//	y = +/- sqrt(r^2 - (-C/A-xc)^2) - yc
		D := PowE2(r) - PowE2(-C/A-xc)
		switch {
		case D < -Eps:
			// no intersection
		case D < Eps:
			// D == 0
			// have one root
			roots = append(roots, Point{X: Line0.X, Y: +yc})
		default:
			// 0 < D
			roots = append(roots,
				Point{X: Line0.X, Y: +math.Sqrt(D) + yc},
				Point{X: Line0.X, Y: -math.Sqrt(D) + yc},
			)
		}

	default:
		//	A*x+B*y+C = 0               :   line function
		//	(x-xc)^2 + (y-yc)^2 = r^2   : circle function
		//
		// solve intersection:
		//	x = -(B*y+C)*1/A
		//	(-(B*y+C)*1/A-xc)^2 + (y-yc)^2 = r^2
		//	(-(B*y+C)-A*xc)^2 + (y-yc)^2*A^2 = r^2*A^2
		//	(-B*y-C-A*xc)^2 + (y-yc)^2*A^2 = r^2*A^2
		//	(-B*y-(C+A*xc))^2 + (y-yc)^2*A^2 = r^2*A^2
		//	(B*y)^2 + 2*(B*y)*(C+A*xc) + (C+A*xc)^2 + (y^2-2*y*yc+yc^2)*A^2 = r^2*A^2
		//	y^2*(B^2 + A^2) + y*(2*B*(C+A*xc) - 2*yc*A^2) + (C+A*xc)^2 + yc^2*A^2 - r^2*A^2 = 0
		//	    ==========      -------------------------   _______________________________
		var (
			a = PowE2(B) + PowE2(A)
			b = 2*B*(C+A*xc) - 2*yc*PowE2(A)
			c = PowE2(C+A*xc) + PowE2(yc*A) - PowE2(r*A)
			D = b*b - 4.0*a*c
		)

		// A and B of line parameters is not zero, so
		// value a is not a zero and more then zero.
		switch {
		case D < -Eps:
			// no intersection
		case D < Eps:
			// D == 0
			// have one root
			y := -b / (2.0 * a)
			x := -(B*y + C) * 1 / A
			roots = append(roots, Point{X: x, Y: y})
		default:
			// 0 < D
			{
				y := (-b + math.Sqrt(D)) / (2.0 * a)
				x := -(B*y + C) * 1 / A
				roots = append(roots, Point{X: x, Y: y})
			}
			{
				y := (-b - math.Sqrt(D)) / (2.0 * a)
				x := -(B*y + C) * 1 / A
				roots = append(roots, Point{X: x, Y: y})
			}
		}
	}

	for _, root := range roots {
		_, _, stBa := PointLine(root, Line0, Line1)
		_, _, stBb := PointArc(root, Arc0, Arc1, Arc2)

		added := false

		if stBa.Has(OnSegment) &&
			(stBb.Has(OnSegment) || stBb.Has(OnPoint0Segment) || stBb.Has(OnPoint1Segment)) {
			added = true
			stA |= OnSegment
		}

		if stBb.Has(OnSegment) &&
			(stBa.Has(OnSegment) || stBa.Has(OnPoint0Segment) || stBa.Has(OnPoint1Segment)) {
			added = true
			stB |= OnSegment
		}

		if !added {
			continue
		}

		pi = append(pi, root)

		for _, c := range [...]struct {
			isTrue   bool
			tiA, tiB State
		}{
			{
				isTrue: SamePoints(Line0, root) &&
					(stBa.Has(OnSegment) || stBa.Has(OnPoint0Segment) || stBa.Has(OnPoint1Segment)),
				tiA: OnPoint0Segment,
			},
			{
				isTrue: SamePoints(Line1, root) &&
					(stBa.Has(OnSegment) || stBa.Has(OnPoint0Segment) || stBa.Has(OnPoint1Segment)),
				tiA: OnPoint1Segment,
			},
			{
				isTrue: SamePoints(Arc0, root) &&
					(stBb.Has(OnSegment) || stBb.Has(OnPoint0Segment) || stBb.Has(OnPoint1Segment)),
				tiB: OnPoint0Segment,
			},
			{
				isTrue: SamePoints(Arc2, root) &&
					(stBb.Has(OnSegment) || stBb.Has(OnPoint0Segment) || stBb.Has(OnPoint1Segment)),
				tiB: OnPoint1Segment,
			},
		} {
			if c.isTrue {
				stA |= c.tiA
				stB |= c.tiB
			}
		}
	}

	return
}

// ArcSplitByPoint return points of arcs with middle point if pi is empty or
// slice of arcs.
//
//	DO NOT CHECKED POINT ON ARC
func ArcSplitByPoint(Arc0 Point, Arc1 Point, Arc2 Point, pi ...Point) (res [][3]Point, err error) {
	switch Orientation(Arc0, Arc1, Arc2) {
	case CollinearPoints:
		et := errorstree.New("ArcSplitByPoint: collinear")
		_ = et.Add(fmt.Errorf("arc0 = %.12e", Arc0))
		_ = et.Add(fmt.Errorf("arc1 = %.12e", Arc1))
		_ = et.Add(fmt.Errorf("arc2 = %.12e", Arc2))
		panic(et)
	case ClockwisePoints:
		res, err = ArcSplitByPoint(Arc2, Arc1, Arc0, pi...)
		for i := range res {
			res[i][0], res[i][2] = res[i][2], res[i][0]
		}
		return
	}
	// CounterClockwisePoints

	for _, c := range [...]struct {
		isTrue bool
	}{
		{isTrue: SamePoints(Arc0, Arc1)},
		{isTrue: SamePoints(Arc1, Arc2)},
		{isTrue: SamePoints(Arc0, Arc2)},
	} {
		if c.isTrue {
			err = fmt.Errorf("invalid points of arc")
			return
		}
	}

	// remove points on corners or same points
againRemove:
	for i, p := range pi {
		for _, c := range [...]struct {
			isTrue bool
		}{
			{isTrue: SamePoints(Arc0, p)},
			{isTrue: SamePoints(Arc2, p)},
		} {
			if c.isTrue {
				pi = append(pi[:i], pi[i+1:]...)
				goto againRemove
			}
		}
		for j := range pi {
			if i < j && SamePoints(pi[i], pi[j]) {
				pi = append(pi[:i], pi[i+1:]...)
				goto againRemove
			}
		}
	}

	// parameter of arc
	xc, yc, r := Arc(Arc0, Arc1, Arc2)

	// angle for rotate
	angle0 := -math.Atan2(Arc0.Y-yc, Arc0.X-xc) - math.Pi + 0.01

	// rotate
	ps := []Point{
		Rotate(xc, yc, +angle0, Arc0),
		Rotate(xc, yc, +angle0, Arc2),
	}
	for i := range pi {
		ps = append(ps, Rotate(xc, yc, +angle0, pi[i]))
	}

	// points angles
	var b []float64
	for i := range ps {
		b = append(b, math.Atan2(ps[i].Y-yc, ps[i].X-xc))
	}
	sort.Float64s(b)

	// remove same angles
again:
	for i := 1; i < len(b); i++ {
		if math.Abs(b[i]-b[i-1]) < Eps {
			b = append(b[:i-1], b[i:]...)
			goto again
		}
	}

	// add middle angles
	if len(pi) == 0 {
		for _, f := range []float64{0.25, 0.5, 0.75} {
			b = append(b, b[0]+f*(b[1]-b[0]))
		}
	} else {
		for i, size := 0, len(b)-1; i < size; i++ {
			b = append(b, b[i]+0.5*(b[i+1]-b[i]))
		}
	}
	sort.Float64s(b)

	if b[0] < -math.Pi {
		panic(fmt.Errorf("debug: %v", b))
	}

	ps = []Point{}
	for _, angle := range b {
		p := Point{
			// X: r*math.Cos(angle-angle0) + xc,
			// Y: r*math.Sin(angle-angle0) + yc,
			X: math.FMA(r, math.Cos(angle-angle0), xc),
			Y: math.FMA(r, math.Sin(angle-angle0), yc),
		}
		ps = append(ps, p)
	}

	// prepare arcs
	// 0-1-2=3=4-5-6
	// len=7 arcs=3
	// len=5 arcs=2
	// len=3 arcs=1
	for i := 0; i <= (len(ps)-1)/2; i += 2 {
		res = append(res, [3]Point{ps[i], ps[i+1], ps[i+2]})
	}

	return
}

// TODO: panic free

var (
	// ErrorDivZero is typical result with not acceptable solving
	ErrorDivZero = fmt.Errorf("div value is too small")

	// ErrorNotValidSystem is typical return only if system of
	// linear equation have not valid data
	ErrorNotValidSystem = fmt.Errorf("not valid system")
)

// Linear equations solving:
//
//	a11*x + a12*y = b1
//	a21*x + a22*y = b2
func Linear(a11 float64, a12 float64, b1 float64, a21 float64, a22 float64, b2 float64) (x float64, y float64, err error) {
	// only for debugging
	// defer func() {
	// 	if err != nil {
	// 		et := eTree.New("Linear")
	// 		et.Add(fmt.Errorf("a11 = %.5e", a11))
	// 		et.Add(fmt.Errorf("a12 = %.5e", a12))
	// 		et.Add(fmt.Errorf("b1  = %.5e", b1))
	// 		et.Add(fmt.Errorf("a21 = %.5e", a21))
	// 		et.Add(fmt.Errorf("a22 = %.5e", a22))
	// 		et.Add(fmt.Errorf("b2 = %.5e", b2))
	// 		err = fmt.Errorf("%v\n%v", err, et)
	// 	}
	// }()
	if math.Abs(a11) < Eps {
		if math.Abs(a12) < Eps {
			err = ErrorNotValidSystem
			return
		}
		// swap parameters
		a11, a12 = a12, a11
		a21, a22 = a22, a21
		defer func() {
			x, y = y, x
		}()
	}
	// algoritm for float64
	// y = (b2*a11 - b1*a21) / (a22*a11 - a21*a12)
	// x = (b1 - a12*y) / a11
	// return

	// algoritm for FMA
	div := math.FMA(a22, a11, -a21*a12)
	if math.Abs(div) < Eps {
		// only for debugging
		// err = fmt.Errorf("error div = %e", div)

		err = ErrorDivZero
		return
	}
	y = math.FMA(b2, a11, -b1*a21) / div
	x = math.FMA(-a12, y, b1) / a11
	return

	// algoritm for float 128
	// const prec = 128
	// var (
	// 	pa11 = new(big.Float).SetPrec(prec).SetFloat64(a11)
	// 	pa12 = new(big.Float).SetPrec(prec).SetFloat64(a12)
	// 	pb1  = new(big.Float).SetPrec(prec).SetFloat64(b1)
	// 	pa21 = new(big.Float).SetPrec(prec).SetFloat64(a21)
	// 	pa22 = new(big.Float).SetPrec(prec).SetFloat64(a22)
	// 	pb2  = new(big.Float).SetPrec(prec).SetFloat64(b2)
	//
	// 	b2a11   = new(big.Float).SetPrec(prec).Mul(pb2, pa11)
	// 	b1a21   = new(big.Float).SetPrec(prec).Mul(pb1, pa21)
	// 	subUp   = new(big.Float).SetPrec(prec).Sub(b2a11, b1a21)
	// 	a22a11  = new(big.Float).SetPrec(prec).Mul(pa22, pa11)
	// 	a21a12  = new(big.Float).SetPrec(prec).Mul(pa21, pa12)
	// 	subDown = new(big.Float).SetPrec(prec).Sub(a22a11, a21a12)
	// 	yQuo    = new(big.Float).SetPrec(prec).Quo(subUp, subDown)
	// )
	// y, _ = yQuo.Float64()
	// var (
	// 	a12y   = new(big.Float).SetPrec(prec).Mul(pa12, yQuo)
	// 	b1a12y = new(big.Float).SetPrec(prec).Sub(pb1, a12y)
	// 	xQuo   = new(big.Float).SetPrec(prec).Quo(b1a12y, pa11)
	// )
	// x, _ = xQuo.Float64()
	//
	// return
}

// Arc return parameters of circle
func Arc(Arc0 Point, Arc1 Point, Arc2 Point) (xc float64, yc float64, r float64) {
	var (
		x1, x2, x3 = Arc0.X, Arc1.X, Arc2.X
		y1, y2, y3 = Arc0.Y, Arc1.Y, Arc2.Y
		a11        = 2 * (x1 - x2)
		a12        = 2 * (y1 - y2)
		a21        = 2 * (x1 - x3)
		a22        = 2 * (y1 - y3)
		b1         = (PowE2(x1) - PowE2(x2)) + (PowE2(y1) - PowE2(y2))
		b2         = (PowE2(x1) - PowE2(x3)) + (PowE2(y1) - PowE2(y3))
	)
	var err error
	xc, yc, err = Linear(a11, a12, b1, a21, a22, b2)
	if err != nil {
		panic(err)
	}

	//	(xi-xc)^2+(yi-yc)^2 = R^2
	r1 := math.Hypot(x1-xc, y1-yc)
	r2 := math.Hypot(x2-xc, y2-yc)
	r3 := math.Hypot(x3-xc, y3-yc)
	r = (r1 + r2 + r3) / 3.0
	// find angles
	return
}

// AngleBetween return true for angle case from <= a <= to
func AngleBetween(center, from, mid, to, a Point) (res bool) {
	switch Orientation(from, mid, to) {
	case CollinearPoints:
		et := errorstree.New("AngleBetween: collinear")
		_ = et.Add(fmt.Errorf("from = %.12e", from))
		_ = et.Add(fmt.Errorf("mid  = %.12e", mid))
		_ = et.Add(fmt.Errorf("to   = %.12e", to))
		panic(et)
	case ClockwisePoints:
		return AngleBetween(center, to, mid, from, a)
	}
	// CounterClockwisePoints

	ps := []Point{from, mid, to, a}
	for i := range ps {
		ps[i] = Point{X: ps[i].X - center.X, Y: ps[i].Y - center.Y}
	}

	// angle for rotate
	angle0 := -math.Atan2(ps[0].Y, ps[0].X) - math.Pi + 0.01

	// rotate
	for i := range ps {
		ps[i] = Rotate(0, 0, +angle0, ps[i])
	}

	// points angles
	var b []float64
	for i := range ps {
		b = append(b, math.Atan2(ps[i].Y, ps[i].X))
	}

	if b[0] < -math.Pi {
		panic(fmt.Errorf("debug : %v", b))
	}

	if b[0] < b[3] && b[3] < b[2] {
		return true
	}

	return false
}

// Area return area of triangle
func Area(tr0 Point, tr1 Point, tr2 Point) float64 {
	var (
		x1, y1 = tr0.X, tr0.Y
		x2, y2 = tr1.X, tr1.Y
		x3, y3 = tr2.X, tr2.Y
	)
	// return math.Abs(0.5 * (x1*(y2-y3) + x2*(y3-y1) + x3*(y1-y2)))
	return math.Abs(0.5 * math.FMA(x1, y2-y3, math.FMA(x2, y3-y1, x3*(y1-y2))))
}

// TriangleSplitByPoint split triangle on triangles only if point inside
// triangle or on triangle edge
func TriangleSplitByPoint(pt Point, tr0 Point, tr1 Point, tr2 Point) (res [][3]Point, lineIntersect int, err error) {
	// check valid triangle
	for _, c := range [...]struct{ isTrue bool }{
		{isTrue: SamePoints(tr0, tr1)},
		{isTrue: SamePoints(tr1, tr2)},
		{isTrue: SamePoints(tr0, tr2)},
	} {
		if c.isTrue {
			err = fmt.Errorf("invalid points of triangle")
			return
		}
	}
	// point in triangle box ?
	{
		var (
			xmax = -math.MaxFloat64
			ymax = -math.MaxFloat64
			xmin = +math.MaxFloat64
			ymin = +math.MaxFloat64
		)
		for _, tr := range []Point{tr0, tr1, tr2} {
			xmax = math.Max(xmax, tr.X)
			ymax = math.Max(ymax, tr.Y)
			xmin = math.Min(xmin, tr.X)
			ymin = math.Min(ymin, tr.Y)
		}
		if pt.X < xmin || xmax < pt.X || pt.Y < ymin || ymax < pt.Y {
			// point outside triangle
			return
		}
	}
	// point on corner ?
	for _, c := range [...]struct{ isTrue bool }{
		{isTrue: SamePoints(tr0, pt)},
		{isTrue: SamePoints(tr1, pt)},
		{isTrue: SamePoints(tr2, pt)},
	} {
		if c.isTrue {
			// point on corner
			// no need a split
			return
		}
	}
	// point on the side ?
	for _, line := range []struct {
		Line  [2]Point
		Free  Point
		state int
	}{
		{
			// tr0 --- pt --- tr1  //
			//   \           /     //
			//    \         /      //
			//     \       /       //
			//      \ tr2 /        //
			Line:  [2]Point{tr0, tr1},
			Free:  tr2,
			state: 0,
		},
		{
			// tr0 ---------- tr1  //
			//   \           /     //
			//    \         pt     //
			//     \       /       //
			//      \ tr2 /        //
			Line:  [2]Point{tr1, tr2},
			Free:  tr0,
			state: 1,
		},
		{
			// tr0 ---------- tr1  //
			//   \           /     //
			//    pt        /      //
			//     \       /       //
			//      \ tr2 /        //
			Line:  [2]Point{tr2, tr0},
			Free:  tr1,
			state: 2,
		},
	} {
		_, _, stBl := PointLine(pt, line.Line[0], line.Line[1])
		if !stBl.Has(OnSegment) {
			// point is outside side
			continue
		}
		// point on side
		switch Orientation(tr0, tr1, tr2) {
		case ClockwisePoints:
			res = [][3]Point{
				{line.Line[0], pt, line.Free},
				{pt, line.Line[1], line.Free},
			}
		case CounterClockwisePoints:
			res = [][3]Point{
				{line.Free, pt, line.Line[0]},
				{line.Free, line.Line[1], pt},
			}
		default:
			panic("strange situation")
		}
		lineIntersect = line.state
		return
	}

	// point in body ?
	orient := [3]OrientationPoints{
		Orientation(tr0, pt, tr1),
		Orientation(tr1, pt, tr2),
		Orientation(tr2, pt, tr0),
	}
	if orient[0] != orient[1] ||
		orient[1] != orient[2] ||
		orient[0] != orient[2] {
		// point is outside triangle
		return
	}
	// point inside triangle
	res = [][3]Point{
		{tr0, tr1, pt},
		{tr1, tr2, pt},
		{tr2, tr0, pt},
	}
	return
}

// PointInCircle return true only if point inside circle based
// on 3 circles points
func PointInCircle(point Point, circle [3]Point) bool {
	xc, yc, r := Arc(circle[0], circle[1], circle[2])
	return Distance(Point{xc, yc}, point)+Eps < r
}

// ConvexHull return chain of convex points
func ConvexHull(points []Point) (chain []Point) {
	if len(points) < 3 {
		// points slice is small
		return
	}

	// copy of points
	{
		c := make([]Point, len(points))
		copy(c, points)
		points = c
	}
	// sorting
	sort.Slice(points, func(i, j int) bool {
		if points[i].Y == points[j].Y {
			return points[i].X < points[j].X
		}
		return points[i].Y < points[j].Y
	})

	// lower hull
	var hull []Point
	for _, point := range points {
		for 2 <= len(hull) && (Orientation(hull[len(hull)-2], hull[len(hull)-1], point) == CollinearPoints ||
			Orientation(hull[len(hull)-2], hull[len(hull)-1], point) == ClockwisePoints) {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, point)
	}
	chain = append(chain, hull...)

	// upper hull
	hull = []Point{}
	for i := len(points) - 1; 0 <= i; i-- {
		point := points[i]
		for 2 <= len(hull) && (Orientation(hull[len(hull)-2], hull[len(hull)-1], point) == CollinearPoints ||
			Orientation(hull[len(hull)-2], hull[len(hull)-1], point) == ClockwisePoints) {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, point)
	}

	// merge hulls
	if 0 < len(chain) && 0 < len(hull) && Distance(chain[len(chain)-1], hull[0]) < Eps {
		hull = hull[1:]
	}
	if 0 < len(chain) && 0 < len(hull) && Distance(chain[0], hull[len(hull)-1]) < Eps {
		hull = hull[:len(hull)-1]
	}
	chain = append(chain, hull...)

	return
}

// SamePoints return true only if point on very distance or
// with same coordinates
func SamePoints(p0 Point, p1 Point) bool {
	if p0.X == p1.X && p0.Y == p1.Y {
		return true
	}
	return Distance(p0, p1) < Eps
}
