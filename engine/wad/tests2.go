package wad

import (
	"fmt"
	"github.com/markel1974/godoom/engine/geometry"
	"github.com/markel1974/godoom/engine/model"
)



func (b * Builder) createNeighbor(ref * model.InputSegment, seg * model.InputSegment) * model.InputSegment {
	neighborId := seg.Parent; if seg.Kind == model.DefinitionWall { neighborId = "wall" }
	kind := seg.Kind; if kind != model.DefinitionWall { kind = model.DefinitionValid }
	neighbor := ref.Clone()
	neighbor.Id = seg.Id
	neighbor.Kind = kind
	neighbor.Neighbor = neighborId
	return neighbor
}

func (b * Builder) createNeighbors(ref * model.InputSegment, segments []*model.InputSegment) []* model.InputSegment{
	var ret [] *model.InputSegment

	line := b.bsp.describeLineF(ref.Start.X, ref.Start.Y, ref.End.X, ref.End.Y)

	for x := 0 ; x < len(line) - 1; x++ {
		point := line[x]
		p0 := geometry.Point{ X: point.X, Y: point.Y }

		var available []*model.InputSegment
		for _, seg := range segments {
			if ref.Parent== seg.Parent { continue }
			l0 := geometry.Point{ X: seg.End.X, Y: seg.End.Y }
			l1 := geometry.Point{ X: seg.Start.X, Y: seg.Start.Y }
			_, sa0, _ := geometry.PointLine(p0, l0, l1)
			if sa0.Has(geometry.OnPoint0Segment) && sa0.Has(geometry.OnPoint1Segment) {
				available = append(available, seg)
			}
		}

		for _, seg := range available {
			if len(ret) == 0 {
				//fmt.Println("- found", seg.Parent, point.X, point.Y)
				z := b.createNeighbor(ref, seg)
				z.Start = model.XY{ X: point.X, Y: point.Y } //ref.Start
				ret = append(ret, z)
			} else {
				last := ret[len(ret) - 1]
				if last.Id != seg.Id {
					//fmt.Println("- found", seg.Parent, point.X, point.Y)
					last.End = model.XY{ X: point.X, Y: point.Y }
					z := b.createNeighbor(ref, seg)
					z.Start = model.XY{ X: point.X, Y: point.Y }

					ret = append(ret, z)
				}
			}
		}
	}
	return ret
}

func (b * Builder) createNeighborsHP(ref * model.InputSegment, compatibles []*model.InputSegment) []* model.InputSegment{
	var ret [] *model.InputSegment

	line := b.bsp.describeLineF(ref.Start.X, ref.Start.Y, ref.End.X, ref.End.Y)

	for x := 0 ; x < len(line) - 1; x++ {
		point := line[x]
		next := line[x + 1]
		p0 := geometry.Point{ X: point.X, Y: point.Y }
		p1 := geometry.Point{ X: next.X, Y: next.Y }

		for _, seg := range compatibles  {
			if ref.Parent== seg.Parent { continue }

			l0 := geometry.Point{ X: seg.End.X, Y: seg.End.Y }
			l1 := geometry.Point{ X: seg.Start.X, Y: seg.Start.Y }

			_, sa0, _ := geometry.PointLine(p0, l0, l1)
			_, sa1, _ := geometry.PointLine(p1, l0, l1)

			if (sa0.Has(geometry.OnPoint0Segment) && sa0.Has(geometry.OnPoint1Segment)) && (sa1.Has(geometry.OnPoint0Segment) && sa1.Has(geometry.OnPoint1Segment)){
				if len(ret) == 0 {
					//fmt.Println("- found on idx", segIdx, seg.Parent, point.X, point.Y)
					z := b.createNeighbor(ref, seg)
					z.Start = ref.Start //model.XY{ X: point.X, Y: point.Y }

					ret = append(ret, z)
				} else {
					last := ret[len(ret) - 1]
					if last.Id != seg.Id {
						//fmt.Println("- found on idx", segIdx, seg.Parent, point.X, point.Y)
						last.End = model.XY{ X: point.X, Y: point.Y }
						z := b.createNeighbor(ref, seg)
						z.Start = model.XY{ X: point.X, Y: point.Y }

						ret = append(ret, z)
					}
				}
			}
		}
	}
	return ret
}


type QuerySegment struct {
	Partial int
	Segment * model.InputSegment
}

func (b * Builder) querySegment(ref * model.InputSegment, miSegments []*model.InputSegment, result map[string][]*QuerySegment) {
	var full []*QuerySegment
	var partial []*QuerySegment
	if ref.Kind == model.DefinitionWall {
		full = append(full, &QuerySegment{Partial: 0, Segment: ref})
	}
	for _, seg := range miSegments {
		if ref.Parent == seg.Parent { continue }
		if _, _, ok := b.segmentOnSegment(ref, seg); ok {
			full = append(full, &QuerySegment{Partial: 0, Segment: seg})
		}
		if b.pointOnSegmentNew(ref.Start, seg) {
			partial = append(partial, &QuerySegment{Partial: 1, Segment: seg})
		}
		if b.pointOnSegmentNew(ref.End, seg) {
			partial = append(partial, &QuerySegment{Partial: 2, Segment: seg})
		}
	}

	var current []*QuerySegment; if len(full) > 0 { current = full } else { current = partial }

	for _, qs := range current {
		n := qs.Segment.Parent; if qs.Segment.Kind == model.DefinitionWall { n = "wall" }
		var out []*QuerySegment
		if v, ok := result [n]; ok { out = v }
		result[n] = append(out, qs)
	}
}

func (b * Builder) testsEntryPoint2(miSectors []*model.InputSector) {
	var miSegments []*model.InputSegment
	var miHulls = make([][]*model.InputSegment, len(miSectors))

	for idx, _ := range miSectors {
		miHulls[idx] = b.createGeometryHull(miSectors[idx].Id, miSectors[idx].Segments)
		miSegments = append(miSegments, miHulls[idx]...)
	}


	/*
	//(2 - 116) 31
	var miNeighbors = make([]map[string][]*QuerySegment, len(miSectors))
	hull := b.createReferenceHull2(31, miSectors)
	result := map[string][]*QuerySegment{}
	for _, seg := range hull {
		b.querySegment(seg, miSegments, result)
	}

	for _, r := range result {
		for _, zz := range r {
			if zz.Partial > 0 {
				fmt.Println("--------")
				n := ""; if zz.Segment.Kind == model.DefinitionWall { n = "wall" }
				fmt.Println(zz.Segment.Parent, n)
			}
		}
	}

	os.Exit(-1)


	for idx := range miSectors {
		hull := b.createReferenceHull2(idx, miSectors)
		result := map[string][]*QuerySegment{}
		for _, seg := range hull {
			b.querySegment(seg, miSegments, result)
		}
		miNeighbors[idx] = result
	}

	os.Exit(-1)

	 */

	//TODO POLIGONI MANCANTI.....


	for _, miSector := range miSectors {
		reference := b.createReferenceHull2(miSector)
		var built []*model.InputSegment

		//if len(miSector.Segments) == 1 {
		//	fmt.Println("One Segment sector:", miSector.Id, "kind:", miSector.Segments[0].Kind)
		//}

		for _, refSegment := range reference {
			//TODO....... UN MURO DEVE ESSERE DIVISO SE HA PIU NEIGHBOR
			if refSegment.Kind == model.DefinitionWall {
				built = append(built, refSegment.Clone())
				continue
			}

			ffound := false
			for _, seg := range miSegments {
				if seg.Parent == refSegment.Parent { continue }
				if refSegment.Start == seg.End && refSegment.End == seg.Start {
					built = append(built, b.createNeighbor(refSegment, seg))
					ffound = true
					continue
				}
			}
			if ffound { continue }

			compatibles := miSegments

			ret := b.createNeighborsHP(refSegment, compatibles)
			if len(ret) == 0 {
				//fmt.Println("NOT FOUND", refSegment.Parent)
				ret = b.createNeighbors(refSegment, compatibles)
			}

			if len(ret) > 0 {
				last := ret[len(ret) - 1]
			    last.End = refSegment.End
				built = append(built, ret...)
			} else {
				fmt.Println("NOT FOUND ", refSegment.Parent)
			}
		}

		if miSector.Id == "31" {
			fmt.Println("---- BEFORE --------- ", miSector.Id)
			for _, z := range reference {
				fmt.Println(z.Kind, z.Parent, z.Start, z.End)
			}
			fmt.Println("------ HULL ---------")
			for _, z := range miHulls[31] {
				fmt.Println(z.Kind, z.Parent, z.Start, z.End)
			}
		}

		miSector.Segments = built

		if miSector.Id == "31" {
			fmt.Println("---- AFTER --------- ", miSector.Id)
			for _, z := range miSector.Segments {
				fmt.Println(z.Kind, z.Neighbor, z.Start, z.End)
			}
		}



		//os.Exit(-1)
	}

	//os.Exit(-1)
}