package portal

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
)

// defaultQueueLen defines the initial size of the queue used for processing rendering or computational tasks.
const (
	defaultQueueLen = 512
)

//TODO culling gerarchico spaziale

// Portal represents a rendering portal used for managing visibility, sectors, and screen dimensions in a 3D environment.
type Portal struct {
	screenWidth      int
	screenWidthHalf  int
	screenHeight     int
	screenHeightHalf float64
	maxSectors       int
	queue            *RingQueue
	sectorQueue      *LinearBatch
	screenHFov       float64
	screenVFov       float64
	compileId        uint64
	sectorsMaxHeight float64
	sectors          []*model.Sector
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
		queue:            NewRingQueue(maxQueue),
		sectorQueue:      NewLinearBatch(256),
		screenHFov:       model.HFov * float64(height),
		screenVFov:       model.VFov * float64(height),
		visibilityCache:  NewVisibilityCache(),
	}
	return r
}

// Setup configures the Portal by assigning sectors, setting the maximum height, and initializing compiled sectors.
func (r *Portal) Setup(sectors []*model.Sector, maxHeight float64) error {
	r.sectors = sectors
	r.sectorsMaxHeight = maxHeight
	r.compiledSectors = make([]*model.CompiledSector, len(r.sectors)*16)

	//debug
	//for _, s := range sectors {
	//	fmt.Println(s.Print(false))
	//}

	for cs := 0; cs < len(r.compiledSectors); cs++ {
		r.compiledSectors[cs] = model.NewCompiledSector()
		r.compiledSectors[cs].Setup(512)
	}
	return nil
}

// Len returns the number of sectors currently managed by the Portal.
func (r *Portal) Len() int {
	return len(r.sectors)
}

// SectorAt retrieves the Sector at the specified index within the Portal's sector list. Returns nil if the index is invalid.
func (r *Portal) SectorAt(idx int) *model.Sector {
	return r.sectors[idx]
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

// Compile processes the active view, traverses sectors, and generates compiled sectors for rendering optimization.
func (r *Portal) Compile(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int) {
	vi.Compute(player)

	r.clear()

	r.queue.Reset()

	// Inizializzazione Root
	wMax := float64(r.screenWidth - 1)
	hMax := float64(r.screenHeight-1) * 3

	qHead := r.queue.GetHead()
	qHead.Update(vi.GetSector(), 0, wMax, -hMax, -hMax, hMax, hMax)

	var qTail *QueueItem

	textures.Tick()

	for !r.queue.IsEmpty() {
		qTail = r.queue.GetTail()
		qTail.sector.Reference(r.compileId)

		sq, sqCount := r.compileSector(vi, qTail.sector, qTail)
		for w := 0; w < sqCount; w++ {
			q := sq[w]
			// Geometric check
			if q.x2 > q.x1 && r.visibilityCache.IsVisible(q.sector, q.x1, q.x2) {
				// Store the span for this sector
				r.visibilityCache.Add(q.sector, q.x1, q.x2)
				if r.queue.IsFull() {
					continue
				}
				qHead = r.queue.GetHead()
				qHead.Update(q.sector, q.x1, q.x2, q.y1t, q.y2t, q.y1b, q.y2b)
			}
		}
	}

	player.Compute(vi)

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
func (r *Portal) compileSector(vi *model.ViewMatrix, sector *model.Sector, qi *QueueItem) ([]QueueItem, int) {
	var cs *model.CompiledSector = nil
	first := false
	outIdx := 0

	for s := 0; s < len(sector.Segments); s++ {
		segment := sector.Segments[s]
		neighbor := sector.Segments[s].Sector
		if neighbor == sector {
			continue
		}

		if segment.Kind == model.DefinitionVoid {
			if neighbor != nil {
				outIdx = r.sectorQueue.UpdateItem(neighbor, outIdx, qi)
			}
			continue
		}

		// Rotate around the player's view
		vx1, vy1, tx1, tz1 := vi.TranslateXY(segment.Start.X, segment.Start.Y)
		vx2, vy2, tx2, tz2 := vi.TranslateXY(segment.End.X, segment.End.Y)

		// If the entire segment is behind the camera, discard it immediately
		if tz1 <= 0 && tz2 <= 0 {
			// Edge-case: if we are EXACTLY on the portal line,
			// float imprecision could give Z <= 0.
			if neighbor != nil && tz1 >= -0.1 && tz2 >= -0.1 {
				outIdx = r.sectorQueue.UpdateItem(neighbor, outIdx, qi)
			}
			continue
		}

		// Calculate real length for texture repetition (world UV mapping)
		// Multiply by TextureScaleFactor if your WAD coordinates have been scaled (e.g. * 100.0)
		u0 := 0.0
		u1 := math.Hypot(vx2-vx1, vy2-vy1) * 100.0 // Adjust multiplier

		// EXACT LINEAR CLIPPING AGAINST THE NEAR-Z PLANE
		if tz1 <= model.NearZ || tz2 <= model.NearZ {
			if tz1 <= model.NearZ && tz2 <= model.NearZ {
				continue // Nessun attraversamento, il portale è alle spalle.
			}

			/*
				if tz1 <= model.NearZ && tz2 <= model.NearZ {
					// The segment is too close to the camera to generate geometry,
					// but if it's a portal, we are physically crossing the threshold.
					// We must necessarily pass visibility to the adjacent sector
					if neighbor != nil {
						outIdx = r.sectorQueue.UpdateItem(neighbor, outIdx, qi)
					}
					continue
				}
			*/

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

		sectorYCeil := vi.ZDistance(sector.CeilY)
		sectorYFloor := vi.ZDistance(sector.FloorY)

		x1Max := mathematic.MaxF(x1, qi.x1)
		x2Min := mathematic.MinF(x2, qi.x2)

		y1a := r.screenHeightHalf + (-vi.ComputeYaw(sectorYCeil, tz1) * yScale1)
		y2a := r.screenHeightHalf + (-vi.ComputeYaw(sectorYCeil, tz2) * yScale2)
		y1b := r.screenHeightHalf + (-vi.ComputeYaw(sectorYFloor, tz1) * yScale1)
		y2b := r.screenHeightHalf + (-vi.ComputeYaw(sectorYFloor, tz2) * yScale2)

		yaStart := (x1Max-x1)*(y2a-y1a)/(x2-x1) + y1a
		yaStop := (x2Min-x1)*(y2a-y1a)/(x2-x1) + y1a
		ybStart := (x1Max-x1)*(y2b-y1b)/(x2-x1) + y1b
		ybStop := (x2Min-x1)*(y2b-y1b)/(x2-x1) + y1b

		zStart := ((x1Max-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8
		if zStart <= 0 {
			zStart = 10e4
		}
		zStop := ((x2Min-x1)*(tz2-tz1)/(x2-x1) + tz1) * 8
		if zStop <= 0 {
			zStop = 10e4
		}

		lightStart := vi.GetLightIntensityFactor(zStart)
		lightStop := vi.GetLightIntensityFactor(zStop)

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

		ceilT := cs.Sector.Animations.Ceil()
		floorT := cs.Sector.Animations.Floor()

		ceilP := cs.Acquire(neighbor, model.IdCeil, ceilT, floorT, ceilT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		ceilP.Rect(x1Max, y1Ceil, yaStart, zStart, lightStart, x2Min, y2Ceil, yaStop, zStop, lightStop)

		floorP := cs.Acquire(neighbor, model.IdFloor, ceilT, floorT, floorT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		floorP.Rect(x1Max, ybStart, y1Floor, zStart, lightStart, x2Min, ybStop, y2Floor, zStop, lightStop)

		if neighbor != nil {
			neighborYCeil := vi.ZDistance(neighbor.CeilY)
			ny1a := r.screenHeightHalf + (-vi.ComputeYaw(neighborYCeil, tz1) * yScale1)
			ny2a := r.screenHeightHalf + (-vi.ComputeYaw(neighborYCeil, tz2) * yScale2)
			nYaStart := (x1Max-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			nYaStop := (x2Min-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			if yaStart-yaStop != 0 || nYaStop-nYaStop != 0 {
				upperT := segment.Animations.Upper()
				upperP := cs.Acquire(neighbor, model.IdUpper, ceilT, floorT, upperT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
				upperP.Rect(x1Max, yaStart, nYaStart, zStart, lightStart, x2Min, yaStop, nYaStop, zStop, lightStop)
			}
			y1Ceil = mathematic.MaxF(yaStart, nYaStart)
			y2Ceil = mathematic.MaxF(yaStop, nYaStop)

			neighborYFloor := vi.ZDistance(neighbor.FloorY)
			ny1b := r.screenHeightHalf + (-vi.ComputeYaw(neighborYFloor, tz1) * yScale1)
			ny2b := r.screenHeightHalf + (-vi.ComputeYaw(neighborYFloor, tz2) * yScale2)
			nYbStart := (x1Max-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			nYbStop := (x2Min-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			if (ybStart-nYbStart) != 0 || (nYbStop-ybStop) != 0 {
				lowerT := segment.Animations.Lower()
				lowerP := cs.Acquire(neighbor, model.IdLower, ceilT, floorT, lowerT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
				lowerP.Rect(x1Max, nYbStart, ybStart, zStart, lightStart, x2Min, nYbStop, ybStop, zStop, lightStop)
			}
			y1Floor = mathematic.MinF(nYbStart, ybStart)
			y2Floor = mathematic.MinF(nYbStop, ybStop)

			outIdx = r.sectorQueue.Update(neighbor, outIdx, x1Max, x2Min, y1Ceil, y2Ceil, y1Floor, y2Floor)
		} else {
			middleT := segment.Animations.Middle()
			wallP := cs.Acquire(neighbor, model.IdWall, ceilT, floorT, middleT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
			wallP.Rect(x1Max, yaStart, ybStart, zStart, lightStart, x2Min, yaStop, ybStop, zStop, lightStop)
		}
	}

	if first && outIdx == 0 {
		for _, s := range sector.Segments {
			if s.Sector != nil && s.Sector != sector {
				outIdx = r.sectorQueue.UpdateItem(s.Sector, outIdx, qi)
			}
		}
	}

	return r.sectorQueue.Items(), outIdx
}
