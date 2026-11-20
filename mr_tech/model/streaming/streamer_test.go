package streaming

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

func TestSectorCoordConversionsAndDistances(t *testing.T) {
	const cellSize = 256.0

	// Positive coordinates
	c1 := WorldToSector(100.0, 300.0, cellSize)
	if c1.X != 0 || c1.Y != 1 {
		t.Fatalf("expected (0, 1), got (%d, %d)", c1.X, c1.Y)
	}

	// Negative coordinates
	c2 := WorldToSector(-50.0, -300.0, cellSize)
	if c2.X != -1 || c2.Y != -2 {
		t.Fatalf("expected (-1, -2), got (%d, %d)", c2.X, c2.Y)
	}

	// Sector bounds
	minX, minY, maxX, maxY := c1.SectorToWorldBounds(cellSize)
	if minX != 0.0 || minY != 256.0 || maxX != 256.0 || maxY != 512.0 {
		t.Fatalf("unexpected bounds: (%.1f, %.1f) - (%.1f, %.1f)", minX, minY, maxX, maxY)
	}

	// Distances
	cA := SectorCoord{X: 0, Y: 0}
	cB := SectorCoord{X: 3, Y: 2}
	if cA.ChebyshevDistance(cB) != 3 {
		t.Fatalf("expected Chebyshev distance 3, got %d", cA.ChebyshevDistance(cB))
	}
	if cA.ManhattanDistance(cB) != 5 {
		t.Fatalf("expected Manhattan distance 5, got %d", cA.ManhattanDistance(cB))
	}
	if cA.DistanceSq(cB) != 13 {
		t.Fatalf("expected DistanceSq 13, got %d", cA.DistanceSq(cB))
	}
}

func TestStreamingRingActivationAndHysteresis(t *testing.T) {
	const cellSize = 256.0
	tree := aabb.NewTree(1024, 2.0)
	streamer := NewWorldStreamer(tree, cellSize, 2, 3, 5)

	// Create an 11x11 city grid of sectors (coords -5 to +5)
	for y := -5; y <= 5; y++ {
		for x := -5; x <= 5; x++ {
			coord := SectorCoord{X: x, Y: y}
			sec := NewSector(coord, cellSize)
			// Mock volume to test tree insertion
			vol := volumes.NewVolume("vol_"+string(rune(x+100))+"_"+string(rune(y+100)), nil, nil, nil)
			sec.AddVolume(vol)
			streamer.RegisterSector(sec)
		}
	}

	// 1. Initial player placement at origin (0, 0)
	streamer.ForceSync(0, 0, 0)

	// With loadRadius = 2, a 5x5 window around (0, 0) must be active = 25 sectors
	if active := streamer.GetActiveSectorCount(); active != 25 {
		t.Fatalf("expected 25 active sectors, got %d", active)
	}

	// 2. Small movement within cell (100, 100) -> same sector (0, 0)
	streamer.Update(100.0, 100.0, 0)
	if active := streamer.GetActiveSectorCount(); active != 25 {
		t.Fatalf("expected 25 active sectors after small move, got %d", active)
	}

	// 3. Move east to sector (1, 0) -> World X = 300.0
	// Load radius = 2 around (1, 0) activates X in [-1..3], Y in [-2..2]
	streamer.ForceSync(300.0, 0, 0)

	// Sectors at X = -2 are at ChebyshevDistance = 3 from (1, 0).
	// Because unloadRadius = 3, sectors at distance 3 remain loaded due to hysteresis!
	if active := streamer.GetActiveSectorCount(); active < 25 {
		t.Fatalf("hysteresis buffer failed, expected >= 25 active sectors, got %d", active)
	}

	// 4. Large vehicle drive to sector (4, 0) -> World X = 1100.0
	streamer.ForceSync(1100.0, 0, 0)

	// Origin sector (0, 0) is now distance 4 > unloadRadius (3) -> MUST BE EVICTED
	originSec := streamer.GetSector(SectorCoord{X: 0, Y: 0})
	if originSec.State == SectorStateActive {
		t.Fatalf("expected origin sector to be evicted after long drive, but is still active")
	}

	targetSec := streamer.GetSector(SectorCoord{X: 4, Y: 0})
	if targetSec.State != SectorStateActive {
		t.Fatalf("expected destination sector (4, 0) to be active")
	}
}

func TestStreamingTimeSlicing(t *testing.T) {
	const cellSize = 256.0
	tree := aabb.NewTree(1024, 2.0)
	streamer := NewWorldStreamer(tree, cellSize, 2, 3, 5)
	streamer.SetMaxOpsPerTick(3) // Process at most 3 sectors per tick

	for y := -2; y <= 2; y++ {
		for x := -2; x <= 2; x++ {
			coord := SectorCoord{X: x, Y: y}
			sec := NewSector(coord, cellSize)
			streamer.RegisterSector(sec)
		}
	}

	// Player appears at origin: 25 sectors requested
	streamer.Update(0, 0, 0)

	// After 1st tick: exactly 3 sectors activated
	if active := streamer.GetActiveSectorCount(); active != 3 {
		t.Fatalf("expected exactly 3 sectors activated on tick 1, got %d", active)
	}

	// After 2nd tick: 6 sectors activated
	streamer.Update(0, 0, 0)
	if active := streamer.GetActiveSectorCount(); active != 6 {
		t.Fatalf("expected 6 sectors activated on tick 2, got %d", active)
	}
}
