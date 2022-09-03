package model

import (
	"fmt"
	"math"
	"sort"
)


type XYId struct {
	XY
	Id int
}

type GrahamScan struct {
	lowest XYId
	pl     []XYId
}

//Implement sort interface
//func (pl PointList) Len() int {
//	return len(pl)
//}

//func (pl PointList) Swap(i, j int) {
//	pl[i], pl[j] = pl[j], pl[i]
//}

func (gs * GrahamScan) compare(i, j int) bool {
	area := gs.area2(gs.lowest.XY, gs.pl[i].XY, gs.pl[j].XY)
	if area == 0 {
		x := math.Abs(gs.pl[i].X - gs.lowest.X) - math.Abs(gs.pl[j].X - gs.lowest.X)
		y := math.Abs(gs.pl[i].Y - gs.lowest.Y) - math.Abs(gs.pl[j].Y - gs.lowest.Y)
		if x < 0 || y < 0 {
			return true
		} else if x > 0 || y > 0 {
			return false
		} else {
			return false
		}
	}
	return area > 0
}

func (gs * GrahamScan) findLowestPoint(pl[]XYId) XYId {
	m := 0
	for i := 1; i < len(pl); i++ {
		//If lowest points are on the same line, take the rightmost point
		if (pl[i].Y < pl[m].Y) || ((pl[i].Y == pl[m].Y) && pl[i].X > pl[m].X) {
			m = i
		}
	}
	return pl[m]
	//pl[0], pl[m] = pl[m], pl[0]
}

func (gs * GrahamScan) isLeft(p0, p1, p2 XY) bool {
	area := gs.area2(p0, p1, p2)
	if area > 0 {
		return true
	}
	if area == 0 {
		//TODO collinear
		return true
	}
	return false
}

func (gs * GrahamScan) area2(a, b, c XY) float64 {
	area := (b.X - a.X) * (c.Y-a.Y) - (c.X-a.X) * (b.Y - a.Y)
	return area
}

func (gs * GrahamScan) Partial(lowest XYId, a XYId, b XYId, pl []XYId) XYId {
	first := pl[0]
	//gs.lowest = gs.findLowestPoint(pl)
	//if middle.X < gs.lowest.X { gs.lowest = middle }
	//if lowest.X < gs.lowest.X { gs.lowest = lowest }
	gs.lowest = lowest

	gs.pl = pl
	//sort.SliceStable(gs.pl, gs.compare)

	for x, curr := range gs.pl {
		if gs.isLeft(a.XY, b.XY, curr.XY) {
			for y, next := range gs.pl {
				if x == y { continue }
				if gs.isLeft(b.XY, curr.XY, next.XY) {
					return curr
				}
			}
		}
	}
	return first
}

func (gs * GrahamScan) Compute(pl []XYId) ([]XYId, bool) {
	if len(pl) < 3 {
		return nil, false
	}
	gs.lowest = gs.findLowestPoint(pl)
	gs.pl = pl
	sort.SliceStable(gs.pl, gs.compare)

	stack := new(Stack)
	stack.Push(gs.pl[0])
	stack.Push(gs.pl[1])

	fmt.Println("-START----------------------------------------")
	fmt.Printf("Sorted Points: %v\n", gs.pl)

	i := 2
	for i < len(gs.pl) {
		pi := gs.pl[i]

		fmt.Println("Stack: ", stack.Print())
		p1 := stack.top.next.value.(XYId)
		p2 := stack.top.value.(XYId)

		if gs.isLeft(p1.XY, p2.XY, pi.XY) {
			stack.Push(pi)
			i++
		} else {
			if stack.Len() > 2 {
				stack.Pop()
			} else {
				i++
			}
		}
	}

	//Copy the hull
	ret := make([]XYId, stack.Len())
	top := stack.top
	count := 0
	for top != nil {
		ret[count] = top.value.(XYId)
		top = top.next
		count++
	}

	fmt.Println("-END------------------------------------------------")
	return ret, true
}