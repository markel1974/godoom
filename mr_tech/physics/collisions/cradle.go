package collisions

import (
	"math"
	"math/bits"
)

// Constants tune cradle support thresholds and angular sector quantization for cradle detection.
const (
	cradleMinSlopeNZ      = 0.25
	cradleSteepSupportNZ  = 0.45
	cradleMinWallBumperNZ = -0.05
	cradleMinPairDotXY    = -0.10
	cradleMinWallDotXY    = -0.35

	cradleSmallThreshold = 16
	cradleSectors        = 64
	cradleSectorWidth    = (2.0 * math.Pi) / float64(cradleSectors)
	cradleSectorScale    = float64(cradleSectors) / (2.0 * math.Pi)
	cradleDedupWindow    = 4
)

// cradleSectorMinCos stores conservative minimum cosines for half-circle sector separations.
var cradleSectorMinCos [cradleSectors/2 + 1]float64

// init populates cradleSectorMinCos with conservative minimum cosine values for sector offsets.
func init() {
	for d := range cradleSectorMinCos {
		if d >= cradleSectors/2-1 {
			cradleSectorMinCos[d] = -1.0
		} else {
			// Widen by 1e-9 rad so boundary quantization rounding never overestimates min cosine.
			cradleSectorMinCos[d] = math.Cos(float64(d+1)*cradleSectorWidth + 1e-9)
		}
	}
}

// cradleNode stores a deduplicated contact normal sample and links it into a sector-local chain.
type cradleNode struct {
	nX, nY, nZ float64
	r          float64
	next       int32
}

// cradleSlopeSector caches aggregate support data for slope contacts within one angular sector.
// It supports fast sector-pair pruning before exact checks on deduplicated contact nodes.
type cradleSlopeSector struct {
	head      int32
	maxNZ     float64
	maxR      float64
	repRX     float64
	repRY     float64
	repRZ     float64
	repZX     float64
	repZY     float64
	repZZ     float64
	steepMaxR float64
	steepRepX float64
	steepRepY float64
}

// cradleBumperSector summarizes wall-bumper contacts within one angular sector for support checks.
type cradleBumperSector struct {
	head int32
	maxR float64
	repX float64
	repY float64
}

// cradleCache stores reusable sector indexes and scratch buffers for cradle support detection.
type cradleCache struct {
	slopeSec    [cradleSectors]cradleSlopeSector
	bumperSec   [cradleSectors]cradleBumperSector
	slopeNodes  []cradleNode
	bumperNodes []cradleNode
	ambigSlopes []uint16
	ambigWalls  []uint16
}

// init initializes reusable cache slices for cradle support checks.
func (c *cradleCache) init() {
	c.slopeNodes = make([]cradleNode, 0, 128)
	c.bumperNodes = make([]cradleNode, 0, 128)
	c.ambigSlopes = make([]uint16, 0, 64)
	c.ambigWalls = make([]uint16, 0, 64)
}

// cradleSectorForXY returns the angular cradle sector containing the vector defined by nx and ny.
func cradleSectorForXY(nx, ny float64) int {
	ang := math.Atan2(ny, nx) + math.Pi
	sec := int(ang * cradleSectorScale)
	if sec < 0 || sec >= cradleSectors {
		return 0
	}
	return sec
}

// cradleSectorDiff returns the shortest distance between two sector indexes on the circular sector ring.
func cradleSectorDiff(s1, s2 int) int {
	d := s2 - s1
	if d < 0 {
		d = -d
	}
	if d > cradleSectors/2 {
		d = cradleSectors - d
	}
	return d
}

// cradleSlopePairValid reports whether two slope normals provide enough upward support and opposing XY directions.
func cradleSlopePairValid(ax, ay, az, bx, by, bz float64) bool {
	return az+bz >= floorNormalThreshold && ax*bx+ay*by < cradleMinPairDotXY
}

// hasCradleSupportSmall reports whether non-step contact normals provide opposing small-cradle support.
func hasCradleSupportSmall(slots []*Entry) bool {
	n := len(slots)
	for i := range n {
		a := slots[i]
		if a.iMode == ImpactStep || a.nZ < cradleMinWallBumperNZ {
			continue
		}
		for j := i + 1; j < n; j++ {
			b := slots[j]
			if b.iMode == ImpactStep || b.nZ < cradleMinWallBumperNZ {
				continue
			}
			hi, lo := a, b
			if lo.nZ > hi.nZ {
				hi, lo = b, a
			}
			if hi.nZ <= cradleMinSlopeNZ {
				continue
			}
			horizDot := hi.nX*lo.nX + hi.nY*lo.nY
			if lo.nZ > cradleMinSlopeNZ {
				if hi.nZ+lo.nZ >= floorNormalThreshold && horizDot < cradleMinPairDotXY {
					return true
				}
			} else if hi.nZ >= cradleSteepSupportNZ && horizDot < cradleMinWallDotXY {
				return true
			}
		}
	}
	return false
}

// hasSupport reports whether the given contact slots provide cradle support from opposing slopes or steep wall bumpers.
func (c *cradleCache) hasSupport(slots []*Entry) bool {
	hasSlope := false
	hasSteep := false
	for i := range slots {
		e := slots[i]
		if e.iMode == ImpactStep {
			continue
		}
		if e.nZ > cradleMinSlopeNZ {
			hasSlope = true
			if e.nZ >= cradleSteepSupportNZ {
				hasSteep = true
				break
			}
		}
	}
	if !hasSlope {
		return false
	}

	c.slopeNodes = c.slopeNodes[:0]
	c.bumperNodes = c.bumperNodes[:0]
	c.ambigSlopes = c.ambigSlopes[:0]
	c.ambigWalls = c.ambigWalls[:0]

	var slopeMask, steepMask, bumperMask uint64

	for i := range slots {
		e := slots[i]
		if e.iMode == ImpactStep || e.nZ < cradleMinWallBumperNZ {
			continue
		}
		nz := e.nZ
		if nz <= cradleMinSlopeNZ && !hasSteep {
			continue
		}
		nx, ny := e.nX, e.nY
		rSq := nx*nx + ny*ny
		if rSq == 0 {
			continue
		}
		// Pad upper-bound radius by 1 ULP margin so sector pruning never drops exact boundary contacts.
		r := math.Sqrt(rSq) * (1.0 + 1e-12)
		sec := cradleSectorForXY(nx, ny)
		bit := uint64(1) << sec

		if nz <= cradleMinSlopeNZ {
			bs := &c.bumperSec[sec]
			if (bumperMask & bit) == 0 {
				bumperMask |= bit
				bs.head = -1
				bs.maxR = r
				bs.repX, bs.repY = nx, ny
			} else if r > bs.maxR {
				bs.maxR = r
				bs.repX, bs.repY = nx, ny
			}
			dup := false
			cur := bs.head
			for k := 0; cur >= 0 && k < cradleDedupWindow; k++ {
				node := &c.bumperNodes[cur]
				if node.nX == nx && node.nY == ny {
					dup = true
					break
				}
				cur = node.next
			}
			if !dup {
				idx := int32(len(c.bumperNodes))
				c.bumperNodes = append(c.bumperNodes, cradleNode{nX: nx, nY: ny, nZ: nz, r: r, next: bs.head})
				bs.head = idx
			}
			continue
		}

		ss := &c.slopeSec[sec]
		if (slopeMask & bit) == 0 {
			slopeMask |= bit
			ss.head = -1
			ss.maxNZ = nz
			ss.maxR = r
			ss.repRX, ss.repRY, ss.repRZ = nx, ny, nz
			ss.repZX, ss.repZY, ss.repZZ = nx, ny, nz
			ss.steepMaxR = -1
		} else {
			if nz > ss.maxNZ {
				ss.maxNZ = nz
				ss.repZX, ss.repZY, ss.repZZ = nx, ny, nz
			}
			if r > ss.maxR {
				ss.maxR = r
				ss.repRX, ss.repRY, ss.repRZ = nx, ny, nz
			}
		}
		if nz >= cradleSteepSupportNZ && ((steepMask&bit) == 0 || r > ss.steepMaxR) {
			steepMask |= bit
			ss.steepMaxR = r
			ss.steepRepX, ss.steepRepY = nx, ny
		}
		dup := false
		cur := ss.head
		for k := 0; cur >= 0 && k < cradleDedupWindow; k++ {
			node := &c.slopeNodes[cur]
			if node.nX == nx && node.nY == ny && node.nZ == nz {
				dup = true
				break
			}
			cur = node.next
		}
		if !dup {
			idx := int32(len(c.slopeNodes))
			c.slopeNodes = append(c.slopeNodes, cradleNode{nX: nx, nY: ny, nZ: nz, r: r, next: ss.head})
			ss.head = idx
		}
	}

	// Phase A: Check extreme sector representatives first for O(1) sector-pair hits or bounds pruning.
	for m1 := slopeMask; m1 != 0; {
		s1 := bits.TrailingZeros64(m1)
		m1 &^= 1 << s1
		a := &c.slopeSec[s1]
		for m2 := m1; m2 != 0; {
			s2 := bits.TrailingZeros64(m2)
			m2 &^= 1 << s2
			minCos := cradleSectorMinCos[cradleSectorDiff(s1, s2)]
			if minCos >= 0 {
				continue
			}
			b := &c.slopeSec[s2]
			if a.maxNZ+b.maxNZ < floorNormalThreshold || a.maxR*b.maxR*minCos >= cradleMinPairDotXY {
				continue
			}
			if cradleSlopePairValid(a.repRX, a.repRY, a.repRZ, b.repRX, b.repRY, b.repRZ) ||
				cradleSlopePairValid(a.repZX, a.repZY, a.repZZ, b.repZX, b.repZY, b.repZZ) ||
				cradleSlopePairValid(a.repRX, a.repRY, a.repRZ, b.repZX, b.repZY, b.repZZ) ||
				cradleSlopePairValid(a.repZX, a.repZY, a.repZZ, b.repRX, b.repRY, b.repRZ) {
				return true
			}
			c.ambigSlopes = append(c.ambigSlopes, uint16((s1<<8)|s2))
		}
	}

	if hasSteep && bumperMask != 0 {
		for m1 := steepMask; m1 != 0; {
			s1 := bits.TrailingZeros64(m1)
			m1 &^= 1 << s1
			a := &c.slopeSec[s1]
			for m2 := bumperMask; m2 != 0; {
				s2 := bits.TrailingZeros64(m2)
				m2 &^= 1 << s2
				minCos := cradleSectorMinCos[cradleSectorDiff(s1, s2)]
				if minCos >= 0 {
					continue
				}
				b := &c.bumperSec[s2]
				if a.steepMaxR*b.maxR*minCos >= cradleMinWallDotXY {
					continue
				}
				if a.steepRepX*b.repX+a.steepRepY*b.repY < cradleMinWallDotXY {
					return true
				}
				c.ambigWalls = append(c.ambigWalls, uint16((s1<<8)|s2))
			}
		}
	}

	// Phase B: Exact fallback only across deduplicated nodes inside surviving hairline sector pairs.
	for _, packed := range c.ambigSlopes {
		s1, s2 := int(packed>>8), int(packed&0xFF)
		minCos := cradleSectorMinCos[cradleSectorDiff(s1, s2)]
		bSec := &c.slopeSec[s2]
		for ni := c.slopeSec[s1].head; ni >= 0; {
			na := &c.slopeNodes[ni]
			ni = na.next
			if na.nZ+bSec.maxNZ < floorNormalThreshold || na.r*bSec.maxR*minCos >= cradleMinPairDotXY {
				continue
			}
			for nj := bSec.head; nj >= 0; {
				nb := &c.slopeNodes[nj]
				nj = nb.next
				if na.nZ+nb.nZ >= floorNormalThreshold && na.nX*nb.nX+na.nY*nb.nY < cradleMinPairDotXY {
					return true
				}
			}
		}
	}

	for _, packed := range c.ambigWalls {
		s1, s2 := int(packed>>8), int(packed&0xFF)
		minCos := cradleSectorMinCos[cradleSectorDiff(s1, s2)]
		bSec := &c.bumperSec[s2]
		for ni := c.slopeSec[s1].head; ni >= 0; {
			na := &c.slopeNodes[ni]
			ni = na.next
			if na.nZ < cradleSteepSupportNZ || na.r*bSec.maxR*minCos >= cradleMinWallDotXY {
				continue
			}
			for nj := bSec.head; nj >= 0; {
				nb := &c.bumperNodes[nj]
				nj = nb.next
				if na.nX*nb.nX+na.nY*nb.nY < cradleMinWallDotXY {
					return true
				}
			}
		}
	}

	return false
}
