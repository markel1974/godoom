package streaming

import (
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// WorldStreamer coordinates the dynamic loading, activation, proxying, and unloading of open-world sectors
// inside an aabb.Tree broadphase structure based on viewer / vehicle coordinates.
type WorldStreamer struct {
	gridSize      float64
	loadRadius    int // Grid cell radius where full detailed geometry + physics are active
	unloadRadius  int // Grid cell radius with hysteresis where sectors are evicted from the active tree
	distantRadius int // Grid cell radius for low-poly skyline / proxy geometry rendering

	tree *aabb.Tree // The active AABB tree queried by physics collision cages and render frustums

	sectors        map[SectorCoord]*Sector
	activeSectors  map[SectorCoord]*Sector
	distantSectors map[SectorCoord]*Sector

	currentCenter SectorCoord
	hasCenter     bool

	maxOpsPerTick int // Maximum number of sector activations/evictions per Update tick (0 = unconstrained)

	// Pending queues for smooth time-sliced loading without frame drops
	pendingActivate []SectorCoord
	pendingEvict    []SectorCoord
}

// NewWorldStreamer creates an open-world spatial streamer managing sectors in an AABB tree.
// gridSize: side length of each square city block sector in world units (e.g. 256.0).
// loadRadius: active simulation and collision radius in cell units (e.g. 2 for 5x5 active cells).
// unloadRadius: eviction distance with hysteresis (e.g. 3 cells, preventing boundary thrashing).
// distantRadius: maximum horizon for low-poly proxy skyline visibility (e.g. 5 cells).
func NewWorldStreamer(tree *aabb.Tree, gridSize float64, loadRadius, unloadRadius, distantRadius int) *WorldStreamer {
	if gridSize <= 0.0 {
		gridSize = 256.0
	}
	if loadRadius <= 0 {
		loadRadius = 2
	}
	if unloadRadius <= loadRadius {
		unloadRadius = loadRadius + 1
	}
	if distantRadius < unloadRadius {
		distantRadius = unloadRadius + 2
	}

	return &WorldStreamer{
		gridSize:        gridSize,
		loadRadius:      loadRadius,
		unloadRadius:    unloadRadius,
		distantRadius:   distantRadius,
		tree:            tree,
		sectors:         make(map[SectorCoord]*Sector),
		activeSectors:   make(map[SectorCoord]*Sector),
		distantSectors:  make(map[SectorCoord]*Sector),
		maxOpsPerTick:   4,
		pendingActivate: make([]SectorCoord, 0, 16),
		pendingEvict:    make([]SectorCoord, 0, 16),
	}
}

// SetMaxOpsPerTick configures the maximum number of sector transitions per tick to avoid CPU time spikes.
func (ws *WorldStreamer) SetMaxOpsPerTick(ops int) {
	ws.maxOpsPerTick = ops
}

// RegisterSector adds a sector definition to the world streamer.
func (ws *WorldStreamer) RegisterSector(s *Sector) {
	if s == nil {
		return
	}
	ws.sectors[s.Coord] = s
}

// GetSector retrieves a sector by its grid coordinate, or nil if not registered.
func (ws *WorldStreamer) GetSector(coord SectorCoord) *Sector {
	return ws.sectors[coord]
}

// GetActiveSectorCount returns the number of sectors currently live in the active AABB tree.
func (ws *WorldStreamer) GetActiveSectorCount() int {
	return len(ws.activeSectors)
}

// GetDistantSectorCount returns the number of sectors currently showing low-poly distant proxies.
func (ws *WorldStreamer) GetDistantSectorCount() int {
	return len(ws.distantSectors)
}

// GetTree returns the underlying active AABB tree.
func (ws *WorldStreamer) GetTree() *aabb.Tree {
	return ws.tree
}

// Update evaluates viewer coordinates, enqueuing sectors for activation or eviction based on distance rings.
func (ws *WorldStreamer) Update(viewerX, viewerY, viewerZ float64) {
	newCenter := WorldToSector(viewerX, viewerY, ws.gridSize)

	if !ws.hasCenter || newCenter != ws.currentCenter {
		ws.currentCenter = newCenter
		ws.hasCenter = true
		ws.replanStreaming(newCenter)
	}

	ws.processPendingQueues()
}

// ForceSync immediately activates and evicts all required sectors synchronously without time-slicing.
func (ws *WorldStreamer) ForceSync(viewerX, viewerY, viewerZ float64) {
	newCenter := WorldToSector(viewerX, viewerY, ws.gridSize)
	ws.currentCenter = newCenter
	ws.hasCenter = true
	ws.replanStreaming(newCenter)

	// Process all pending without budget limits
	for len(ws.pendingEvict) > 0 {
		coord := ws.pendingEvict[0]
		ws.pendingEvict = ws.pendingEvict[1:]
		ws.executeEviction(coord)
	}
	for len(ws.pendingActivate) > 0 {
		coord := ws.pendingActivate[0]
		ws.pendingActivate = ws.pendingActivate[1:]
		ws.executeActivation(coord)
	}
}

// replanStreaming determines which sectors should be loaded, proxied, or evicted relative to center.
func (ws *WorldStreamer) replanStreaming(center SectorCoord) {
	ws.pendingActivate = ws.pendingActivate[:0]
	ws.pendingEvict = ws.pendingEvict[:0]

	// 1. Identify sectors that need to be evicted (distance > unloadRadius)
	for coord, s := range ws.activeSectors {
		dist := coord.ChebyshevDistance(center)
		if dist > ws.unloadRadius {
			ws.pendingEvict = append(ws.pendingEvict, coord)
		} else if dist > ws.loadRadius && s.ProxyVolume != nil {
			// Transition active to distant proxy
			ws.pendingEvict = append(ws.pendingEvict, coord)
		}
	}

	for coord := range ws.distantSectors {
		dist := coord.ChebyshevDistance(center)
		if dist > ws.distantRadius || dist <= ws.loadRadius {
			ws.pendingEvict = append(ws.pendingEvict, coord)
		}
	}

	// 2. Identify sectors that need to be activated (distance <= loadRadius)
	for dy := -ws.loadRadius; dy <= ws.loadRadius; dy++ {
		for dx := -ws.loadRadius; dx <= ws.loadRadius; dx++ {
			targetCoord := SectorCoord{X: center.X + dx, Y: center.Y + dy}
			if _, isActive := ws.activeSectors[targetCoord]; !isActive {
				if s := ws.sectors[targetCoord]; s != nil {
					ws.pendingActivate = append(ws.pendingActivate, targetCoord)
				}
			}
		}
	}

	// 3. Identify distant proxy sectors
	for dy := -ws.distantRadius; dy <= ws.distantRadius; dy++ {
		for dx := -ws.distantRadius; dx <= ws.distantRadius; dx++ {
			targetCoord := SectorCoord{X: center.X + dx, Y: center.Y + dy}
			dist := targetCoord.ChebyshevDistance(center)
			if dist > ws.loadRadius && dist <= ws.distantRadius {
				if _, isDistant := ws.distantSectors[targetCoord]; !isDistant {
					if s := ws.sectors[targetCoord]; s != nil && s.ProxyVolume != nil {
						ws.pendingActivate = append(ws.pendingActivate, targetCoord)
					}
				}
			}
		}
	}
}

// processPendingQueues applies up to maxOpsPerTick transitions per frame.
func (ws *WorldStreamer) processPendingQueues() {
	ops := 0

	// Always prioritize evictions to free up spatial tree density first
	for len(ws.pendingEvict) > 0 && (ws.maxOpsPerTick <= 0 || ops < ws.maxOpsPerTick) {
		coord := ws.pendingEvict[0]
		ws.pendingEvict = ws.pendingEvict[1:]
		ws.executeEviction(coord)
		ops++
	}

	for len(ws.pendingActivate) > 0 && (ws.maxOpsPerTick <= 0 || ops < ws.maxOpsPerTick) {
		coord := ws.pendingActivate[0]
		ws.pendingActivate = ws.pendingActivate[1:]
		ws.executeActivation(coord)
		ops++
	}
}

// executeActivation inserts the sector's volumes or proxy into the active AABB tree.
func (ws *WorldStreamer) executeActivation(coord SectorCoord) {
	s := ws.sectors[coord]
	if s == nil {
		return
	}

	dist := coord.ChebyshevDistance(ws.currentCenter)
	if dist <= ws.loadRadius {
		// Activate full detailed geometry + physics
		if s.State == SectorStateDistant && s.ProxyVolume != nil {
			ws.tree.RemoveObject(s.ProxyVolume)
			delete(ws.distantSectors, coord)
		}

		if s.State != SectorStateActive {
			for i := 0; i < s.volumesLen; i++ {
				ws.tree.InsertObject(s.Volumes[i])
			}
			s.State = SectorStateActive
			ws.activeSectors[coord] = s
		}
	} else if dist <= ws.distantRadius && s.ProxyVolume != nil {
		// Activate distant skyline proxy
		if s.State == SectorStateActive {
			for i := 0; i < s.volumesLen; i++ {
				ws.tree.RemoveObject(s.Volumes[i])
			}
			delete(ws.activeSectors, coord)
		}

		if s.State != SectorStateDistant {
			ws.tree.InsertObject(s.ProxyVolume)
			s.State = SectorStateDistant
			ws.distantSectors[coord] = s
		}
	}
}

// executeEviction removes a sector from the active AABB tree.
func (ws *WorldStreamer) executeEviction(coord SectorCoord) {
	s := ws.sectors[coord]
	if s == nil {
		return
	}

	if s.State == SectorStateActive {
		for i := 0; i < s.volumesLen; i++ {
			ws.tree.RemoveObject(s.Volumes[i])
		}
		delete(ws.activeSectors, coord)
	} else if s.State == SectorStateDistant && s.ProxyVolume != nil {
		ws.tree.RemoveObject(s.ProxyVolume)
		delete(ws.distantSectors, coord)
	}
	s.State = SectorStateUnloaded
}

// Raycast tests a 3D ray against all detailed volumes currently active in the streamed world.
// It directly fulfills the RaycastFn contract required by the vehicle simulation.
func (ws *WorldStreamer) Raycast(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist float64) (hitDist float64, nx, ny, nz float64, hit bool) {
	closest := maxDist + 1.0

	ws.tree.QueryRay(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist, func(obj aabb.IObject, d float64) (float64, bool) {
		vol, ok := obj.(*volumes.Volume)
		if !ok {
			return closest, false
		}

		facesTree := vol.GetFacesTree()
		if facesTree == nil {
			return closest, false
		}

		facesTree.QueryRay(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist, func(face aabb.IObject, fDist float64) (float64, bool) {
			fc, okFc := face.(*volumes.Face)
			if !okFc || !fc.IsSolid() {
				return closest, false
			}
			tHit, okHit := fc.IntersectRay(fromX, fromY, fromZ, dirX, dirY, dirZ, closest)
			if okHit && tHit < closest {
				closest = tHit
				fnx, fny, fnz := fc.GetNormal()
				nx, ny, nz = fnx, fny, fnz
				hit = true
				return closest, true
			}
			return closest, false
		})

		return closest, hit
	})

	if hit && closest <= maxDist {
		return closest, nx, ny, nz, true
	}
	return 0, 0, 0, 1, false
}
