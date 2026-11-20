package model

import (
	"fmt"
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/streaming"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func createTestVolumeAt(id string, cx, cy, cz, size float64) *volumes.Volume {
	mat := textures.NewMaterial("solid", nil, 0)
	vol := volumes.NewVolumeConcrete(0, id, "test")

	h := size * 0.5
	// Simple cube/box made of triangular faces
	v0 := geometry.XYZ{X: cx - h, Y: cy - h, Z: cz - h}
	v1 := geometry.XYZ{X: cx + h, Y: cy - h, Z: cz - h}
	v2 := geometry.XYZ{X: cx + h, Y: cy + h, Z: cz - h}
	v3 := geometry.XYZ{X: cx - h, Y: cy + h, Z: cz - h}
	v4 := geometry.XYZ{X: cx - h, Y: cy - h, Z: cz + h}
	v5 := geometry.XYZ{X: cx + h, Y: cy - h, Z: cz + h}
	v6 := geometry.XYZ{X: cx + h, Y: cy + h, Z: cz + h}
	v7 := geometry.XYZ{X: cx - h, Y: cy + h, Z: cz + h}

	// Floor
	vol.AddFace(volumes.NewFace([3]geometry.XYZ{v0, v1, v2}, "solid", mat))
	vol.AddFace(volumes.NewFace([3]geometry.XYZ{v0, v2, v3}, "solid", mat))
	// Ceiling
	vol.AddFace(volumes.NewFace([3]geometry.XYZ{v4, v6, v5}, "solid", mat))
	vol.AddFace(volumes.NewFace([3]geometry.XYZ{v4, v7, v6}, "solid", mat))

	_ = vol.Setup(vol)
	_ = vol.Rebuild()
	return vol
}

func TestVolumesStreamingIntegration(t *testing.T) {
	// Create a 5x5 grid of volumes spaced by 256 units
	const cellSize = 256.0
	var allVols []*volumes.Volume

	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			id := fmt.Sprintf("vol_%d_%d", dx, dy)
			cx := float64(dx)*cellSize + cellSize*0.5
			cy := float64(dy)*cellSize + cellSize*0.5
			vol := createTestVolumeAt(id, cx, cy, 0.0, 50.0)
			allVols = append(allVols, vol)
		}
	}

	vs := NewVolumesFromSlice(allVols)
	if err := vs.Setup(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Without streaming: all 25 volumes are immediately in the tree
	if vs.HasStreamer() {
		t.Fatalf("expected HasStreamer() to be false initially")
	}

	// Enable streaming and auto-sectorize
	ws := vs.EnableStreaming(cellSize, 1, 2, 3)
	if !vs.HasStreamer() || vs.GetStreamer() != ws {
		t.Fatalf("expected streamer to be active")
	}

	vs.AutoSectorize(cellSize)

	// Streamer should have 25 registered sectors (from -2..2 in X and Y)
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			coord := streaming.SectorCoord{X: dx, Y: dy}
			sec := ws.GetSector(coord)
			if sec == nil {
				t.Fatalf("expected sector at %v to be registered", coord)
			}
			if sec.VolumesLen() != 1 {
				t.Fatalf("expected 1 volume in sector %v, got %d", coord, sec.VolumesLen())
			}
		}
	}

	// Initially, tree was cleared on EnableStreaming, active sectors = 0
	if ws.GetActiveSectorCount() != 0 {
		t.Fatalf("expected 0 active sectors before sync, got %d", ws.GetActiveSectorCount())
	}

	// Move viewer to center (0, 0, 0) - coord {0, 0}
	// With loadRadius = 1, active sectors should be a 3x3 block: 9 sectors
	vs.ForceSyncStreaming(10.0, 10.0, 0.0)

	if ws.GetActiveSectorCount() != 9 {
		t.Fatalf("expected 9 active sectors around center with loadRadius=1, got %d", ws.GetActiveSectorCount())
	}

	// Query at center should find the center volume
	centerVol := vs.GetVolume("vol_0_0")
	if centerVol == nil {
		t.Fatalf("expected vol_0_0 in cache")
	}
	overlaps := vs.Query(centerVol.GetAABB())
	if len(overlaps) == 0 {
		t.Fatalf("expected center volume to be queried in active tree")
	}

	// A volume far away ({2, 2}) is outside loadRadius (dist = 2 > loadRadius 1)
	// so it should NOT be in the active tree
	farVol := vs.GetVolume("vol_2_2")
	if farVol == nil {
		t.Fatalf("expected vol_2_2 in cache")
	}
	farOverlaps := vs.Query(farVol.GetAABB())
	if len(farOverlaps) != 0 {
		t.Fatalf("expected distant vol_2_2 to not be present in active tree, found %d overlaps", len(farOverlaps))
	}

	// Now move viewer to ({2, 2})
	viewerX := 2.0*cellSize + cellSize*0.5
	viewerY := 2.0*cellSize + cellSize*0.5
	vs.ForceSyncStreaming(viewerX, viewerY, 0.0)

	// Now vol_2_2 should be in the active tree
	farOverlapsAfterMove := vs.Query(farVol.GetAABB())
	if len(farOverlapsAfterMove) == 0 {
		t.Fatalf("expected vol_2_2 to be in active tree after moving viewer")
	}

	// And the old center vol_0_0 (Chebyshev distance = 2 > unloadRadius is not > 2, but dist = 2 > loadRadius)
	// unloadRadius is 2, Chebyshev dist between {0,0} and {2,2} is 2,
	// let's move further to {5, 5} to test complete eviction
	vs.ForceSyncStreaming(5.0*cellSize, 5.0*cellSize, 0.0)
	if ws.GetActiveSectorCount() != 0 {
		t.Fatalf("expected 0 active sectors when viewer is at {5,5}, got %d", ws.GetActiveSectorCount())
	}
}
