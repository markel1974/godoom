package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// BucketType represents a categorized type for spatial partitioning or collision detection in 3D environments.
type BucketType int

// BucketWallWest represents the west (-X) wall bucket type.
// BucketWallEast represents the east (+X) wall bucket type.
// BucketWallNorth represents the north (-Y) wall bucket type.
// BucketWallSouth represents the south (+Y) wall bucket type.
// BucketCeiling represents the ceiling (-Z) bucket type.
// BucketFloor represents the floor (+Z) bucket type.
const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
)

// String returns the string representation of the BucketType value.
func (p BucketType) String() string {
	switch p {
	case BucketWallWest:
		return "BucketWallWest"
	case BucketWallEast:
		return "BucketWallEast"
	case BucketWallNorth:
		return "BucketWallNorth"
	case BucketWallSouth:
		return "BucketWallSouth"
	case BucketCeiling:
		return "BucketCeiling"
	case BucketFloor:
		return "BucketFloor"
	default:
		return "BucketUnknown"
	}
}

// IsWall determines if the BucketType value represents a wall (west, east, north, or south).
func (p BucketType) IsWall() bool {
	return p == BucketWallWest || p == BucketWallEast || p == BucketWallNorth || p == BucketWallSouth
}

// Opposite returns the opposite directional BucketType.
func (p BucketType) Opposite() BucketType {
	switch p {
	case BucketWallWest:
		return BucketWallEast
	case BucketWallEast:
		return BucketWallWest
	case BucketWallNorth:
		return BucketWallSouth
	case BucketWallSouth:
		return BucketWallNorth
	case BucketCeiling:
		return BucketFloor
	case BucketFloor:
		return BucketCeiling
	default:
		return p
	}
}

// BucketSize defines the size of a single bucket, calculated as BucketFloor + 1.
const BucketSize = BucketFloor + 1

// FacesPerBucket specifies the maximum number of manifold/face entries stored per bucket.
const FacesPerBucket = 16

// warmStartFactor controls the fraction of previous-frame impulses carried over to persistent contacts.
const warmStartFactor = 0.80

// Bucket holds collision data for a specific bucket, using double-buffered Entry pointer pools for zero-copy warm starting.
type Bucket struct {
	bucket           BucketType
	poolA            [FacesPerBucket]*Entry
	poolB            [FacesPerBucket]*Entry
	currPool         *[FacesPerBucket]*Entry
	prevPool         *[FacesPerBucket]*Entry
	containerCounter int
	prevCount        int
}

// NewBucket creates a new Bucket for a given BucketType, preallocating its double-buffered Entry pools.
func NewBucket(bucket BucketType) *Bucket {
	b := &Bucket{
		bucket: bucket,
	}
	b.currPool = &b.poolA
	b.prevPool = &b.poolB
	for i := 0; i < FacesPerBucket; i++ {
		b.poolA[i] = NewCageEntry()
		b.poolB[i] = NewCageEntry()
	}
	return b
}

// Rebuild swaps current and previous Entry pools in O(1) without copying and resets containerCounter to 0.
func (b *Bucket) Rebuild() {
	b.currPool, b.prevPool = b.prevPool, b.currPool
	b.prevCount = b.containerCounter
	b.containerCounter = 0
}

// Count returns the total number of entries currently stored in the Bucket.
func (b *Bucket) Count() int {
	return b.containerCounter
}

// HasEntryFor reports whether the Bucket currently holds an active dynamic entry for entityId.
func (b *Bucket) HasEntryFor(entityId uint64) bool {
	for i := 0; i < b.containerCounter; i++ {
		if rCage := b.currPool[i].rCage; rCage != nil && rCage.object.GetEntity().GetId() == entityId {
			return true
		}
	}
	return false
}

// Penetrable determines if the Bucket can be penetrated starting from a specified entity ID.
// Returns false if any entity in the bucket provides infinite inelastic resistance or cannot be recursively pushed.
func (b *Bucket) Penetrable(from uint64) bool {
	return b.penetrable(from, 0)
}

func (b *Bucket) penetrable(from uint64, depth int) bool {
	if depth >= maxPushDepth {
		return true
	}
	// Scorriamo TUTTE le entità in questo bucket (multi-pushing)
	for i := 0; i < b.containerCounter; i++ {
		entry := b.currPool[i]
		// Se tocchiamo direttamente un muro, il bucket è bloccato
		if entry.iMode == ImpactInelastic {
			return false
		}
		if entry.rCage != nil && entry.rCage.object.GetEntity().GetId() == from {
			continue
		}
		// Altrimenti, verifichiamo se l'entità dinamica che stiamo toccando
		// può essere spinta a sua volta (Attraversamento Ricorsivo del Grafo).
		// Se anche UNA SOLA entità è bloccata, tutto il nostro fronte di spinta è bloccato!
		if !entry.penetrable(depth) {
			return false
		}
	}
	// Se nessuna entità oppone resistenza anelastica infinita, il bucket cede.
	return true
}

// Add inserts or updates a collision entry in the bucket based on specific conditions, ensuring deduplication and prioritization.
func (b *Bucket) Add(bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) *Entry {
	var coplanarIdx [aabb.MaxManifoldPoints]int
	coplanarCount := 0
	var mergeDistSq float64
	if rCage == nil {
		minExt := math.Min(lCage.eRadX, math.Min(lCage.eRadY, lCage.eRadZ))
		mergeDist := math.Min(0.5, math.Max(planeEpsilon, minExt*0.1))
		mergeDistSq = mergeDist * mergeDist
	}

	for i := 0; i < b.containerCounter; i++ {
		existing := b.currPool[i]
		// Deduplicate coincident dynamic contact points for the same body, preserving up to 4 manifold corners.
		if rCage != nil && existing.rCage != nil {
			if rCage.object.GetEntity().GetId() == existing.rCage.object.GetEntity().GetId() {
				dx := p0x - existing.p0X
				dy := p0y - existing.p0Y
				dz := p0z - existing.p0Z
				if dx*dx+dy*dy+dz*dz < planeEpsilon*planeEpsilon {
					if penetration > existing.penetration {
						b.rebuildKeepingWarm(existing, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
					}
					return nil
				}
			}
		}
		// Coplanar static manifold deduplication: preserve up to MaxManifoldPoints distinct support points per static plane.
		if rCage == nil && existing.rCage == nil {
			if dot := (normalX * existing.nX) + (normalY * existing.nY) + (normalZ * existing.nZ); dot > 0.999 {
				planeDistance := math.Abs(
					(p0x-existing.p0X)*existing.nX +
						(p0y-existing.p0Y)*existing.nY +
						(p0z-existing.p0Z)*existing.nZ,
				)
				if planeDistance < planeEpsilon {
					// Solid walls/floors take single-entry priority over step-ups; step-ups stay single-entry per plane.
					if existing.iMode == ImpactStep && iMode == ImpactInelastic {
						b.rebuildKeepingWarm(existing, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
						return nil
					}
					if existing.iMode == ImpactInelastic && iMode == ImpactStep {
						return nil
					}
					if iMode == ImpactStep {
						if penetration > existing.penetration || (math.Abs(penetration-existing.penetration) <= planeEpsilon && maxZ > existing.maxZ) {
							b.rebuildKeepingWarm(existing, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
						}
						return nil
					}
					dx := p0x - existing.p0X
					dy := p0y - existing.p0Y
					dz := p0z - existing.p0Z
					if dx*dx+dy*dy+dz*dz < mergeDistSq {
						if penetration > existing.penetration {
							b.rebuildKeepingWarm(existing, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
						}
						return nil
					}
					if coplanarCount < len(coplanarIdx) {
						coplanarIdx[coplanarCount] = i
						coplanarCount++
					}
				}
			}
		}
	}

	if rCage == nil && coplanarCount >= aabb.MaxManifoldPoints {
		b.replaceCoplanarManifold(&coplanarIdx, coplanarCount, bucket, lCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		return nil
	}

	// Insert a new plane into the non-full bucket
	if b.containerCounter < FacesPerBucket {
		target := b.currPool[b.containerCounter]
		target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.matchWarmStart(target)
		b.containerCounter++
		return target
	}

	// Replace the least relevant face (respecting impactModePriority first, then penetration depth)
	minIdx := 0
	minPrio := b.currPool[0].iMode.Priority()
	minPen := b.currPool[0].penetration
	for i := 1; i < FacesPerBucket; i++ {
		cPrio := b.currPool[i].iMode.Priority()
		cPen := b.currPool[i].penetration
		if cPrio < minPrio || (cPrio == minPrio && cPen < minPen) {
			minPrio = cPrio
			minPen = cPen
			minIdx = i
		}
	}
	inPrio := iMode.Priority()
	if inPrio > minPrio || (inPrio == minPrio && penetration > minPen) {
		target := b.currPool[minIdx]
		target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.matchWarmStart(target)
	}
	return nil
}

func (b *Bucket) replaceCoplanarManifold(
	coplanarIdx *[aabb.MaxManifoldPoints]int,
	coplanarCount int,
	bucket BucketType,
	lCage *Cage,
	rFace IFace,
	dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ float64,
	iMode ImpactMode,
) {
	minPenIdx := coplanarIdx[0]
	minPen := b.currPool[minPenIdx].penetration
	for k := 1; k < coplanarCount; k++ {
		idx := coplanarIdx[k]
		if p := b.currPool[idx].penetration; p < minPen {
			minPen = p
			minPenIdx = idx
		}
	}
	if penetration > minPen+1e-4 {
		b.rebuildKeepingWarm(b.currPool[minPenIdx], bucket, lCage, nil, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		return
	}

	worstSlot := -1
	worstMinDistSq := math.MaxFloat64
	for s := 0; s < coplanarCount; s++ {
		es := b.currPool[coplanarIdx[s]]
		if es.penetration > penetration+1e-4 {
			continue
		}
		minDSq := math.MaxFloat64
		for k := 0; k < coplanarCount; k++ {
			if k == s {
				continue
			}
			ek := b.currPool[coplanarIdx[k]]
			dx := es.p0X - ek.p0X
			dy := es.p0Y - ek.p0Y
			dz := es.p0Z - ek.p0Z
			if dSq := dx*dx + dy*dy + dz*dz; dSq < minDSq {
				minDSq = dSq
			}
		}
		if minDSq < worstMinDistSq {
			worstMinDistSq = minDSq
			worstSlot = s
		}
	}
	if worstSlot < 0 {
		return
	}

	candMinDistSq := math.MaxFloat64
	for k := 0; k < coplanarCount; k++ {
		if k == worstSlot {
			continue
		}
		ek := b.currPool[coplanarIdx[k]]
		dx := p0x - ek.p0X
		dy := p0y - ek.p0Y
		dz := p0z - ek.p0Z
		if dSq := dx*dx + dy*dy + dz*dz; dSq < candMinDistSq {
			candMinDistSq = dSq
		}
	}
	if candMinDistSq > worstMinDistSq*(1.0+1e-3) {
		b.rebuildKeepingWarm(b.currPool[coplanarIdx[worstSlot]], bucket, lCage, nil, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	}
}

func (b *Bucket) rebuildKeepingWarm(entry *Entry, bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ float64, iMode ImpactMode) {
	wN, wTx, wTy, wTz := entry.GetWarmImpulse()
	entry.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	if iMode == ImpactStep {
		return
	}
	if wN > 0.0 {
		entry.SetWarmImpulse(wN, wTx, wTy, wTz)
		return
	}
	b.matchWarmStart(entry)
}

func (b *Bucket) matchWarmStart(entry *Entry) {
	if b.prevCount == 0 || entry.iMode == ImpactStep {
		return
	}
	var targetId uint64
	if entry.rCage != nil {
		targetId = entry.rCage.object.GetEntity().GetId()
	}

	lCage := entry.lCage
	warmRadius := math.Max(0.5, math.Min(lCage.eRadX, math.Min(lCage.eRadY, lCage.eRadZ))*0.25)
	bestDistSq := warmRadius * warmRadius
	var bestPrev *Entry

	for i := 0; i < b.prevCount; i++ {
		w := b.prevPool[i]
		if w.warmConsumed || w.accNormal <= 0.0 {
			continue
		}
		var wId uint64
		if w.rCage != nil {
			wId = w.rCage.object.GetEntity().GetId()
		}
		if wId != targetId {
			continue
		}
		nDot := entry.nX*w.nX + entry.nY*w.nY + entry.nZ*w.nZ
		if nDot <= 0.90 {
			continue
		}
		if targetId == 0 && w.rFace != entry.rFace && nDot <= 0.99 {
			continue
		}
		dx := entry.p0X - w.p0X
		dy := entry.p0Y - w.p0Y
		dz := entry.p0Z - w.p0Z
		if distSq := dx*dx + dy*dy + dz*dz; distSq < bestDistSq {
			bestDistSq = distSq
			bestPrev = w
		}
	}
	if bestPrev != nil {
		bestPrev.warmConsumed = true
		entry.SetWarmImpulse(
			bestPrev.accNormal*warmStartFactor,
			bestPrev.accTx*warmStartFactor,
			bestPrev.accTy*warmStartFactor,
			bestPrev.accTz*warmStartFactor,
		)
	}
}
