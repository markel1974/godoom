package volumes

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

func TestFaceDeform(t *testing.T) {
	// A horizontal triangle on Z=0 facing upwards (+Z normal)
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}
	face := NewFace(tri, "test_floor", nil)

	nx, ny, nz := face.GetNormal()
	if math.Abs(nz-1.0) > 1e-6 || math.Abs(nx) > 1e-6 || math.Abs(ny) > 1e-6 {
		t.Fatalf("expected initial normal (0, 0, 1), got (%f, %f, %f)", nx, ny, nz)
	}

	// Impact near vertex 0 at (0, 0, 0) pushing downwards along -Z
	impact := geometry.XYZ{X: 0, Y: 0, Z: 0}
	impulse := geometry.XYZ{X: 0, Y: 0, Z: -5.0}
	radius := 6.0

	deformed := face.Deform(impact, impulse, radius)
	if !deformed {
		t.Fatalf("expected face to be deformed")
	}

	p0x, p0y, p0z := face.Get0()
	p1x, p1y, p1z := face.Get1()
	p2x, p2y, p2z := face.Get2()

	// Vertex 0 was at distance 0, so smoothstep falloff factor is 1.0 -> Z should be -5.0
	if math.Abs(p0z-(-5.0)) > 1e-4 {
		t.Errorf("vertex 0 Z expected -5.0, got %f", p0z)
	}

	// Vertex 1 at (10, 0, 0) is at distance 10 > radius 6.0 -> unaffected
	if p1x != 10.0 || p1y != 0.0 || p1z != 0.0 {
		t.Errorf("vertex 1 should be unaffected, got (%f, %f, %f)", p1x, p1y, p1z)
	}

	// Vertex 2 at (0, 10, 0) is at distance 10 > radius 6.0 -> unaffected
	if p2x != 0.0 || p2y != 10.0 || p2z != 0.0 {
		t.Errorf("vertex 2 should be unaffected, got (%f, %f, %f)", p2x, p2y, p2z)
	}

	// Normal should have been recomputed by Rebuild()
	newNx, newNy, newNz := face.GetNormal()
	if newNx == nx && newNy == ny && newNz == nz {
		t.Errorf("expected normal to change after deformation")
	}

	// AABB should encompass the deformed vertex Z = -5.0
	aabbMinX, aabbMinY, aabbMinZ, _, _, _ := face.GetAABB().GetBounds()
	if aabbMinZ > -5.0 {
		t.Errorf("expected AABB minZ <= -5.0, got %f", aabbMinZ)
	}
	_ = aabbMinX
	_ = aabbMinY
}

func TestFaceDeformDent(t *testing.T) {
	// A vertical wall on X=0 facing +X
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 0, Y: 0, Z: 10},
	}
	face := NewFace(tri, "test_wall", nil)

	vol := NewVolumeConcrete(1, "vol_test", "wall")
	face.SetParent(vol)
	vol.SetDirty(false)

	// Impact near vertex 0 at (0, 0, 0) creating a dent of depth 3.0 with radius 5.0
	impact := geometry.XYZ{X: 0, Y: 0, Z: 0}
	deformed := face.DeformDent(impact, 3.0, 5.0)
	if !deformed {
		t.Fatalf("expected face to be deformed")
	}

	// The wall normal was facing +X, so negative normal is -X -> X should be -3.0
	p0x, _, _ := face.Get0()
	if math.Abs(p0x-(-3.0)) > 1e-4 {
		t.Errorf("vertex 0 X expected -3.0 after dent, got %f", p0x)
	}

	// Parent volume should have been marked dirty
	if !vol.IsDirty() {
		t.Errorf("expected parent volume to be marked dirty after deformation")
	}
}

func TestFaceIntersectRay(t *testing.T) {
	// Triangle on XY plane at Z=0, facing +Z
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}
	face := NewFace(tri, "floor", nil)

	// Ray from (2, 2, 10) pointing straight down (0, 0, -1)
	dist, hit := face.IntersectRay(2, 2, 10, 0, 0, -1, 100)
	if !hit {
		t.Fatalf("expected ray to hit the face")
	}
	if math.Abs(dist-10.0) > 1e-4 {
		t.Errorf("expected hit distance 10.0, got %f", dist)
	}

	// Ray pointing away (0, 0, 1) should not hit
	_, hitAway := face.IntersectRay(2, 2, 10, 0, 0, 1, 100)
	if hitAway {
		t.Errorf("ray pointing away should not hit")
	}

	// Ray outside triangle (20, 20, 10) pointing down should not hit
	_, hitOut := face.IntersectRay(20, 20, 10, 0, 0, -1, 100)
	if hitOut {
		t.Errorf("ray outside triangle should not hit")
	}
}

func TestVolumeDeformDent(t *testing.T) {
	// A quad composed of two triangles sharing the hypotenuse edge
	t1 := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}
	t2 := [3]geometry.XYZ{
		{X: 10, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}

	vol := NewVolumeConcrete(1, "vol_quad", "floor")
	f1 := NewFace(t1, "f1", nil)
	f2 := NewFace(t2, "f2", nil)
	f1.SetParent(vol)
	f2.SetParent(vol)

	vol.faces.container = append(vol.faces.container, f1, f2)
	vol.faces.count = 2
	vol.Rebuild()
	vol.SetDirty(false)

	// Impact right on the shared edge near (0, 10, 0)
	impact := geometry.XYZ{X: 0, Y: 10, Z: 0}
	normal := geometry.XYZ{X: 0, Y: 0, Z: 1}
	deformed := vol.DeformDent(impact, normal, 4.0, 6.0)
	if !deformed {
		t.Fatalf("expected volume to be deformed")
	}

	// Both faces sharing (0, 10, 0) should have their vertex deformed down along -Z
	_, _, f1z := f1.Get2() // point 2 in t1 is (0, 10, 0)
	_, _, f2z := f2.Get2() // point 2 in t2 is (0, 10, 0)

	if math.Abs(f1z-(-4.0)) > 1e-4 {
		t.Errorf("expected f1 vertex 2 Z -4.0, got %f", f1z)
	}
	if math.Abs(f2z-(-4.0)) > 1e-4 {
		t.Errorf("expected f2 vertex 2 Z -4.0, got %f", f2z)
	}
	if !vol.IsDirty() {
		t.Errorf("volume should be marked dirty")
	}
}

func TestLargePolygonBarycentricDeform(t *testing.T) {
	// A massive triangle (500x500 units)
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 500, Y: 0, Z: 0},
		{X: 0, Y: 500, Z: 0},
	}
	face := NewFace(tri, "huge_floor", nil)

	// Impact in the middle at (100, 100, 0), where distance to all vertices is > 100 units
	impact := geometry.XYZ{X: 100, Y: 100, Z: 0}
	impulse := geometry.XYZ{X: 0, Y: 0, Z: -10.0}
	radius := 16.0 // Small radius: no vertex is within 16 units!

	deformed := face.Deform(impact, impulse, radius)
	if !deformed {
		t.Fatalf("expected large face to be deformed via barycentric fallback")
	}

	// Vertices should have moved downwards along Z proportional to barycentric weights
	_, _, p0z := face.Get0()
	_, _, p1z := face.Get1()
	_, _, p2z := face.Get2()

	if p0z >= 0.0 || p1z >= 0.0 || p2z >= 0.0 {
		t.Errorf("expected all vertices of large polygon to be dented downward, got (%f, %f, %f)", p0z, p1z, p2z)
	}

	// P0 at (0, 0) is closest to (100, 100) than P1 (500, 0) or P2 (0, 500), so |p0z| should be largest
	if math.Abs(p0z) <= math.Abs(p1z) || math.Abs(p0z) <= math.Abs(p2z) {
		t.Errorf("expected p0 to have largest displacement, got p0z=%f, p1z=%f, p2z=%f", p0z, p1z, p2z)
	}
}

func TestFacesTessellateMinEdgeAndBuffers(t *testing.T) {
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}
	f := NewFaces(16, false)
	f.Add(NewFace(tri, "floor", nil))

	impact := geometry.XYZ{X: 0, Y: 0, Z: 0}

	// First tessellation with radius 6.0 and target edge 2.0
	added1, err := f.TessellateNear(impact, 6.0, 2.0, deformMaxFaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added1 == 0 {
		t.Fatalf("expected faces to be added during first subdivision")
	}

	countAfterFirst := f.Len()

	// Second tessellation requesting micro-edges smaller than deformMinEdge
	// deformMinEdge (1.0) must prevent infinite or sub-unit subdivision
	added2, err := f.TessellateNear(impact, 6.0, 0.1, deformMaxFaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = added2

	// Check that no edge in the mesh has length < deformMinEdge (unless it was already smaller originally)
	for i := 0; i < f.Len(); i++ {
		face := f.At(i)
		p := face.tri.Points
		edges := [3][2]geometry.XYZ{{p[0], p[1]}, {p[1], p[2]}, {p[2], p[0]}}
		for _, e := range edges {
			lenEdge := math.Hypot(math.Hypot(e[0].X-e[1].X, e[0].Y-e[1].Y), e[0].Z-e[1].Z)
			// Midpoint split can produce diagonal edges slightly under 1.0 (e.g. 1.0/sqrt(2) = 0.707)
			// but must never produce sub-millimetric slivers (e.g. < 0.1)
			if lenEdge < 0.5 {
				t.Errorf("found degenerate sliver edge of length %f", lenEdge)
			}
		}
	}

	// Running TessellateNear again with the same parameters should add 0 faces (convergence)
	added3, err := f.TessellateNear(impact, 6.0, 1.0, deformMaxFaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added3 != 0 {
		t.Errorf("expected 0 additional faces once converged, got %d", added3)
	}

	_ = countAfterFirst
}
