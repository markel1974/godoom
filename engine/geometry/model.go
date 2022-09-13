package geometry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Model of points, lines, arcs for prepare of triangulation
type Model struct {
	Points    []Point  // Points is slice of points
	Lines     [][3]int // Lines store 2 index of Points and last for tag
	Arcs      [][4]int // Arcs store 3 index of Points and last for tag
	Triangles [][4]int // Triangles store 3 index of Points and last for tag/material
}

func NewModel() *Model{
	return &Model{}
}

// TagProperty return length of lines, area of triangles for each tag.
// Arcs are ignored
func (m *Model) TagProperty() (length []float64, area []float64) {
	// prepare slices
	max := 0
	for i := range m.Lines {
		if max < m.Lines[i][2] {
			max = m.Lines[i][2]
		}
	}
	for i := range m.Triangles {
		if max < m.Triangles[i][3] {
			max = m.Triangles[i][3]
		}
	}
	length = make([]float64, max+1)
	area = make([]float64, max+1)
	// calculate data
	for i := range m.Lines {
		length[m.Lines[i][2]] += math.Hypot(
			m.Points[m.Lines[i][1]].X-m.Points[m.Lines[i][0]].X,
			m.Points[m.Lines[i][1]].Y-m.Points[m.Lines[i][0]].Y,
		)
	}
	for i := range m.Triangles {
		area[m.Triangles[i][3]] += Area(
			m.Points[m.Triangles[i][0]],
			m.Points[m.Triangles[i][1]],
			m.Points[m.Triangles[i][2]],
		)
	}
	return
}

// Copy return copy of Model
func (m *Model) Copy() (dst Model) {
	// Points
	dst.Points = make([]Point, len(m.Points))
	copy(dst.Points, m.Points)
	// Lines
	dst.Lines = make([][3]int, len(m.Lines))
	copy(dst.Lines, m.Lines)
	// Arcs
	dst.Arcs = make([][4]int, len(m.Arcs))
	copy(dst.Arcs, m.Arcs)
	// Triangles
	dst.Triangles = make([][4]int, len(m.Triangles))
	copy(dst.Triangles, m.Triangles)
	return
}

// String return a standard model view
func (m *Model) String() string {
	var str string
	if 0 < len(m.Points) {
		str += "Points:\n"
	}
	for i := range m.Points {
		str += fmt.Sprintf("%03d\t%+.4f\n", i, m.Points[i])
	}
	if 0 < len(m.Lines) {
		str += "Lines:\n"
	}
	for i := range m.Lines {
		str += fmt.Sprintf("%03d\t%3d\n", i, m.Lines[i])
	}
	if 0 < len(m.Arcs) {
		str += "Arcs:\n"
	}
	for i := range m.Arcs {
		str += fmt.Sprintf("%03d\t%3d\n", i, m.Arcs[i])
	}
	if 0 < len(m.Triangles) {
		str += "Triangles:\n"
	}
	for i := range m.Triangles {
		str += fmt.Sprintf("%03d\t%3d\n", i, m.Triangles[i])
	}
	return str
}

// Dxf return string in dxf drawing format
// https://images.autodesk.com/adsk/files/autocad_2012_pdf_dxf-reference_enu.pdf
func (m *Model) Dxf() string {
	// create buffer
	var buf bytes.Buffer

	// start dxf
	_, _ = fmt.Fprintf(&buf, "0\nSECTION\n")
	_, _ = fmt.Fprintf(&buf, "2\nENTITIES\n")

	line := func(st, en Point, layer string) {
		_, _ = fmt.Fprintf(&buf, "0\nLINE\n")
		_, _ = fmt.Fprintf(&buf, "8\n%s\n", layer) // layer
		_, _ = fmt.Fprintf(&buf, "10\n%f\n", st.X) // start point X
		_, _ = fmt.Fprintf(&buf, "20\n%f\n", st.Y) // start point Y
		_, _ = fmt.Fprintf(&buf, "30\n%f\n", 0.0)  // start point Z
		_, _ = fmt.Fprintf(&buf, "11\n%f\n", en.X) // end point X
		_, _ = fmt.Fprintf(&buf, "21\n%f\n", en.Y) // end point Y
		_, _ = fmt.Fprintf(&buf, "31\n%f\n", 0.0)  // end point Z
	}

	// TODO
	// text := func(str string, p1, p2 Point) {
	// }

	if 1 < len(m.Points) {
		// draw convex
		{
			cps := ConvexHull(m.Points)
			for i := 1; i < len(cps); i++ {
				line(cps[i-1], cps[i], "convex")
			}
			line(cps[len(cps)-1], cps[0], "convex")
		}
		// draw lines
		for i := range m.Lines {
			name := fmt.Sprintf("lines%+2d", m.Lines[i][2])
			line(m.Points[m.Lines[i][0]], m.Points[m.Lines[i][1]], name)
		}
		// draw arc
		for i := range m.Arcs {
			name := fmt.Sprintf("arcs%+2d", m.Arcs[i][3])
			line(m.Points[m.Arcs[i][0]], m.Points[m.Arcs[i][1]], name)
			line(m.Points[m.Arcs[i][1]], m.Points[m.Arcs[i][2]], name)
		}
		// draw triangles
		for i := range m.Triangles {
			name := fmt.Sprintf("triangles%+2d", m.Triangles[i][3])
			line(m.Points[m.Triangles[i][0]], m.Points[m.Triangles[i][1]], name)
			line(m.Points[m.Triangles[i][1]], m.Points[m.Triangles[i][2]], name)
			line(m.Points[m.Triangles[i][2]], m.Points[m.Triangles[i][0]], name)
		}
	}

	// end dxf
	_, _ = fmt.Fprintf(&buf, "0\nENDSEC\n")
	_, _ = fmt.Fprintf(&buf, "0\nEOF\n")

	return buf.String()
}

// AddPoint return index in model slice point
func (m *Model) AddPoint(p Point) (index int) {
	// search in exist points
	for i := range m.Points {
		if p.X == m.Points[i].X && p.Y == m.Points[i].Y {
			return i
		}
	}
	for i := range m.Points {
		if SamePoints(p, m.Points[i]) {
			return i
		}
	}
	// new point
	m.Points = append(m.Points, p)
	return len(m.Points) - 1
}

// AddLine add line into model with specific tag
func (m *Model) AddLine(start Point, end Point, tag int) {
	// add points
	st := m.AddPoint(start)
	en := m.AddPoint(end)

	// do not add line with same id
	for i := range m.Lines {
		if m.Lines[i][0] == st && m.Lines[i][1] == en {
			m.Lines[i][2] = tag
			return
		}
	}
	// add line
	m.Lines = append(m.Lines, [3]int{st, en, tag})
}

// AddMultiline add many lines with specific tag
func (m *Model) AddMultiline(tag int, ps ...Point) {
	if len(ps) < 2 {
		return
	}
	for i := range ps {
		if i == 0 {
			continue
		}
		m.AddLine(ps[i-1], ps[i], tag)
	}
}

// AddArc add arc into model with specific tag
func (m *Model) AddArc(start, middle, end Point, tag int) {
	// add points
	st := m.AddPoint(start)
	mi := m.AddPoint(middle)
	en := m.AddPoint(end)

	// do not add line with same id
	for i := range m.Arcs {
		if (m.Arcs[i][0] == st && m.Arcs[i][1] == mi && m.Arcs[i][2] == en) ||
			(m.Arcs[i][2] == st && m.Arcs[i][1] == mi && m.Arcs[i][0] == en) {
			m.Arcs[i][3] = tag
			return
		}
	}
	// add arc
	m.Arcs = append(m.Arcs, [4]int{st, mi, en, tag})
}

// AddTriangle add triangle into model with specific tag/material
func (m *Model) AddTriangle(start, middle, end Point, tag int) {
	// add points
	st := m.AddPoint(start)
	mi := m.AddPoint(middle)
	en := m.AddPoint(end)

	// do not add line with same id
	for i := range m.Triangles {
		if (m.Triangles[i][0] == st && m.Triangles[i][1] == mi && m.Triangles[i][2] == en) ||
			(m.Triangles[i][2] == st && m.Triangles[i][1] == mi && m.Triangles[i][0] == en) {
			m.Triangles[i][3] = tag
			return
		}
	}
	// add arc
	m.Triangles = append(m.Triangles, [4]int{st, mi, en, tag})
}

// AddCircle add arcs based on circle geometry into model with specific tag
func (m *Model) AddCircle(xc, yc, r float64, tag int) {
	// add points
	up    := Point{X: xc, Y: yc + r}
	down  := Point{X: xc, Y: yc - r}
	left  := Point{X: xc - r, Y: yc}
	right := Point{X: xc + r, Y: yc}

	// add arcs
	// CounterClockwisePoints
	m.AddArc(down, right, up, tag)
	m.AddArc(up, left, down, tag)
}

// AddModel inject model into model
func (m *Model) AddModel(from Model) {
	for _, l := range from.Lines {
		m.AddLine(from.Points[l[0]], from.Points[l[1]], l[2])
	}
	for _, a := range from.Arcs {
		m.AddArc(from.Points[a[0]], from.Points[a[1]], from.Points[a[2]], a[3])
	}
	for _, t := range from.Triangles {
		m.AddTriangle(from.Points[t[0]], from.Points[t[1]], from.Points[t[2]], t[3])
	}
}

// Intersection change model with finding all model intersections
func (m *Model) Intersection() {
	// value `ai` is amount of intersections
	box := func(ps []Point) (xmin, xmax, ymin, ymax float64) {
		xmin = +math.MaxFloat64
		xmax = -math.MaxFloat64
		ymin = +math.MaxFloat64
		ymax = -math.MaxFloat64
		for i := range ps {
			xmin = math.Min(xmin, ps[i].X)
			xmax = math.Max(xmax, ps[i].X)
			ymin = math.Min(ymin, ps[i].Y)
			ymax = math.Max(ymax, ps[i].Y)
		}
		return
	}
	boxIntersect := func(A, B []Point) bool {
		Axin, Axax, Ayin, Ayax := box(A)
		Bxin, Bxax, Byin, Byax := box(B)
		if Axax < Bxin {
			return false
		}
		if Bxax < Axin {
			return false
		}
		if Ayax < Byin {
			return false
		}
		if Byax < Ayin {
			return false
		}
		// try or may-be
		return true
	}

	// find intersections
	fs := []func() int{
		// line-line intersection
		func() (ai int) {
			var (
				intersect = make([]bool, len(m.Lines))
				size      = len(m.Lines)
				s1        = make([]Point, 2)
				s2        = make([]Point, 2)
			)
			for il := 0; il < size; il++ {
				for jl := 0; jl < size; jl++ {
					// ignore intersection lines
					if il <= jl || intersect[il] || intersect[jl] {
						continue
					}
					s1[0] = m.Points[m.Lines[il][0]]
					s1[1] = m.Points[m.Lines[il][1]]
					s2[0] = m.Points[m.Lines[jl][0]]
					s2[1] = m.Points[m.Lines[jl][1]]
					if !boxIntersect(s1, s2) {
						continue
					}
					// analyse
					pi, stA, stB := LineLine(
						m.Points[m.Lines[il][0]], m.Points[m.Lines[il][1]],
						m.Points[m.Lines[jl][0]], m.Points[m.Lines[jl][1]],
					)
					// no intersections
					if 0 == len(pi) {
						continue
					}
					// debug test
					if 1 < len(pi) {
						panic("not valid")
					}
					// not acceptable zero length lines
					if stA.Has(ZeroLengthSegment) ||
						stB.Has(ZeroLengthSegment) {
						panic(fmt.Errorf("zero lenght error"))
					}

					if stA.Has(OnPoint0Segment) && stA.Has(OnPoint1Segment) &&
						stB.Has(OnPoint0Segment) && stB.Has(OnPoint1Segment) {
						intersect[il] = true
						continue
					}

					// intersection on line A
					//
					// for cases - no need update the line:
					// OnPoint0SegmentA, OnPoint1SegmentA
					//
					if stA.Has(OnSegment) && !(stA.Has(OnPoint0Segment) || stA.Has(OnPoint1Segment)) {
						intersect[il] = true
						tag := m.Lines[il][2]
						m.AddLine(m.Points[m.Lines[il][0]], pi[0], tag)
						m.AddLine(pi[0], m.Points[m.Lines[il][1]], tag)
					}
					// intersection on line B
					//
					// for cases - no need update the line:
					// OnPoint0SegmentB, OnPoint1SegmentB
					//
					if stB.Has(OnSegment) && !(stB.Has(OnPoint0Segment) || stB.Has(OnPoint1Segment)) {
						intersect[jl] = true
						tag := m.Lines[jl][2]
						m.AddLine(m.Points[m.Lines[jl][0]], pi[0], tag)
						m.AddLine(pi[0], m.Points[m.Lines[jl][1]], tag)
					}
				}
			}
			for i := size - 1; 0 <= i; i-- {
				if intersect[i] {
					// add to amount intersections
					ai++
					// remove intersection line
					m.Lines = append(m.Lines[:i], m.Lines[i+1:]...)
				}
			}
			return
		},

		// arc-line intersection
		func() (ai int) {
			var (
				intersectLines = make([]bool, len(m.Lines))
				intersectArcs  = make([]bool, len(m.Arcs))
				sizeLines      = len(m.Lines)
				sizeArcs       = len(m.Arcs)
				s1             = make([]Point, 2)
				s2             = make([]Point, 3)
			)
			for il := 0; il < sizeLines; il++ {
				for ja := 0; ja < sizeArcs; ja++ {
					// ignore intersection lines
					if intersectLines[il] || intersectArcs[ja] {
						continue
					}
					s1[0] = m.Points[m.Lines[il][0]]
					s1[1] = m.Points[m.Lines[il][1]]
					s2[0] = m.Points[m.Arcs[ja][0]]
					s2[1] = m.Points[m.Arcs[ja][1]]
					s2[2] = m.Points[m.Arcs[ja][2]]
					if !boxIntersect(s1, s2) {
						continue
					}
					// analyse
					pi, stA, stB := LineArc(
						// Line
						m.Points[m.Lines[il][0]], m.Points[m.Lines[il][1]],
						// Arc
						m.Points[m.Arcs[ja][0]],
						m.Points[m.Arcs[ja][1]],
						m.Points[m.Arcs[ja][2]],
					)
					// not acceptable zero length lines
					if stA.Has(ZeroLengthSegment) ||
						stB.Has(ZeroLengthSegment) {
						panic(fmt.Errorf("zero lenght error"))
					}
					// intersection on line A
					//
					// for cases - no need update the line:
					// OnPoint0Segment, OnPoint1Segment
					//
					if stA.Has(OnSegment) {
						// remove OnPoint
						roots := make([]Point, len(pi))
						copy(roots, pi)

					same1:
						for i := range roots {
							for j := 0; j < 2; j++ {
								if Distance(roots[i], m.Points[m.Lines[il][j]]) < Eps {
									roots = append(roots[:i], roots[i+1:]...)
									goto same1
								}
							}
						}

						if 0 < len(roots) {
							intersectLines[il] = true
							tag := m.Lines[il][2]
							switch len(roots) {
							case 1:
								m.AddLine(m.Points[m.Lines[il][0]], roots[0], tag)
								m.AddLine(roots[0], m.Points[m.Lines[il][1]], tag)
							case 2:
								if stA.Has(VerticalSegment) {
									if roots[1].Y < roots[0].Y {
										roots[0], roots[1] = roots[1], roots[0]
									}
									// roots[0].Y < roots[1].Y
									if m.Points[m.Lines[il][0]].Y < m.Points[m.Lines[il][1]].Y {
										// Design:
										//
										//	| Lines [1]
										//	| roots[1]
										//	| roots[0]
										//	| Lines [0]
										m.AddLine(m.Points[m.Lines[il][0]], roots[0], tag)
										m.AddLine(roots[0], roots[1], tag)
										m.AddLine(roots[1], m.Points[m.Lines[il][1]], tag)
									} else {
										// Design:
										//
										//	| Lines [0]
										//	| roots[1]
										//	| roots[0]
										//	| Lines [1]
										m.AddLine(m.Points[m.Lines[il][1]], roots[0], tag)
										m.AddLine(roots[0], roots[1], tag)
										m.AddLine(roots[1], m.Points[m.Lines[il][0]], tag)
									}
								} else {
									// Not vertical line
									if roots[1].X < roots[0].X {
										roots[0], roots[1] = roots[1], roots[0]
									}
									// roots[0].X < roots[1].X
									if m.Points[m.Lines[il][0]].X < m.Points[m.Lines[il][1]].X {
										// Design:
										//
										//	 Lines[0]    roots[0]   roots[1]   Lines[1]
										m.AddLine(m.Points[m.Lines[il][0]], roots[0], tag)
										m.AddLine(roots[0], roots[1], tag)
										m.AddLine(roots[1], m.Points[m.Lines[il][1]], tag)
									} else {
										// Design:
										//
										//	 Lines[1]    roots[0]   roots[1]   Lines[0]
										m.AddLine(m.Points[m.Lines[il][1]], roots[0], tag)
										m.AddLine(roots[0], roots[1], tag)
										m.AddLine(roots[1], m.Points[m.Lines[il][0]], tag)
									}
								}
							default:
								panic("not valid")
							}
						}
					}
					// intersection on arc B
					//
					// for cases - no need update the line:
					// OnPoint0SegmentB, OnPoint1SegmentB
					//
					if stB.Has(OnSegment) {
						tag := m.Arcs[ja][3]
						res, err := ArcSplitByPoint(
							m.Points[m.Arcs[ja][0]],
							m.Points[m.Arcs[ja][1]],
							m.Points[m.Arcs[ja][2]],
							pi...)
						if err != nil {
							// TODO	panic(err)
							err = nil
						} else {
							for i := range res {
								intersectArcs[ja] = true
								m.AddArc(res[i][0], res[i][1], res[i][2], tag)
							}
						}
					}
				}
			}
			for i := sizeLines - 1; 0 <= i; i-- {
				if intersectLines[i] {
					// add to amount intersections
					ai++
					// remove intersection line
					m.Lines = append(m.Lines[:i], m.Lines[i+1:]...)
				}
			}
			for i := sizeArcs - 1; 0 <= i; i-- {
				if intersectArcs[i] {
					// add to amount intersections
					ai++
					// remove intersection arcs
					m.Arcs = append(m.Arcs[:i], m.Arcs[i+1:]...)
				}
			}
			return
		},

		// point-arc intersection
		func() (ai int) {
			intersectArcs := make([]bool, len(m.Arcs))
			sizeArcs      := len(m.Arcs)
			s1            := make([]Point, 1)
			s2            := make([]Point, 3)

			for ip := 0; ip < len(m.Points); ip++ {
				for ja := 0; ja < sizeArcs; ja++ {
					// ignore intersection lines
					if intersectArcs[ja] {
						continue
					}
					// ignore arc middle points only if not by another
					// line or arc
					if Distance(m.Points[m.Arcs[ja][1]], m.Points[ip]) < Eps {
						ignore := true
						for i := range m.Lines {
							if m.Lines[i][0] == ip || m.Lines[i][1] == ip {
								ignore = false
							}
						}
						for i := range m.Arcs {
							if i == ja {
								continue
							}
							if m.Arcs[i][0] == ip || m.Arcs[i][1] == ip || m.Arcs[i][2] == ip {
								ignore = false
							}
						}
						if ignore {
							continue
						}
					}
					s1[0] = m.Points[ip]
					s2[0] = m.Points[m.Arcs[ja][0]]
					s2[1] = m.Points[m.Arcs[ja][1]]
					s2[2] = m.Points[m.Arcs[ja][2]]
					if !boxIntersect(s1, s2) {
						continue
					}

					// analyse
					pi, _, stB := PointArc(m.Points[ip], m.Points[m.Arcs[ja][0]], m.Points[m.Arcs[ja][1]], m.Points[m.Arcs[ja][2]])
					// not acceptable zero length lines
					if stB.Has(ZeroLengthSegment) {
						panic(fmt.Errorf("zero lenght error"))
					}
					// intersection on arc B
					//
					// for cases - no need update the line:
					// OnPoint0SegmentB, OnPoint1SegmentB
					//
					if stB.Has(OnSegment) && 0 < len(pi) {
						tag := m.Arcs[ja][3]
						res, err := ArcSplitByPoint(m.Points[m.Arcs[ja][0]], m.Points[m.Arcs[ja][1]], m.Points[m.Arcs[ja][2]], pi...)
						if err != nil {
							// TODO	panic(err)
							err = nil
						} else {
							for i := range res {
								intersectArcs[ja] = true
								m.AddArc(res[i][0], res[i][1], res[i][2], tag)
							}
						}
					}
				}
			}
			for i := sizeArcs - 1; 0 <= i; i-- {
				if intersectArcs[i] {
					// add to amount intersections
					ai++
					// remove intersection arcs
					m.Arcs = append(m.Arcs[:i], m.Arcs[i+1:]...)
				}
			}
			return
		},

		// point-line intersection
		func() (ai int) {
			var (
				intersectLines = make([]bool, len(m.Lines))
				sizeLines      = len(m.Lines)
				s1             = make([]Point, 1)
				s2             = make([]Point, 2)
			)
			for ip := 0; ip < len(m.Points); ip++ {
				for ja := 0; ja < sizeLines; ja++ {
					// ignore intersection lines
					if intersectLines[ja] {
						continue
					}
					s1[0] = m.Points[ip]
					s2[0] = m.Points[m.Lines[ja][0]]
					s2[1] = m.Points[m.Lines[ja][1]]
					if !boxIntersect(s1, s2) {
						continue
					}
					// analyse
					pi, _, stB := PointLine(
						// Point
						m.Points[ip],
						// Arc
						m.Points[m.Lines[ja][0]],
						m.Points[m.Lines[ja][1]],
					)
					// not acceptable zero length lines
					if stB.Has(ZeroLengthSegment) {
						panic(fmt.Errorf("zero lenght error for line: %v", m.Lines[ja]))
					}
					// intersection on line B
					//
					// for cases - no need update the line:
					// OnPoint0SegmentB, OnPoint1SegmentB
					//
					if stB.Has(OnSegment) {
						tag := m.Lines[ja][2]
						for _, p := range pi {
							intersectLines[ja] = true
							m.AddLine(m.Points[m.Lines[ja][0]], p, tag)
							m.AddLine(m.Points[m.Lines[ja][1]], p, tag)
						}
					}
				}
			}
			for i := sizeLines - 1; 0 <= i; i-- {
				if intersectLines[i] {
					// add to amount intersections
					ai++
					// remove intersection arcs
					m.Lines = append(m.Lines[:i], m.Lines[i+1:]...)
				}
			}
			return
		},

		// point-point intersection
		// TODO

		// point-triangle intersection
		func() (ai int) {
			var (
				intersectTr = make([]bool, len(m.Triangles))
				sizeTrs     = len(m.Triangles)
				s1          = make([]Point, 1)
				s2          = make([]Point, 3)
			)
			for ip := 0; ip < len(m.Points); ip++ {
				for jt := 0; jt < sizeTrs; jt++ {
					// ignore intersection lines
					if intersectTr[jt] {
						continue
					}
					s1[0] = m.Points[ip]
					s2[0] = m.Points[m.Triangles[jt][0]]
					s2[1] = m.Points[m.Triangles[jt][1]]
					s2[2] = m.Points[m.Triangles[jt][2]]
					if !boxIntersect(s1, s2) {
						continue
					}
					tag := m.Triangles[jt][3]
					res, _, err := TriangleSplitByPoint(
						// Point
						m.Points[ip],
						// Triangle
						m.Points[m.Triangles[jt][0]],
						m.Points[m.Triangles[jt][1]],
						m.Points[m.Triangles[jt][2]],
					)
					if err != nil {
						// TODO	panic(err)
						err = nil
					} else {
						for i := range res {
							intersectTr[jt] = true
							m.AddTriangle(res[i][0], res[i][1], res[i][2], tag)
						}
					}
				}
			}
			for i := sizeTrs - 1; 0 <= i; i-- {
				if intersectTr[i] {
					// add to amount intersections
					ai++
					// remove intersection triangles
					m.Triangles = append(m.Triangles[:i], m.Triangles[i+1:]...)
				}
			}
			return
		},
	}
	for iter := 0; ; iter++ {
		ai := 0
		for i := range fs {
			ai += fs[i]()
		}
		if ai == 0 {
			break
		}
		if iter == 1000 {
			panic("too many intersections")
		}
	}
}


/*
func (m *Model) Get(mesh *Mesh) {
	if mesh.Log {
		log.Printf("Get")
	}
	for _, tr := range mesh.model.Triangles {
		if tr[0] == Removed {
			continue
		}
		m.AddTriangle(
			mesh.model.Points[tr[0]],
			mesh.model.Points[tr[1]],
			mesh.model.Points[tr[2]],
			tr[3],
		)
	}
	m.Intersection()
}
*/

func (m *Model) Merge(from Model) {
	for i := range from.Points {
		m.AddPoint(from.Points[i])
	}
	for i := range from.Lines {
		m.AddLine(
			from.Points[from.Lines[i][0]],
			from.Points[from.Lines[i][1]],
			from.Lines[i][2],
		)
	}
	for i := range from.Arcs {
		m.AddArc(
			from.Points[from.Arcs[i][0]],
			from.Points[from.Arcs[i][1]],
			from.Points[from.Arcs[i][2]],
			from.Arcs[i][3],
		)
	}
	for i := range from.Triangles {
		m.AddTriangle(
			from.Points[from.Triangles[i][0]],
			from.Points[from.Triangles[i][1]],
			from.Points[from.Triangles[i][2]],
			from.Triangles[i][3],
		)
	}
}

// Rotate all points of model around point {xc,yc}
func (m *Model) Rotate(xc, yc, angle float64) {
	for i := range m.Points {
		m.Points[i] = Rotate(xc, yc, angle, m.Points[i])
	}
}

// Move all points of model
func (m *Model) Move(dx, dy float64) {
	for i := range m.Points {
		m.Points[i] = Point{
			X: m.Points[i].X + dx,
			Y: m.Points[i].Y + dy,
		}
	}
}

// RemovePoint removed point in accoding to function `filter`
func (m *Model) RemovePoint(remove func(p Point) bool) {
	pt := make([]bool, len(m.Points))
	for i := range m.Points {
		pt[i] = !remove(m.Points[i])
	}
	var rs []int
	for i := range pt {
		if pt[i] {
			continue
		}
		rs = append(rs, i)
	}
	m.removePointByIndex(rs...)
}

// RemoveEmptyPoints removed point not connected to line, arcs, triangles
func (m *Model) RemoveEmptyPoints() {
	// find all used points
	pt := make([]bool, len(m.Points))
	for i := range m.Lines {
		for j := 0; j < 2; j++ {
			pt[m.Lines[i][j]] = true
		}
	}
	for i := range m.Arcs {
		for j := 0; j < 3; j++ {
			pt[m.Arcs[i][j]] = true
		}
	}
	for i := range m.Triangles {
		for j := 0; j < 3; j++ {
			pt[m.Triangles[i][j]] = true
		}
	}
	var remove []int
	for i := range pt {
		if pt[i] {
			continue
		}
		remove = append(remove, i)
	}
	m.removePointByIndex(remove...)
}

func (m *Model) removePointByIndex(remove ...int) {
	if len(remove) == 0 {
		return
	}
	// sort
	sort.Ints(remove)
	// check
	for i := range remove {
		if i == 0 {
			continue
		}
		if remove[i-1] == remove[i] {
			panic("same indexes")
		}
	}
	// reverse
	for i := len(remove)/2 - 1; i >= 0; i-- {
		opp := len(remove) - 1 - i
		remove[i], remove[opp] = remove[opp], remove[i]
	}
	// removing
	for _, r := range remove {
		// remove points in lines
		for i, size := len(m.Lines)-1, 2; 0 <= i; i-- {
			found := false
			for j := 0; j < size; j++ {
				if r == m.Lines[i][j] {
					found = true
				}
			}
			if !found {
				continue
			}
			m.Lines = append(m.Lines[:i], m.Lines[i+1:]...)
		}
		// correction of point index
		for i := range m.Lines {
			for j := 0; j < 2; j++ {
				if r < m.Lines[i][j] {
					m.Lines[i][j]--
				}
			}
		}
		// remove points in arcs
		for i, size := len(m.Arcs)-1, 3; 0 <= i; i-- {
			found := false
			for j := 0; j < size; j++ {
				if r == m.Arcs[i][j] {
					found = true
				}
			}
			if !found {
				continue
			}
			m.Arcs = append(m.Arcs[:i], m.Arcs[i+1:]...)
		}
		// correction of point index
		for i := range m.Arcs {
			for j := 0; j < 3; j++ {
				if r < m.Arcs[i][j] {
					m.Arcs[i][j]--
				}
			}
		}
		// remove points in triangles
		for i, size := len(m.Triangles)-1, 3; 0 <= i; i-- {
			found := false
			for j := 0; j < size; j++ {
				if r == m.Triangles[i][j] {
					found = true
				}
			}
			if !found {
				continue
			}
			m.Triangles = append(m.Triangles[:i], m.Triangles[i+1:]...)
		}
		// correction of point index
		for i := range m.Triangles {
			for j := 0; j < 3; j++ {
				if r < m.Triangles[i][j] {
					m.Triangles[i][j]--
				}
			}
		}
		// remove points
		m.Points = append(m.Points[:r], m.Points[r+1:]...)
	}
}

// Split all model lines, arcs by distance `d`
func (m *Model) Split(d float64) {
	if d <= 0 {
		panic("negative or zero split distance")
	}
	{
		// split lines
		size := len(m.Lines)
		split := make([]bool, size)
		for il := 0; il < size; il++ {
			distance := Distance(m.Points[m.Lines[il][0]], m.Points[m.Lines[il][1]])
			if distance <= d {
				continue
			}
			split[il] = true
			var (
				// amount intermediant points
				am = int(distance/d) + 1
				// step
				dx = (m.Points[m.Lines[il][1]].X - m.Points[m.Lines[il][0]].X) / float64(am)
				dy = (m.Points[m.Lines[il][1]].Y - m.Points[m.Lines[il][0]].Y) / float64(am)
			)
			// add new lines
			for i := 1; i <= am; i++ {
				m.AddLine(
					Point{
						X: m.Points[m.Lines[il][0]].X + dx*float64(i-1),
						Y: m.Points[m.Lines[il][0]].Y + dy*float64(i-1),
					},
					Point{
						X: m.Points[m.Lines[il][0]].X + dx*float64(i),
						Y: m.Points[m.Lines[il][0]].Y + dy*float64(i),
					},
					m.Lines[il][2],
				)
			}
		}
		// remove split lines
		for il := size - 1; 0 <= il; il-- {
			if !split[il] {
				continue
			}
			m.Lines = append(m.Lines[:il], m.Lines[il+1:]...)
		}
	}
	{
		// split arcs
		size := len(m.Arcs)
		split := make([]bool, size)
		for ia := 0; ia < size; ia++ {
			arcs := [][3]Point{{
				m.Points[m.Arcs[ia][0]],
				m.Points[m.Arcs[ia][1]],
				m.Points[m.Arcs[ia][2]],
			}}

			for iter := 0; iter < 100; iter++ {
				// preliminary calculation arc length
				distance := 2.0 * Distance(arcs[len(arcs)-1][0], arcs[len(arcs)-1][1])
				if distance <= d {
					break
				}
				var arcs2 [][3]Point
				for i := range arcs {
					res, err := ArcSplitByPoint(arcs[i][0], arcs[i][1], arcs[i][2])
					if err != nil {
						panic(fmt.Errorf("Arc: %v", arcs[len(arcs)-1]))
					}
					arcs2 = append(arcs2, res...)
				}
				arcs = arcs2
			}

			if len(arcs) == 1 {
				continue
			}
			split[ia] = true
			// add new arcs
			for i := range arcs {
				m.AddArc(arcs[i][0], arcs[i][1], arcs[i][2], m.Arcs[ia][3])
			}
		}
		// remove split lines
		for ia := size - 1; 0 <= ia; ia-- {
			if !split[ia] {
				continue
			}
			m.Arcs = append(m.Arcs[:ia], m.Arcs[ia+1:]...)
		}
	}
}

// MinPointDistance return minimal between 2 points
func (m *Model) MinPointDistance() (distance float64) {
	distance = math.MaxFloat64 // default value of distance
	for i := range m.Points {
		for j := range m.Points {
			// ignore
			if i <= j {
				continue
			}
			// calculation
			d := math.Hypot(
				m.Points[i].X-m.Points[j].X,
				m.Points[i].Y-m.Points[j].Y,
			)
			if d < distance {
				distance = d
			}
		}
	}
	return
}

// ArcsToLines convert arc to lines
func (m *Model) ArcsToLines() {
	// center point of arc is ignore
	for i := range m.Arcs {
		m.AddLine(m.Points[m.Arcs[i][0]], m.Points[m.Arcs[i][2]], m.Arcs[i][3])
	}
	// remove arcs
	m.Arcs = nil
}

// ConvexHullTriangles add triangles of model convex hull
func (m *Model) ConvexHullTriangles() {
	cps := ConvexHull(m.Points) // points on convex hull
	for i := 2; i < len(cps); i++ {
		m.AddTriangle(cps[0], cps[i-2], cps[i-1], -1)
	}
}

// Write model into file with filename in JSON format
func (m *Model) Write(filename string) (err error) {
	out, err := m.JSON()
	if err != nil {
		return
	}
	// write into file
	err = os.WriteFile(filename, []byte(out), 0666)
	if err != nil {
		return
	}
	return nil
}

// JSON convert model in JSON format
func (m *Model) JSON() (_ string, err error) {
	// convert into json
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, b, " ", "\t")
	if err != nil {
		return
	}
	return buf.String(), nil
}

// Read model from file with filename in JSON format
func (m *Model) Read(filename string) (err error) {
	// read our opened file as a byte array.
	var dat []byte
	dat, err = os.ReadFile(filename)
	if err != nil {
		return
	}
	if len(dat) == 0 {
		err = fmt.Errorf("file `%s` is empty", filename)
		return
	}
	// we unmarshal our data which contains our slice
	err = json.Unmarshal(dat, m)
	if err != nil {
		return
	}
	return
}
