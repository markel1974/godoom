package open_gl

import (
	"math"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type windingTestCtx struct{ api.IContext }

func (windingTestCtx) Ptr(_ interface{}) unsafe.Pointer    { return nil }
func (windingTestCtx) PtrOffset(offset int) unsafe.Pointer { return unsafe.Pointer(uintptr(offset)) }
func (windingTestCtx) GenVertexArrays(_ int32, a *uint32) {
	if a != nil {
		*a = 1
	}
}
func (windingTestCtx) GenBuffers(_ int32, b *uint32) {
	if b != nil {
		*b = 1
	}
}
func (windingTestCtx) DeleteVertexArrays(_ int32, _ *uint32)                  {}
func (windingTestCtx) DeleteBuffers(_ int32, _ *uint32)                       {}
func (windingTestCtx) BindVertexArray(_ uint32)                               {}
func (windingTestCtx) BindBuffer(_, _ uint32)                                 {}
func (windingTestCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (windingTestCtx) BufferSubData(_ uint32, _, _ int, _ unsafe.Pointer)     {}
func (windingTestCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}
func (windingTestCtx) VertexAttribDivisor(_, _ uint32)  {}
func (windingTestCtx) EnableVertexAttribArray(_ uint32) {}

func TestFaceOutwardWindingAndCreaseNormalsAndTangents(t *testing.T) {
	tex := textures.NewTexture("TEST", 1, 64, 64, false)
	mat := textures.NewMaterial("brick", []*textures.Texture{tex}, int(config.MaterialKindSolid))

	// Face 0: inverted clockwise winding relative to expected outward normal (0, 0, 1).
	// Expected outward normal = +Z, initial vertices give (p1-p0)x(p2-p0) = (0,10,0)x(10,0,0) = (0,0,-100).
	f0 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 10, Y: 0, Z: 0},
	}, mat)
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
	}, mat)
	fA.SetUV(0, 0, 1, 0, 0, 1)

	fB := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: -10 * invSqrt2, Z: -10 * invSqrt2},
		{X: 10, Y: 0, Z: 0},
	}, mat)
	fB.SetUV(0, 0, 1, 1, 1, 0)

	fC := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: -10},
	}, mat)
	fC.SetUV(0, 0, 1, 0, 1, 1)

	faces := volumes.NewFaces(aabb.NewTree(4, 0.0), false)
	faces.Add(fA)
	faces.Add(fB)
	faces.Add(fC)
	faces.Rebuild()

	// Vertex 0 (0,0,0) on fA is shared with fB (dot = ~0.707 >= 0.5) and fC (dot = 0 < 0.5).
	// So fA's vertex 0 must blend A + B only (positive Z < 1 and negative Y), excluding C (-X)!
	nx0, ny0, nz0, _, _, _, _, _, _ := fA.GetVertexNormalsFlat()
	if math.Abs(float64(nx0)) > 1e-6 {
		t.Fatalf("expected 90-degree crease neighbor fC (-X) excluded from fA vertex 0 normal, got nx0=%.6f", nx0)
	}
	if !(ny0 < -0.1 && nz0 > 0.7 && nz0 < 0.99) {
		t.Fatalf("expected 45-degree neighbor fB smoothed into fA vertex 0 normal, got (%v, %v, %v)", nx0, ny0, nz0)
	}

	// Hard corner on fC vertex 2 (0,10,-10) belongs only to fC -> pure (-1, 0, 0).
	_, _, _, _, _, _, cnx2, cny2, cnz2 := fC.GetVertexNormalsFlat()
	if math.Abs(float64(cnx2)+1.0) > 1e-4 || math.Abs(float64(cny2)) > 1e-4 || math.Abs(float64(cnz2)) > 1e-4 {
		t.Fatalf("expected hard normal (-1,0,0) on fC corner, got (%v, %v, %v)", cnx2, cny2, cnz2)
	}

	// Verify GL (u, -v) tangent & handedness sign + compressed 16-word (64-byte) deduplicated GPU vertex packing.
	fA.SetLightmap(2, 0.25, 0.75, 0.5, 0.75, 0.25, 0.5)
	ctx := windingTestCtx{}
	rm := buffers.NewResidentMesh(ctx)
	ok := rm.Rebuild(faces, func(_ *textures.Texture) (float32, bool) { return 3.0, true })
	vdata := rm.VertexData()
	if !ok || len(vdata) == 0 || len(vdata)%buffers.StaticVertexWords != 0 {
		t.Fatalf("expected valid 16-word compressed ResidentMesh vertex buffer, ok=%v words=%d", ok, len(vdata))
	}

	// Vertex 2 of fA (0,10,0): u=0, v=1 -> v_gl = -1. Check UV flip & tangent handedness sign non-zero & normalized.
	tx0, ty0, tz0, s0, _, _, _, _, _, _, _, _ := fA.GetTangentsFlat()
	tLen := math.Sqrt(float64(tx0*tx0 + ty0*ty0 + tz0*tz0))
	if math.Abs(tLen-1.0) > 1e-4 || (s0 != 1.0 && s0 != -1.0) {
		t.Fatalf("expected unit tangent and ±1 handedness sign on fA, got T=(%v,%v,%v) len=%.5f sign=%.2f", tx0, ty0, tz0, tLen, s0)
	}

	// Verify ResidentMesh uploaded packed GL normal (nx, nz, -ny) and HALF_FLOAT lightmap (0.25, 0.75, 2) for fA vertex 0.
	vf := vdata[:buffers.StaticVertexWords]
	lmU0, lmV0 := buffers.UnpackHalf2(vf[8])
	lmID0, _ := buffers.UnpackHalf2(vf[9])
	if math.Abs(float64(lmU0-0.25)) > 1e-4 || math.Abs(float64(lmV0-0.75)) > 1e-4 || math.Abs(float64(lmID0-2.0)) > 1e-4 {
		t.Fatalf("unexpected HALF_FLOAT lightmap attribute in ResidentMesh vertex: (%v, %v, %v)", lmU0, lmV0, lmID0)
	}
	gnx, gny, gnz, _ := buffers.UnpackSnorm10x3_2(vf[6])
	if math.Abs(float64(gnx-nx0)) > 3e-3 || math.Abs(float64(gny-nz0)) > 3e-3 || math.Abs(float64(gnz+ny0)) > 3e-3 {
		t.Fatalf("expected packed GL normal transform (nx, nz, -ny)=(%v,%v,%v), got (%v,%v,%v)", nx0, nz0, -ny0, gnx, gny, gnz)
	}
	gtx, gty, gtz, gtw := buffers.UnpackSnorm10x3_2(vf[7])
	if math.Abs(float64(gtx-tx0)) > 3e-3 || math.Abs(float64(gty-tz0)) > 3e-3 || math.Abs(float64(gtz+ty0)) > 3e-3 || gtw != s0 {
		t.Fatalf("expected packed GL tangent (%v,%v,%v,%v), got (%v,%v,%v,%v)", tx0, tz0, -ty0, s0, gtx, gty, gtz, gtw)
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
	}, animMat)
	f.SetUV(0, 0, 1, 0, 0, 1)
	faces := volumes.NewFaces(aabb.NewTree(2, 0.0), false)
	faces.Add(f)
	faces.Rebuild()

	ctx := windingTestCtx{}
	rm := buffers.NewResidentMesh(ctx)
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

	fv := draws.NewStagingFrameVertices(64)
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

func TestFaceFloat32PresentationAndFloat64PhysicsBoundary(t *testing.T) {
	var _ collisions.IFace = (*volumes.Face)(nil)
	if sz := unsafe.Sizeof(volumes.Face{}); sz > 336 {
		t.Fatalf("expected compact float32 Face struct <= 336 bytes, got %d", sz)
	}

	tex := textures.NewTexture("WORLD", 1, 64, 64, false)
	mat := textures.NewMaterial("world_wall", []*textures.Texture{tex}, int(config.MaterialKindSolid))

	const base = 16000.0
	f := volumes.NewFace([3]geometry.XYZ{
		{X: base, Y: base, Z: 0},
		{X: base + 12, Y: base, Z: 0},
		{X: base, Y: base + 12, Z: 0},
	}, mat)
	f.SetUV(0, 0, 1, 0, 0, 1)
	f.SetLightmap(4, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6)

	// Presentation boundary returns native float32 tuples cleanly.
	var p0x float32
	p0x, _, _, _, _, _, _, _, _ = f.GetPointsFlat()
	u0, _, _, _, _, _ := f.GetUVFlat()
	_, _, nz0, _, _, _, _, _, _ := f.GetVertexNormalsFlat()
	tx0, _, _, s0, _, _, _, _, _, _, _, _ := f.GetTangentsFlat()
	lmID, lmU, _ := f.GetLightmap()
	if p0x != float32(base) || u0 != 0 || math.Abs(float64(nz0-1.0)) > 1e-5 || math.Abs(float64(tx0-1.0)) > 1e-5 || s0 == 0 || lmID != 4 || math.Abs(float64(lmU[0]-0.1)) > 1e-6 {
		t.Fatalf("unexpected float32 presentation outputs: p0x=%v nz=%v tx=%v s0=%v lmID=%d lmU0=%v", p0x, nz0, tx0, s0, lmID, lmU[0])
	}

	// Physics/model boundary preserves float64 API and precision at large world coordinates.
	var gx0, gy0, gz0 float64 = f.Get0()
	var nx, ny, nz float64 = f.GetNormal()
	if gx0 != base || gy0 != base || gz0 != 0 || math.Abs(nx) > 1e-6 || math.Abs(ny) > 1e-6 || math.Abs(nz-1.0) > 1e-5 {
		t.Fatalf("unexpected float64 physics getters: pos=(%v,%v,%v) n=(%v,%v,%v)", gx0, gy0, gz0, nx, ny, nz)
	}
	minX, minY, _, maxX, maxY, _ := f.GetAABB().GetBounds()
	if !(minX < float64(f.GetTri().Points[0].X) && maxX > float64(f.GetTri().Points[1].X) && minY < float64(f.GetTri().Points[0].Y) && maxY > float64(f.GetTri().Points[2].Y)) {
		t.Fatalf("expected AABB bounds [%v..%v, %v..%v] to strictly contain quantized float32 triangle", minX, maxX, minY, maxY)
	}
	dist, hit := f.IntersectRay(base+2, base+2, 50, 0, 0, -1, 100)
	if !hit || math.Abs(dist-50.0) > 1e-4 {
		t.Fatalf("expected float64 IntersectRay hit dist=50, got hit=%v dist=%.5f", hit, dist)
	}
	w0, w1, w2, inside := f.Barycentric(base+3, base+3, 0)
	if !inside || math.Abs((w0+w1+w2)-1.0) > 1e-5 {
		t.Fatalf("expected valid float64 Barycentric inside triangle, got inside=%v weights=(%v,%v,%v)", inside, w0, w1, w2)
	}
}
