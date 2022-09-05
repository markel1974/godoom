package model

import (
	"math"
	"sort"
)


type CHSegment struct {
	Ref     string
	Start   XY
	End     XY
	Data    interface{}
}

func NewCHSegment(ref string, data interface{}, start XY, end XY) * CHSegment {
	chs := &CHSegment{
		Ref:   ref,
		Data:  data,
		Start: start,
		End:   end,
	}
	return chs
}


type ConvexHull struct {
}

func NewConvexHull() * ConvexHull {
	return &ConvexHull{}
}

func (ch * ConvexHull) Create(id string, inputSegments []*CHSegment) []*CHSegment {
	//head := ch.getHead(sect.Segments)
	head := ch.findLowest(inputSegments)
	headSeg := inputSegments[head]
	out := []*CHSegment{ headSeg }
	var lower XY; if ch.isLower(headSeg.Start, headSeg.End) { lower = headSeg.Start } else { lower = headSeg.End }

	var segments []*CHSegment
	for x, seg := range inputSegments {
		if x == head { continue }
		segments = append(segments, seg)
	}

	sort.SliceStable(segments, func (i, j int) bool {
		area := ch.area2(lower, segments[i].Start, segments[j].Start)
		if area == 0 {
			x := math.Abs(segments[i].Start.X - lower.X) - math.Abs(segments[j].Start.X - lower.X)
			y := math.Abs(segments[i].Start.Y - lower.Y) - math.Abs(segments[j].Start.Y - lower.Y)
			if x < 0 || y < 0 { return true }
			if x > 0 || y > 0 { return false }
			return false
		}
		return area > 0
	})

	curr := headSeg

	for len(segments) > 0 {
		target := -1
		if len(segments) == 1 {
			target = 0
		} else if target = ch.getConnect(curr, segments); target < 0 {
			target = ch.getConvex(curr, segments)
		}
		curr = segments[target]
		segments = append(segments[:target], segments[target+1:]...)

		last := out[len(out)-1]
		if last.End.X != curr.Start.X || last.End.Y != curr.Start.Y {
			chs := NewCHSegment(id,nil, last.End, curr.Start)
			out = append(out, chs)
		}
		out = append(out, curr)
	}

	if len(out) > 1 {
		first := out[0]
		last := out[len(out)-1]
		if first.Start.X != last.End.X || first.Start.Y != last.End.Y {
			chs := NewCHSegment(id,nil, last.End, first.Start)
			out = append(out, chs)
		}
	}
	//fmt.Println("------ Original")
	//for idx, seg := range inputSegments {
	//	fmt.Printf("%d: %.0f %.0f %.0f %.0f\n", idx, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y)
	//}
	//fmt.Println("------ Altered", id)
	//for _, seg := range out {
	//	created := false; if seg.Data == nil { created = true }
	//	fmt.Printf("%.0f %.0f %.0f %.0f: %v\n", seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y, created)
	//}
	return out
}

func (ch * ConvexHull) getConnect(curr *CHSegment, segments[]*CHSegment) int {
	for idx, target := range segments {
		if curr.End.X == target.Start.X && curr.End.Y == target.Start.Y { return idx }
	}
	return -1
}

func (ch * ConvexHull) getConvex(mainPoint *CHSegment, segments []*CHSegment) int {
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

func (ch * ConvexHull) findLowest(segments []*CHSegment) int {
	lowest := 0
	for i := 1; i < len(segments); i++ {
		if ch.isLower(segments[i].Start, segments[lowest].Start) { lowest = i }
		if ch.isLower(segments[i].End, segments[lowest].End) { lowest = i }
	}
	return lowest
}

func (ch * ConvexHull) isLower(a XY, b XY) bool {
	//If lowest points are on the same line, take the rightmost point
	if (a.Y < b.Y) || ((a.Y == b.Y) && a.X > b.X) {
		return true
	}
	return false
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

func (ch * ConvexHull) FromSector(sector * Sector) []*Segment {
	var chs []*CHSegment
	var out []*Segment
	for _, s := range sector.Segments {
		chs = append(chs, NewCHSegment(sector.Id, s, s.Start, s.End))
	}
	for _, s := range ch.Create(sector.Id, chs) {
		if s.Data != nil {
			out = append(out, s.Data.(*Segment))
		} else {
			ns := NewSegment(sector.Id, sector, DefinitionVoid, s.Start, s.End, "ADDED")
			out = append(out, ns)
		}
	}
	return out
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