package wad

import (
	"fmt"
	"github.com/markel1974/godoom/engine/geometry"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/quickhull"
	"os"
)



func (b * Builder) testsEntryPoint(miSectors []*model.InputSector) {
	b.createMeshes(miSectors)
	b.compileSectorBySegment(miSectors)
}

func (b * Builder) compileSectorBySegment(miSectors []*model.InputSector) {
	var miSegments []*model.InputSegment
	var miHulls = make([][]*model.InputSegment, len(miSectors))

	for idx := range miSectors {
		miHulls[idx] = b.createGeometryHull(idx, miSectors)
		miSegments = append(miSegments, miHulls[idx]...)
	}

	getInternalSegment := func(curr *model.InputSegment, internal []*model.InputSegment) []*model.InputSegment{
		var ss []*model.InputSegment
		for _, seg := range internal {
			if seg.EqualCoords(curr) {
				res := seg.Clone()
				return []*model.InputSegment{ res }
			}
			if a, b, ok := b.segmentOnSegment(seg, curr); ok {
				res := seg.Clone()
				//TODO ORDER
				res.Start = a
				res.End = b
				res.Tag += fmt.Sprintf(" internal real: %d %d %d %d", int(seg.Start.X), int(seg.Start.Y), int(seg.End.X), int(seg.End.Y))
				ss = append(ss, res)
			}
		}
		if len(ss) == 0 {
			res := curr.Clone()
			ss = append(ss, res)
		}
		return ss
	}

	getSubSegments := func(curr *model.InputSegment) []*model.InputSegment {
		var ss []*model.InputSegment

		for _, seg := range miSegments {
			if seg.Parent == curr.Parent { continue }
			if seg.EqualCoords(curr) {
				res := seg.Clone()
				res.Parent = curr.Parent
				res.Neighbor = seg.Parent
				res.Tag += fmt.Sprintf("external real: %d %d %d %d", int(seg.Start.X), int(seg.Start.Y), int(seg.End.X), int(seg.End.Y))
				return []*model.InputSegment{res}
			}
			if a, b, ok := b.segmentOnSegment(curr, seg); ok {
				res := seg.Clone()
				//TODO ORDER
				res.Start = a
				res.End = b
				res.Parent = curr.Parent
				res.Neighbor = seg.Parent
				res.Tag += fmt.Sprintf("external real: %d %d %d %d", int(seg.Start.X), int(seg.Start.Y), int(seg.End.X), int(seg.End.Y))
				ss = append(ss, res)
			}
		}
		if len(ss) == 0 {
			res := curr.Clone()
			res.Neighbor = "nil"
			ss = append(ss, res)
		}
		return ss
	}

	testSectorIdx := 122
	testSector := miSectors[testSectorIdx]
	convex := miHulls[testSectorIdx]

	for _, curr := range convex {
		var out[]*model.InputSegment
		for _, seg := range getInternalSegment(curr, testSector.Segments) {
			if seg.Kind == model.DefinitionWall {
				out = append(out, seg)
				continue
			}
			out = append(out, getSubSegments(seg)...)
		}
		fmt.Println("------")
		fmt.Println("base:", curr.Neighbor, curr.Start.X, curr.Start.Y, curr.End.X, curr.End.Y)
		for _, r := range out {
			//TODO SORT
			fmt.Println("\tsub:", r.Neighbor, r.Start.X, r.Start.Y, r.End.X, r.End.Y, r.Tag)
		}
	}

	os.Exit(-1)
}


func (b * Builder) createMeshes(miSectors []*model.InputSector) {
	m := &geometry.Model{}
	//for idx, seg := range miSectors[2].Segments {
	//	m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, idx)
	//}
	//out  := m.Dxf()
	//_ = os.WriteFile("test1.dxf", []byte(out), 0644)

	//for idx, seg := range miSectors[116].Segments {
	//	m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, idx)
	//}
	//out  := m.Dxf()
	//_ = os.WriteFile("test2.dxf", []byte(out), 0644)

	converter := map[int]*model.InputSegment{}
	counter := 0
	for _, sector := range miSectors {
		for _, seg := range sector.Segments {
			m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, counter)
			converter[counter] = seg
			counter++
		}
	}

	mesh := geometry.NewMesh()

	err := mesh.Compile(m)
	if err == nil {
		err := mesh.Materials()
		if err != nil {
			fmt.Println(err)
		}

		n := mesh.Model()

		for _, t := range n.Triangles {
			seg := converter[t[3]]

			fmt.Println("------", seg.Parent, seg.Tag)
			for i := 0; i < len(t) -1; i++ {
				p := n.Points[t[i]]
				fmt.Print(p.X, p.Y, " - ")
			}
			fmt.Println()
			//fmt.Println(t)
		}
	}
	os.Exit(-1)
	//m.Intersection()
	//m.ConvexHullTriangles()
}






func (b * Builder) createIdealHull(testSector *model.InputSector) []*model.InputSegment {
	var qVector []quickhull.Vector
	for _, seg := range testSector.Segments {
		qVector = append(qVector, quickhull.Vector{ X: seg.Start.X, Y: seg.Start.Y, Z: 1.0, Data: nil })
		qVector = append(qVector, quickhull.Vector{ X: seg.End.X, Y: seg.End.Y, Z: 1.0, Data: nil })
	}
	hull := new(quickhull.QuickHull).ConvexHull(qVector, true, false, 0)
	var ideals []*model.InputSegment
	for idx := 0; idx < len(hull.Vertices) - 1; idx++{
		v := hull.Vertices[idx]
		n := hull.Vertices[idx+1]
		ideals = append(ideals, model.NewInputSegment("null",-1, model.XY{X:v.X, Y:v.Y}, model.XY{X:n.X, Y:n.Y}))
	}

	ch := model.NewConvexHull()

	var chs []*model.CHSegment
	for _, s := range ideals {
		c := model.NewCHSegment(testSector.Id, s, s.Start, s.End)
		chs = append(chs, c)
	}

	res := ch.Create(testSector.Id, chs)
	var out []*model.InputSegment
	for _, r := range res {
		if seg, ok := r.Data.(*model.InputSegment); ok {
			out = append(out, seg)
		} else {
			out = append(out, model.NewInputSegment("null",-1, r.Start, r.End))
		}
	}
	//for _, seg := range testSector.Segments {
	//	out = append(out, seg)
	//}
	return out
}


func (b * Builder) segmentOnSegment(refSeg *model.InputSegment, currSeg * model.InputSegment) (model.XY, model.XY, bool) {
	refStart := geometry.Point{X: refSeg.Start.X, Y: refSeg.Start.Y}
	refEnd := geometry.Point{X: refSeg.End.X, Y: refSeg.End.Y}
	currStart := geometry.Point{X: currSeg.Start.X, Y: currSeg.Start.Y}
	currEnd := geometry.Point{X: currSeg.End.X, Y: currSeg.End.Y}

	var p1[]model.XY

	if _, sa0, _ := geometry.PointLine(refStart, currStart, currEnd);
		sa0.Has(geometry.OnPoint0Segment) || sa0.Has(geometry.OnPoint1Segment) {
		p1 = append(p1, model.XY{X: refStart.X, Y: refStart.Y})
	}
	if _, sb0, _ := geometry.PointLine(refEnd, currStart, currEnd);
		sb0.Has(geometry.OnPoint0Segment) || sb0.Has(geometry.OnPoint1Segment) {
		p1 = append(p1, model.XY{X: refEnd.X, Y: refEnd.Y})
	}
	if len(p1) == 2 {
		return p1[0], p1[1], true
	}
	var p2[]model.XY
	if _, sc0, _ := geometry.PointLine(currStart, refStart, refEnd);
		sc0.Has(geometry.OnPoint0Segment) || sc0.Has(geometry.OnPoint1Segment) {
		p2 = append(p2, model.XY{X: currStart.X, Y: currStart.Y})
	}
	if _, sd0, _ := geometry.PointLine(currEnd, refStart, refEnd);
		sd0.Has(geometry.OnPoint0Segment) || sd0.Has(geometry.OnPoint1Segment) {
		p2 = append(p2, model.XY{X: currEnd.X, Y: currEnd.Y})
	}
	if len(p2) >= 2 {
		return p2[0], p2[1], true
	}

	if len(p1) > 0 && len(p2) > 0 {
		//return p1[0], p2[0], true
		if p1[0] != p2[0] {
			return p1[0], p2[0], true
		}
	}

	return model.XY{}, model.XY{}, false

	/*
		fmt.Println("sa0", sa0.String())
		fmt.Println("sa1", sa1.String())
		fmt.Println("sb0", sb0.String())
		fmt.Println("sb1", sb1.String())
		fmt.Println("sc0", sc0.String())
		fmt.Println("sc1", sc1.String())
		fmt.Println("sd0", sd0.String())
		fmt.Println("sd1", sd1.String())

		fmt.Println("-------")

	*/

}


func(b * Builder) intersectRect(x float64, y float64, top float64, left float64, bottom float64, right float64) bool {
	return x >= left && x <= right && y >= bottom && y <= top
}

func(b * Builder) compileRect(miSectors []*model.InputSector) {

	b.createReferenceHull3(2, miSectors)

	/*
		b.describeSegments(2, miSectors)
		b.describeSegments(116, miSectors)

		m := &geometry.Model{}


		//for _, seg := range miSectors[2].Segments {
		//	m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, 0)
		//}
		//out  := m.Dxf()
		//_ = os.WriteFile("test1.dxf", []byte(out), 0644)

		for _, seg := range miSectors[116].Segments {
			m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, 0)
		}
		out  := m.Dxf()
		_ = os.WriteFile("test2.dxf", []byte(out), 0644)

		//for _, sector := range miSectors {
		//	for _, seg := range sector.Segments {
		//		m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, 0)
		//	}
		//}
		//m.Intersection()
		//m.ConvexHullTriangles()
		os.Exit(-1)
	*/





	ideals := make([][]*model.InputSegment, len(miSectors))
	for idx, _ := range miSectors {
		ideals[idx] = b.createReferenceHull2(idx, miSectors)
	}

	targetIdx := 116
	targetSector := miSectors[targetIdx]

	for _, currSeg := range ideals[targetIdx] {
		fmt.Println("---------------------", currSeg.Start, currSeg.End)
		if currSeg.Kind == model.DefinitionWall {
			fmt.Println("wall", currSeg.Start, currSeg.End)
		} else {
			found := false
			for idealIdx, idealHull := range ideals {
				idealSector := miSectors[idealIdx]
				if idealSector.Id == targetSector.Id { continue }

				for _, idealSeg := range idealHull {
					if cs, ce, ok := b.segmentOnSegment(currSeg, idealSeg); ok {
						fmt.Println(idealSector.Id, cs, ce)
						found = true
					}
				}
			}

			if !found {
				fmt.Println("undefined", currSeg.Start, currSeg.End)
			}
		}
	}

	os.Exit(-1)
}


func (b * Builder) createReferenceHull3(targetIdx int, miSectors []*model.InputSector) []*model.InputSegment {
	targetSector := miSectors[targetIdx]

	testSeg := targetSector.Segments[0]

	//TODO VERIFICARE - A VOLTE IL RECT RESTITUITO E' SBAGLIATO......
	r, _ := b.bsp.FindRect(int16(testSeg.End.X), int16(-testSeg.End.Y))

	//I SEGMENTI DA VERIFICARE SONO 2 - 116
	fmt.Println(r)
	top := float64(-r.Top); left := float64(r.Left); bottom := float64(-r.Bottom); right := float64(r.Right)

	if left > right {
		t := right
		right = left
		left = t
	}
	if bottom > top {
		t := bottom
		bottom = top
		top = t
	}
	var hull2 []geometry.Point


	for _, seg := range targetSector.Segments {
		if b.intersectRect(seg.Start.X, seg.Start.Y, top, left, bottom, right) {
			fmt.Println("Start found")
		} else {
			fmt.Println("Start not found")
		}
		if b.intersectRect(seg.End.X, seg.End.Y, top, left, bottom, right) {
			fmt.Println("End found")
		} else {
			fmt.Println("End not found")
		}
	}

	//os.Exit(-1)



	points := map[geometry.Point]bool{}
	neighbors := map[string]bool{}
	for _, sec := range miSectors {
		for _, seg := range sec.Segments {
			if b.intersectRect(seg.Start.X, seg.Start.Y, top, left, bottom, right) {
				neighbors[sec.Id] = true
				hull2 = append(hull2, geometry.Point{X: seg.Start.X, Y: seg.Start.Y})
				points[geometry.Point{X: seg.Start.X, Y: seg.Start.Y}] = true
				//hull = append(hull, geometry.Point{X: seg.End.X, Y: seg.End.Y})
			}
			if b.intersectRect(seg.End.X, seg.End.Y, top, left, bottom, right) {
				neighbors[sec.Id] = true
				//hull = append(hull, geometry.Point{X: seg.Start.X, Y: seg.Start.Y})
				hull2 = append(hull2, geometry.Point{X: seg.End.X, Y: seg.End.Y})
				points[geometry.Point{X: seg.End.X, Y: seg.End.Y}] = true
			}
		}
	}
	fmt.Println("rect", r)
	fmt.Println(neighbors)
	for v := range points {
		fmt.Println(v.X, v.Y)
	}

	fmt.Println("------------")

	for _, v := range geometry.ConvexHull(hull2) {
		fmt.Println(int(v.X), int(v.Y))
	}

	os.Exit(-1)

	return nil
}



func (b * Builder) createReferenceHull2(targetIdx int, miSectors []*model.InputSector) []*model.InputSegment {
	ch := model.NewConvexHull()
	miSector := miSectors[targetIdx]

	var chs []*model.CHSegment
	for _, s := range miSector.Segments {
		c := model.NewCHSegment(miSector.Id, s, s.Start, s.End)
		chs = append(chs, c)
	}
	var out []*model.InputSegment
	for _, s := range ch.Create(miSector.Id, chs) {
		if s.Data != nil {
			out = append(out, s.Data.(*model.InputSegment))
		} else {
			ns := model.NewInputSegment(miSector.Id, model.DefinitionVoid, s.Start, s.End)
			ns.Tag = "missing"
			out = append(out, ns)
		}
	}
	return out
}



func(b * Builder) createGeometryHull(targetIdx int, miSectors []*model.InputSector) []*model.InputSegment {
	targetSector := miSectors[targetIdx]

	var hull []geometry.Point
	for _, seg := range targetSector.Segments {
		hull = append(hull, geometry.Point{X: seg.Start.X, Y: seg.Start.Y})
		hull = append(hull, geometry.Point{X: seg.End.X, Y: seg.End.Y})
	}
	convexHull := geometry.ConvexHull(hull)
	var reference []*model.InputSegment
	for idx := 0; idx < len(convexHull) - 1; idx++{
		v := convexHull[idx]
		n := convexHull[idx+1]
		reference = append(reference, model.NewInputSegment(targetSector.Id, model.DefinitionVoid, model.XY{X:v.X, Y:v.Y}, model.XY{X:n.X, Y:n.Y}))
	}
	if len(reference) > 0 {
		start := reference[0]
		end := reference[len(reference) - 1]
		reference = append(reference, model.NewInputSegment(targetSector.Id, model.DefinitionVoid, end.End, start.Start))
	}

	if len(reference) == 0 {
		for _, seg := range miSectors[targetIdx].Segments {
			reference = append(reference, model.NewInputSegment(targetSector.Id, model.DefinitionVoid, seg.Start, seg.End))
		}
	}
	return reference
}



func (b * Builder) segmentOnSegmentOld(refSeg *model.InputSegment, currSeg * model.InputSegment) bool {
	t1 := b.pointOnSegment(refSeg.Start, currSeg.Start, currSeg.End)
	t2 := b.pointOnSegment(refSeg.End, currSeg.Start, currSeg.End)

	t3 := b.pointOnSegment(currSeg.Start, refSeg.Start, refSeg.End)
	t4 := b.pointOnSegment(currSeg.End, refSeg.Start, refSeg.End)
	return (t1 && t2) || (t3 && t4)
}

func (b * Builder) compileSegmentRelations2(miSectors []*model.InputSector) {

	b.compileRect(miSectors)

	type segData struct { xy model.XY; kind int; wall bool; neighbor string }

	ideals := make([][]*model.InputSegment, len(miSectors))
	for idx, testSector := range miSectors {
		ideals[idx] = b.createIdealHull(testSector)
	}


	/*

		testSector := 15
		fmt.Println("-------- IDEAL --------")
		for _, t := range ideals[testSector] {
			fmt.Println(t.Start, t.End)
		}
		fmt.Println("-------- REAL --------")
		for _, t := range miSectors[testSector].Segments {
			fmt.Println(t.Start, t.End)
		}
		os.Exit(-1)
	*/

	//testSectors := miSectors[2:3]
	//refIdx := 2

	for refIdx , refSector := range miSectors {
		fmt.Println()
		fmt.Println("--------------------", refSector.Id, "len", len(refSector.Segments))

		idealSector := ideals[refIdx]
		for currSegIdx, currSeg := range idealSector {
			fmt.Println("\t------------------ Seg - parent:", refSector.Id, "index", currSegIdx, currSeg.Start, currSeg.End)
			var segment []*segData

			neighborsCheck := 0
			found := false
			for _, eq :=  range refSector.Segments {
				if ok := b.segmentOnSegmentOld(eq, currSeg); !ok {
					continue
				}
				found = true
				if eq.Kind == model.DefinitionWall {
					segment = append(segment, &segData{xy: eq.End, kind: 0, wall: true, neighbor: "wall" })
					segment = append(segment, &segData{xy: eq.Start, kind: 1, wall: true, neighbor: "wall" })
				} else {
					neighborsCheck ++
				}
			}
			if !found {
				neighborsCheck++
			}

			if neighborsCheck > 0 {
				for idealIdx, idealHull := range ideals {
					idealSector := miSectors[idealIdx]

					if idealSector.Id == refSector.Id { continue }

					neighbor := idealSector.Id

					for _, idealSeg := range idealHull {
						if cs, ce, ok := b.segmentOnSegment(currSeg, idealSeg); ok {
							found = true
							segment = append(segment, &segData{xy: cs, kind: 0, wall: false, neighbor: neighbor })
							segment = append(segment, &segData{xy: ce, kind: 1, wall: false, neighbor: neighbor })
						}
					}
				}
			}

			if !found {
				fmt.Println("NOT FOUND", currSeg.Start, currSeg.End, currSeg.Tag)
			}

			for _, n := range segment {
				var kind string; if n.kind == 0 { kind = "start" } else if  n.kind == 1 { kind = "stop"}
				neighbor := n.neighbor
				//fmt.Println(kind, neighbor, n.xy, refSeg.Upper, refSeg.Middle, refSeg.Lower)
				fmt.Println(kind, neighbor, n.xy)
			}
		}
		//if refSector.Id == "0" {
		//	os.Exit(-1)
		//}
	}

	os.Exit(-1)
}