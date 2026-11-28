package collisions

import (
	"math"
	"math/bits"
)

// Cradle tuning constants define slope, dot-product, angular sector, and deduplication thresholds.
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

// _cradleSectorMinCos stores lower-bound cosines for angular differences between cradle sectors.
var _cradleSectorMinCos [cradleSectors/2 + 1]float64

// init precomputes minimum cosine bounds for cradle sector separations.
func init() {
	for d := range _cradleSectorMinCos {
		if d >= cradleSectors/2-1 {
			_cradleSectorMinCos[d] = -1.0
		} else {
			// Widen by 1e-9 rad so boundary quantization rounding never overestimates min cosine.
			_cradleSectorMinCos[d] = math.Cos(float64(d+1)*cradleSectorWidth + 1e-9)
		}
	}
}

// cradleSectorForXY returns the angular cradle sector index for the horizontal vector nx, ny.
func cradleSectorForXY(nx, ny float64) int {
	ang := math.Atan2(ny, nx) + math.Pi
	sec := int(ang * cradleSectorScale)
	if sec < 0 || sec >= cradleSectors {
		return 0
	}
	return sec
}

// cradleSectorDiff returns the shortest circular distance between two cradle sector indices.
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

// cradleSlopePairValid reports whether two slope entries can jointly provide valid cradle ground support.
func cradleSlopePairValid(a, b *Entry) bool {
	return a.nZ+b.nZ >= floorNormalThreshold && a.nX*b.nX+a.nY*b.nY < cradleMinPairDotXY
}

// cradleSlopeSector summarizes slope contacts in one angular sector using indices into cage.slots.
type cradleSlopeSector struct {
	maxR      float64
	steepMaxR float64
	repR      int32
	repZ      int32
	steepRep  int32
	head      int32
}

// cradleBumperSector summarizes wall-bumper contacts in one angular sector using an index into cage.slots.
type cradleBumperSector struct {
	maxR float64
	rep  int32
	head int32
}

// CradleCache detects multi-contact cradle support directly over cage.slots without copying contact entries.
type CradleCache struct {
	slopeSec  [cradleSectors]*cradleSlopeSector
	bumperSec [cradleSectors]*cradleBumperSector
	nextIdx   []int32

	dirty  bool
	cached bool
}

// NewCradleCache creates and returns an empty zero-copy cradle support cache.
func NewCradleCache() *CradleCache {
	cc := &CradleCache{
		dirty:  true,
		cached: false,
	}
	for x := range cradleSectors {
		cc.slopeSec[x] = &cradleSlopeSector{}
		cc.bumperSec[x] = &cradleBumperSector{}
	}
	return cc
}

// Reset marks the cradle support cache dirty and clears its cached state.
func (c *CradleCache) Reset() {
	c.dirty = true
	c.cached = false
}

// SetDirty marks the cradle support cache as dirty.
func (c *CradleCache) SetDirty() {
	c.dirty = true
}

// HasSupport reports whether slots contain contacts that cradle a free 6-DoF body, using a cached result when clean.
func (c *CradleCache) HasSupport(slots *Slots) bool {
	if !c.dirty {
		return c.cached
	}
	n := slots.Len()
	if n <= cradleSmallThreshold {
		c.cached = c.hasSupportSmall(slots)
	} else {
		c.cached = c.hasSupportFull(slots)
	}
	c.dirty = false
	return c.cached
}

// HasSupportSmall reports whether the cage slots contain a small paired-normal cradle support.
func (c *CradleCache) hasSupportSmall(slots *Slots) bool {
	n := slots.Len()
	for i := range n {
		a := slots.At(i)
		if a.iMode == ImpactStep || a.nZ < cradleMinWallBumperNZ {
			continue
		}
		for j := i + 1; j < n; j++ {
			b := slots.At(j)
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

// HasSupport reports whether cage.slots contain opposing slope or steep-slope bumper contacts
// that cradle a free 6-DoF body, reading cage.slots in-place without copying entry normals.
func (c *CradleCache) hasSupportFull(slots *Slots) bool {
	n := slots.Len()
	hasSlope := false
	hasSteep := false
	for idx := range n {
		e := slots.At(idx)
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

	var slopeMask, steepMask, bumperMask uint64
	var prevValid bool
	var prevBX, prevBY uint64
	var prevNZ, prevR float64
	var prevSec int

	for idx := range n {
		e := slots.At(idx)
		if e.iMode == ImpactStep || e.nZ < cradleMinWallBumperNZ {
			continue
		}
		nz := e.nZ
		if nz <= cradleMinSlopeNZ && !hasSteep {
			continue
		}
		nx, ny := e.nX, e.nY
		bx, by := math.Float64bits(nx), math.Float64bits(ny)
		var r float64
		var sec int
		if prevValid && bx == prevBX && by == prevBY {
			if nz == prevNZ {
				continue
			}
			r = prevR
			sec = prevSec
		} else {
			rSq := nx*nx + ny*ny
			if rSq == 0 {
				continue
			}
			// Pad upper-bound radius by 1 ULP margin so sector pruning never drops exact boundary contacts.
			r = math.Sqrt(rSq) * (1.0 + 1e-12)
			sec = cradleSectorForXY(nx, ny)
			prevBX, prevBY = bx, by
			prevR = r
			prevSec = sec
			prevValid = true
		}
		prevNZ = nz
		bit := uint64(1) << sec

		if nz <= cradleMinSlopeNZ {
			bs := c.bumperSec[sec]
			if (bumperMask & bit) == 0 {
				bumperMask |= bit
				bs.maxR = r
				bs.rep = int32(idx)
			} else if r > bs.maxR {
				bs.maxR = r
				bs.rep = int32(idx)
			}
			continue
		}

		ss := c.slopeSec[sec]
		if (slopeMask & bit) == 0 {
			slopeMask |= bit
			ss.maxR = r
			ss.repR = int32(idx)
			ss.repZ = int32(idx)
			ss.steepMaxR = -1
			ss.steepRep = -1
		} else {
			if nz > slots.At(int(ss.repZ)).nZ {
				ss.repZ = int32(idx)
			}
			if r > ss.maxR {
				ss.maxR = r
				ss.repR = int32(idx)
			}
		}
		if nz >= cradleSteepSupportNZ && ((steepMask&bit) == 0 || r > ss.steepMaxR) {
			steepMask |= bit
			ss.steepMaxR = r
			ss.steepRep = int32(idx)
		}
	}

	var ambigSlope [cradleSectors]uint64
	var ambigWall [cradleSectors]uint64
	var ambigSlopeS1, needSlope uint64
	var ambigWallS1, needSteep, needBumper uint64

	// Phase A: Check extreme sector representatives directly via slots indices for O(1) hits or pruning.
	for m1 := slopeMask; m1 != 0; {
		s1 := bits.TrailingZeros64(m1)
		m1 &^= 1 << s1
		a := c.slopeSec[s1]
		aRepR := slots.At(int(a.repR))
		aRepZ := slots.At(int(a.repZ))
		for m2 := m1; m2 != 0; {
			s2 := bits.TrailingZeros64(m2)
			m2 &^= 1 << s2
			minCos := _cradleSectorMinCos[cradleSectorDiff(s1, s2)]
			if minCos >= 0 {
				continue
			}
			b := c.slopeSec[s2]
			bRepZ := slots.At(int(b.repZ))
			if aRepZ.nZ+bRepZ.nZ < floorNormalThreshold || a.maxR*b.maxR*minCos >= cradleMinPairDotXY {
				continue
			}
			bRepR := slots.At(int(b.repR))
			if cradleSlopePairValid(aRepR, bRepR) ||
				cradleSlopePairValid(aRepZ, bRepZ) ||
				cradleSlopePairValid(aRepR, bRepZ) ||
				cradleSlopePairValid(aRepZ, bRepR) {
				return true
			}
			ambigSlope[s1] |= uint64(1) << s2
			ambigSlopeS1 |= uint64(1) << s1
			needSlope |= (uint64(1) << s1) | (uint64(1) << s2)
		}
	}

	if hasSteep && bumperMask != 0 {
		for m1 := steepMask; m1 != 0; {
			s1 := bits.TrailingZeros64(m1)
			m1 &^= 1 << s1
			a := c.slopeSec[s1]
			aSteep := slots.At(int(a.steepRep))
			for m2 := bumperMask; m2 != 0; {
				s2 := bits.TrailingZeros64(m2)
				m2 &^= 1 << s2
				minCos := _cradleSectorMinCos[cradleSectorDiff(s1, s2)]
				if minCos >= 0 {
					continue
				}
				b := c.bumperSec[s2]
				if a.steepMaxR*b.maxR*minCos >= cradleMinWallDotXY {
					continue
				}
				bRep := slots.At(int(b.rep))
				if aSteep.nX*bRep.nX+aSteep.nY*bRep.nY < cradleMinWallDotXY {
					return true
				}
				ambigWall[s1] |= uint64(1) << s2
				ambigWallS1 |= uint64(1) << s1
				needSteep |= uint64(1) << s1
				needBumper |= uint64(1) << s2
			}
		}
	}

	if (ambigSlopeS1 | ambigWallS1) == 0 {
		return false
	}

	// Phase B: Index surviving hairline sectors lazily in-place using only nextIdx slot indices (zero Entry copies).
	for m := needSlope | needSteep; m != 0; {
		s := bits.TrailingZeros64(m)
		m &^= 1 << s
		c.slopeSec[s].head = -1
	}
	for m := needBumper; m != 0; {
		s := bits.TrailingZeros64(m)
		m &^= 1 << s
		c.bumperSec[s].head = -1
	}

	if cap(c.nextIdx) < n {
		c.nextIdx = make([]int32, n)
	} else {
		c.nextIdx = c.nextIdx[:n]
	}

	allSlopeMask := needSlope | needSteep
	prevValid = false
	for idx := range n {
		e := slots.At(idx)
		if e.iMode == ImpactStep || e.nZ < cradleMinWallBumperNZ {
			continue
		}
		nz := e.nZ
		isSlope := nz > cradleMinSlopeNZ
		if isSlope {
			if (nz < cradleSteepSupportNZ && needSlope == 0) || allSlopeMask == 0 {
				continue
			}
		} else if needBumper == 0 {
			continue
		}

		nx, ny := e.nX, e.nY
		bx, by := math.Float64bits(nx), math.Float64bits(ny)
		var sec int
		if prevValid && bx == prevBX && by == prevBY {
			if nz == prevNZ {
				continue
			}
			sec = prevSec
		} else {
			if nx*nx+ny*ny == 0 {
				continue
			}
			sec = cradleSectorForXY(nx, ny)
			prevBX, prevBY = bx, by
			prevSec = sec
			prevValid = true
		}
		prevNZ = nz
		bit := uint64(1) << sec

		if !isSlope {
			if (needBumper & bit) == 0 {
				continue
			}
			bs := c.bumperSec[sec]
			dup := false
			cur := bs.head
			for k := 0; cur >= 0 && k < cradleDedupWindow; k++ {
				node := slots.At(int(cur))
				if node.nX == nx && node.nY == ny {
					dup = true
					break
				}
				cur = c.nextIdx[cur]
			}
			if !dup {
				c.nextIdx[idx] = bs.head
				bs.head = int32(idx)
			}
			continue
		}

		if (needSlope&bit) == 0 && (nz < cradleSteepSupportNZ || (needSteep&bit) == 0) {
			continue
		}
		ss := c.slopeSec[sec]
		dup := false
		cur := ss.head
		for k := 0; cur >= 0 && k < cradleDedupWindow; k++ {
			node := slots.At(int(cur))
			if node.nX == nx && node.nY == ny && node.nZ == nz {
				dup = true
				break
			}
			cur = c.nextIdx[cur]
		}
		if !dup {
			c.nextIdx[idx] = ss.head
			ss.head = int32(idx)
		}
	}

	for m1 := ambigSlopeS1; m1 != 0; {
		s1 := bits.TrailingZeros64(m1)
		m1 &^= 1 << s1
		aSec := c.slopeSec[s1]
		for m2 := ambigSlope[s1]; m2 != 0; {
			s2 := bits.TrailingZeros64(m2)
			m2 &^= 1 << s2
			minCos := _cradleSectorMinCos[cradleSectorDiff(s1, s2)]
			bSec := c.slopeSec[s2]
			bMaxNZ := slots.At(int(bSec.repZ)).nZ
			for ni := aSec.head; ni >= 0; ni = c.nextIdx[ni] {
				na := slots.At(int(ni))
				rA := math.Sqrt(na.nX*na.nX+na.nY*na.nY) * (1.0 + 1e-12)
				if na.nZ+bMaxNZ < floorNormalThreshold || rA*bSec.maxR*minCos >= cradleMinPairDotXY {
					continue
				}
				for nj := bSec.head; nj >= 0; nj = c.nextIdx[nj] {
					if cradleSlopePairValid(na, slots.At(int(nj))) {
						return true
					}
				}
			}
		}
	}

	for m1 := ambigWallS1; m1 != 0; {
		s1 := bits.TrailingZeros64(m1)
		m1 &^= 1 << s1
		aSec := c.slopeSec[s1]
		for m2 := ambigWall[s1]; m2 != 0; {
			s2 := bits.TrailingZeros64(m2)
			m2 &^= 1 << s2
			minCos := _cradleSectorMinCos[cradleSectorDiff(s1, s2)]
			bSec := c.bumperSec[s2]
			for ni := aSec.head; ni >= 0; ni = c.nextIdx[ni] {
				na := slots.At(int(ni))
				if na.nZ < cradleSteepSupportNZ {
					continue
				}
				rA := math.Sqrt(na.nX*na.nX+na.nY*na.nY) * (1.0 + 1e-12)
				if rA*bSec.maxR*minCos >= cradleMinWallDotXY {
					continue
				}
				for nj := bSec.head; nj >= 0; nj = c.nextIdx[nj] {
					nb := slots.At(int(nj))
					if na.nX*nb.nX+na.nY*nb.nY < cradleMinWallDotXY {
						return true
					}
				}
			}
		}
	}

	return false
}
