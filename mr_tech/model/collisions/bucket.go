package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/volumes"
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

// FacesPerBucket specifies the number of faces allocated per bucket.
const FacesPerBucket = 4

// Bucket holds collision data for a specific bucket, including spare and active cage entries and counters.
type Bucket struct {
	bucket           BucketType
	spare            [FacesPerBucket]*Entry
	container        [FacesPerBucket]*Entry
	containerCounter int
}

// NewBucket creates a new Bucket for a given BucketType, initializing its spare and container arrays.
func NewBucket(bucket BucketType) *Bucket {
	b := &Bucket{
		bucket:           bucket,
		containerCounter: 0,
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.spare[i] = NewCageEntry()
		b.container[i] = nil
	}
	return b
}

// Rebuild resets the containerCounter to 0, clearing the current collision state for the bucket.
func (b *Bucket) Rebuild() {
	b.containerCounter = 0
}

// Count returns the total number of entries currently stored in the Bucket.
func (b *Bucket) Count() int {
	return b.containerCounter
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
		entry := b.container[i]
		// Se tocchiamo direttamente un muro, il bucket è bloccato
		if entry.iMode == ImpactInelastic {
			return false
		}
		if entry.rCage != nil && entry.rCage.object.VerticesGetEntity().GetId() == from {
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
func (b *Bucket) Add(bucket BucketType, lCage *Cage, rCage *Cage, rFace *volumes.Face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) *Entry {
	for i := 0; i < b.containerCounter; i++ {
		existing := b.container[i]
		// DEDUPLICAZIONE PER ENTITÀ DINAMICHE (Contact Reduction)
		// Se questa faccia appartiene allo STESSO oggetto dinamico che abbiamo già registrato in QUESTO bucket...
		if rCage != nil && existing.rCage != nil {
			if rCage.object.VerticesGetEntity().GetId() == existing.rCage.object.VerticesGetEntity().GetId() {
				if penetration > existing.penetration {
					existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
				}
				return nil // Interrompiamo: abbiamo già gestito questo oggetto in questo bucket
			}
		}
		// DEDUPLICAZIONE TOPOLOGICA (Geometria Statica Coplanare)
		// Se il dot product è ~1.0, i due triangoli formano un piano continuo (Triangle Soup statica)
		if rCage == nil && existing.rCage == nil {
			if dot := (normalX * existing.nX) + (normalY * existing.nY) + (normalZ * existing.nZ); dot > 0.999 {
				planeDistance := math.Abs(
					(p0x-existing.p0X)*existing.nX +
						(p0y-existing.p0Y)*existing.nY +
						(p0z-existing.p0Z)*existing.nZ,
				)
				if planeDistance < planeEpsilon {
					// Un vincolo anelastico (muro solido) ha sempre priorità su uno scalino (ImpactStep) coplanare
					if existing.iMode == ImpactStep && iMode == ImpactInelastic {
						existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
						return nil
					}
					if existing.iMode == ImpactInelastic && iMode == ImpactStep {
						return nil
					}
					if penetration > existing.penetration || (iMode == ImpactStep && math.Abs(penetration-existing.penetration) <= planeEpsilon && maxZ > existing.maxZ) {
						existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
					}
					return nil
				}
			}
		}
	}

	// Insert a new plane into the non-full bucket
	if b.containerCounter < FacesPerBucket {
		target := b.spare[b.containerCounter]
		target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.container[b.containerCounter] = target
		b.containerCounter++
		return target
	}

	// Replace the least relevant face (respecting impactModePriority first, then penetration depth)
	minIdx := 0
	minPrio := b.container[0].iMode.Priority()
	minPen := b.container[0].penetration
	for i := 1; i < FacesPerBucket; i++ {
		cPrio := b.container[i].iMode.Priority()
		cPen := b.container[i].penetration
		if cPrio < minPrio || (cPrio == minPrio && cPen < minPen) {
			minPrio = cPrio
			minPen = cPen
			minIdx = i
		}
	}
	inPrio := iMode.Priority()
	if inPrio > minPrio || (inPrio == minPrio && penetration > minPen) {
		b.container[minIdx].Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	}
	return nil
}
