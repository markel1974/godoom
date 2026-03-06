package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// DefinitionJoin represents a valid definition state with a value of 3.
// DefinitionVoid represents a void definition state with a value of 1.
// DefinitionWall represents a wall definition state with a value of 2.
// DefinitionUnknown represents an unknown definition state with a value of 0.
const (
	DefinitionJoin    = 3
	DefinitionVoid    = 1
	DefinitionWall    = 2
	DefinitionUnknown = 0
)

// lineDef2 represents a line segment in a 2D space with start and end Points, belonging to a specific sector.
type lineDef2 struct {
	start  XY
	end    XY
	sector *Sector
	np     int
}

type edgeKey struct {
	x1, y1, x2, y2 int64
}

// lineDefHash generates a unique string representation of a line defined by its start and end XY Points.
func makeEdgeKey(start XY, end XY) edgeKey {
	const scale = 1000.0
	return edgeKey{
		x1: int64(math.Round(start.X * scale)),
		y1: int64(math.Round(start.Y * scale)),
		x2: int64(math.Round(end.X * scale)),
		y2: int64(math.Round(end.Y * scale)),
	}
}

// Compiler represents a compiler entity responsible for managing and operating on a collection of 3D space sectors.
type Compiler struct {
	sectors          []*Sector
	sectorsMaxHeight float64
	cache            map[string]*Sector
}

// NewCompiler creates and returns a new Compiler instance initialized with default values.
func NewCompiler() *Compiler {
	return &Compiler{
		sectors:          nil,
		sectorsMaxHeight: 0,
		cache:            make(map[string]*Sector),
	}
}

// Setup initializes and configures the Compiler object based on the provided configuration and texture data.
// It processes the input sectors, creates segments, applies textures, resolves loops, and ensures Sector consistency.
// Returns an error in case of configuration issues or invalid state encountered during the setup process.
func (r *Compiler) Setup(cfg *ConfigRoot) error {
	modelSectorId := uint16(0)
	for idx, cs := range cfg.Sectors {
		var segments []*Segment
		var tags []string
		for _, cn := range cs.Segments {
			tags = append(tags, cn.Tag)
			tUpper := cfg.Textures.Get(cn.TextureUpper)
			tMiddle := cfg.Textures.Get(cn.TextureMiddle)
			tLower := cfg.Textures.Get(cn.TextureLower)
			segments = append(segments, NewSegment(cn.Neighbor, nil, cn.Kind, cn.Start, cn.End, cn.Tag, tUpper, tMiddle, tLower))
		}

		if len(segments) == 0 {
			fmt.Printf("Sector %s (idx: %d): vertices as zero len, removing\n", cs.Id, idx)
			continue
		}

		s := NewSector(modelSectorId, cs.Id, segments)
		modelSectorId++
		s.Tag = cs.Tag + "[" + strings.Join(tags, ";") + "]"
		s.Ceil = cs.Ceil
		s.Floor = cs.Floor
		s.TextureFloor = cfg.Textures.Get(cs.TextureFloor)
		s.TextureCeil = cfg.Textures.Get(cs.TextureCeil)
		s.TextureScaleFactor = cs.TextureScaleFactor
		s.LightDistance = cs.LightDistance
		r.sectors = append(r.sectors, s)
		r.cache[cs.Id] = s
	}

	for _, sect := range r.sectors {
		for _, segment := range sect.Segments {
			if segment.Kind != DefinitionWall {
				if s, ok := r.cache[segment.Ref]; ok {
					segment.SetSector(s.Id, s)
				} else {
					//fmt.Println("OUT", segment.Ref)
					//os.Exit(-1)
				}
			}
		}
	}

	//TODO 207 - 225
	//ch := &ConvexHull{}

	//for _, sect := range r.sectors {
	//	fmt.Println("-----------------------", sect.Id)
	//	ch.FromSector(sect)
	//}

	//ch.FromSector(r.sectors[54])
	//ch.FromSector(r.sectors[15])
	//ch.FromSector(r.sectors[38])
	//ch.FromSector(r.sectors[15])
	//ch.FromSector(r.sectors[54])

	//ch.FromSector(r.sectors[207])
	//fmt.Println("----")
	//ch.FromSector(r.sectors[225])

	//os.Exit(-1)

	if !cfg.DisableLoop {
		//Verify Loop
		for _, sector := range r.sectors {
			if len(sector.Segments) == 1 {
				continue
			}
			vFirst := sector.Segments[0]
			vLast := sector.Segments[len(sector.Segments)-1]
			hasLoop := vFirst.Start.X == vLast.End.X && vFirst.Start.Y == vLast.End.Y
			if !hasLoop {
				//TODO StubOld2 funziona solo se viene aggiunto in testa.....
				//vLast := sect.Vertices[len(sect.Vertices)-1]
				//sect.Vertices = append([]XY{vLast}, sect.Vertices...)
				fmt.Printf("creating loop for Sector %s\n", sector.Id)
				k := vLast.Copy()
				k.Start = k.End
				k.End = vFirst.Start
				sector.Segments = append(sector.Segments, k)
			} else {
				//TODO
				//fmt.Println("Adding an extra vertex")
				//vLast := sect.Segments[len(sect.Segments)-1]
				//sect.Segments = append(sect.Segments, vLast.Clone())
				//sect.NPoints = uint64(len(sect.Vertices) - 1)
			}
		}

		//Rescan:
		// Verify that for each edge that has a neighbor, the neighbor has this same neighbor.
		fixed := 0
		undefined := 0
		lineDefsCache := r.makeLineDefsCache()
		for _, sector := range r.sectors {
			for np, s := range sector.Segments {
				if s.Kind != DefinitionWall {
					//if s.Kind == DefinitionUnknown {
					if ld, ok := lineDefsCache[makeEdgeKey(s.End, s.Start)]; ok {
						if s.Ref != ld.sector.Id {
							fmt.Printf("p1 - Sector %s (segment: %d): Neighbor behind line (%g, %g) - (%g, %g) should be %s, %s found instead. Fixing...\n", sector.Id, np, s.Start.X, s.Start.Y, s.End.X, s.End.Y, ld.sector.Id, s.Ref)
							if s.Kind == DefinitionUnknown {
								s.Kind = DefinitionJoin
							}
							s.SetSector(ld.sector.Id, ld.sector)
							fixed++
							//goto Rescan
						}
					} else {
						fmt.Printf("p1 - Sector %s (segment: %d): Neighbor behind line (%g, %g) - (%g, %g) %s %s. Opposite not found\n", sector.Id, np, s.Start.X, s.Start.Y, s.End.X, s.End.Y, s.Ref, s.Tag)
						//s.Kind = DefinitionWall
						//s.Sector = nil
						//v1start.Update("wall", nil, DefinitionWall, v1start.XY)

						//if s.Kind == DefinitionVoid  {
						//	s.Kind = DefinitionWall
						//}
						undefined++
					}
					//}
				}
			}
		}
		fmt.Println("undefined:", undefined, "fixed:", fixed)
	}

	/*
		//TODO SISTEMARE
			// Verify that the vertexes form a convex hull.
			for idx, sect := range r.sectors {
				vert := sect.Vertices
				for b := uint64(0); b < sect.NPoints; b++ {
					c := (b + 1) % sect.NPoints
					d := (b + 2) % sect.NPoints
					x0 := vert[b].X;
					y0 := vert[b].Y;
					X1 := vert[c].X;
					y1 := vert[c].Y
					switch mathematic.PointSideF(vert[d].X, vert[d].Y, x0, y0, X1, y1) {
					case 0:
						continue
						//Note: This used to be a problem for my engine, but is not anymore, so it is disabled.
						//if you enable this change, you will not need the IntersectBox calls in some locations anymore.
						//if sect.NeighborsRefs[b] == sect.NeighborsRefs[c] { continue }
						//fmt.Printf("Sector %d: Edges %d-%d and %d-%d are parallel, but have different neighbors. This would pose problems for collision detection.\n", idx, b, c, c, d)
					case -1:
						fmt.Printf("Sector %d: Edges %d-%d and %d-%d create a concave turn. This would be rendered wrong.\n", idx, b, c, c, d)
					default:
						continue
					}

					fmt.Printf("- splitting Sector, using (%g,%g) as anchor\n", vert[c].X, vert[c].Y)

					// Insert an edge between (c) and (e), where e is the nearest point to (c), under the following rules:
					// e cannot be c, c-1 or c+1
					// line (c)-(e) cannot intersect with any edge in this Sector
					nearestDist := 1e29
					nearestPoint := ^uint64(0)
					for n := (d + 1) % sect.NPoints; n != b; n = (n + 1) % sect.NPoints {
						// Don't go through b, c, d
						X2 := vert[n].X
						y2 := vert[n].Y
						distX := X2 - X1
						distY := y2 - y1
						dist := distX*distX + distY*distY
						if dist >= nearestDist {
							continue
						}
						if mathematic.PointSideF(X2, y2, x0, y0, X1, y1) != 1 {
							continue
						}
						ok := true
						X1 += distX * 1e-4;
						X2 -= distX * 1e-4;
						y1 += distY * 1e-4;
						y2 -= distY * 1e-4
						for f := 0; f < int(sect.NPoints); f++ {
							if mathematic.IntersectLineSegmentsF(X1, y1, X2, y2, vert[f].X, vert[f].Y, vert[f+1].X, vert[f+1].Y) {
								ok = false
								break
							}
						}
						if !ok {
							continue
						}
						// Check whether this split would resolve the original problem
						if mathematic.PointSideF(X2, y2, vert[d].X, vert[d].Y, X1, y1) == 1 {
							dist += 1e6
						}
						if dist >= nearestDist {
							continue
						}
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

					var vert1 []*XYKind2
					//var neigh1 []int
					// Create chain 1: from c to e.
					for n := uint64(0); n < sect.NPoints; n++ {
						m := (c + n) % sect.NPoints
						vert1 = append(vert1, sect.Vertices[m].Clone())
						//neigh1 = append(neigh1, sect.NeighborsRefs[m])
						//vert1 = append(vert1, sect.Vertices[m])
						if m == e {
							vert1 = append(vert1, vert1[0])
							break
						}
					}

					//TODO ??????
					//neigh1Idx := len(r.sectors)
					//neigh1 = append(neigh1, neigh1Idx)

					var vert2 []*XYKind2
					//var neigh2 []int
					// Create chain 2: from e to c.
					for n := uint64(0); n < sect.NPoints; n++ {
						m := (e + n) % sect.NPoints
						//neigh2 = append(neigh2, sect.NeighborsRefs[m])
						//vert2 = append(vert2, sect.Vertices[m])
						vert2 = append(vert2, sect.Vertices[m].Clone())
						if m == c {
							vert2 = append(vert2, vert2[0])
							break
						}
					}
					//neigh2 = append(neigh2, idx)

					// using chain1
					sect.Vertices = vert1
					//sect.NeighborsRefs = neigh1
					sect.NPoints = uint64(len(vert1) - 1)
					sect = r.sectors[idx]

					ns := NewSector("AutoGenerated_"+NextUUId(), uint64(len(vert2)-1), vert2)
					//ns.NeighborsRefs = neigh2
					ns.Floor = sect.Floor
					ns.Ceil = sect.Ceil
					ns.FileTextures = sect.FileTextures
					ns.TextureFloor = sect.TextureFloor
					ns.TextureCeil = sect.TextureCeil
					ns.TextureUpper = sect.TextureUpper
					ns.TextureLower = sect.TextureLower
					ns.TextureWall = sect.TextureWall

					r.sectors = append(r.sectors, ns)

					r.cache[sect.Id] = sect

					// We needs to rescan
					goto Rescan
				}
			}
	*/

	r.finalize(cfg)

	fmt.Println("Scan complete")

	return nil
}

// finalize adjusts Sector vertex coordinates by the configured scale factor and updates the maximum Sector height.
func (r *Compiler) finalize(cfg *ConfigRoot) {
	scale := cfg.ScaleFactor
	if scale < 1 {
		scale = 1
	}

	cfg.Player.Position.X /= scale
	cfg.Player.Position.Y /= scale

	r.sectorsMaxHeight = 0
	for _, sect := range r.sectors {
		//vertex scale
		if scale != 1 {
			for s := 0; s < len(sect.Segments); s++ {
				sect.Segments[s].Start.X /= scale
				sect.Segments[s].Start.Y /= scale
				sect.Segments[s].End.X /= scale
				sect.Segments[s].End.Y /= scale
			}
		}
		//maxHeight
		if h := math.Abs(sect.Ceil - sect.Floor); h > r.sectorsMaxHeight {
			r.sectorsMaxHeight = h
		}
	}
}

// GetSectors returns a slice of pointers to Sector, representing all sectors in the Compiler instance.
func (r *Compiler) GetSectors() []*Sector {
	return r.sectors
}

// Get retrieves a Sector from the cache using the provided sectorId. Returns an error if the sectorId is invalid.
func (r *Compiler) Get(sectorId string) (*Sector, error) {
	s, ok := r.cache[sectorId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("invalid Sector: %s", sectorId))
	}
	return s, nil
}

// GetMaxHeight returns the maximum height difference between the ceiling and floor among the sectors in the compiler.
func (r *Compiler) GetMaxHeight() float64 {
	return r.sectorsMaxHeight
}

// makeLineDefsCache constructs a cache of line definitions mapped by a unique hash key generated from segment coordinates.
func (r *Compiler) makeLineDefsCache() map[edgeKey]*lineDef2 {
	t := make(map[edgeKey]*lineDef2)
	for _, sect := range r.sectors {
		for np := 0; np < len(sect.Segments); np++ {
			s := sect.Segments[np]
			hash := makeEdgeKey(s.Start, s.End)
			ld := &lineDef2{sector: sect, np: np, start: s.Start, end: s.End}
			if fld, ok := t[hash]; ok {
				if sect.Id != fld.sector.Id {
					//fmt.Println("line segment already added", sect.Id, fld.Sector.Id, hash, np)
				}
			} else {
				t[hash] = ld
			}
		}
	}
	return t
}
