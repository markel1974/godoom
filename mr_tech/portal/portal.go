package portal

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
)

// defaultQueueLen defines the initial size of the queue used for processing rendering or computational tasks.
const (
	defaultQueueLen = 512
)

// Portal represents a rendering portal used for managing visibility, sectors, and screen dimensions in a 3D environment.
type Portal struct {
	maxSectors             int
	queue                  *RingQueue
	sectorQueue            *LinearBatch
	compileId              uint64
	sectorsMaxHeight       float64
	sectors                []*model.Sector
	compiledSectors        []*model.CompiledSector
	compiledCount          int
	visibilityCache        *VisibilityCache
	viewFactor             float64
	textureScaleRepetition float64
}

// NewPortal creates and initializes a new Portal instance with the specified screen dimensions and queue length.
// If maxQueue is 0, it defaults to a predefined constant value.
func NewPortal(maxQueue int, viewFactor float64) *Portal {
	if maxQueue <= 0 {
		maxQueue = defaultQueueLen
	}
	if viewFactor <= 1.0 {
		viewFactor = 1.0
	}
	r := &Portal{
		viewFactor:             viewFactor,
		queue:                  NewRingQueue(maxQueue),
		sectorQueue:            NewLinearBatch(256),
		visibilityCache:        NewVisibilityCache(),
		textureScaleRepetition: 100.0,
	}
	return r
}

func (r *Portal) Grow() {
	oldSize := len(r.compiledSectors)
	newSize := len(r.sectors) * 2
	if oldSize == 0 {
		r.compiledSectors = make([]*model.CompiledSector, newSize)
	} else {
		newSize = oldSize * 2
		newData := make([]*model.CompiledSector, newSize)
		copy(newData, r.compiledSectors)
		r.compiledSectors = newData
	}
	for cs := oldSize; cs < newSize; cs++ {
		r.compiledSectors[cs] = model.NewCompiledSector()
		r.compiledSectors[cs].Setup()
	}
}

// Setup configures the Portal by assigning sectors, setting the maximum height, and initializing compiled sectors.
func (r *Portal) Setup(sectors []*model.Sector) error {
	r.sectors = sectors
	r.Grow()
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

// clear resets the Portal's state by incrementing the compile ID, setting compiled sector count to 0, and clearing the visibility cache.
func (r *Portal) clear() {
	r.compileId++
	r.compiledCount = 0
	r.visibilityCache.Clear()
}

// Build compiles all sectors within the Portal, updates the compiled sector list, and returns the results.
func (r *Portal) Build() ([]*model.CompiledSector, int) {
	r.clear()
	for _, sector := range r.sectors {
		if cs, _ := r.getCompiledSector(sector); cs != nil {
			r.compile(sector, cs)
		}
	}
	return r.compiledSectors, r.compiledCount
}

// compile processes a given sector, generating compiled geometry for rendering, including walls, floors, and ceilings.
func (r *Portal) compile(sector *model.Sector, cs *model.CompiledSector) {
	for s := 0; s < len(sector.Segments); s++ {
		segment := sector.Segments[s]
		neighbor := segment.Neighbor
		//if segment.Kind == config.DefinitionVoid {
		//	continue
		//}
		// Coordinate World assolute sul piano XZ (niente proiezioni fotocamera!)
		wx1, wz1 := segment.Start.X, segment.Start.Y
		wx2, wz2 := segment.End.X, segment.End.Y

		// UV mapping basato sulla lunghezza reale del muro
		u0 := 0.0
		u1 := math.Hypot(wx2-wx1, wz2-wz1) * r.textureScaleRepetition

		ceilT := cs.Sector.Ceil
		floorT := cs.Sector.Floor
		sectorCeilY := sector.CeilY
		sectorFloorY := sector.FloorY

		// 1. Muri di connessione (Portali verso altri settori)
		if neighbor != nil {
			neighborCeilY := neighbor.CeilY
			neighborFloorY := neighbor.FloorY
			// Upper Wall (dal soffitto corrente scende al soffitto del vicino)
			if sectorCeilY > neighborCeilY {
				upperP := cs.Acquire(neighbor, model.IdUpper, ceilT, floorT, segment.Upper, wx1, wx2, wx1, wx2, wz1, wz2, u0, u1)
				// x1, Y_top, Y_bottom, z1, x2, Y_top, Y_bottom, z2
				upperP.Rect(wx1, sectorCeilY, neighborCeilY, wz1, wx2, sectorCeilY, neighborCeilY, wz2)
			}
			// Lower Wall (dal pavimento del vicino scende al pavimento corrente)
			if sectorFloorY < neighborFloorY {
				lowerP := cs.Acquire(neighbor, model.IdLower, ceilT, floorT, segment.Lower, wx1, wx2, wx1, wx2, wz1, wz2, u0, u1)
				lowerP.Rect(wx1, neighborFloorY, sectorFloorY, wz1, wx2, neighborFloorY, sectorFloorY, wz2)
			}
		} else {
			// 2. Muro Solido (Middle Wall) - connette soffitto e pavimento del settore
			wallP := cs.Acquire(nil, model.IdWall, ceilT, floorT, segment.Middle, wx1, wx2, wx1, wx2, wz1, wz2, u0, u1)
			wallP.Rect(wx1, sectorCeilY, sectorFloorY, wz1, wx2, sectorCeilY, sectorFloorY, wz2)
		}
		center := sector.GetCentroid()

		ceilP := cs.Acquire(neighbor, model.IdCeil, ceilT, floorT, ceilT, wx1, wx2, wx1, wx2, wz1, wz2, u0, u1)
		// Generi un triangolo orizzontale: Centro, P1, P2
		ceilP.Triangle(center.X, sectorCeilY, center.Y, wx1, sectorCeilY, wz1, wx2, sectorCeilY, wz2)

		floorP := cs.Acquire(neighbor, model.IdFloor, ceilT, floorT, floorT, wx1, wx2, wx1, wx2, wz1, wz2, u0, u1)
		floorP.Triangle(center.X, sectorFloorY, center.Y, wx1, sectorFloorY, wz1, wx2, sectorFloorY, wz2)
	}
}

// Traverse processes the active view, traverses sectors, and generates compiled sectors for rendering optimization.
func (r *Portal) Traverse(fbw, fbh int32, vi *model.ViewMatrix) ([]*model.CompiledSector, int) {
	wMin := float64(-fbw) * r.viewFactor
	wMax := float64(fbw) * r.viewFactor
	hMax := float64(fbh-1) * r.viewFactor

	r.clear()

	r.queue.Reset()

	qHead := r.queue.GetHead()
	qHead.Update(vi.GetSector(), wMin, wMax, -hMax, -hMax, hMax, hMax)

	var qTail *QueueItem

	for !r.queue.IsEmpty() {
		qTail = r.queue.GetTail()
		qTail.sector.Reference(r.compileId)

		sq, sqCount := r.compileProjection(fbw, fbh, vi, qTail.sector, qTail)
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

	return r.compiledSectors, r.compiledCount
}

// compileSector determines visible geometry and propagates visibility to adjacent sectors based on the current view matrix.
func (r *Portal) compileProjection(fbw, fbh int32, vi *model.ViewMatrix, sector *model.Sector, qi *QueueItem) ([]QueueItem, int) {
	screenWidthHalf := fbw / 2
	screenHeightHalf := float64(fbh) / 2
	screenHFov := model.HFov * float64(fbw)
	screenVFov := model.VFov * float64(fbh)

	var cs *model.CompiledSector = nil
	first := false
	outIdx := 0

	for s := 0; s < len(sector.Segments); s++ {
		segment := sector.Segments[s]
		neighbor := sector.Segments[s].Neighbor
		if neighbor == sector {
			continue
		}

		//if segment.Kind == config.DefinitionVoid {
		//	if neighbor != nil {
		//		outIdx = r.sectorQueue.UpdateItem(neighbor, outIdx, qi)
		//	}
		//	continue
		//}

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
		u0 := 0.0
		u1 := math.Hypot(vx2-vx1, vy2-vy1) * r.textureScaleRepetition

		// EXACT LINEAR CLIPPING AGAINST THE NEAR-Z PLANE
		if tz1 <= model.NearZ || tz2 <= model.NearZ {
			if tz1 <= model.NearZ && tz2 <= model.NearZ {
				continue // No traversal, the portal is behind the camera.
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
		xScale1 := screenHFov / tz1
		yScale1 := screenVFov / tz1
		x1 := float64(screenWidthHalf) - (tx1 * xScale1)

		xScale2 := screenHFov / tz2
		yScale2 := screenVFov / tz2
		x2 := float64(screenWidthHalf) - (tx2 * xScale2)

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

		y1a := screenHeightHalf + (-vi.ComputeYaw(sectorYCeil, tz1) * yScale1)
		y2a := screenHeightHalf + (-vi.ComputeYaw(sectorYCeil, tz2) * yScale2)
		y1b := screenHeightHalf + (-vi.ComputeYaw(sectorYFloor, tz1) * yScale1)
		y2b := screenHeightHalf + (-vi.ComputeYaw(sectorYFloor, tz2) * yScale2)

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
			if cs, first = r.getCompiledSector(sector); cs == nil {
				return nil, 0
			}
		}

		ceilT := cs.Sector.Ceil
		floorT := cs.Sector.Floor

		ceilP := cs.Acquire(neighbor, model.IdCeil, ceilT, floorT, ceilT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		ceilP.Rect(x1Max, y1Ceil, yaStart, zStart, x2Min, y2Ceil, yaStop, zStop)

		floorP := cs.Acquire(neighbor, model.IdFloor, ceilT, floorT, floorT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
		floorP.Rect(x1Max, ybStart, y1Floor, zStart, x2Min, ybStop, y2Floor, zStop)

		if neighbor != nil {
			neighborYCeil := vi.ZDistance(neighbor.CeilY)
			ny1a := screenHeightHalf + (-vi.ComputeYaw(neighborYCeil, tz1) * yScale1)
			ny2a := screenHeightHalf + (-vi.ComputeYaw(neighborYCeil, tz2) * yScale2)
			nYaStart := (x1Max-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			nYaStop := (x2Min-x1)*(ny2a-ny1a)/(x2-x1) + ny1a
			if yaStart-yaStop != 0 || nYaStop-nYaStop != 0 {
				upperT := segment.Upper
				upperP := cs.Acquire(neighbor, model.IdUpper, ceilT, floorT, upperT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
				upperP.Rect(x1Max, yaStart, nYaStart, zStart, x2Min, yaStop, nYaStop, zStop)
			}
			y1Ceil = mathematic.MaxF(yaStart, nYaStart)
			y2Ceil = mathematic.MaxF(yaStop, nYaStop)

			neighborYFloor := vi.ZDistance(neighbor.FloorY)
			ny1b := screenHeightHalf + (-vi.ComputeYaw(neighborYFloor, tz1) * yScale1)
			ny2b := screenHeightHalf + (-vi.ComputeYaw(neighborYFloor, tz2) * yScale2)
			nYbStart := (x1Max-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			nYbStop := (x2Min-x1)*(ny2b-ny1b)/(x2-x1) + ny1b
			if (ybStart-nYbStart) != 0 || (nYbStop-ybStop) != 0 {
				lowerT := segment.Lower
				lowerP := cs.Acquire(neighbor, model.IdLower, ceilT, floorT, lowerT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
				lowerP.Rect(x1Max, nYbStart, ybStart, zStart, x2Min, nYbStop, ybStop, zStop)
			}
			y1Floor = mathematic.MinF(nYbStart, ybStart)
			y2Floor = mathematic.MinF(nYbStop, ybStop)

			outIdx = r.sectorQueue.Update(neighbor, outIdx, x1Max, x2Min, y1Ceil, y2Ceil, y1Floor, y2Floor)
		} else {
			middleT := segment.Middle
			wallP := cs.Acquire(neighbor, model.IdWall, ceilT, floorT, middleT, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
			wallP.Rect(x1Max, yaStart, ybStart, zStart, x2Min, yaStop, ybStop, zStop)
		}
	}

	if first && outIdx == 0 {
		for _, s := range sector.Segments {
			if s.Neighbor != nil && s.Neighbor != sector {
				outIdx = r.sectorQueue.UpdateItem(s.Neighbor, outIdx, qi)
			}
		}
	}

	return r.sectorQueue.Items(), outIdx
}

// getCompiledSector retrieves a CompiledSector and binds it to the provided Sector, returning whether it is the first retrieval.
func (r *Portal) getCompiledSector(sector *model.Sector) (*model.CompiledSector, bool) {
	first := r.compiledCount == 0
	if r.compiledCount >= len(r.compiledSectors) {
		r.Grow()
	}
	cs := r.compiledSectors[r.compiledCount]
	r.compiledCount++
	cs.Bind(sector)
	return cs, first
}
