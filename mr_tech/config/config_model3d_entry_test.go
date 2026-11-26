package config

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

func makeBoxTri(mat *Material, minPt, maxPt geometry.XYZ) []Model3DEntryTriangle {
	t := NewModel3DEntryTriangle(mat)
	t.Vertices[0].Pos = minPt
	t.Vertices[1].Pos = geometry.XYZ{X: maxPt.X, Y: minPt.Y, Z: (minPt.Z + maxPt.Z) * 0.5}
	t.Vertices[2].Pos = maxPt
	return []Model3DEntryTriangle{t}
}

func TestModel3DEntryCenterWithBoundsUsesReferenceFrameAllAxes(t *testing.T) {
	mat := NewConfigMaterial("mat", []string{"t"}, MaterialKindSolid)
	m := NewModel3DEntry(3, []string{"attak01", "stand01", "death01"}, []string{"death"})

	// Frame 0 (attak01) is shorter and shifted in X/Z compared to stand01.
	m.Frames[0] = NewModel3DEntryFrame(makeBoxTri(mat, geometry.XYZ{X: 10, Y: -5, Z: 10}, geometry.XYZ{X: 20, Y: 5, Z: 30}))
	// Frame 1 (stand01) spans X in [-10..10], Y in [-10..10], Z in [-24..31] (height = 55).
	m.Frames[1] = NewModel3DEntryFrame(makeBoxTri(mat, geometry.XYZ{X: -10, Y: -10, Z: -24}, geometry.XYZ{X: 10, Y: 10, Z: 31}))
	// Frame 2 (death01) is flattened near the floor.
	m.Frames[2] = NewModel3DEntryFrame(makeBoxTri(mat, geometry.XYZ{X: -10, Y: -10, Z: -26}, geometry.XYZ{X: 10, Y: 10, Z: -10}))

	center, size := m.CenterWithBounds([]string{"stand", "idle", "legs_idle", "hover"})
	if math.Abs(center.X) > 1e-6 || math.Abs(center.Y) > 1e-6 || math.Abs(center.Z-3.5) > 1e-6 {
		t.Fatalf("expected center from stand01 = (0,0,3.5), got (%v,%v,%v)", center.X, center.Y, center.Z)
	}
	if math.Abs(size.Z-55.0) > 1e-6 {
		t.Fatalf("expected height 55.0 from stand01, got %v", size.Z)
	}

	minZ, maxZ := 1e9, -1e9
	for _, v := range m.Frames[1].Triangles[0].Vertices {
		if v.Pos.Z < minZ {
			minZ = v.Pos.Z
		}
		if v.Pos.Z > maxZ {
			maxZ = v.Pos.Z
		}
	}
	if math.Abs(minZ+27.5) > 1e-6 || math.Abs(maxZ-27.5) > 1e-6 {
		t.Fatalf("expected stand01 centered at [-27.5..+27.5], got [%v..%v]", minZ, maxZ)
	}
}
