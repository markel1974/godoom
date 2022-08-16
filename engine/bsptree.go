package main

import (
	"fmt"
	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	"math"
)

const (
	defaultQueueLen = 32
	hFov            = 0.73
	vFov            = 0.2

	nearZ    = 1e-4
	nearSide = 1e-5
	farZ     = 5.0
	farSide  = 20.0
)

type viewItem struct {
	where    model.XYZ
	angleSin float64
	angleCos float64
	yaw      float64
	sector   *model.Sector
	zoom     float64
}

type queueItem struct {
	sector *model.Sector
	x1     float64
	x2     float64
	y1t    float64
	y2t    float64
	y1b    float64
	y2b    float64
}

func (qi *queueItem) Update(sector *model.Sector, x1 float64, x2 float64, y1t float64, y2t float64, y1b float64, y2b float64) {
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
	textures         *textures.Textures
	sectors          []*model.Sector
	compiledSectors  []*CompiledSector
	compiledCount    int
}

func NewBSPTree(width int, height int, maxQueue int, textures *textures.Textures) *BSPTree {
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

func (r *BSPTree) Setup(sectors []*model.Sector, maxHeight float64) error {
	r.sectors = sectors
	r.sectorsMaxHeight = maxHeight
	r.compiledSectors = make([]*CompiledSector, len(r.sectors) * 16)
	for cs := 0; cs < len(r.compiledSectors); cs++ {
		r.compiledSectors[cs] = NewCompiledSector()
		r.compiledSectors[cs].Setup(512)
	}
	return nil
}

func (r *BSPTree) clear() {
	r.compileId++
	r.compiledCount = 0
}



//var __patch map[*model.Sector]int

func (r *BSPTree) Compile(vi *viewItem) ([]*CompiledSector, int) {
	r.clear()

	 //__patch = make(map[*model.Sector]int)

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
			if q.x2 > q.x1 && (headIdx + queueLen + 1 - tailIdx) % queueLen != 0 {
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
func (r *BSPTree) compileSector(vi *viewItem, sector *model.Sector, qi *queueItem) ([]*queueItem, int) {
	first := r.compiledCount == 0
	cs := r.compiledSectors[r.compiledCount]
	r.compiledCount++
	cs.Bind(sector)

	outIdx := 0

	//ceilPComplete := cs.Acquire( nil, IdCeilTest, 0, 0, 0, 0, 0, 0)
	//floorPComplete := cs.Acquire(nil, IdFloorTest, 0, 0, 0, 0, 0, 0)

	for s := uint64(0); s < sector.NPoints; s++ {
		vertexCurr := sector.Vertices[s]
		vertexNext := sector.Vertices[s + 1]
		neighbor := sector.Vertices[s].Sector

		if vertexCurr.Kind == model.DefinitionVoid {
			if neighbor != nil && neighbor != sector {
				r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
				outIdx++
			}
			continue
		}

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

		//TODO RIATTIVARE
		if tz1 <= 0 && tz2 <= 0 { continue }

		u0 := 0.0
		u1 := float64(textures.TextureEnd)

		// If partially in front of the player, clip it against player's view frustum
		if tz1 <= 0 || tz2 <= 0 {
			// Find an intersection between the wall and the approximate edges of player's view
			i1X, i1Y, _ := mathematic.IntersectFn(tx1, tz1, tx2, tz2, -nearSide, nearZ, -farSide, farZ)
			i2X, i2Y, _ := mathematic.IntersectFn(tx1, tz1, tx2, tz2, nearSide, nearZ, farSide, farZ)
			org1x := tx1; org1y := tz1; org2x := tx2; org2y := tz2
			if tz1 < nearZ { if i1Y > 0 { tx1 = i1X; tz1 = i1Y } else { tx1 = i2X; tz1 = i2Y } }
			if tz2 < nearZ { if i1Y > 0 { tx2 = i1X; tz2 = i1Y} else {tx2 = i2X; tz2 = i2Y } }

			//https://en.wikipedia.org/wiki/Texture_mapping
			if math.Abs(tx2 - tx1) > math.Abs(tz2 - tz1) {
				u0 = (tx1 - org1x) * textures.TextureEnd / (org2x - org1x)
				u1 = (tx2 - org1x) * textures.TextureEnd / (org2x - org1x)
			} else {
				u0 = (tz1 - org1y) * textures.TextureEnd / (org2y - org1y)
				u1 = (tz2 - org1y) * textures.TextureEnd / (org2y - org1y)
			}
		}

		// Perspective transformation
		xScale1 := r.screenHFov / tz1
		yScale1 := r.screenVFov / tz1
		x1 := float64(r.screenWidthHalf) - (tx1 * xScale1)
		xScale2 := r.screenHFov / tz2
		yScale2 := r.screenVFov / tz2
		x2 := float64(r.screenWidthHalf) - (tx2 * xScale2)

		//TODO RIATTIVARE
		if x1 > x2 || x2 < qi.x1 || x1 > qi.x2 {
			continue
		}
		x1Max := mathematic.MaxF(x1, qi.x1)
		x2Min := mathematic.MinF(x2, qi.x2)

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
			if _, i1, ok := mathematic.IntersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x1Max, ybStart, x1Max, qi.y1t); ok { y1Ceil = i1 }
			if _, i1, ok := mathematic.IntersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x1Max, ybStart, x1Max, qi.y1b); ok { y1Floor = i1 }
		}
		if x2Min != qi.x2 {
			if _, i2, ok := mathematic.IntersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x2Min, ybStop, x2Min, qi.y2t); ok { y2Ceil = i2 }
			if _, i2, ok := mathematic.IntersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x2Min, ybStop, x2Min, qi.y2b); ok { y2Floor = i2 }
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
			y1Ceil = mathematic.MaxF(yaStart, nYaStart)
			y2Ceil = mathematic.MaxF(yaStop, nYaStop)

			neighborYFloor := neighbor.Floor - vi.where.Z
			ny1b := screenHeightHalf + (-Yaw(neighborYFloor, tz1, vi.yaw) * yScale1)
			ny2b := screenHeightHalf + (-Yaw(neighborYFloor, tz2, vi.yaw) * yScale2)
			nYbStart := (x1Max - x1) * (ny2b - ny1b) / (x2 - x1) + ny1b
			nYbStop :=  (x2Min - x1) * (ny2b - ny1b) / (x2 - x1) + ny1b
			if ybStart-nYbStart != 0 || nYbStop-ybStop != 0 {
				lowerP := cs.Acquire(neighbor, IdLower, x1, x2, tz1, tz2, u0, u1)
				lowerP.Rect(x1Max, nYbStart, ybStart, zStart, lightStart, x2Min, nYbStop, ybStop, zStop, lightStop)
			}
			y1Floor = mathematic.MinF(nYbStart, ybStart)
			y2Floor = mathematic.MinF(nYbStop, ybStop)



			/*
			//TODO RIMUOVERE!!!!!!!!
			if v, ok := __patch[neighbor]; ok {
				if v >= 3 {
					continue
				}
				__patch[neighbor] = v + 1
			} else {
				__patch[neighbor] = 1
			}

			 */

			//TODO RIATTIVARE
			if neighbor != sector { //circuit breaker
				r.sectorQueue[outIdx].Update(neighbor, x1Max, x2Min, y1Ceil, y2Ceil, y1Floor, y2Floor)
				outIdx++
			}
		} else {
			wallP := cs.Acquire(neighbor, IdWall, x1, x2, tz1, tz2, u0, u1)
			wallP.Rect(x1Max, yaStart, ybStart, zStart, lightStart, x2Min, yaStop, ybStop, zStop, lightStop)
		}
	}

	//ceilPComplete.Finalize()
	//floorPComplete.Finalize()

	if first && outIdx == 0 {
		for s := uint64(0); s < sector.NPoints; s++ {
			neighbor := sector.Vertices[s].Sector
			if neighbor != nil {
				//TODO RIATTIVARE
				if neighbor != sector {
					r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
					outIdx++
				}
			}
		}
	}

	return r.sectorQueue, outIdx
}
