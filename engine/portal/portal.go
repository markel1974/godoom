package portal

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/renderers"
	"github.com/markel1974/godoom/engine/textures"
)

// defaultQueueLen defines the default length of the processing queue.
const (
	defaultQueueLen = 32
)

// Render is responsible for rendering portals and managing Sectors, queues, and compiled Sector states.
type Render struct {
	screenWidth      int
	screenWidthHalf  int
	screenHeight     int
	screenHeightHalf int
	maxSectors       int
	queue            []*QueueItem
	sectorQueue      []*QueueItem
	screenHFov       float64
	screenVFov       float64
	compileId        uint64
	SectorsMaxHeight float64
	textures         *textures.Textures
	Sectors          []*model.Sector
	compiledSectors  []*model.CompiledSector
	compiledCount    int
}

// NewPortal creates and initializes a new PortalRender instance with the specified screen dimensions and queue size.
func NewPortal(width int, height int, maxQueue int, textures *textures.Textures) *Render {
	if maxQueue != 0 && (maxQueue&(maxQueue-1)) != 0 {
		fmt.Printf("maxQueue is not a power of two, using %d\n", defaultQueueLen)
		maxQueue = defaultQueueLen
	}
	r := &Render{
		textures:         textures,
		screenWidth:      width,
		screenWidthHalf:  width / 2,
		screenHeight:     height,
		screenHeightHalf: height / 2,

		SectorsMaxHeight: 0,

		queue:         make([]*QueueItem, maxQueue),
		sectorQueue:   make([]*QueueItem, 256),
		maxSectors:    maxQueue + 1,
		screenHFov:    renderers.HFov * float64(height),
		screenVFov:    renderers.VFov * float64(height),
		compileId:     0,
		compiledCount: 0,
	}
	r.compileId--
	for x := 0; x < len(r.queue); x++ {
		r.queue[x] = &QueueItem{}
	}
	for x := 0; x < len(r.sectorQueue); x++ {
		r.sectorQueue[x] = &QueueItem{}
	}
	return r
}

// Setup initializes the PortalRender by setting up Sectors, maximum height, and allocating compiled Sector data.
func (r *Render) Setup(sectors []*model.Sector, maxHeight float64) error {
	r.Sectors = sectors
	r.SectorsMaxHeight = maxHeight
	r.compiledSectors = make([]*model.CompiledSector, len(r.Sectors)*16)

	//debug
	for _, s := range sectors {
		fmt.Println(s.Print(false))
	}

	for cs := 0; cs < len(r.compiledSectors); cs++ {
		r.compiledSectors[cs] = model.NewCompiledSector()
		r.compiledSectors[cs].Setup(512)
	}
	return nil
}

// clear increments the compileId and resets the compiledCount to 0.
func (r *Render) clear() {
	r.compileId++
	r.compiledCount = 0
}

// growSectorQueue doubles the size of the sectorQueue and initializes new QueueItem instances for the added capacity.
func (r *Render) growSectorQueue() {
	oldLen := len(r.sectorQueue)
	newLen := oldLen * 2
	newQueue := make([]*QueueItem, newLen)
	copy(newQueue, r.sectorQueue)
	for i := oldLen; i < newLen; i++ {
		newQueue[i] = &QueueItem{}
	}
	r.sectorQueue = newQueue
}

// Compile processes a ViewItem, performing recursive Sector visibility computation and returns compiled Sectors and their count.
func (r *Render) Compile(vi *renderers.ViewItem) ([]*model.CompiledSector, int) {
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
	head.sector = vi.Sector
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
		if tailIdx == queueLen {
			tailIdx = 0
		}
		tail = r.queue[tailIdx]

		sector := current.sector

		sector.Reference(r.compileId)

		sq, sqCount := r.compileSector(vi, sector, current)
		for w := 0; w < sqCount; w++ {
			q := sq[w]
			if q.x2 > q.x1 && (headIdx+queueLen+1-tailIdx)%queueLen != 0 {
				hash := q.Hash64()
				if !sector.Has(hash) {
					sector.Add(hash)
					head.Update(q.sector, q.x1, q.x2, q.y1t, q.y2t, q.y1b, q.y2b)
					headIdx++
					if headIdx >= queueLen {
						headIdx = 0
					}
					head = r.queue[headIdx]
				}
			}
		}
	}

	return r.compiledSectors, r.compiledCount
}

// GetCS retrieves a compiled Sector for the given Sector and indicates if it is the first retrieval in the current cycle.
func (r *Render) GetCS(sector *model.Sector) (*model.CompiledSector, bool) {
	first := r.compiledCount == 0
	if r.compiledCount >= len(r.compiledSectors) {
		fmt.Println("OUT OF COMPILED SECTORS!")
		return nil, false
	}
	cs := r.compiledSectors[r.compiledCount]
	r.compiledCount++
	cs.Bind(sector)
	return cs, first
}

// compileSector processes a Sector for rendering by partitioning segments and updating the render queue with adjacent Sectors.
func (r *Render) compileSector(vi *renderers.ViewItem, sector *model.Sector, qi *QueueItem) ([]*QueueItem, int) {
	var cs *model.CompiledSector = nil
	first := false
	outIdx := 0

	for s := 0; s < len(sector.Segments); s++ {
		segment := sector.Segments[s]
		vertexCurr := sector.Segments[s].Start
		vertexNext := sector.Segments[s].End
		neighbor := sector.Segments[s].Sector

		if neighbor == sector {
			continue
		}

		if segment.Kind == model.DefinitionVoid {
			if neighbor != nil {
				if outIdx >= len(r.sectorQueue) {
					r.growSectorQueue()
				}
				r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
				outIdx++
			}
			continue
		}

		sectorYCeil := sector.Ceil - vi.Where.Z
		sectorYFloor := sector.Floor - vi.Where.Z

		vx1 := vertexCurr.X - vi.Where.X
		vy1 := vertexCurr.Y - vi.Where.Y
		vx2 := vertexNext.X - vi.Where.X
		vy2 := vertexNext.Y - vi.Where.Y

		// Rotate around the player's view
		tx1 := (vx1 * vi.AngleSin) - (vy1 * vi.AngleCos)
		tz1 := (vx1 * vi.AngleCos) + (vy1 * vi.AngleSin)
		tx2 := (vx2 * vi.AngleSin) - (vy2 * vi.AngleCos)
		tz2 := (vx2 * vi.AngleCos) + (vy2 * vi.AngleSin)

		if tz1 <= 0 && tz2 <= 0 {
			continue
		}

		u0 := 0.0
		u1 := float64(textures.TextureEnd)

		// If partially in front of the player, clip it against player's view frustum
		if tz1 <= 0 || tz2 <= 0 {
			i1X, i1Y, _ := mathematic.IntersectFn(tx1, tz1, tx2, tz2, -renderers.NearSide, renderers.NearZ, -renderers.FarSide, renderers.FarZ)
			i2X, i2Y, _ := mathematic.IntersectFn(tx1, tz1, tx2, tz2, renderers.NearSide, renderers.NearZ, renderers.FarSide, renderers.FarZ)
			org1x := tx1
			org1y := tz1
			org2x := tx2
			org2y := tz2
			if tz1 < renderers.NearZ {
				if i1Y > 0 {
					tx1 = i1X
					tz1 = i1Y
				} else {
					tx1 = i2X
					tz1 = i2Y
				}
			}
			if tz2 < renderers.NearZ {
				if i1Y > 0 {
					tx2 = i1X
					tz2 = i1Y
				} else {
					tx2 = i2X
					tz2 = i2Y
				}
			}

			if math.Abs(tx2-tx1) > math.Abs(tz2-tz1) {
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

		if x1 > x2 {
			continue
		}
		if x2 < qi.x1 || x1 > qi.x2 {
			continue
		}

		x1Max := mathematic.MaxF(x1, qi.x1)
		x2Min := mathematic.MinF(x2, qi.x2)

		screenHeightHalf := float64(r.screenHeightHalf)

		y1a := screenHeightHalf + (-renderers.Yaw(sectorYCeil, tz1, vi.Yaw) * yScale1)
		y2a := screenHeightHalf + (-renderers.Yaw(sectorYCeil, tz2, vi.Yaw) * yScale2)
		y1b := screenHeightHalf + (-renderers.Yaw(sectorYFloor, tz1, vi.Yaw) * yScale1)
		y2b := screenHeightHalf + (-renderers.Yaw(sectorYFloor, tz2, vi.Yaw) * yScale2)
		yaStart := (x1Max-x1)*(y2a-y1a)/(x2-x1) + y1a
		yaStop := (x2Min-x1)*(y2a-y1a)/(x2-x1) + y1a
		ybStart := (x1Max-x1)*(y2b-y1b)/(x2-x1) + y1b
		ybStop := (x2Min-x1)*(y2b-y1b)/(x2-x1) + y1b
		zStart := ((x1Max-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8
		zStop := ((x2Min-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8
		lightStart := 1 - (zStart * renderers.FullLightDistance)
		lightStop := 1 - (zStop * renderers.FullLightDistance)

		if zStart <= 0 {
			zStart = 10e4
		}
		if zStop <= 0 {
			zStop = 10e4
		}
		if lightStart < 0 {
			lightStart = 0
		}
		if lightStop < 0 {
			lightStop = 0
		}

		y1Ceil := qi.y1t
		y2Ceil := qi.y2t
		y1Floor := qi.y1b
		y2Floor := qi.y2b
		if x1Max != qi.x1 {
			if _, i1, ok := mathematic.IntersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x1Max, ybStart, x1Max, qi.y1t); ok {
				y1Ceil = i1
			}
			if _, i1, ok := mathematic.IntersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x1Max, ybStart, x1Max, qi.y1b); ok {
				y1Floor = i1
			}
		}
		if x2Min != qi.x2 {
			if _, i2, ok := mathematic.IntersectFn(qi.x1, qi.y1t, qi.x2, qi.y2t, x2Min, ybStop, x2Min, qi.y2t); ok {
				y2Ceil = i2
			}
			if _, i2, ok := mathematic.IntersectFn(qi.x1, qi.y1b, qi.x2, qi.y2b, x2Min, ybStop, x2Min, qi.y2b); ok {
				y2Floor = i2
			}
		}

		if cs == nil {
			if cs, first = r.GetCS(sector); cs == nil {
				return nil, 0
			}
		}

		ceilP := cs.Acquire(neighbor, model.IdCeil, x1, x2, tz1, tz2, u0, u1)
		ceilP.Rect(x1Max, y1Ceil, yaStart, zStart, lightStart, x2Min, y2Ceil, yaStop, zStop, lightStop)

		floorP := cs.Acquire(neighbor, model.IdFloor, x1, x2, tz1, tz2, u0, u1)
		floorP.Rect(x1Max, ybStart, y1Floor, zStart, lightStart, x2Min, ybStop, y2Floor, zStop, lightStop)

		if neighbor != nil {
			neighborYCeil := neighbor.Ceil - vi.Where.Z
			ny1a := screenHeightHalf + (-renderers.Yaw(neighborYCeil, tz1, vi.Yaw) * yScale1)
			ny2a := screenHeightHalf + (-renderers.Yaw(neighborYCeil, tz2, vi.Yaw) * yScale2)
			nYaStart := (x1Max-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			nYaStop := (x2Min-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			if yaStart-yaStop != 0 || nYaStop-nYaStop != 0 {
				upperP := cs.Acquire(neighbor, model.IdUpper, x1, x2, tz1, tz2, u0, u1)
				upperP.Rect(x1Max, yaStart, nYaStart, zStart, lightStart, x2Min, yaStop, nYaStop, zStop, lightStop)
			}
			y1Ceil = mathematic.MaxF(yaStart, nYaStart)
			y2Ceil = mathematic.MaxF(yaStop, nYaStop)

			neighborYFloor := neighbor.Floor - vi.Where.Z
			ny1b := screenHeightHalf + (-renderers.Yaw(neighborYFloor, tz1, vi.Yaw) * yScale1)
			ny2b := screenHeightHalf + (-renderers.Yaw(neighborYFloor, tz2, vi.Yaw) * yScale2)
			nYbStart := (x1Max-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			nYbStop := (x2Min-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			if ybStart-nYbStart != 0 || nYbStop-ybStop != 0 {
				lowerP := cs.Acquire(neighbor, model.IdLower, x1, x2, tz1, tz2, u0, u1)
				lowerP.Rect(x1Max, nYbStart, ybStart, zStart, lightStart, x2Min, nYbStop, ybStop, zStop, lightStop)
			}
			y1Floor = mathematic.MinF(nYbStart, ybStart)
			y2Floor = mathematic.MinF(nYbStop, ybStop)

			if outIdx >= len(r.sectorQueue) {
				r.growSectorQueue()
			}
			r.sectorQueue[outIdx].Update(neighbor, x1Max, x2Min, y1Ceil, y2Ceil, y1Floor, y2Floor)
			outIdx++
		} else {
			wallP := cs.Acquire(neighbor, model.IdWall, x1, x2, tz1, tz2, u0, u1)
			wallP.Rect(x1Max, yaStart, ybStart, zStart, lightStart, x2Min, yaStop, ybStop, zStop, lightStop)
		}
	}

	if first && outIdx == 0 {
		for s := 0; s < len(sector.Segments); s++ {
			segment := sector.Segments[s]
			neighbor := segment.Sector
			if neighbor != nil && neighbor != sector {
				if outIdx >= len(r.sectorQueue) {
					r.growSectorQueue()
				}
				r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
				outIdx++
			}
		}
	}

	return r.sectorQueue, outIdx
}
