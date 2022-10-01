package wad

import (
	"fmt"
	"github.com/markel1974/godoom/engine/geometry"
	"github.com/markel1974/godoom/engine/model"
	"strconv"
)


func emptySegments(r * model.InputSegment) []*model.InputSegment{
	u := r.Clone()
	u.Neighbor = ""
	u.Kind = model.DefinitionUnknown
	return []*model.InputSegment {u}
}

func swapXY(a model.XY, b model.XY) (model.XY, model.XY) {
	return b, a
}

func segmentIntersect(ref *model.InputSegment, miSectors []*model.InputSector) bool {
	p0 := geometry.Point{X: ref.Start.X, Y: ref.Start.Y }
	p1 := geometry.Point{X: ref.End.X, Y: ref.End.Y }
	for _, sec := range miSectors {
		for _, seg := range sec.Segments {
			p2 := geometry.Point{X: seg.Start.X, Y: seg.Start.Y }
			p3 := geometry.Point{X: seg.End.X, Y: seg.End.Y }
			res, _, _ := geometry.LineLine(p0, p1, p2, p3)
			if len(res) > 0 {
				return true
			}
		}
	}
	return false
}

func (b * Builder) rebuildSegmentStraight(ref *model.InputSegment, in []*model.InputSegment) []*model.InputSegment {
	var out []*model.InputSegment
	start := ref.Start
	for len(in) > 0 {
		found := false
		for x := 0; x < len(in); x++ {
			if in[x].Start == start || in[x].End == start {
				if in[x].End == start { in[x].Start, in[x].End = swapXY(in[x].Start, in[x].End) }
				out = append(out, in[x])
				start = in[x].End
				in = append(in[:x], in[x+1:]...)
				found = true
				break
			}
		}
		if !found { break }
	}
	if len(out) == 0 { return emptySegments(ref) }
	last := out[len(out) - 1]
	if last.End != ref.End {
		u := ref.Clone()
		u.Start = last.End
		u.End = ref.End
		u.Kind = model.DefinitionUnknown
		out = append(out, u)
	}
	return out
}

func (b * Builder) rebuildSegmentReverse(ref *model.InputSegment, in []*model.InputSegment) []*model.InputSegment {
	var out []*model.InputSegment
	end := ref.End
	for len(in) > 0 {
		found := false
		for x := 0; x < len(in); x++ {
			if end == in[x].End || end == in[x].Start {
				if end == in[x].Start { in[x].Start, in[x].End = swapXY(in[x].Start, in[x].End) }
				out = append([]*model.InputSegment{in[x]}, out...)
				end = in[x].Start
				in = append(in[:x], in[x+1:]...)
				found = true
				break
			}
		}
		if !found { break }
	}
	if len(out) == 0 { return emptySegments(ref) }
	first := out[0]
	if ref.Start != first.Start {
		u := ref.Clone()
		u.Start = ref.Start
		u.End = first.Start
		u.Kind = model.DefinitionUnknown
		out = append([]*model.InputSegment{u}, out...)
	}

	return out
}

func (b * Builder) rebuildSegment(ref *model.InputSegment, in []*model.InputSegment) []*model.InputSegment {
	reverse := -1
	for x := 0; x < len(in); x++ {
		if ref.Start == in[x].Start || ref.Start == in[x].End { reverse = 0; break }
		if ref.End == in[x].End || ref.End == in[x].Start { reverse = 1; break }
	}
	switch reverse {
		case 0: return b.rebuildSegmentStraight(ref, in)
		case 1: return b.rebuildSegmentReverse(ref, in)
	}
	return emptySegments(ref)
}

func (b * Builder) compileRemoteSegment(ref * model.InputSegment, sectors []*model.InputSector, walls []*model.InputSegment, textures []*model.InputSegment) []* model.InputSegment {
	var out[]*model.InputSegment

	cacheHas := func(cs * model.InputSegment) bool {
		for _, s := range out {
			if start, end, ok := b.segmentOnSegment(s, cs); ok {
				if start == cs.Start && end == cs.End { return true }
				if end == cs.Start && start == cs.End { return true }
			}
		}
		return false
	}

	if ref.Kind == model.DefinitionWall { return nil }

	//TODO REMOVE.....

	for _, wall := range walls {
		if start, end, ok := b.segmentOnSegment(ref, wall); ok {
			cs := ref.Clone()
			cs.Kind = model.DefinitionWall
			cs.Neighbor = "wall"//wall.Parent
			cs.Start = start
			cs.End = end
			cs.Upper = wall.Upper
			cs.Lower = wall.Lower
			cs.Middle = wall.Middle

			if !cacheHas(cs) {
				out = append(out, cs)
			}
		}
	}


	for _, hull := range sectors {
		for _, tst := range hull.Segments {
			if ref.Parent == tst.Parent { continue }
			if start, end, ok := b.segmentOnSegment(ref, tst); ok {
				cs := ref.Clone()
				cs.Kind = model.DefinitionValid
				cs.Neighbor = tst.Parent
				cs.Start = start
				cs.End = end

				for _, texture := range textures {
					if _, _ , ok := b.segmentOnSegment(ref, texture); ok {
						cs.Upper = texture.Upper
						cs.Lower = texture.Lower
						cs.Middle = texture.Middle
						break
					}
				}
				if !cacheHas(cs) {
					out = append(out, cs)
				}
			}
		}
	}

	result := b.rebuildSegment(ref, out)
	return result
}

func (b * Builder) compileRemoteSector(miSector *model.InputSector, sectors []*model.InputSector) *model.InputSector {
	//var newSectors []*model.InputSector

	for _, refSeg := range miSector.Segments {
		if refSeg.Kind != model.DefinitionUnknown { continue }

		for _, sec := range sectors {
			if miSector.Id == sec.Id { continue }

			for _, seg := range sec.Segments {
				if seg.Kind == model.DefinitionUnknown {
					if ok := refSeg.AnyCoords(seg); ok {
						testId := strconv.Itoa(len(sectors))

						hull := b.createGeometryHull(testId, []*model.InputSegment{refSeg, seg})
						for _, z := range hull {
							if z.SameCoords(refSeg) {
								//z.Neighbor = refSeg.Parent
								//z.Kind = model.DefinitionValid
							} else if z.SameCoords(seg) {
								//z.Neighbor = seg.Parent
								//z.Kind = model.DefinitionValid
							} else {
								if !segmentIntersect(z, sectors) {
									kkk, _ := strconv.Atoi(seg.Parent)
									testSector := sectors[kkk]

									newSector := model.NewInputSector(testId)
									newSector.Segments = hull
									newSector.Ceil = testSector.Ceil
									newSector.Floor = testSector.Floor
									newSector.LowerTexture = testSector.LowerTexture
									newSector.UpperTexture = testSector.UpperTexture
									newSector.FloorTexture = testSector.FloorTexture
									newSector.CeilTexture = testSector.CeilTexture
									newSector.WallTexture = testSector.WallTexture
									//newSector.Textures = testSector.Textures
									//sectors = append(sectors, newSector)

									return newSector
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}


func (b * Builder) testsEntryPoint3(miSectors []*model.InputSector) []*model.InputSector {
	var walls []*model.InputSegment
	var textures []*model.InputSegment
	var hulls = make([]*model.InputSector, len(miSectors))

	for idx, sec := range miSectors {
		for _, seg := range sec.Segments {
			if seg.Kind == model.DefinitionWall {
				walls = append(walls, seg)
			} else {
				textures = append(textures, seg)
			}
		}

		cloned := sec.Clone(false)
		hulls[idx] = cloned
		if len(sec.Segments) > 1 {
			cloned.Segments = b.createReferenceHull2(sec)
			//cloned.Segments = b.createGeometryHull(sec.Id, sec.Segments)
		} else {
			sec.Segments = nil
			cloned.Segments = nil
		}
	}


	for idx, hull := range hulls {
		var segments []*model.InputSegment
		for _, seg := range hull.Segments {
			if cs := b.compileRemoteSegment(seg, hulls, walls, textures); cs != nil {
				segments = append(segments, cs...)
			} else {
				segments = append(segments, seg)
			}
		}
		miSectors[idx].Segments = segments
	}


	l := len(miSectors)


	for idx := 0; idx < l; idx++ {
		fmt.Println("CURRENT", idx)
		if newSector := b.compileRemoteSector(miSectors[idx], miSectors); newSector != nil {
			miSectors = append(miSectors, newSector)

			for _, sec := range miSectors {
				var segments []*model.InputSegment
				for _, seg := range sec.Segments {
					if cs := b.compileRemoteSegment(seg, miSectors, walls, textures); cs != nil {
						segments = append(segments, cs...)
					} else {
						segments = append(segments, seg)
					}
				}
				sec.Segments = segments
			}
			idx = 0
			l = len(miSectors)
		}
	}



	notFound := 0
	for idx, sec := range miSectors {
		fmt.Println("-----------------", idx)
		for _, seg := range sec.Segments {
			fmt.Println("SEGMENT", seg.Start, seg.End)
			neighbor := seg.Neighbor
			if seg.Kind == model.DefinitionWall {
				neighbor = "wall"
			} else if seg.Kind == model.DefinitionUnknown {
				neighbor = "unknown"
				notFound++
			}
			fmt.Println("\t", neighbor, seg.Start, seg.End, seg.Upper, seg.Middle, seg.Lower)
		}
	}

	fmt.Println("NOT FOUND", notFound)

	//os.Exit(-1)
	return miSectors
}



/*
func (b * Builder) compileLocalSegment(ref * model.InputSegment, local []*model.InputSegment) []*model.InputSegment {
	var out []*model.InputSegment
	for _, seg := range local {
		if start, end, ok := b.segmentOnSegment(ref, seg); ok {
			v := seg.Clone()
			v.Start = start
			v.End = end
			out = append(out, v)
		}
	}
	if out == nil {
		return []*model.InputSegment{ ref }
	}
	result := b.rebuildSegment(ref, out)
	return result
}
*/



/*
func (b * Builder) createMeshes(miSectors []*model.InputSector) {
	m := geometry.NewModel()
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
	cache := make(map[string]*model.InputSegment)
	counter := 0

	for idx, _ := range miSectors {
		hull := b.createGeometryHull(idx, miSectors)

		for _, seg := range hull {
			m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, counter)
			converter[counter] = seg
			hash1 := fmt.Sprintf("%f%f%f%f", seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y)
			hash2 := fmt.Sprintf("%f%f%f%f", seg.End.X, seg.End.Y, seg.Start.X, seg.Start.Y)
			cache[hash1] = seg
			cache[hash2] = seg
			counter++
		}
	}

	//for _, sector := range miSectors {
	//	for _, seg := range sector.Segments {
	//		m.AddLine(geometry.Point{X: seg.Start.X, Y: seg.Start.Y}, geometry.Point{X: seg.End.X, Y: seg.End.Y}, counter)
	//		converter[counter] = seg
	//		hash1 := fmt.Sprintf("%f%f%f%f", seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y)
	//		hash2 := fmt.Sprintf("%f%f%f%f", seg.End.X, seg.End.Y, seg.Start.X, seg.Start.Y)
	//		cache[hash1] = seg
	//		cache[hash2] = seg
	//		counter++
	//	}
	//}

	mesh := geometry.NewMesh()

	err := mesh.Compile(m)
	if err == nil {
		//err := mesh.Materials()
		//if err != nil {
		//	fmt.Println(err)
		//}

		n := mesh.Model()

		//count := 0

		for _, t := range n.Triangles {
			seg := converter[t[3]]

			p1 := n.Points[t[0]]
			p2 := n.Points[t[1]]
			p3 := n.Points[t[1]]

			hash1 := fmt.Sprintf("%f%f%f%f", p1.X, p1.Y, p2.X, p2.Y)
			hash2 := fmt.Sprintf("%f%f%f%f", p2.X, p2.Y, p3.X, p3.Y)
			hash3 := fmt.Sprintf("%f%f%f%f", p3.X, p3.Y, p1.X, p1.Y)

			if _, ok := cache[hash1]; ok {
				//fmt.Println("HASH1 - OK")
			} else {
				fmt.Println("HASH1 - NOT OK")
			}

			if _, ok := cache[hash2]; ok {
				//fmt.Println("HASH2 - OK")
			} else {
				fmt.Println("HASH2 - NOT OK")
			}

			if _, ok := cache[hash3]; ok {
				//fmt.Println("HASH3 - OK")
			} else {
				fmt.Println("HASH3 - NOT OK")
			}


			fmt.Println("------", seg.Parent, seg.Tag, seg.Start, seg.End)
			for i := 0; i < len(t) -1; i++ {
				p := n.Points[t[i]]
				fmt.Print(p.X, p.Y, " - ")
			}
			fmt.Println()

			//fmt.Println(t)
		}
	} else {
		fmt.Println(err.Error())
	}
	os.Exit(-1)
	//m.Intersection()
	//m.ConvexHullTriangles()
}
*/
