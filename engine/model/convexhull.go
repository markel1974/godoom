package model

import (
	"fmt"
	"math"
	"sort"
)

type ConvexHull struct {
}

func (ch * ConvexHull) Create(sect * Sector) []*Segment {
	//head := ch.getHead(sect.Segments)
	head := ch.findLowest(sect.Segments)
	lowest := sect.Segments[head]
	curr := lowest
	out := []*Segment{ curr }

	var segments []*Segment
	for x := 0; x < len(sect.Segments); x++ {
		if x == head { continue }
		segments = append(segments, sect.Segments[x])
	}

	sort.SliceStable(segments, func (i, j int) bool {
		area := ch.area2(lowest.Start, segments[i].Start, segments[j].Start)
		if area == 0 {
			x := math.Abs(segments[i].Start.X - lowest.Start.X) - math.Abs(segments[j].Start.X - lowest.Start.X)
			y := math.Abs(segments[i].Start.Y - lowest.Start.Y) - math.Abs(segments[j].Start.Y - lowest.Start.Y)
			if x < 0 || y < 0 { return true }
			if x > 0 || y > 0 { return false }
			return false
		}
		return area > 0
	})

	for len(segments) > 0 {
		target := -1
		if len(segments) == 1 {
			target = 0
		} else if target = ch.getConnect(curr, segments); target < 0 {
			target = ch.getConvex(lowest, curr, segments)
		}
		curr = segments[target]
		segments = append(segments[:target], segments[target + 1:]...)
		out = append(out, curr)
	}
	//fmt.Println("------ Original")
	for idx, seg := range sect.Segments {
		fmt.Printf("%d: %.0f %.0f %.0f %.0f\n", idx, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y)
	}
	fmt.Println("------ Altered", sect.Id)
	for _, seg := range out {
		fmt.Printf("%.0f %.0f %.0f %.0f\n", seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y)
	}
	first := out[0]
	last := out[len(out) -1]

	if first.Start.X != last.End.X ||  first.Start.Y != last.End.Y {
		ns := NewSegment("TEST", sect, DefinitionWall, last.End, first.Start, "VERIFICA")
		out = append(out, ns)
	}
	return out
}

func (ch * ConvexHull) getConnect(curr *Segment, segments[]*Segment) int {
	for idx, target := range segments {
		if curr.End.X == target.Start.X && curr.End.Y == target.Start.Y { return idx }
	}
	return -1
}

func (ch * ConvexHull) getConvex(_ *Segment, mainPoint *Segment, segments []*Segment) int {
	/*
	var test []XYId
	for idx, p := range segments {
		test = append(test, XYId{XY: p.Start, Id: idx})
	}
	gs := &GrahamScan{}
	v := gs.Partial(XYId{XY:head.Start, Id: -3}, XYId{XY:mainPoint.Start, Id: -2}, XYId{XY:mainPoint.End, Id: -1}, test)
	return v.Id
	*/
	for x, curr := range segments {
		if ch.isLeft(mainPoint.Start, mainPoint.End, curr.Start) {
			for y, next := range segments {
				if x == y { continue }
				if ch.isLeft(mainPoint.End, curr.Start, next.Start) {
					return x
				}
			}
		}
	}
	return 0
}

func (ch * ConvexHull) findLowest(segments []*Segment) int {
	lowest := 0
	for i := 1; i < len(segments); i++ {
		start := segments[i].Start
		end := segments[i].End
		lowestStart := segments[lowest].Start
		//If lowest points are on the same line, take the rightmost point
		if (start.Y < lowestStart.Y) || ((start.Y == lowestStart.Y) && start.X > lowestStart.X) {
			lowest = i
		}
		lowestEnd := segments[lowest].End
		if (end.Y < lowestEnd.Y) || ((end.Y == lowestEnd.Y) && end.X > lowestEnd.X) {
			lowest = i
		}
	}
	return lowest
}

func (ch * ConvexHull) area2(a XY, b XY, c XY) float64 {
	area := (b.X - a.X) * (c.Y - a.Y) - (c.X - a.X) * (b.Y - a.Y)
	return area
}

func (ch * ConvexHull) isLeft(p0 XY, p1 XY, p2 XY) bool {
	area := ch.area2(p0, p1, p2)
	if area > 0 {
		return true
	}
	if area == 0 {
		//fmt.Println(ch.sameLine(p0, p1, p2))
		//TODO collinear
		//return true
		return ch.sameLine(p0, p1, p2)
	}
	return false
}

//judge the position of three points on the same line
func (ch * ConvexHull) sameLine(a XY, b XY, c XY) bool {
	//dot product
	d := (b.X - a.X) * (c.Y - a.Y) + (c.X - a.X) * (b.Y - a.Y)
	return d <= 0
}

/*

func (ch * ConvexHull) getHead(segments []*Segment) int {
	var connected []int
	for x := 0; x < len(segments); x++ {
		source := segments[x]
		for y, seg := range segments {
			if x == y { continue }
			if source.End.X == seg.Start.X && source.End.Y == seg.Start.Y {
				connected = append(connected, x)
			}
		}
	}
	if len(connected) == 0 { return 0 }
	if len (connected) == 1 { return connected[0] }

	for x := 0; x < len(connected); x++ {
		index := connected[x]
		source := segments[index]
		found := false
		for y, target := range segments {
			if x == y { continue }
			if source.Start.X == target.End.X && source.Start.Y == target.End.Y {
				found = true
				break
			}
		}
		if !found { return index }
	}
	return connected[0]
}

func (ch * ConvexHull) findAngle(l1s XY, l1e XY, l2s XY, l2e XY) float64 {
	M1 := ch.slope(l1s.X, l1s.Y, l1e.X, l1e.Y)
	M2 := ch.slope(l2s.X, l2s.Y, l2e.X, l2e.Y)
	angle := math.Abs((M2 - M1) / (1 + M1 * M2))
	ret := math.Atan(angle)
	val := (ret * 180) / math.Pi
	return val
}

func (ch * ConvexHull) slope(x1 float64, y1 float64, x2 float64, y2 float64) float64 {
	d := x2 - x1
	if d == 0 { return 0 }
	return (y2 - y1) / d
}
*/