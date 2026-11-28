package collisions

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func makeCradleEntry(nX, nY, nZ float64, mode ImpactMode) *Entry {
	e := NewCageEntry()
	e.Rebuild(Normal2BucketType(nX, nY, nZ), nil, nil, nil, 0, 0.1, nX, nY, nZ, 0, 0, 0, 0, mode)
	return e
}

func makeCradleUnitSlope(angle, nZ float64, radialScale float64, mode ImpactMode) *Entry {
	rXY := math.Sqrt(math.Max(0, 1.0-nZ*nZ)) * radialScale
	return makeCradleEntry(math.Cos(angle)*rXY, math.Sin(angle)*rXY, nZ*radialScale, mode)
}

func padToCachePath(entries []*Entry) []*Entry {
	if len(entries) > cradleSmallThreshold {
		return entries
	}
	out := make([]*Entry, len(entries), cradleSmallThreshold+4)
	copy(out, entries)
	for len(out) <= cradleSmallThreshold {
		// Degenerate zero-XY entry safely skipped by both implementations.
		out = append(out, makeCradleEntry(0, 0, 0.5, ImpactInelastic))
	}
	return out
}

func makeTestSlots(entries []*Entry) *Slots {
	return &Slots{data: entries, len: len(entries)}
}

func TestCradleSupportEquivalenceBoundaryCases(t *testing.T) {
	cache := NewCradleCache()

	verify := func(name string, entries []*Entry) {
		t.Helper()
		rawSlots := makeTestSlots(entries)
		want := cache.hasSupportSmall(rawSlots)

		padded := padToCachePath(entries)
		padSlots := makeTestSlots(padded)
		cache.Reset()
		got := cache.HasSupport(padSlots)
		if got != want {
			t.Fatalf("%s: HasSupport=%v, want %v (hasSupportSmall)", name, got, want)
		}
		if wantPadded := cache.hasSupportSmall(padSlots); wantPadded != want {
			t.Fatalf("%s: internal test helper mismatch padded=%v vs want=%v", name, wantPadded, want)
		}
	}

	// 1. Slope threshold nZ > 0.25
	verify("nZ exactly 0.25 opposite", []*Entry{
		makeCradleUnitSlope(0, 0.25, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, 0.25, 1, ImpactInelastic),
	})
	verify("nZ just above 0.25 opposite", []*Entry{
		makeCradleUnitSlope(0, 0.2500001, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, 0.2500001, 1, ImpactInelastic),
	})

	// 2. ImpactStep always excluded
	verify("opposite ImpactStep ignored", []*Entry{
		makeCradleUnitSlope(0, 0.55, 1, ImpactStep),
		makeCradleUnitSlope(math.Pi, 0.55, 1, ImpactStep),
	})
	verify("steep slope with ImpactStep wall ignored", []*Entry{
		makeCradleUnitSlope(0, 0.60, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, 0.0, 1, ImpactStep),
	})

	// 3. Steep threshold nZ >= 0.45 for slope+bumper
	verify("sub-steep 0.449999 + pure wall bumper", []*Entry{
		makeCradleUnitSlope(0, 0.449999, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, 0.0, 1, ImpactInelastic),
	})
	verify("steep 0.45 + bumper at -0.05", []*Entry{
		makeCradleUnitSlope(0, 0.45, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, -0.05, 1, ImpactInelastic),
	})
	verify("steep 0.45 + bumper below -0.05", []*Entry{
		makeCradleUnitSlope(0, 0.45, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi, -0.050001, 1, ImpactInelastic),
	})

	// 4. Opposite atan2 seam around -pi / +pi (and negative zero)
	verify("seam -pi+eps vs +pi-eps (same direction)", []*Entry{
		makeCradleUnitSlope(-math.Pi+1e-7, 0.55, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi-1e-7, 0.55, 1, ImpactInelastic),
	})
	verify("seam 0 vs +pi-eps (opposite across seam)", []*Entry{
		makeCradleUnitSlope(1e-7, 0.55, 1, ImpactInelastic),
		makeCradleUnitSlope(math.Pi-1e-7, 0.55, 1, ImpactInelastic),
	})
	verify("negative zero vs +0 along -X", []*Entry{
		makeCradleEntry(-0.5, math.Copysign(0, -1), 0.5, ImpactInelastic),
		makeCradleEntry(-0.5, 0.0, 0.5, ImpactInelastic),
	})

	// 5. Hairline dot thresholds (-0.10 for slope-slope, -0.35 for steep-bumper)
	slopeThreshAngle := math.Acos(-0.10)
	wallThreshAngle := math.Acos(-0.35)
	for baseDeg := 0; baseDeg < 360; baseDeg += 7 {
		base := float64(baseDeg) * math.Pi / 180.0
		verify(fmt.Sprintf("hairline slope+ base=%d", baseDeg), []*Entry{
			makeCradleUnitSlope(base, 0.35, 1, ImpactInelastic),
			makeCradleUnitSlope(base+slopeThreshAngle+1e-5, 0.35, 1, ImpactInelastic),
		})
		verify(fmt.Sprintf("hairline slope- base=%d", baseDeg), []*Entry{
			makeCradleUnitSlope(base, 0.35, 1, ImpactInelastic),
			makeCradleUnitSlope(base+slopeThreshAngle-1e-5, 0.35, 1, ImpactInelastic),
		})
		verify(fmt.Sprintf("hairline wall+ base=%d", baseDeg), []*Entry{
			makeCradleUnitSlope(base, 0.55, 1, ImpactInelastic),
			makeCradleUnitSlope(base+wallThreshAngle+1e-5, 0.05, 1, ImpactInelastic),
		})
		verify(fmt.Sprintf("hairline wall- base=%d", baseDeg), []*Entry{
			makeCradleUnitSlope(base, 0.55, 1, ImpactInelastic),
			makeCradleUnitSlope(base+wallThreshAngle-1e-5, 0.05, 1, ImpactInelastic),
		})
	}
}

func TestCradleSupportEquivalenceRandomized(t *testing.T) {
	cache := NewCradleCache()
	rng := rand.New(rand.NewSource(20261127))
	entries := make([]*Entry, 0, 320)

	for trial := 0; trial < 800; trial++ {
		entries = entries[:0]
		n := 17 + rng.Intn(260)
		pattern := trial % 5

		baseAngle := rng.Float64() * 2 * math.Pi
		switch pattern {
		case 0:
			// Narrow single-side arc (~70 deg) -> impossible cradle
			for i := 0; i < n; i++ {
				ang := baseAngle + (rng.Float64()-0.5)*1.2
				nz := 0.26 + rng.Float64()*0.40
				scale := 0.4 + rng.Float64()*2.0
				entries = append(entries, makeCradleUnitSlope(ang, nz, scale, ImpactInelastic))
			}
		case 1:
			// Heavy tessellation: 5 base normals repeated with tiny jitter
			var bases [5]struct {
				ang, nz float64
				mode    ImpactMode
			}
			for b := range bases {
				bases[b].ang = baseAngle + float64(b)*0.32
				bases[b].nz = 0.05 + rng.Float64()*0.60
				bases[b].mode = ImpactInelastic
				if b == 4 && rng.Intn(2) == 0 {
					bases[b].mode = ImpactStep
				}
			}
			for i := 0; i < n; i++ {
				b := bases[rng.Intn(len(bases))]
				ang := b.ang + (rng.Float64()-0.5)*1e-3
				nz := b.nz + (rng.Float64()-0.5)*1e-4
				entries = append(entries, makeCradleUnitSlope(ang, nz, 1.0, b.mode))
			}
		case 2:
			// Hairline ambiguous cone near dot = -0.10 (105 deg span around baseAngle)
			span := math.Acos(-0.10) + (rng.Float64()-0.5)*0.08
			for i := 0; i < n; i++ {
				ang := baseAngle + rng.Float64()*span
				nz := 0.25 + rng.Float64()*0.44
				entries = append(entries, makeCradleUnitSlope(ang, nz, 0.8+rng.Float64()*0.5, ImpactInelastic))
			}
		case 3:
			// Hairline steep + wall cone near dot = -0.35
			span := math.Acos(-0.35) + (rng.Float64()-0.5)*0.08
			for i := 0; i < n; i++ {
				if i%3 == 0 {
					ang := baseAngle + rng.Float64()*0.1
					nz := 0.44 + rng.Float64()*0.25
					entries = append(entries, makeCradleUnitSlope(ang, nz, 1.0, ImpactInelastic))
				} else {
					ang := baseAngle + span - rng.Float64()*0.05
					nz := -0.06 + rng.Float64()*0.30
					entries = append(entries, makeCradleUnitSlope(ang, nz, 1.0, ImpactInelastic))
				}
			}
		default:
			// Full chaotic mix including degenerates and different radial lengths
			for i := 0; i < n; i++ {
				if rng.Intn(12) == 0 {
					entries = append(entries, makeCradleEntry(0, 0, rng.Float64()*0.6, ImpactInelastic))
					continue
				}
				ang := (rng.Float64() - 0.5) * 2 * math.Pi
				nz := -0.15 + rng.Float64()*0.84
				mode := ImpactInelastic
				if rng.Intn(5) == 0 {
					mode = ImpactStep
				}
				scale := 0.25 + rng.Float64()*2.5
				entries = append(entries, makeCradleUnitSlope(ang, nz, scale, mode))
			}
		}

		testSlots := makeTestSlots(entries)
		want := cache.hasSupportSmall(testSlots)
		cache.Reset()
		got := cache.HasSupport(testSlots)
		if got != want {
			t.Fatalf("trial %d (pattern=%d, n=%d): HasSupport=%v, want %v", trial, pattern, n, got, want)
		}
	}
}

func TestCageHasFloorCachingAndTessellation(t *testing.T) {
	obj := newTestDynamicObject(0, 0, 50, 20, 20, 40)
	cage := newTestCage(1, obj)
	cage.Rebuild(18.0, 1)

	if cage.HasFloor() {
		t.Fatal("empty cage unexpectedly reports HasFloor() == true")
	}

	// Add 1024 tessellated slots along a single slope direction (+X/+Y, nZ=0.50).
	// Use distinct IFace objects and XY contact offsets so the mesh simulates heavy tessellation > 128 slots.
	invSqrt2 := 1.0 / math.Sqrt2
	for i := 0; i < 1024; i++ {
		off := float64(i) * 5.0
		ang := 0.25 * math.Sin(float64(i)*0.1)
		nX := math.Cos(ang) * invSqrt2
		nY := math.Sin(ang) * invSqrt2
		f := newTestFace([3]float64{off, off, 0}, [3]float64{off + 1, off, 0}, [3]float64{off, off + 1, 0}, nX, nY, 0.50)
		cage.addToBucketEntry(BucketWallEast, nil, f, 0, 0.05, nX, nY, 0.50, off, off, 0, 10, ImpactInelastic)
	}
	if cage.GetSlotsLen() <= 128 {
		t.Fatalf("expected > 128 slots for high tessellation test, got %d", cage.GetSlotsLen())
	}

	if cage.HasFloor() {
		t.Fatal("one-sided 1024-slot slope falsely reported cradle floor support")
	}
	// Repeat call hits cache.
	if cage.HasFloor() {
		t.Fatal("cached HasFloor() changed unexpectedly")
	}

	// Add an opposite steep-compatible bumper -> invalidates cache -> HasFloor becomes true.
	oppFace := newTestFace([3]float64{-50, -50, 0}, [3]float64{-50, -49, 0}, [3]float64{-50, -50, 1}, -1, 0, 0)
	cage.addToBucketEntry(BucketWallWest, nil, oppFace, 0, 0.05, -1.0, 0.0, 0.0, -50, -50, 0, 10, ImpactInelastic)
	if !cage.HasFloor() {
		t.Fatal("expected cradle floor support after adding opposite bumper")
	}
	if !cage.HasFloor() {
		t.Fatal("cached HasFloor() == true became false")
	}

	// Rebuild invalidates cache and clears slots.
	cage.Rebuild(18.0, 2)
	if cage.HasFloor() {
		t.Fatal("expected HasFloor() == false immediately after Rebuild")
	}
}

func TestCradleSupportZeroAllocs(t *testing.T) {
	cache := NewCradleCache()
	slotsSlice := make([]*Entry, 256)
	for i := range slotsSlice {
		ang := -0.4 + 0.8*float64(i)/float64(len(slotsSlice))
		slotsSlice[i] = makeCradleUnitSlope(ang, 0.50, 1.0, ImpactInelastic)
	}
	slots := makeTestSlots(slotsSlice)

	// Phase A-only path: 0 allocs even on very first call without warmup.
	allocs := testing.AllocsPerRun(50, func() {
		if cache.HasSupport(slots) {
			t.Fatal("unexpected true on one-sided arc")
		}
	})
	if allocs != 0 {
		t.Fatalf("cache.HasSupport(256 slots, Phase A) allocated %v times, want 0", allocs)
	}

	// Phase B hairline fallback: 0 allocs after warmup.
	hairline := makeTestSlots(makeHairlineBenchmarkSlots(256))
	if cache.HasSupport(hairline) {
		t.Fatal("unexpected true on sub-threshold hairline wedge")
	}
	hairAllocs := testing.AllocsPerRun(50, func() {
		if cache.HasSupport(hairline) {
			t.Fatal("unexpected true on sub-threshold hairline wedge")
		}
	})
	if hairAllocs != 0 {
		t.Fatalf("cache.HasSupport(256 slots, Phase B warmup) allocated %v times, want 0", hairAllocs)
	}

	obj := newTestDynamicObject(0, 0, 50, 20, 20, 40)
	dynCage := newTestCage(1, obj)
	dynCage.Rebuild(18.0, 1)
	for i := 0; i < 256; i++ {
		off := float64(i) * 5.0
		f := newTestFace([3]float64{off, 0, 0}, [3]float64{off + 1, 0, 0}, [3]float64{off, 1, 0}, 0.86, 0, 0.50)
		dynCage.addToBucketEntry(BucketWallEast, nil, f, 0, 0.05, 0.86, 0.0, 0.50, off, 0, 0, 10, ImpactInelastic)
	}
	_ = dynCage.HasFloor() // populate cache
	cageAllocs := testing.AllocsPerRun(50, func() {
		if dynCage.HasFloor() {
			t.Fatal("unexpected floor")
		}
	})
	if cageAllocs != 0 {
		t.Fatalf("cached dynCage.HasFloor() allocated %v times, want 0", cageAllocs)
	}
}

func makeBenchmarkSlots(n int, coplanar bool, includeBumper bool) []*Entry {
	slots := make([]*Entry, n)
	for i := 0; i < n; i++ {
		var ang, nz float64
		if coplanar {
			ang = float64(i%4) * 0.15
			nz = 0.50
		} else {
			// Single-sided 75-degree arc: worst case scanning all entries with false return.
			ang = float64(i) * 1.25 / float64(n)
			nz = 0.50
			if includeBumper && i%3 == 0 {
				nz = 0.05
			}
		}
		slots[i] = makeCradleUnitSlope(ang, nz, 1.0, ImpactInelastic)
	}
	return slots
}

func makeHairlineBenchmarkSlots(n int) []*Entry {
	slots := make([]*Entry, n)
	// Sub-threshold hairline wedge (~94 deg) forcing Phase B ambiguous-sector check while remaining false.
	halfSpan := (math.Acos(-0.10) - 0.03) * 0.5
	for i := 0; i < n; i++ {
		ang := halfSpan
		if i%2 == 0 {
			ang = -halfSpan + 1e-4*float64(i%8)
		} else {
			ang = halfSpan - 1e-4*float64(i%8)
		}
		slots[i] = makeCradleUnitSlope(ang, 0.50, 1.0, ImpactInelastic)
	}
	return slots
}

func BenchmarkCradleSupport(b *testing.B) {
	for _, n := range []int{128, 256, 1024} {
		for _, mode := range []struct {
			name     string
			coplanar bool
		}{
			{name: "Coplanar", coplanar: true},
			{name: "CurvedArc", coplanar: false},
		} {
			slots := makeTestSlots(makeBenchmarkSlots(n, mode.coplanar, true))

			b.Run(fmt.Sprintf("Old_O_N2/%s/N=%d", mode.name, n), func(b *testing.B) {
				cache := NewCradleCache()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_ = cache.hasSupportSmall(slots)
				}
			})

			b.Run(fmt.Sprintf("New_O_N/%s/N=%d", mode.name, n), func(b *testing.B) {
				cache := NewCradleCache()
				_ = cache.HasSupport(slots)
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_ = cache.HasSupport(slots)
				}
			})
		}

		hairSlots := makeTestSlots(makeHairlineBenchmarkSlots(n))
		b.Run(fmt.Sprintf("New_O_N/HairlinePhaseB/N=%d", n), func(b *testing.B) {
			cache := NewCradleCache()
			_ = cache.HasSupport(hairSlots)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = cache.HasSupport(hairSlots)
			}
		})
	}
}
