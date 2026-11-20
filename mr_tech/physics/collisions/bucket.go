package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// BucketSize defines the size of a single bucket, calculated as BucketFloor + 1.
const BucketSize = BucketFloor + 1

// FacesPerBucket specifies the maximum number of manifold/face entries stored per bucket.
const FacesPerBucket = 16

// warmStartFactor controls the fraction of previous-frame impulses carried over to persistent contacts.
const warmStartFactor = 0.80

// maxPushQueue caps unique remote cages expanded during a single iterative BFS push-chain check.
const maxPushQueue = 64

// noEntityId marks an absent entity ID in push-chain tracking (valid entity IDs start at 0).
const noEntityId = ^uint64(0)

// Bucket holds collision data for a specific bucket, using double-buffered Entry pointer pools for zero-copy warm starting.
type Bucket struct {
	bucket           BucketType
	currPoll         [FacesPerBucket]*Entry
	prevPool         [FacesPerBucket]*Entry
	currPoolPtr      *[FacesPerBucket]*Entry
	prevPoolPtr      *[FacesPerBucket]*Entry
	containerCounter int
	prevCount        int
	inelasticCount   int8
	dynamicCount     int8
	cache            BucketCache
}

// NewBucket creates a new Bucket for a given BucketType, preallocating its double-buffered Entry pools.
func NewBucket(bucket BucketType) *Bucket {
	b := &Bucket{
		bucket: bucket,
	}
	b.currPoolPtr = &b.currPoll
	b.prevPoolPtr = &b.prevPool
	for i := 0; i < FacesPerBucket; i++ {
		b.currPoll[i] = NewCageEntry()
		b.prevPool[i] = NewCageEntry()
	}
	return b
}

// Rebuild swaps current and previous Entry pools in O(1) without copying and resets containerCounter to 0.
func (b *Bucket) Rebuild() {
	b.currPoolPtr, b.prevPoolPtr = b.prevPoolPtr, b.currPoolPtr
	b.prevCount = b.containerCounter
	b.containerCounter = 0
	b.inelasticCount = 0
	b.dynamicCount = 0
	b.cache.Reset()
}

// Count returns the total number of entries currently stored in the Bucket.
func (b *Bucket) Count() int {
	return b.containerCounter
}

// RemoveEntry swaps target out of the active pool while preserving all preallocated Entry buffers.
func (b *Bucket) RemoveEntry(target *Entry) bool {
	for i := 0; i < b.containerCounter; i++ {
		if b.currPoolPtr[i] == target {
			b.countEntry(target, -1)
			b.cache.ClearPenetrable()
			last := b.containerCounter - 1
			b.currPoolPtr[i], b.currPoolPtr[last] = b.currPoolPtr[last], b.currPoolPtr[i]
			b.containerCounter = last
			return true
		}
	}
	return false
}

// countEntry adjusts the bucket's inelastic or dynamic entry counters by delta based on the entry impact type.
func (b *Bucket) countEntry(e *Entry, delta int8) {
	if e.iMode == ImpactInelastic {
		b.inelasticCount += delta
	} else if e.rCage != nil {
		b.dynamicCount += delta
	} else if e.iMode != ImpactStep {
		b.inelasticCount += delta
	}
}

// rebuildEntry rebuilds an entry, updates bucket counters for its impact type, and clears cached penetrability.
func (b *Bucket) rebuildEntry(target *Entry, bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) {
	b.countEntry(target, -1)
	target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	b.countEntry(target, 1)
	b.cache.ClearPenetrable()
}

// HasEntryFor reports whether the Bucket currently holds an active dynamic entry for entityId.
func (b *Bucket) HasEntryFor(entityId uint64) bool {
	for i := 0; i < b.containerCounter; i++ {
		if rCage := b.currPoolPtr[i].rCage; rCage != nil && rCage.GetEntity().GetId() == entityId {
			return true
		}
	}
	return false
}

// extractDynamicRemoteId returns the remote dynamic entity ID for entry.
// It reports blocked for inelastic impacts or non-step static impacts, and ok when an ID was found.
func extractDynamicRemoteId(entry *Entry) (rId uint64, blocked bool, ok bool) {
	if entry == nil {
		return 0, false, false
	}
	if entry.iMode == ImpactInelastic {
		return 0, true, false
	}
	if entry.rCage == nil {
		return 0, entry.iMode != ImpactStep, false
	}
	rEnt := entry.rCage.GetEntity()
	if rEnt == nil {
		return 0, false, false
	}
	return rEnt.GetId(), false, true
}

// Penetrable reports whether this bucket can yield to a push from the given entity within the recursion depth.
func (b *Bucket) Penetrable(from uint64, depth int) bool {
	if b.inelasticCount > 0 {
		return false
	}
	if b.dynamicCount == 0 || depth >= maxPushDepth {
		return true
	}
	if ok, val := b.cache.Check(from, depth); ok {
		return val
	}
	selfId := noEntityId
	if e := b.currPoolPtr[0]; e != nil && e.lCage != nil {
		if ent := e.lCage.GetEntity(); ent != nil {
			selfId = ent.GetId()
		}
	}
	res := b.doPenetrable(depth, from, selfId)
	b.cache.Store(from, depth, res)
	return res
}

// doPenetrable reports whether this bucket and chained remote buckets can yield within the push depth limit.
// It runs a zero-allocation shallow scan first and falls back to an iterative BFS only for deeper multi-body chains.
func (b *Bucket) doPenetrable(depth int, seen1, seen2 uint64) bool {
	if b.inelasticCount > 0 {
		return false
	}
	if b.dynamicCount == 0 || depth >= maxPushDepth {
		return true
	}

	hasDeep := false
	nextDepth := depth + 1
	checkGrandchildren := nextDepth+1 < maxPushDepth
	prevRId := noEntityId

	for i := 0; i < b.containerCounter; i++ {
		entry := b.currPoolPtr[i]
		rId, blocked, ok := extractDynamicRemoteId(entry)
		if blocked {
			return false
		}
		if !ok || rId == prevRId || rId == seen1 || rId == seen2 {
			continue
		}
		prevRId = rId
		if nextDepth >= maxPushDepth {
			continue
		}
		nextB := entry.rCage.buckets[entry.bucket]
		if nextB == nil {
			continue
		}
		if nextB.inelasticCount > 0 {
			return false
		}
		if checkGrandchildren && nextB.dynamicCount > 0 {
			hasDeep = true
		}
	}
	if !hasDeep {
		return true
	}
	return b.bfsPenetrable(depth, seen1, seen2)
}

// bfsPenetrable reports whether the bucket's dynamic push chain can yield within the configured depth limit.
func (b *Bucket) bfsPenetrable(startDepth int, seen1, seen2 uint64) bool {
	var queue [maxPushQueue]*Bucket
	var depths [maxPushQueue]uint8
	var seen [maxPushQueue]uint64

	seenCount := 0
	if seen1 != noEntityId {
		seen[seenCount] = seen1
		seenCount++
	}
	if seen2 != noEntityId && seen2 != seen1 {
		seen[seenCount] = seen2
		seenCount++
	}

	queue[0] = b
	depths[0] = uint8(startDepth)
	head := 0
	tail := 1

	for head < tail {
		curr := queue[head]
		nextDepth := int(depths[head]) + 1
		head++

		if curr.inelasticCount > 0 {
			return false
		}
		if nextDepth >= maxPushDepth {
			continue
		}

		expandChildren := nextDepth+1 < maxPushDepth
		prevRId := noEntityId

		for i := 0; i < curr.containerCounter; i++ {
			entry := curr.currPoolPtr[i]
			rId, blocked, ok := extractDynamicRemoteId(entry)
			if blocked {
				return false
			}
			if !ok || rId == prevRId {
				continue
			}
			prevRId = rId

			alreadySeen := false
			for s := 0; s < seenCount; s++ {
				if seen[s] == rId {
					alreadySeen = true
					break
				}
			}
			if alreadySeen {
				continue
			}

			nextB := entry.rCage.buckets[entry.bucket]
			if nextB == nil {
				continue
			}
			if nextB.inelasticCount > 0 {
				return false
			}
			if !expandChildren || nextB.dynamicCount == 0 {
				if seenCount < maxPushQueue {
					seen[seenCount] = rId
					seenCount++
				}
				continue
			}
			if seenCount >= maxPushQueue || tail >= maxPushQueue {
				continue
			}
			seen[seenCount] = rId
			seenCount++
			queue[tail] = nextB
			depths[tail] = uint8(nextDepth)
			tail++
		}
	}
	return true
}

// Add inserts or updates a collision entry in the bucket based on specific conditions, ensuring deduplication and prioritization.
func (b *Bucket) Add(bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) *Entry {
	var coplanar [aabb.MaxManifoldPoints]int
	coplanarCount := 0
	var mergeDistSq float64
	if rCage == nil {
		minExt := math.Min(lCage.eRadX, math.Min(lCage.eRadY, lCage.eRadZ))
		mergeDist := math.Min(0.5, math.Max(planeEpsilon, minExt*0.1))
		mergeDistSq = mergeDist * mergeDist
	}

	for i := 0; i < b.containerCounter; i++ {
		existing := b.currPoolPtr[i]
		// Deduplicate coincident dynamic contact points for the same body, preserving up to 4 manifold corners.
		if rCage != nil && existing.rCage != nil {
			if rCage.GetEntity().GetId() == existing.rCage.GetEntity().GetId() {
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
				planeDistance := math.Abs((p0x-existing.p0X)*existing.nX + (p0y-existing.p0Y)*existing.nY + (p0z-existing.p0Z)*existing.nZ)
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
					if coplanarCount < len(coplanar) {
						coplanar[coplanarCount] = i
						coplanarCount++
					}
				}
			}
		}
	}

	if rCage == nil && coplanarCount >= aabb.MaxManifoldPoints {
		b.replaceCoplanarManifold(&coplanar, coplanarCount, bucket, lCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		return nil
	}

	// Insert a new plane into the non-full bucket
	if b.containerCounter < FacesPerBucket {
		target := b.currPoolPtr[b.containerCounter]
		target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.countEntry(target, 1)
		b.cache.ClearPenetrable()
		b.matchWarmStart(target)
		b.containerCounter++
		return target
	}

	// Replace the least relevant face (respecting impactModePriority first, then penetration depth)
	minIdx := 0
	minPrio := b.currPoolPtr[0].iMode.Priority()
	minPen := b.currPoolPtr[0].penetration
	for i := 1; i < FacesPerBucket; i++ {
		cPrio := b.currPoolPtr[i].iMode.Priority()
		cPen := b.currPoolPtr[i].penetration
		if cPrio < minPrio || (cPrio == minPrio && cPen < minPen) {
			minPrio = cPrio
			minPen = cPen
			minIdx = i
		}
	}
	inPrio := iMode.Priority()
	if inPrio > minPrio || (inPrio == minPrio && penetration > minPen) {
		target := b.currPoolPtr[minIdx]
		b.rebuildEntry(target, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.matchWarmStart(target)
	}
	return nil
}

// replaceCoplanarManifold replaces a coplanar contact when the candidate improves penetration or manifold spread.
func (b *Bucket) replaceCoplanarManifold(coplanar *[aabb.MaxManifoldPoints]int, coplanarCount int, bucket BucketType, lCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ float64, iMode ImpactMode) {
	minPenIdx := coplanar[0]
	minPen := b.currPoolPtr[minPenIdx].penetration
	for k := 1; k < coplanarCount; k++ {
		idx := coplanar[k]
		if p := b.currPoolPtr[idx].penetration; p < minPen {
			minPen = p
			minPenIdx = idx
		}
	}
	if penetration > minPen+1e-4 {
		target := b.currPoolPtr[minPenIdx]
		b.rebuildEntry(target, bucket, lCage, nil, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.matchWarmStart(target)
		return
	}

	worstSlot := -1
	worstMinDistSq := math.MaxFloat64
	for s := 0; s < coplanarCount; s++ {
		es := b.currPoolPtr[coplanar[s]]
		if es.penetration > penetration+1e-4 {
			continue
		}
		minDSq := math.MaxFloat64
		for k := 0; k < coplanarCount; k++ {
			if k == s {
				continue
			}
			ek := b.currPoolPtr[coplanar[k]]
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
		ek := b.currPoolPtr[coplanar[k]]
		dx := p0x - ek.p0X
		dy := p0y - ek.p0Y
		dz := p0z - ek.p0Z
		if dSq := dx*dx + dy*dy + dz*dz; dSq < candMinDistSq {
			candMinDistSq = dSq
		}
	}
	if candMinDistSq > worstMinDistSq*(1.0+1e-3) {
		target := b.currPoolPtr[coplanar[worstSlot]]
		b.rebuildEntry(target, bucket, lCage, nil, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.matchWarmStart(target)
	}
}

// rebuildKeepingWarm rebuilds entry and preserves or restores warm-start impulses when applicable.
func (b *Bucket) rebuildKeepingWarm(entry *Entry, bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ float64, iMode ImpactMode) {
	wN, wTx, wTy, wTz := entry.GetWarmImpulse()
	wasWarmEligible := entry.warmEligible
	b.rebuildEntry(entry, bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	if iMode == ImpactStep {
		return
	}
	if wN > 0.0 {
		entry.warmEligible = wasWarmEligible
		entry.SetWarmImpulse(wN, wTx, wTy, wTz)
		return
	}
	b.matchWarmStart(entry)
}

// matchWarmStart restores scaled warm impulses from the best matching previous entry, if one is available.
func (b *Bucket) matchWarmStart(entry *Entry) {
	if b.prevCount == 0 || entry.iMode == ImpactStep {
		return
	}
	var targetId uint64
	if entry.rCage != nil {
		targetId = entry.rCage.GetEntity().GetId()
	}

	lCage := entry.lCage
	warmRadius := math.Max(0.5, math.Min(lCage.eRadX, math.Min(lCage.eRadY, lCage.eRadZ))*0.25)
	bestDistSq := warmRadius * warmRadius
	var bestPrev *Entry

	for i := 0; i < b.prevCount; i++ {
		w := b.prevPoolPtr[i]
		if w.warmConsumed || w.accNormal <= 0.0 {
			continue
		}
		var wId uint64
		if w.rCage != nil {
			wId = w.rCage.GetEntity().GetId()
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
		// Ignore impulses from bouncing contacts or initial single-frame crash transients;
		// once persistent across consecutive frames without rebound, seed warm-starting from frame 2+.
		if bestPrev.restTarget > 1e-3 {
			return
		}
		entry.warmEligible = true
		if !bestPrev.warmEligible {
			return
		}
		entry.SetWarmImpulse(
			bestPrev.accNormal*warmStartFactor,
			bestPrev.accTx*warmStartFactor,
			bestPrev.accTy*warmStartFactor,
			bestPrev.accTz*warmStartFactor,
		)
	}
}
