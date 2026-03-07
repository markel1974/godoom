package portal

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
)

// defaultQueueLen defines the initial size of the queue used for processing rendering or computational tasks.
const (
	defaultQueueLen = 512
)

// Yaw computes a transformed y-coordinate based on input y, z, and yaw values, returning the adjusted result.
func Yaw(y float64, z float64, yaw float64) float64 {
	return y + (z * yaw)
}

// Portal represents a rendering portal used for managing visibility, sectors, and screen dimensions in a 3D environment.
type Portal struct {
	screenWidth      int
	screenWidthHalf  int
	screenHeight     int
	screenHeightHalf float64
	maxSectors       int
	queue            []*QueueItem
	sectorQueue      []*QueueItem
	screenHFov       float64
	screenVFov       float64
	compileId        uint64
	sectorsMaxHeight float64
	Sectors          []*model.Sector
	compiledSectors  []*model.CompiledSector
	compiledCount    int
	visibilityCache  *VisibilityCache
}

// NewPortal creates and initializes a new Portal instance with the specified screen dimensions and queue length.
// If maxQueue is 0, it defaults to a predefined constant value.
func NewPortal(width int, height int, maxQueue int) *Portal {
	if maxQueue == 0 {
		maxQueue = defaultQueueLen
	}
	r := &Portal{
		screenWidth:      width,
		screenWidthHalf:  width / 2,
		screenHeight:     height,
		screenHeightHalf: float64(height) / 2,
		queue:            make([]*QueueItem, maxQueue),
		sectorQueue:      make([]*QueueItem, 256),
		screenHFov:       model.HFov * float64(height),
		screenVFov:       model.VFov * float64(height),
		visibilityCache:  NewVisibilityCache(),
	}
	for x := 0; x < len(r.queue); x++ {
		r.queue[x] = &QueueItem{}
	}
	for x := 0; x < len(r.sectorQueue); x++ {
		r.sectorQueue[x] = &QueueItem{}
	}
	return r
}

// Setup configures the Portal by assigning sectors, setting the maximum height, and initializing compiled sectors.
func (r *Portal) Setup(sectors []*model.Sector, maxHeight float64) error {
	r.Sectors = sectors
	r.sectorsMaxHeight = maxHeight
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

// SectorsMaxHeight returns the maximum height of all sectors in the portal as a float64 value.
func (r *Portal) SectorsMaxHeight() float64 {
	return r.sectorsMaxHeight
}

// ScreenWidth returns the current width of the screen associated with the Portal instance.
func (r *Portal) ScreenWidth() int {
	return r.screenWidth
}

// ScreenHeight returns the height of the screen in pixels.
func (r *Portal) ScreenHeight() int {
	return r.screenHeight
}

// clear resets the Portal's state by incrementing the compile ID, setting compiled sector count to 0, and clearing the visibility cache.
func (r *Portal) clear() {
	r.compileId++
	r.compiledCount = 0
	r.visibilityCache.Clear()
}

// growSectorQueue doubles the size of the sectorQueue and initializes new QueueItem instances for the expanded slots.
func (r *Portal) growSectorQueue() {
	oldLen := len(r.sectorQueue)
	newLen := oldLen * 2
	newQueue := make([]*QueueItem, newLen)
	copy(newQueue, r.sectorQueue)
	for i := oldLen; i < newLen; i++ {
		newQueue[i] = &QueueItem{}
	}
	r.sectorQueue = newQueue
}

// Compile processes the active view, traverses sectors, and generates compiled sectors for rendering optimization.
func (r *Portal) Compile(vi *model.ViewItem) ([]*model.CompiledSector, int) {
	r.clear()

	queueLen := len(r.queue)
	headIdx := 0
	tailIdx := 0

	// Inizializzazione Root
	wMax := float64(r.screenWidth - 1)
	hMax := float64(r.screenHeight-1) * 3

	r.queue[headIdx].Update(vi.Sector, 0, wMax, -hMax, -hMax, hMax, hMax)
	headIdx = (headIdx + 1) % queueLen

	for headIdx != tailIdx {
		current := r.queue[tailIdx]
		tailIdx = (tailIdx + 1) % queueLen

		sector := current.sector
		sector.Reference(r.compileId)

		sq, sqCount := r.compileSector(vi, sector, current)
		for w := 0; w < sqCount; w++ {
			q := sq[w]

			// Geometric check
			if q.x2 > q.x1 && r.visibilityCache.IsVisible(q.sector, q.x1, q.x2) {
				// Store the span for this sector
				r.visibilityCache.Add(q.sector, q.x1, q.x2)

				// Check queue overflow
				if (headIdx+1)%queueLen == tailIdx {
					continue
				}

				r.queue[headIdx].Update(q.sector, q.x1, q.x2, q.y1t, q.y2t, q.y1b, q.y2b)
				headIdx = (headIdx + 1) % queueLen
			}
		}
	}

	return r.compiledSectors, r.compiledCount
}

// GetCS retrieves a CompiledSector and binds it to the provided Sector, returning whether it is the first retrieval.
func (r *Portal) GetCS(sector *model.Sector) (*model.CompiledSector, bool) {
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

// compileSector determines visible geometry and propagates visibility to adjacent sectors based on the current view matrix.
func (r *Portal) compileSector(vi *model.ViewItem, sector *model.Sector, qi *QueueItem) ([]*QueueItem, int) {
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

		// If the entire segment is behind the camera, discard it immediately
		if tz1 <= 0 && tz2 <= 0 {
			// Edge-case: if we are EXACTLY on the portal line,
			// float imprecision could give Z <= 0.
			if neighbor != nil && tz1 >= -0.1 && tz2 >= -0.1 {
				if outIdx >= len(r.sectorQueue) {
					r.growSectorQueue()
				}
				r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
				outIdx++
			}
			continue
		}

		// Calculate real length for texture repetition (world UV mapping)
		// Multiply by TextureScaleFactor if your WAD coordinates have been scaled (e.g. * 100.0)
		segLength := math.Hypot(vx2-vx1, vy2-vy1) * 100.0 // Adjust this multiplier for your engine
		u0 := 0.0
		u1 := segLength

		// EXACT LINEAR CLIPPING AGAINST THE NEAR-Z PLANE
		if tz1 <= model.NearZ || tz2 <= model.NearZ {
			if tz1 <= model.NearZ && tz2 <= model.NearZ {
				// CRITICAL FIX: The segment is too close to the camera to generate geometry,
				// but if it's a portal, we are physically crossing the threshold.
				// We must necessarily pass visibility to the adjacent sector!
				if neighbor != nil {
					if outIdx >= len(r.sectorQueue) {
						r.growSectorQueue()
					}
					// Propagate the current screen opening to the new sector
					r.sectorQueue[outIdx].Update(neighbor, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
					outIdx++
				}
				continue
			}

			// Calculate the exact intersection point (t from 0.0 to 1.0)
			t := (model.NearZ - tz1) / (tz2 - tz1)
			ix := tx1 + t*(tx2-tx1)
			iz := model.NearZ
			iu := u0 + t*(u1-u0) // Interpolate the UV exactly at the cut

			if tz1 <= model.NearZ {
				tx1 = ix
				tz1 = iz
				u0 = iu
			} else {
				tx2 = ix
				tz2 = iz
				u1 = iu
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

		//screenHeightHalf := float64(r.screenHeightHalf)

		y1a := r.screenHeightHalf + (-Yaw(sectorYCeil, tz1, vi.Yaw) * yScale1)
		y2a := r.screenHeightHalf + (-Yaw(sectorYCeil, tz2, vi.Yaw) * yScale2)
		y1b := r.screenHeightHalf + (-Yaw(sectorYFloor, tz1, vi.Yaw) * yScale1)
		y2b := r.screenHeightHalf + (-Yaw(sectorYFloor, tz2, vi.Yaw) * yScale2)

		yaStart := (x1Max-x1)*(y2a-y1a)/(x2-x1) + y1a
		yaStop := (x2Min-x1)*(y2a-y1a)/(x2-x1) + y1a
		ybStart := (x1Max-x1)*(y2b-y1b)/(x2-x1) + y1b
		ybStop := (x2Min-x1)*(y2b-y1b)/(x2-x1) + y1b

		zStart := ((x1Max-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8
		zStop := ((x2Min-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8

		lightStart := 1 - (zStart * vi.LightDistance)
		lightStop := 1 - (zStop * vi.LightDistance)

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

		ceilP := cs.Acquire(neighbor, model.IdCeil, cs.Sector.TextureCeil, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		ceilP.Rect(x1Max, y1Ceil, yaStart, zStart, lightStart, x2Min, y2Ceil, yaStop, zStop, lightStop)

		floorP := cs.Acquire(neighbor, model.IdFloor, cs.Sector.TextureFloor, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		floorP.Rect(x1Max, ybStart, y1Floor, zStart, lightStart, x2Min, ybStop, y2Floor, zStop, lightStop)

		if neighbor != nil {
			neighborYCeil := neighbor.Ceil - vi.Where.Z
			ny1a := r.screenHeightHalf + (-Yaw(neighborYCeil, tz1, vi.Yaw) * yScale1)
			ny2a := r.screenHeightHalf + (-Yaw(neighborYCeil, tz2, vi.Yaw) * yScale2)
			nYaStart := (x1Max-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			nYaStop := (x2Min-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			if yaStart-yaStop != 0 || nYaStop-nYaStop != 0 {
				upperP := cs.Acquire(neighbor, model.IdUpper, segment.TextureUpper, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
				upperP.Rect(x1Max, yaStart, nYaStart, zStart, lightStart, x2Min, yaStop, nYaStop, zStop, lightStop)
			}
			y1Ceil = mathematic.MaxF(yaStart, nYaStart)
			y2Ceil = mathematic.MaxF(yaStop, nYaStop)

			neighborYFloor := neighbor.Floor - vi.Where.Z
			ny1b := r.screenHeightHalf + (-Yaw(neighborYFloor, tz1, vi.Yaw) * yScale1)
			ny2b := r.screenHeightHalf + (-Yaw(neighborYFloor, tz2, vi.Yaw) * yScale2)
			nYbStart := (x1Max-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			nYbStop := (x2Min-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			if ybStart-nYbStart != 0 || nYbStop-ybStop != 0 {
				lowerP := cs.Acquire(neighbor, model.IdLower, segment.TextureLower, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
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
			wallP := cs.Acquire(neighbor, model.IdWall, segment.TextureMiddle, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
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
