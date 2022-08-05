package main

import (
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/config"
	"math"
	"strconv"
	"strings"
)

const (
	defaultQueueLen = 32
	wallDefinition  = -1
	hFov            = 0.73
	vFov            = 0.2

	nearZ    = 1e-4
	farZ     = 5.0
	nearSide = 1e-5
	farSide  = 20.0
)

type viewItem struct {
	where    XYZ
	angleSin float64
	angleCos float64
	yaw      float64
	sector   *Sector
	zoom     float64
}

type queueItem struct {
	sector *Sector
	x1     float64
	x2     float64
	y1t    float64
	y2t    float64
	y1b    float64
	y2b    float64
}

func (qi *queueItem) Update(sector *Sector, x1 float64, x2 float64, y1t float64, y2t float64, y1b float64, y2b float64) {
	qi.sector = sector
	qi.x1 = x1
	qi.x2 = x2
	qi.y1t = y1t
	qi.y2t = y2t
	qi.y1b = y1b
	qi.y2b = y2b
}

type BSPTree struct {
	screenWidth      int
	screenWidthHalf  int
	screenHeight     int
	screenHeightHalf int
	maxSectors       int
	queue            []*queueItem
	sectorQueue      []*queueItem
	screenHFov       float64
	screenVFov       float64
	compileId        int
	sectorsMaxHeight float64
	textures         *Textures
	sectors          []*Sector
	compiledSectors  []*CompiledSector
	compiledCount    int
}

func NewBSPTree(width int, height int, maxQueue int, textures *Textures) *BSPTree {
	if maxQueue != 0 && (maxQueue&(maxQueue-1)) != 0 {
		fmt.Printf("maxQueue is not a power of two, using %d\n", defaultQueueLen)
		maxQueue = defaultQueueLen
	}
	r := &BSPTree{
		textures:         textures,
		screenWidth:      width,
		screenWidthHalf:  width / 2,
		screenHeight:     height,
		screenHeightHalf: height / 2,

		sectorsMaxHeight: 0,

		queue:         make([]*queueItem, maxQueue),
		sectorQueue:   make([]*queueItem, 256), //TODO DYNAMIC
		maxSectors:    maxQueue + 1,
		screenHFov:    hFov * float64(height),
		screenVFov:    vFov * float64(height),
		compileId:     0,
		compiledCount: 0,
	}
	for x := 0; x < len(r.queue); x++ {
		r.queue[x] = &queueItem{}
	}
	for x := 0; x < len(r.sectorQueue); x++ {
		r.sectorQueue[x] = &queueItem{}
	}
	return r
}

func (r *BSPTree) Setup(playerSector string, cfgSectors []*config.Sector) (*Sector, error) {
	cache := make(map[string]int)

	for idx, cfgSector := range cfgSectors {
		var vertices []XY
		var neighborsIds []string

		for _, cfgNeighbor := range cfgSector.Neighbors {
			vertices = append(vertices, XY{X: cfgNeighbor.X, Y: cfgNeighbor.Y})
			neighborsIds = append(neighborsIds, cfgNeighbor.Id)
		}

		s := NewSector(cfgSector.Id, uint64(len(vertices)), vertices, neighborsIds)
		s.Ceil = cfgSector.Ceil
		s.Floor = cfgSector.Floor
		s.Textures = cfgSector.Textures
		if s.Textures {
			s.FloorTexture = r.textures.Get(cfgSector.FloorTexture)
			s.CeilTexture = r.textures.Get(cfgSector.CeilTexture)
			s.UpperTexture = r.textures.Get(cfgSector.UpperTexture)
			s.LowerTexture = r.textures.Get(cfgSector.LowerTexture)
			s.WallTexture = r.textures.Get(cfgSector.WallTexture)
			if s.FloorTexture == nil || s.CeilTexture == nil && s.UpperTexture == nil || s.LowerTexture == nil || s.WallTexture == nil {
				fmt.Println("invalid textures configuration for sector", s.Id)
				s.Textures = false
			}
		}
		r.sectors = append(r.sectors, s)
		cache[cfgSector.Id] = idx
	}

	for _, sect := range r.sectors {
		for _, id := range sect.NeighborsIds {
			switch strings.Trim(strings.ToLower(id), " \t\n") {
			case "", "-1", "wall":
				sect.NeighborsRefs = append(sect.NeighborsRefs, wallDefinition)
			default:
				idx, ok := cache[id]
				if !ok {
					return nil, errors.New(fmt.Sprintf("can't find secor id: %s", id))
				}
				sect.NeighborsRefs = append(sect.NeighborsRefs, idx)
			}
		}
	}

	playerSectorIdx, ok := cache[playerSector]
	if !ok {
		return nil, errors.New(fmt.Sprintf("invalid player sector: %s", playerSector))
	}

	r.compiledSectors = make([]*CompiledSector, len(r.sectors)*16)

	for cs := 0; cs < len(r.compiledSectors); cs++ {
		r.compiledSectors[cs] = NewCompiledSector()
		r.compiledSectors[cs].Setup(128)
	}

	//Verify Loop
	for idx, sect := range r.sectors {
		if len(sect.Vertices) == 0 {
			return nil, errors.New(fmt.Sprintf("sector %d: vertices as zero len", idx))
		}
		hasLoop := false
		vFirst := sect.Vertices[0]
		if len(sect.Vertices) > 1 {
			vLast := sect.Vertices[len(sect.Vertices)-1]
			hasLoop = vFirst.X == vLast.X && vFirst.Y == vLast.Y
		}
		if !hasLoop {
			vLast := sect.Vertices[len(sect.Vertices)-1]
			sect.Vertices = append([]XY{vLast}, sect.Vertices...)
			fmt.Printf("creating loop for sector %d\n", idx)
		}
	}

Rescan:
	// Verify that for each edge that has a neighbor, the neighbor has this same neighbor.
	for idx, sect := range r.sectors {
		vert := sect.Vertices
		for b := uint64(0); b < sect.NPoints; b++ {
			p1 := vert[b]
			p2 := vert[b+1]
			found := 0
			for d, neighbor := range r.sectors {
				for c := uint64(0); c < neighbor.NPoints; c++ {
					c0x := neighbor.Vertices[c+0].X
					c0y := neighbor.Vertices[c+0].Y
					c1x := neighbor.Vertices[c+1].X
					c1y := neighbor.Vertices[c+1].Y
					if c1x == p1.X && c1y == p1.Y && c0x == p2.X && c0y == p2.Y {
						neighborIdx := neighbor.NeighborsRefs[c]
						if idx != neighborIdx {
							fmt.Printf("sector %d: Neighbor behind line (%g,%g)-(%g,%g) should be %d, %d found instead. Fixing.\n", c, p2.X, p2.Y, p1.Y, p1.Y, idx, neighbor.NeighborsRefs[c])
							neighbor.NeighborsRefs[c] = idx
							goto Rescan
						}
						if d != sect.NeighborsRefs[b] {
							fmt.Printf("sector %d: Neighbor behind line (%g,%g)-(%g,%g) should be %d, %d found instead. Fixing.\n", c, p1.X, p1.Y, p2.X, p2.Y, idx, sect.NeighborsRefs[b])
							sect.NeighborsRefs[b] = d
							goto Rescan
						} else {
							found++
						}
					}
				}
			}
			if sect.NeighborsRefs[b] >= 0 && sect.NeighborsRefs[b] < len(r.sectors) && found != 1 {
				fmt.Printf("sectors %d and its neighbor %d don't share line (%g,%g)-(%g,%g)\n", idx, sect.NeighborsRefs[b], p1.X, p1.Y, p2.X, p2.Y)
			}
		}
	}

	// Verify that the vertexes form a convex hull.
	for idx, sect := range r.sectors {
		vert := sect.Vertices
		for b := uint64(0); b < sect.NPoints; b++ {
			c := (b + 1) % sect.NPoints
			d := (b + 2) % sect.NPoints
			x0 := vert[b].X
			y0 := vert[b].Y
			x1 := vert[c].X
			y1 := vert[c].Y
			switch pointSideF(vert[d].X, vert[d].Y, x0, y0, x1, y1) {
			case 0:
				continue
				//Note: This used to be a problem for my engine, but is not anymore, so it is disabled. if you enable this change, you will not need the IntersectBox calls in some locations anymore.
				//if sect.Neighbors[b] == sect.Neighbors[c] { continue }
				//fmt.Printf("sector %d: Edges %d-%d and %d-%d are parallel, but have different neighbors. This would pose problems for collision detection.\n", a, b, c, c, d)
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
				if dist >= nearestDist {
					continue
				}
				if pointSideF(x2, y2, x0, y0, x1, y1) != 1 {
					continue
				}
				ok := true
				x1 += distX * 1e-4
				x2 -= distX * 1e-4
				y1 += distY * 1e-4
				y2 -= distY * 1e-4
				for f := 0; f < int(sect.NPoints); f++ {
					if intersectLineSegmentsF(x1, y1, x2, y2, vert[f].X, vert[f].Y, vert[f+1].X, vert[f+1].Y) {
						ok = false
						break
					}
				}
				if !ok {
					continue
				}
				// Check whether this split would resolve the original problem
				if pointSideF(x2, y2, vert[d].X, vert[d].Y, x1, y1) == 1 {
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

	r.sectorsMaxHeight = 0
	for _, sect := range r.sectors {
		sect.Neighbors = make([]*Sector, sect.NPoints)

		if h := sect.Ceil - sect.Floor; h > r.sectorsMaxHeight {
			r.sectorsMaxHeight = h
		}

		for s := uint64(0); s < sect.NPoints; s++ {
			neighborIdx := sect.NeighborsRefs[s]
			if neighborIdx > wallDefinition {
				neighbor := r.sectors[neighborIdx]
				sect.Neighbors[s] = r.sectors[neighborIdx]
				if h := neighbor.Ceil - neighbor.Floor; h > r.sectorsMaxHeight {
					r.sectorsMaxHeight = h
				}
			}
		}
	}

	return r.sectors[playerSectorIdx], nil
}

func (r *BSPTree) clear() {
	r.compileId++
	r.compiledCount = 0
}

func (r *BSPTree) Compile(vi *viewItem) ([]*CompiledSector, int) {
	r.clear()

	queueLen := len(r.queue)
	headIdx := 0
	tailIdx := 0
	head := r.queue[headIdx]
	tail := r.queue[tailIdx]

	const wFactor = 1
	const hFactor = 3

	wMax := (float64(r.screenWidth) - 1) * wFactor
	hMax := (float64(r.screenHeight) - 1) * hFactor
	head.sector = vi.sector
	head.x1 = 0
	head.x2 = wMax

	head.y1t = -hMax
	head.y2t = -hMax
	head.y1b = hMax
	head.y2b = hMax

	headIdx++
	if headIdx == queueLen {
		headIdx = 0
	}
	head = r.queue[headIdx]

	for head != tail {
		current := tail
		tailIdx++
		if tailIdx == queueLen { tailIdx = 0 }
		tail = r.queue[tailIdx]

		sector := current.sector

		sector.Reference(r.compileId)
		if sector.GetUsage() & r.maxSectors != 0 { continue }
		sector.AddUsage()

		sq, sqCount := r.compileSector(vi, sector, current)
		for w := 0; w < sqCount; w++ {
			q := sq[w]
			if q.x2 >= q.x1 && (headIdx + queueLen + 1 - tailIdx) % queueLen != 0 {
				head.Update(q.sector, q.x1, q.x2, q.y1t, q.y2t, q.y1b, q.y2b)
				headIdx++
				if headIdx >= queueLen { headIdx = 0 }
				head = r.queue[headIdx]
			}
		}
		sector.AddUsage()
	}

	return r.compiledSectors, r.compiledCount
}


//TODO MOVE TO SoftwareRender
func (r *BSPTree) compileSector(vi *viewItem, sector *Sector, qi *queueItem) ([]*queueItem, int) {
	first := r.compiledCount == 0
	cs := r.compiledSectors[r.compiledCount]
	r.compiledCount++
	cs.Bind(sector)

	outIdx := 0

	//ceilPComplete := cs.Acquire( nil, IdCeilTest, 0, 0, 0, 0, 0, 0)
	//floorPComplete := cs.Acquire(nil, IdFloorTest, 0, 0, 0, 0, 0, 0)

	for s := uint64(0); s < sector.NPoints; s++ {
		vertexCurr := sector.Vertices[s]
		vertexNext := sector.Vertices[s+1]
		neighbor := sector.Neighbors[s]
		sectorYCeil := sector.Ceil - vi.where.Z
		sectorYFloor := sector.Floor - vi.where.Z

		vx1 := vertexCurr.X - vi.where.X
		vy1 := vertexCurr.Y - vi.where.Y
		vx2 := vertexNext.X - vi.where.X
		vy2 := vertexNext.Y - vi.where.Y

		// Rotate around the player's view
		tx1 := (vx1 * vi.angleSin) - (vy1 * vi.angleCos)
		tz1 := (vx1 * vi.angleCos) + (vy1 * vi.angleSin)
		tx2 := (vx2 * vi.angleSin) - (vy2 * vi.angleCos)
		tz2 := (vx2 * vi.angleCos) + (vy2 * vi.angleSin)

		// If is partially in front of the player continue
		if tz1 <= 0 && tz2 <= 0 { continue }

		u0 := 0.0
		u1 := float64(TextureEnd)

		// If partially in front of the player, clip it against player's view frustum
		if tz1 <= 0 || tz2 <= 0 {
			// Find an intersection between the wall and the approximate edges of player's view
			i1X, i1Y, _ := intersectFn(tx1, tz1, tx2, tz2, -nearSide, nearZ, -farSide, farZ)
			i2X, i2Y, _ := intersectFn(tx1, tz1, tx2, tz2, nearSide, nearZ, farSide, farZ)
			org1x := tx1; org1y := tz1; org2x := tx2; org2y := tz2
			if tz1 < nearZ { if i1Y > 0 { tx1 = i1X; tz1 = i1Y } else { tx1 = i2X; tz1 = i2Y } }
			if tz2 < nearZ { if i1Y > 0 { tx2 = i1X; tz2 = i1Y} else {tx2 = i2X; tz2 = i2Y } }

			//https://en.wikipedia.org/wiki/Texture_mapping
			if math.Abs(tx2 - tx1) > math.Abs(tz2 - tz1) {
				u0 = (tx1 - org1x) * TextureEnd / (org2x - org1x)
				u1 = (tx2 - org1x) * TextureEnd / (org2x - org1x)
			} else {
				u0 = (tz1 - org1y) * TextureEnd / (org2y - org1y)
				u1 = (tz2 - org1y) * TextureEnd / (org2y - org1y)
			}
		}

		// Perspective transformation
		xScale1 := r.screenHFov / tz1
		yScale1 := r.screenVFov / tz1
		x1 := float64(r.screenWidthHalf) - (tx1 * xScale1)
		xScale2 := r.screenHFov / tz2
		yScale2 := r.screenVFov / tz2
		x2 := float64(r.screenWidthHalf) - (tx2 * xScale2)

		// Compile if visible
		if x1 >= x2 || x2 < qi.x1 || x1 > qi.x2 { continue }
		x1Max := maxF(x1, qi.x1)
		x2Min := minF(x2, qi.x2)

		screenHeightHalf := float64(r.screenHeightHalf)

		// Project ceiling and floor into Y coordinates
		y1a := screenHeightHalf + (-Yaw(sectorYCeil, tz1, vi.yaw) * yScale1)
		y2a := screenHeightHalf + (-Yaw(sectorYCeil, tz2, vi.yaw) * yScale2)
		y1b := screenHeightHalf + (-Yaw(sectorYFloor, tz1, vi.yaw) * yScale1)
		y2b := screenHeightHalf + (-Yaw(sectorYFloor, tz2, vi.yaw) * yScale2)
		yaStart := (x1Max - x1) * (y2a - y1a) / (x2 - x1) + y1a
		yaStop :=  (x2Min - x1) * (y2a - y1a) / (x2 - x1) + y1a
		ybStart := (x1Max - x1) * (y2b - y1b) / (x2 - x1) + y1b
		ybStop :=  (x2Min - x1) * (y2b - y1b) / (x2 - x1) + y1b
		zStart := ((x1Max - x1) * (tz2 - tz1) / (x2 - x1) + tz1) * 8
		zStop :=  ((x2Min - x1) * (tz2 - tz1) / (x2 - x1) + tz1) * 8
		lightStart := 1 - (zStart * fullLightDistance)
		lightStop :=  1 - (zStop  * fullLightDistance)

		if zStart <= 0 { zStart = 10e4 }
		if zStop <= 0 { zStop = 10e4 }
		if lightStart < 0 { lightStart = 0 }
		if lightStop < 0 { lightStop = 0 }

		y1Ceil := qi.y1t; y2Ceil := qi.y2t; y1Floor := qi.y1b; y2Floor := qi.y2b
		if x1Max != qi.x1 {
			if _, i1, ok := intersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x1Max, ybStart, x1Max, qi.y1t); ok { y1Ceil = i1 }
			if _, i1, ok := intersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x1Max, ybStart, x1Max, qi.y1b); ok { y1Floor = i1 }
		}
		if x2Min != qi.x2 {
			if _, i2, ok := intersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x2Min, ybStop, x2Min, qi.y2t); ok { y2Ceil = i2 }
			if _, i2, ok := intersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x2Min, ybStop, x2Min, qi.y2b); ok { y2Floor = i2 }
		}

		ceilP := cs.Acquire(neighbor, IdCeil, x1, x2, tz1, tz2, u0, u1)
		ceilP.Rect(x1Max, y1Ceil, yaStart, zStart, lightStart, x2Min, y2Ceil, yaStop, zStop, lightStop)
		//ceilPComplete.AddPoint(x1, yaStart, zStart, lightStart, x2, yaStop, zStop, lightStop)

		floorP := cs.Acquire(neighbor, IdFloor, x1, x2, tz1, tz2, u0, u1)
		floorP.Rect(x1Max, ybStart, y1Floor, zStart, lightStart, x2Min, ybStop, y2Floor, zStop, lightStop)
		//floorPComplete.AddPoint(x1Max, ybStart, zStart, lightStart, x2Min, ybStop, zStop, lightStop)

		if neighbor != nil {
			neighborYCeil := neighbor.Ceil - vi.where.Z
			ny1a := screenHeightHalf + (-Yaw(neighborYCeil, tz1, vi.yaw) * yScale1)
			ny2a := screenHeightHalf + (-Yaw(neighborYCeil, tz2, vi.yaw) * yScale2)
			nYaStart := (x1Max - x1) * (ny2a - ny1a) / (x2 - x1) + ny1a
			nYaStop :=  (x2Min - x1) * (ny2a - ny1a) / (x2 - x1) + ny1a
			if yaStart-yaStop != 0 || nYaStop-nYaStop != 0 {
				upperP := cs.Acquire(neighbor, IdUpper, x1, x2, tz1, tz2, u0, u1)
				upperP.Rect(x1Max, yaStart, nYaStart, zStart, lightStart, x2Min, yaStop, nYaStop, zStop, lightStop)
			}
			y1Ceil = maxF(yaStart, nYaStart)
			y2Ceil = maxF(yaStop, nYaStop)

			neighborYFloor := neighbor.Floor - vi.where.Z
			ny1b := screenHeightHalf + (-Yaw(neighborYFloor, tz1, vi.yaw) * yScale1)
			ny2b := screenHeightHalf + (-Yaw(neighborYFloor, tz2, vi.yaw) * yScale2)
			nYbStart := (x1Max - x1) * (ny2b - ny1b) / (x2 - x1) + ny1b
			nYbStop :=  (x2Min - x1) * (ny2b - ny1b) / (x2 - x1) + ny1b
			if ybStart-nYbStart != 0 || nYbStop-ybStop != 0 {
				lowerP := cs.Acquire(neighbor, IdLower, x1, x2, tz1, tz2, u0, u1)
				lowerP.Rect(x1Max, nYbStart, ybStart, zStart, lightStart, x2Min, nYbStop, ybStop, zStop, lightStop)
			}
			y1Floor = minF(nYbStart, ybStart)
			y2Floor = minF(nYbStop, ybStop)

			r.sectorQueue[outIdx].Update(neighbor, x1Max, x2Min, y1Ceil, y2Ceil, y1Floor, y2Floor)
			outIdx++
		} else {
			wallP := cs.Acquire(neighbor, IdWall, x1, x2, tz1, tz2, u0, u1)
			wallP.Rect(x1Max, yaStart, ybStart, zStart, lightStart, x2Min, yaStop, ybStop, zStop, lightStop)
		}
	}

	//ceilPComplete.Finalize()
	//floorPComplete.Finalize()

	if first && outIdx == 0 {
		for s := uint64(0); s < sector.NPoints; s++ {
			neighbor := sector.Neighbors[s]
			if neighbor != nil {
				r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
				outIdx++
			}
		}
	}

	return r.sectorQueue, outIdx
}
