package model

import (
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/textures"
	"math"
	"strconv"
	"strings"
)

const (
	wallDefinition = -1
	unknownDefinition = -2
)

type lineDef struct { start XY; end XY; sectorId int; np int }


func lineDefHash(start XY, end XY) string {
	startX := strconv.FormatFloat(start.X, 'f', -1, 64)
	startY := strconv.FormatFloat(start.Y, 'f', -1, 64)
	endX := strconv.FormatFloat(end.X, 'f', -1, 64)
	endY := strconv.FormatFloat(end.Y, 'f', -1, 64)
	return startX + "|" + startY + "=>" +  endX + "|" + endY
}



type Compiler struct {
	sectors          []*Sector
	sectorsMaxHeight float64
	cache            map[string]int
}

func NewCompiler() * Compiler{
	return &Compiler{
		sectors:          nil,
		sectorsMaxHeight: 0,
		cache:            make(map[string]int),
	}
}

func (r *Compiler) Setup(cfg *Input, text * textures.Textures) error {
	for idx, cfgSector := range cfg.Sectors {
		var vertices []XY
		var neighborsIds []string
		var tags[] string
		for _, cfgNeighbor := range cfgSector.Neighbors {
			tags = append(tags, cfgNeighbor.Tag)
			vertices = append(vertices, XY{X: cfgNeighbor.X, Y: cfgNeighbor.Y})
			neighborsIds = append(neighborsIds, cfgNeighbor.Neighbor)
		}

		s := NewSector(cfgSector.Id, uint64(len(vertices)), vertices, neighborsIds)
		s.Tag = cfgSector.Tag + "[" + strings.Join(tags, ";") + "]"
		s.Ceil = cfgSector.Ceil
		s.Floor = cfgSector.Floor
		s.Textures = cfgSector.Textures
		if s.Textures {
			s.FloorTexture = text.Get(cfgSector.FloorTexture)
			s.CeilTexture = text.Get(cfgSector.CeilTexture)
			s.UpperTexture = text.Get(cfgSector.UpperTexture)
			s.LowerTexture = text.Get(cfgSector.LowerTexture)
			s.WallTexture = text.Get(cfgSector.WallTexture)
			if s.FloorTexture == nil || s.CeilTexture == nil && s.UpperTexture == nil || s.LowerTexture == nil || s.WallTexture == nil {
				fmt.Println("invalid textures configuration for sector", s.Id)
				s.Textures = false
			}
		}
		r.sectors = append(r.sectors, s)
		r.cache[cfgSector.Id] = idx
	}

	for _, sect := range r.sectors {

		for idx := 0; idx < len(sect.NeighborsIds); idx++ {
			id := sect.NeighborsIds[idx]
			//	for _, id := range sect.NeighborsIds {
			switch strings.Trim(strings.ToLower(id), " \t\n") {
			case "-1", "wall":
				sect.NeighborsRefs = append(sect.NeighborsRefs, wallDefinition)
			case "", "-2", "unknown":
				sect.NeighborsRefs = append(sect.NeighborsRefs, unknownDefinition)
			default:
				idx, ok := r.cache[id]
				if !ok {
					fmt.Printf("sector %s: can't find neighbor id %s\n", sect.Id, id)
					//return errors.New(fmt.Sprintf("sector %s: can't find neighbor id %s", sect.Id, id))
					//sect.NeighborsRefs = append(sect.NeighborsRefs, unknownDefinition)
					continue
				}
				sect.NeighborsRefs = append(sect.NeighborsRefs, idx)
			}
		}
	}

	//Verify Loop
	for idx, sect := range r.sectors {
		if len(sect.Vertices) == 0 {
			return errors.New(fmt.Sprintf("sector %s (idx: %d): vertices as zero len", sect.Id, idx))
		}
		hasLoop := false
		vFirst := sect.Vertices[0]
		if len(sect.Vertices) > 1 {
			vLast := sect.Vertices[len(sect.Vertices)-1]
			hasLoop = vFirst.X == vLast.X && vFirst.Y == vLast.Y
		}
		if !hasLoop {
			//TODO StubOld2 funziona solo se viene aggiunto in testa.....
			//vLast := sect.Vertices[len(sect.Vertices)-1]
			//sect.Vertices = append([]XY{vLast}, sect.Vertices...)
			//fmt.Printf("creating loop for sector %d\n", idx)
			sect.Vertices = append(sect.Vertices, vFirst)
		} else {
			fmt.Println("Adding an extra vertex")
			sect.Vertices = append(sect.Vertices, XY{})
			//vLast := sect.Vertices[len(sect.Vertices) - 1]
			//sect.Vertices = append(sect.Vertices, vLast)
		}
		sect.NPoints = uint64(len(sect.Vertices) - 1)
	}

Rescan:
	// Verify that for each edge that has a neighbor, the neighbor has this same neighbor.
	lineDefsCache := r.makeLineDefsCache()
	fixed := 0
	unmatch := 0
	for _, sector := range r.sectors {
		vert := sector.Vertices
		for np1 := uint64(0); np1 < sector.NPoints; np1++ {
			//if sector.NeighborsRefs[np1] >= wallDefinition { continue }
			np2 := np1 + 1
			v1start := vert[np1]
			v1end := vert[np2]
			if sector.NeighborsRefs[np1] < wallDefinition {
				if ld, ok := lineDefsCache[lineDefHash(v1end, v1start)]; ok {
					if ld.sectorId != sector.NeighborsRefs[np1] {
						fmt.Printf("p1 - sector %s (line: %d - %d): Neighbor behind line (%g, %g) - (%g, %g) should be %d, %d found instead. Fixing...\n", sector.Id, np1, np2, v1start.X, v1start.Y, v1end.X, v1end.Y, ld.sectorId, sector.NeighborsRefs[np1])
						sector.NeighborsRefs[np1] = ld.sectorId
						fixed++
					}
				} else {
					fmt.Printf("p1 - sector %s (line: %d - %d): Neighbor behind line (%g, %g) - (%g, %g) should be %d, %d found instead. Can't fix setting unknown...\n", sector.Id, np1, np2, v1start.X, v1start.Y, v1end.X, v1end.Y, ld.sectorId, sector.NeighborsRefs[np1])
					//sector.NeighborsRefs[np1] = unknownDefinition

					//sector.Vertices = append(sector.Vertices[:np1], sector.Vertices[np1+1:]...)
					//sector.NeighborsRefs = append(sector.NeighborsRefs[:np1], sector.NeighborsRefs[np1+1:]...)
					//sector.NPoints = uint64(len(sector.Vertices) - 1)
					unmatch++
				}
			}
		}
	}
	fmt.Println("unmatch:", unmatch, "fixed:", fixed)


	// Verify that the vertexes form a convex hull.
	for idx, sect := range r.sectors {
		vert := sect.Vertices
		for b := uint64(0); b < sect.NPoints; b++ {
			c := (b + 1) % sect.NPoints
			d := (b + 2) % sect.NPoints
			x0 := vert[b].X; y0 := vert[b].Y; x1 := vert[c].X; y1 := vert[c].Y
			switch mathematic.PointSideF(vert[d].X, vert[d].Y, x0, y0, x1, y1) {
			case 0:
				continue
				//Note: This used to be a problem for my engine, but is not anymore, so it is disabled.
				//if you enable this change, you will not need the IntersectBox calls in some locations anymore.
				//if sect.NeighborsRefs[b] == sect.NeighborsRefs[c] { continue }
				//fmt.Printf("sector %d: Edges %d-%d and %d-%d are parallel, but have different neighbors. This would pose problems for collision detection.\n", idx, b, c, c, d)
			case -1:
				fmt.Printf("Sector %d: Edges %d-%d and %d-%d create a concave turn. This would be rendered wrong.\n", idx, b, c, c, d)
			default:
				continue
			}

			fmt.Printf("- splitting sector, using (%g,%g) as anchor\n", vert[c].X, vert[c].Y)

			// Insert an edge between (c) and (e), where e is the nearest point to (c), under the following rules:
			// e cannot be c, c-1 or c+1
			// line (c)-(e) cannot intersect with any edge in this sector
			nearestDist := 1e29
			nearestPoint := ^uint64(0)
			for n := (d + 1) % sect.NPoints; n != b; n = (n + 1) % sect.NPoints {
				// Don't go through b, c, d
				x2 := vert[n].X
				y2 := vert[n].Y
				distX := x2 - x1
				distY := y2 - y1
				dist := distX*distX + distY*distY
				if dist >= nearestDist { continue }
				if mathematic.PointSideF(x2, y2, x0, y0, x1, y1) != 1 { continue }
				ok := true
				x1 += distX * 1e-4;	x2 -= distX * 1e-4;	y1 += distY * 1e-4; y2 -= distY * 1e-4
				for f := 0; f < int(sect.NPoints); f++ {
					if mathematic.IntersectLineSegmentsF(x1, y1, x2, y2, vert[f].X, vert[f].Y, vert[f+1].X, vert[f+1].Y) {
						ok = false
						break
					}
				}
				if !ok { continue }
				// Check whether this split would resolve the original problem
				if mathematic.PointSideF(x2, y2, vert[d].X, vert[d].Y, x1, y1) == 1 { dist += 1e6 }
				if dist >= nearestDist { continue }
				nearestDist = dist
				nearestPoint = n
			}

			if nearestPoint == ^uint64(0) {
				fmt.Printf("  ERROR: Could not find a vertex to pair with\n")
				continue
			}
			e := nearestPoint
			fmt.Printf(" - and point %d - (%g,%g) as the far point\n", e, vert[e].X, vert[e].Y)

			// Now that we have a chain: a b c d e f g h
			// And we're supposed to split it at "c" and "e", the outcome should be two chains:
			// c d e         (c)
			// e f g h a b c (e)

			var vert1 []XY
			var neigh1 []int
			// Create chain 1: from c to e.
			for n := uint64(0); n < sect.NPoints; n++ {
				m := (c + n) % sect.NPoints
				neigh1 = append(neigh1, sect.NeighborsRefs[m])
				vert1 = append(vert1, sect.Vertices[m])
				if m == e {
					vert1 = append(vert1, vert1[0])
					break
				}
			}

			neigh1Idx := len(r.sectors)
			neigh1 = append(neigh1, neigh1Idx)

			var vert2 []XY
			var neigh2 []int
			// Create chain 2: from e to c.
			for n := uint64(0); n < sect.NPoints; n++ {
				m := (e + n) % sect.NPoints
				neigh2 = append(neigh2, sect.NeighborsRefs[m])
				vert2 = append(vert2, sect.Vertices[m])
				if m == c {
					vert2 = append(vert2, vert2[0])
					break
				}
			}
			neigh2 = append(neigh2, idx)

			// using chain1
			sect.Vertices = vert1
			sect.NeighborsRefs = neigh1
			sect.NPoints = uint64(len(vert1) - 1)
			sect = r.sectors[idx]

			ns := NewSector("AutoGenerated_"+strconv.Itoa(neigh1Idx), uint64(len(vert2)-1), vert2, sect.NeighborsIds)
			ns.NeighborsRefs = neigh2
			ns.Floor = sect.Floor
			ns.Ceil = sect.Ceil
			ns.Textures = sect.Textures
			ns.FloorTexture = sect.FloorTexture
			ns.CeilTexture = sect.CeilTexture
			ns.UpperTexture = sect.UpperTexture
			ns.LowerTexture = sect.LowerTexture
			ns.WallTexture = sect.WallTexture

			r.sectors = append(r.sectors, ns)

			// We needs to rescan
			goto Rescan
		}
	}

	scale := cfg.ScaleFactor
	if scale < 1 { scale = 1 }

	r.sectorsMaxHeight = 0
	for _, sect := range r.sectors {
		//vertex scale
		for s := uint64(0); s <= sect.NPoints; s++ {
			sect.Vertices[s].X /= scale
			sect.Vertices[s].Y /= scale
		}
		//maxHeight
		if h := math.Abs(sect.Ceil - sect.Floor); h > r.sectorsMaxHeight {
			r.sectorsMaxHeight = h
		}
		//create neighbors from neighborIdx
		sect.Neighbors = make([]*Sector, sect.NPoints)
		for s := uint64(0); s < sect.NPoints; s++ {
			neighborIdx := sect.NeighborsRefs[s]
			if neighborIdx > wallDefinition {
				sect.Neighbors[s] = r.sectors[neighborIdx]
			}
		}
	}

	//out, _ := json.MarshalIndent(r.sectors, "", " ")
	//fmt.Println(string(out))

	fmt.Println("Scan complete")

	return nil
}

func (r * Compiler) GetSectors() []*Sector {
	return r.sectors
}

func (r * Compiler) Get(sectorId string) (*Sector, error) {
	idx, ok := r.cache[sectorId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("invalid sector: %s", sectorId))
	}
	return r.sectors[idx], nil
}

func (r * Compiler) GetMaxHeight() float64 {
	return r.sectorsMaxHeight
}


func (r * Compiler) makeLineDefsCache() map[string]lineDef {
	t := make(map[string] lineDef)
	for sectorId, sect := range r.sectors {
		for np := uint64(0); np < sect.NPoints; np++ {
			v1start := sect.Vertices[np]
			v1end := sect.Vertices[np + 1]
			hash := lineDefHash(v1start, v1end)
			ld := lineDef{sectorId: sectorId, np: int(np), start: v1start, end: v1end }
			if fld, ok := t[hash]; ok {
				if sectorId != fld.sectorId {
					//if test { even = false } else { odd = false }
					fmt.Println("line segment already added", sectorId, fld.sectorId, hash, np)
				}
			} else {
				t[hash] = ld
			}
		}
	}
	return t
}




/*
	for s1Idx, s1 := range r.sectors {
		vert := s1.Vertices
		for np1 := uint64(0); np1 < s1.NPoints; np1++ {
			v1start := vert[np1]
			v1end := vert[np1 + 1]
			found := 0

			for s2Idx, s2 := range r.sectors {
				for np2 := uint64(0); np2 < s2.NPoints; np2++ {
					v2start := s2.Vertices[np2]
					v2end := s2.Vertices[np2 + 1]

					if v2end.X == v1start.X && v2end.Y == v1start.Y && v2start.X == v1end.X && v2start.Y == v1end.Y {
						if s1Idx != s2.NeighborsRefs[np2] {
							fmt.Printf("[1] sector %s (idx: %d): Neighbor behind line (%g, %g) - (%g, %g) should be %d, %d found instead. Fixing...\n", s1.Id, np2, v1end.X, v1end.Y, v1start.Y, v1start.Y, s1Idx, s2.NeighborsRefs[np2])
							s2.NeighborsRefs[np2] = s1Idx
							goto Rescan
						}
						if s2Idx != s1.NeighborsRefs[np1] {
							fmt.Printf("[2] sector %s (idx: %d): Neighbor behind line (%g, %g) - (%g, %g) should be %d, %d found instead. Fixing...\n", s1.Id, np1, v1start.X, v1start.Y, v1end.X, v1end.Y, s2Idx, s1.NeighborsRefs[np1])
							s1.NeighborsRefs[np1] = s2Idx
							goto Rescan
						} else {
							found++
						}
					}
				}
			}
			if s1.NeighborsRefs[np1] >= 0 && s1.NeighborsRefs[np1] < len(r.sectors) && found != 1 {
				fmt.Printf("sector %s (idx: %d) and its neighbor %d don't share line (%g, %g)-(%g, %g)\n", s1.Id, s1Idx, s1.NeighborsRefs[np1], v1start.X, v1start.Y, v1end.X, v1end.Y)
			}
		}
	}
*/