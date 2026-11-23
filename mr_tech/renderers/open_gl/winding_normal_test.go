package open_gl

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func TestFaceOutwardWindingAndCreaseNormalsAndTangents(t *testing.T) {
	tex := textures.NewTexture("TEST", 1, 64, 64, false)
	mat := textures.NewMaterial("brick", []*textures.Texture{tex}, int(config.MaterialKindSolid))

	// Face 0: inverted clockwise winding relative to expected outward normal (0, 0, 1).
	// Expected outward normal = +Z, initial vertices give (p1-p0)x(p2-p0) = (0,10,0)x(10,0,0) = (0,0,-100).
	f0 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 10, Y: 0, Z: 0},
	}, "f0", mat)
	f0.SetUV(0, 0, 0, 1, 1, 0)
	flipped := f0.EnsureOutwardWinding(geometry.XYZ{X: 0, Y: 0, Z: 1})
	if !flipped {
		t.Fatalf("expected EnsureOutwardWinding to flip inverted triangle")
	}
	// Calling again must be idempotent.
	if f0.EnsureOutwardWinding(geometry.XYZ{X: 0, Y: 0, Z: 1}) {
		t.Fatalf("expected second EnsureOutwardWinding call to be a no-op")
	}

	// Build a crease-angle test:
	// Face A lies in Z=0 plane, normal = (0, 0, 1).
	// Face B meets A along edge (0,0,0)-(10,0,0) tilted 45 deg: normal = (0, -sin45, cos45), dot(A,B) = 0.707 > 0.5 -> SMOOTH!
	// Face C meets A along edge (0,0,0)-(0,10,0) at 90 deg: normal = (-1, 0, 0), dot(A,C) = 0 < 0.5 -> HARD CREASE!
	invSqrt2 := 1.0 / math.Sqrt2
	fA := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, "A", mat)
	fA.SetUV(0, 0, 1, 0, 0, 1)

	fB := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: -10 * invSqrt2, Z: -10 * invSqrt2},
		{X: 10, Y: 0, Z: 0},
	}, "B", mat)
	fB.SetUV(0, 0, 1, 1, 1, 0)

	fC := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: -10},
	}, "C", mat)
	fC.SetUV(0, 0, 1, 0, 1, 1)

	faces := volumes.NewFaces(4, false)
	faces.Add(fA)
	faces.Add(fB)
	faces.Add(fC)
	faces.Rebuild()

	// Vertex 0 (0,0,0) on fA is shared with fB (dot = ~0.707 >= 0.5) and fC (dot = 0 < 0.5).
	// So fA's vertex 0 must blend A + B only (positive Z < 1 and negative Y), excluding C (-X)!
	nx0, ny0, nz0, _, _, _, _, _, _ := fA.GetVertexNormalsFlat()
	if math.Abs(nx0) > 1e-6 {
		t.Fatalf("expected 90-degree crease neighbor fC (-X) excluded from fA vertex 0 normal, got nx0=%.6f", nx0)
	}
	if !(ny0 < -0.1 && nz0 > 0.7 && nz0 < 0.99) {
		t.Fatalf("expected 45-degree neighbor fB smoothed into fA vertex 0 normal, got (%v, %v, %v)", nx0, ny0, nz0)
	}

	// Hard corner on fC vertex 2 (0,10,-10) belongs only to fC -> pure (-1, 0, 0).
	_, _, _, _, _, _, cnx2, cny2, cnz2 := fC.GetVertexNormalsFlat()
	if math.Abs(cnx2+1.0) > 1e-4 || math.Abs(cny2) > 1e-4 || math.Abs(cnz2) > 1e-4 {
		t.Fatalf("expected hard normal (-1,0,0) on fC corner, got (%v, %v, %v)", cnx2, cny2, cnz2)
	}

	// Verify GL (u, -v) tangent & handedness sign + ResidentMesh 24-float GPU vertex packing.
	fA.SetLightmap(2, 0.25, 0.75, 0.5, 0.75, 0.25, 0.5)
	ctx := &noopRenderCtx{}
	rm := gpu.NewResidentMesh(ctx)
	ok := rm.Rebuild(faces, func(_ *textures.Texture) (float32, bool) { return 3.0, true })
	if !ok || len(rm.VertFloats) == 0 || len(rm.VertFloats)%gpu.StaticVertexFloats != 0 {
		t.Fatalf("expected valid 24-float ResidentMesh vertex buffer, ok=%v floats=%d", ok, len(rm.VertFloats))
	}

	// Vertex 2 of fA (0,10,0): u=0, v=1 -> v_gl = -1. Check UV flip & tangent handedness sign non-zero & normalized.
	tx0, ty0, tz0, s0, _, _, _, _, _, _, _, _ := fA.GetTangentsFlat()
	tLen := math.Sqrt(tx0*tx0 + ty0*ty0 + tz0*tz0)
	if math.Abs(tLen-1.0) > 1e-4 || (s0 != 1.0 && s0 != -1.0) {
		t.Fatalf("expected unit tangent and ±1 handedness sign on fA, got T=(%v,%v,%v) len=%.5f sign=%.2f", tx0, ty0, tz0, tLen, s0)
	}

	// Verify ResidentMesh uploaded GL normal (nx, nz, -ny) and lightmap (0.25, 0.75, 2) for fA vertex 0.
	vf := rm.VertFloats[:gpu.StaticVertexFloats]
	if math.Abs(float64(vf[13]-0.25)) > 1e-5 || math.Abs(float64(vf[14]-0.75)) > 1e-5 || math.Abs(float64(vf[15]-2.0)) > 1e-5 {
		t.Fatalf("unexpected lightmap attribute in ResidentMesh vertex: (%v, %v, %v)", vf[13], vf[14], vf[15])
	}
	if math.Abs(float64(vf[6])-nx0) > 1e-4 || math.Abs(float64(vf[7])-nz0) > 1e-4 || math.Abs(float64(vf[8])+ny0) > 1e-4 {
		t.Fatalf("expected GL normal transform (nx, nz, -ny)=(%v,%v,%v), got (%v,%v,%v)", nx0, nz0, -ny0, vf[6], vf[7], vf[8])
	}
}

func TestResidentMeshAndFrameVerticesZeroAllocsSteadyState(t *testing.T) {
	tA := textures.NewTexture("ANIM_A", 1, 64, 64, false)
	tB := textures.NewTexture("ANIM_B", 1, 64, 64, false)
	animMat := textures.NewMaterial("anim", []*textures.Texture{tA, tB}, int(config.MaterialKindSolid))

	f := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, "anim_face", animMat)
	f.SetUV(0, 0, 1, 0, 0, 1)
	faces := volumes.NewFaces(2, false)
	faces.Add(f)
	faces.Rebuild()

	ctx := &noopRenderCtx{}
	rm := gpu.NewResidentMesh(ctx)
	texFn := func(_ *textures.Texture) (float32, bool) { return 1.0, true }
	if !rm.Rebuild(faces, texFn) {
		t.Fatalf("expected initial ResidentMesh.Rebuild to succeed")
	}
	subCapLen := len(rm.SubMeshes)
	animCapLen := len(rm.AnimMats)

	rmAllocs := testing.AllocsPerRun(50, func() {
		if !rm.Rebuild(faces, texFn) {
			t.Fatalf("expected steady-state ResidentMesh.Rebuild to succeed")
		}
	})
	if rmAllocs != 0 {
		t.Fatalf("expected 0 heap allocations on steady-state ResidentMesh.Rebuild, got %.2f", rmAllocs)
	}
	if len(rm.SubMeshes) != subCapLen || len(rm.AnimMats) != animCapLen || rm.SubMeshesLen != 1 || rm.AnimLen != 1 {
		t.Fatalf("unexpected slice cap/count after rebuild: subLen=%d/%d animLen=%d/%d", rm.SubMeshesLen, len(rm.SubMeshes), rm.AnimLen, len(rm.AnimMats))
	}

	fv := gpu.NewStagingFrameVertices(64)
	fv.AddAnimRef(animMat, tA)
	fv.AddBatchCentroid(config.BlendModeOpaque, 3, animMat, 1, 2, 3)

	fvAllocs := testing.AllocsPerRun(50, func() {
		fv.DeepReset()
		fv.AddAnimRef(animMat, tA)
		fv.AddBatchCentroid(config.BlendModeOpaque, 3, animMat, 1, 2, 3)
	})
	if fvAllocs != 0 {
		t.Fatalf("expected 0 heap allocations on steady-state FrameVertices cycle, got %.2f", fvAllocs)
	}
}
