package volumes

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func TestFaceDeform(t *testing.T) {
	// A horizontal triangle on Z=0 facing upwards (+Z normal)
	tri := [3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}
	face := NewFace(tri, nil)

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

	_, _, p0z := face.Get0()
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
	face := NewFace(tri, nil)

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
	face := NewFace(tri, nil)

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
	f1 := NewFace(t1, nil)
	f2 := NewFace(t2, nil)
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
	face := NewFace(tri, nil)

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
	f := NewFaces(aabb.NewTree(16, 0.0), false)
	f.Add(NewFace(tri, nil))

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
		edges := [3][2]geometry.XYZ32{{p[0], p[1]}, {p[1], p[2]}, {p[2], p[0]}}
		for _, e := range edges {
			lenEdge := math.Hypot(math.Hypot(float64(e[0].X-e[1].X), float64(e[0].Y-e[1].Y)), float64(e[0].Z-e[1].Z))
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

func TestFacesRebuildActiveEdges(t *testing.T) {
	t.Run("coplanar_quad", func(t *testing.T) {
		f1 := NewFace([3]geometry.XYZ{
			{X: 0, Y: 0, Z: 0},
			{X: 10, Y: 0, Z: 0},
			{X: 0, Y: 10, Z: 0},
		}, nil)
		f2 := NewFace([3]geometry.XYZ{
			{X: 10, Y: 0, Z: 0},
			{X: 10, Y: 10, Z: 0},
			{X: 0, Y: 10, Z: 0},
		}, nil)

		faces := NewFaces(aabb.NewTree(4, 0.0), false)
		faces.Add(f1)
		faces.Add(f2)
		faces.Rebuild()

		// Shared diagonal: f1 edge 1 (BC) and f2 edge 2 (CA) must be inactive.
		if f1.IsEdgeActive(1) {
			t.Errorf("expected f1 shared diagonal edge 1 (BC) to be inactive, mask=%03b", f1.ActiveEdgeMask())
		}
		if f2.IsEdgeActive(2) {
			t.Errorf("expected f2 shared diagonal edge 2 (CA) to be inactive, mask=%03b", f2.ActiveEdgeMask())
		}
		// Outer boundary edges must stay active.
		if !f1.IsEdgeActive(0) || !f1.IsEdgeActive(2) {
			t.Errorf("expected f1 boundary edges 0 and 2 to remain active, mask=%03b", f1.ActiveEdgeMask())
		}
		if !f2.IsEdgeActive(0) || !f2.IsEdgeActive(1) {
			t.Errorf("expected f2 boundary edges 0 and 1 to remain active, mask=%03b", f2.ActiveEdgeMask())
		}
	})

	t.Run("concave_floor_wall_crease", func(t *testing.T) {
		// Floor faces +Z (y > 0), wall faces +Y (z > 0), forming an interior concave 90-degree crease at y=0, z=0.
		floorTri := NewFace([3]geometry.XYZ{
			{X: 0, Y: 0, Z: 0},
			{X: 10, Y: 0, Z: 0},
			{X: 0, Y: 10, Z: 0},
		}, nil)
		wallTri := NewFace([3]geometry.XYZ{
			{X: 10, Y: 0, Z: 0},
			{X: 0, Y: 0, Z: 0},
			{X: 0, Y: 0, Z: 10},
		}, nil)

		faces := NewFaces(aabb.NewTree(4, 0.0), false)
		faces.Add(floorTri)
		faces.Add(wallTri)
		faces.Rebuild()

		if floorTri.IsEdgeActive(0) {
			t.Errorf("expected concave shared edge 0 on floor to be inactive, mask=%03b", floorTri.ActiveEdgeMask())
		}
		if wallTri.IsEdgeActive(0) {
			t.Errorf("expected concave shared edge 0 on wall to be inactive, mask=%03b", wallTri.ActiveEdgeMask())
		}
		if !floorTri.IsEdgeActive(1) || !floorTri.IsEdgeActive(2) || !wallTri.IsEdgeActive(1) || !wallTri.IsEdgeActive(2) {
			t.Errorf("expected boundary edges to stay active, floor=%03b wall=%03b", floorTri.ActiveEdgeMask(), wallTri.ActiveEdgeMask())
		}
	})

	t.Run("convex_step_edge", func(t *testing.T) {
		// Top faces +Z (y > 0), front riser faces -Y (z < 10), forming an exterior convex 90-degree step edge at y=0, z=10.
		topTri := NewFace([3]geometry.XYZ{
			{X: 0, Y: 0, Z: 10},
			{X: 10, Y: 0, Z: 10},
			{X: 0, Y: 10, Z: 10},
		}, nil)
		frontTri := NewFace([3]geometry.XYZ{
			{X: 10, Y: 0, Z: 10},
			{X: 0, Y: 0, Z: 10},
			{X: 0, Y: 0, Z: 0},
		}, nil)

		faces := NewFaces(aabb.NewTree(4, 0.0), false)
		faces.Add(topTri)
		faces.Add(frontTri)
		faces.Rebuild()

		if topTri.ActiveEdgeMask() != EdgeAllActive {
			t.Errorf("expected all edges active on convex step top, got %03b", topTri.ActiveEdgeMask())
		}
		if frontTri.ActiveEdgeMask() != EdgeAllActive {
			t.Errorf("expected all edges active on convex step front, got %03b", frontTri.ActiveEdgeMask())
		}
	})
}

func TestVolumeDeformUpdatesActiveEdges(t *testing.T) {
	solidMat := textures.NewMaterial("stone", nil, int(config.MaterialKindSolid))

	// Part A: fixed topology toggling coplanar -> convex ridge -> concave valley.
	vol := NewVolumeConcrete(1, "deform_edges", "floor")
	vol.SetAllowTessellate(false)

	f1 := NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, solidMat)
	f2 := NewFace([3]geometry.XYZ{
		{X: 10, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, solidMat)
	vol.AddFace(f1)
	vol.AddFace(f2)
	vol.Rebuild()

	if f1.IsEdgeActive(1) || f2.IsEdgeActive(2) {
		t.Fatalf("expected initially coplanar shared diagonal to be inactive, f1=%03b f2=%03b", f1.ActiveEdgeMask(), f2.ActiveEdgeMask())
	}

	// Push vertex (0,10,0) upward along +Z to form a convex ridge along the shared diagonal.
	if !vol.Deform(geometry.XYZ{X: 0, Y: 10, Z: 0}, geometry.XYZ{X: 0, Y: 0, Z: 4}, 6.0) {
		t.Fatalf("expected upward bump deform to succeed")
	}
	if !f1.IsEdgeActive(1) || !f2.IsEdgeActive(2) {
		t.Errorf("expected convex ridge shared diagonal to become active after upward bump, f1=%03b f2=%03b", f1.ActiveEdgeMask(), f2.ActiveEdgeMask())
	}

	// Push vertex (0,10,4) downward into -Z (to Z=-4) to form a concave crease along the shared diagonal.
	if !vol.Deform(geometry.XYZ{X: 0, Y: 10, Z: 4}, geometry.XYZ{X: 0, Y: 0, Z: -8}, 6.0) {
		t.Fatalf("expected downward dent deform to succeed")
	}
	if f1.IsEdgeActive(1) || f2.IsEdgeActive(2) {
		t.Errorf("expected concave crease shared diagonal to become inactive after dent, f1=%03b f2=%03b", f1.ActiveEdgeMask(), f2.ActiveEdgeMask())
	}

	// Part B: with dynamic tessellation enabled, DeformDent subdivides and reclassifies all shared edges via Rebuild().
	volTess := NewVolumeConcrete(2, "deform_tess_edges", "floor")
	t1 := NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 40, Y: 0, Z: 0},
		{X: 0, Y: 40, Z: 0},
	}, solidMat)
	t2 := NewFace([3]geometry.XYZ{
		{X: 40, Y: 0, Z: 0},
		{X: 40, Y: 40, Z: 0},
		{X: 0, Y: 40, Z: 0},
	}, solidMat)
	volTess.AddFace(t1)
	volTess.AddFace(t2)
	volTess.Rebuild()

	if !volTess.DeformDent(geometry.XYZ{X: 20, Y: 20, Z: 0}, geometry.XYZ{X: 0, Y: 0, Z: 1}, 3.0, 12.0) {
		t.Fatalf("expected tessellated DeformDent to succeed")
	}
	if volTess.faces.Len() <= 2 {
		t.Fatalf("expected dynamic tessellation to split faces, got count=%d", volTess.faces.Len())
	}

	// Verify at least one shared internal edge became inactive (coplanar/concave) after tessellation+dent.
	inactiveCount := 0
	for i := 0; i < volTess.faces.Len(); i++ {
		fc := volTess.faces.At(i)
		if fc.ActiveEdgeMask() != EdgeAllActive {
			inactiveCount++
		}
	}
	if inactiveCount == 0 {
		t.Errorf("expected tessellated volume to classify coplanar/concave shared edges as inactive")
	}
}

func TestVolumeDeformLocalityAndIncrementalTreeSync(t *testing.T) {
	solidMat := textures.NewMaterial("stone", nil, int(config.MaterialKindSolid))
	vol := NewVolumeConcrete(3, "wall_grid", "wall")
	for ix := 0; ix < 10; ix++ {
		for iz := 0; iz < 10; iz++ {
			x0, x1 := float64(ix)*64.0, float64(ix+1)*64.0
			z0, z1 := float64(iz)*64.0, float64(iz+1)*64.0
			vol.AddFace(NewFace([3]geometry.XYZ{{X: x0, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z1}}, solidMat))
			vol.AddFace(NewFace([3]geometry.XYZ{{X: x0, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z1}, {X: x0, Y: 0, Z: z1}}, solidMat))
		}
	}
	vol.Rebuild()

	impact := geometry.XYZ{X: 320, Y: 0, Z: 320}
	normal := geometry.XYZ{X: 0, Y: -1, Z: 0}
	if !vol.DeformDent(impact, normal, 3.0, 15.0) {
		t.Fatalf("expected wall dent to deform")
	}
	if n := vol.GetFaceCount(); n <= 200 || n > 600 {
		t.Fatalf("expected localized crater tessellation cleanly bounded from 200 initial faces, got %d", n)
	}

	faces := vol.GetFaces()
	tree := faces.GetTree()
	for i := 0; i < faces.Len(); i++ {
		fc := faces.At(i)
		if !tree.ContainsObject(fc) {
			t.Fatalf("face %d missing from incremental tree", i)
		}
	}

	// Verify ray query directly against the deformed spot hits a dented face inside the crater.
	var hitFace *Face
	bestDist := 200.0
	tree.QueryRay(impact.X, -50, impact.Z, 0, 1, 0, bestDist, func(obj aabb.IObject, _ float64) (float64, bool) {
		fc, ok := obj.(*Face)
		if !ok {
			return bestDist, false
		}
		if d, hit := fc.IntersectRay(impact.X, -50, impact.Z, 0, 1, 0, bestDist); hit {
			bestDist = d
			hitFace = fc
			return d, true
		}
		return bestDist, false
	})
	if hitFace == nil {
		t.Fatalf("expected ray query to hit dented face in incremental AABB tree")
	}
	if math.Abs(bestDist-53.0) > 1.0 {
		t.Fatalf("expected hit distance near dented surface (~53), got %.3f", bestDist)
	}

	// Subsequent hit on the same spot must not explode faces or fail tree consistency.
	beforeSecond := faces.Len()
	if !vol.DeformDent(impact, normal, 2.0, 15.0) {
		t.Fatalf("expected second wall dent to succeed")
	}
	if faces.Len() > beforeSecond+32 {
		t.Fatalf("expected converged crater on repeat shot, grew from %d to %d", beforeSecond, faces.Len())
	}
	for i := 0; i < faces.Len(); i++ {
		if !tree.ContainsObject(faces.At(i)) {
			t.Fatalf("face %d missing from tree after repeat dent", i)
		}
	}
}

func BenchmarkVolumeDeformRepeat(b *testing.B) {
	solidMat := textures.NewMaterial("stone", nil, int(config.MaterialKindSolid))
	vol := NewVolumeConcrete(4, "bench_wall", "wall")
	for ix := 0; ix < 12; ix++ {
		for iz := 0; iz < 12; iz++ {
			x0, x1 := float64(ix)*64.0, float64(ix+1)*64.0
			z0, z1 := float64(iz)*64.0, float64(iz+1)*64.0
			vol.AddFace(NewFace([3]geometry.XYZ{{X: x0, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z1}}, solidMat))
			vol.AddFace(NewFace([3]geometry.XYZ{{X: x0, Y: 0, Z: z0}, {X: x1, Y: 0, Z: z1}, {X: x0, Y: 0, Z: z1}}, solidMat))
		}
	}
	vol.Rebuild()
	impact := geometry.XYZ{X: 350, Y: 0, Z: 350}
	normal := geometry.XYZ{X: 0, Y: -1, Z: 0}
	vol.DeformDent(impact, normal, 2.5, 15.0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		imp := geometry.XYZ{X: 350 + float64(i%5)*0.5, Y: 0, Z: 350}
		vol.DeformDent(imp, normal, 0.2, 15.0)
	}
}
