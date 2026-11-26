package buffers

import (
	"math"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type attribCall struct {
	loc        uint32
	size       int32
	xtype      uint32
	normalized bool
	stride     int32
	offset     uintptr
}

type bufferUpload struct {
	target uint32
	size   int
}

type meshTestCtx struct {
	api.IContext
	nextID  uint32
	attribs []attribCall
	uploads []bufferUpload
}

func (c *meshTestCtx) Ptr(data interface{}) unsafe.Pointer {
	switch v := data.(type) {
	case []uint32:
		if len(v) > 0 {
			return unsafe.Pointer(&v[0])
		}
	case []float32:
		if len(v) > 0 {
			return unsafe.Pointer(&v[0])
		}
	}
	return nil
}

func (c *meshTestCtx) PtrOffset(offset int) unsafe.Pointer { return unsafe.Pointer(uintptr(offset)) }

func (c *meshTestCtx) GenVertexArrays(n int32, arrays *uint32) {
	for i := range unsafe.Slice(arrays, int(n)) {
		c.nextID++
		unsafe.Slice(arrays, int(n))[i] = c.nextID
	}
}

func (c *meshTestCtx) GenBuffers(n int32, bufs *uint32) {
	for i := range unsafe.Slice(bufs, int(n)) {
		c.nextID++
		unsafe.Slice(bufs, int(n))[i] = c.nextID
	}
}

func (c *meshTestCtx) DeleteVertexArrays(_ int32, _ *uint32) {}
func (c *meshTestCtx) DeleteBuffers(_ int32, _ *uint32)      {}
func (c *meshTestCtx) BindVertexArray(_ uint32)              {}
func (c *meshTestCtx) BindBuffer(_, _ uint32)                {}
func (c *meshTestCtx) EnableVertexAttribArray(_ uint32)      {}
func (c *meshTestCtx) VertexAttribDivisor(_, _ uint32)       {}

func (c *meshTestCtx) BufferData(target uint32, size int, _ unsafe.Pointer, _ uint32) {
	c.uploads = append(c.uploads, bufferUpload{target: target, size: size})
}

func (c *meshTestCtx) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, ptr unsafe.Pointer) {
	c.attribs = append(c.attribs, attribCall{
		loc:        index,
		size:       size,
		xtype:      xtype,
		normalized: normalized,
		stride:     stride,
		offset:     uintptr(ptr),
	})
}

func TestVertexAttributePackingAndConfigureAttribs(t *testing.T) {
	if StaticVertexBytes != 64 || StaticVertexWords != 16 || StreamingVertexBytes != 96 || StreamingVertexWords != 24 {
		t.Fatalf("unexpected compressed vertex sizes: static=%dB (%dw) streaming=%dB (%dw)",
			StaticVertexBytes, StaticVertexWords, StreamingVertexBytes, StreamingVertexWords)
	}

	// Verify Float32 <-> Float16 precision & special bounds.
	for _, want := range []float32{0, -1, 0.25, 0.5, 0.75, 2.0, 40.0, 115.0, 0.1} {
		h := Float32ToFloat16(want)
		got := Float16ToFloat32(h)
		if math.Abs(float64(got-want)) > 1e-3 {
			t.Fatalf("Float16 roundtrip(%v): got %v (bits=%#x)", want, got, h)
		}
	}
	if ov := Float16ToFloat32(Float32ToFloat16(100000)); math.IsInf(float64(ov), 0) || ov < 65000 {
		t.Fatalf("expected finite clamped max half-float for overflow, got %v", ov)
	}
	if nanH := Float32ToFloat16(float32(math.NaN())); nanH != 0 {
		t.Fatalf("expected NaN encoded as 0, got %#x", nanH)
	}

	// Verify INT_2_10_10_10_REV packing including normalization, negative tangent handedness, and zero vector.
	pPos := PackSnorm10x3_2(0, 2, -2, 1.0, true)
	ux, uy, uz, uw := UnpackSnorm10x3_2(pPos)
	wantComp := float32(1.0 / math.Sqrt2)
	if math.Abs(float64(ux)) > 2e-3 || math.Abs(float64(uy-wantComp)) > 2e-3 || math.Abs(float64(uz+wantComp)) > 2e-3 || uw != 1.0 {
		t.Fatalf("PackSnorm10x3_2(+1) unpacked (%v,%v,%v,%v), want (0,%v,%v,1)", ux, uy, uz, uw, wantComp, -wantComp)
	}
	pNeg := PackSnorm10x3_2(-1, 0, 0, -1.0, false)
	nx, ny, nz, nw := UnpackSnorm10x3_2(pNeg)
	if math.Abs(float64(nx+1.0)) > 2e-3 || ny != 0 || nz != 0 || nw != -1.0 {
		t.Fatalf("PackSnorm10x3_2(-1) unpacked (%v,%v,%v,%v), want (-1,0,0,-1)", nx, ny, nz, nw)
	}
	zx, zy, zz, zw := UnpackSnorm10x3_2(PackSnorm10x3_2(0, 0, 0, 0, true))
	if zx != 0 || zy != 0 || zz != 0 || zw != 0 {
		t.Fatalf("expected zero vector preserved, got (%v,%v,%v,%v)", zx, zy, zz, zw)
	}

	ctx := &meshTestCtx{}
	ConfigureVertexAttribs(ctx, StreamingVertexBytes, true)
	if len(ctx.attribs) != 10 {
		t.Fatalf("expected 10 streaming attributes, got %d", len(ctx.attribs))
	}
	wantAttribs := []attribCall{
		{0, 3, api.FLOAT, false, StreamingVertexBytes, 0},
		{1, 3, api.FLOAT, false, StreamingVertexBytes, 12},
		{2, 4, api.INT_2_10_10_10_REV, true, StreamingVertexBytes, 24},
		{3, 4, api.INT_2_10_10_10_REV, true, StreamingVertexBytes, 28},
		{4, 4, api.HALF_FLOAT, false, StreamingVertexBytes, 32},
		{5, 4, api.HALF_FLOAT, false, StreamingVertexBytes, 40},
		{6, 4, api.FLOAT, false, StreamingVertexBytes, 48},
		{7, 3, api.FLOAT, false, StreamingVertexBytes, 64},
		{8, 3, api.FLOAT, false, StreamingVertexBytes, 76},
		{9, 2, api.FLOAT, false, StreamingVertexBytes, 88},
	}
	for i, want := range wantAttribs {
		if ctx.attribs[i] != want {
			t.Fatalf("attrib[%d] = %+v, want %+v", i, ctx.attribs[i], want)
		}
	}
}

func TestResidentMeshVertexDeduplicationAndZeroAllocs(t *testing.T) {
	tex := textures.NewTexture("WALL", 1, 64, 64, false)
	mat := textures.NewMaterial("stone", []*textures.Texture{tex}, int(config.MaterialKindSolid))
	mat.SetSpecularBoost(0.75)
	mat.SetShininess(64.0)
	texFn := func(_ *textures.Texture) (float32, bool) { return 2.0, true }

	// Case 1: Quad built from 2 coplanar triangles sharing 2 edge vertices -> 4 unique vertices, 6 indices.
	f0 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
	}, mat)
	f0.SetUV(0, 0, 1, 0, 1, 1)
	f1 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, mat)
	f1.SetUV(0, 0, 1, 1, 0, 1)

	quadFaces := volumes.NewFaces(aabb.NewTree(4, 0.0), false)
	quadFaces.Add(f0)
	quadFaces.Add(f1)
	quadFaces.Rebuild()

	ctx := &meshTestCtx{}
	rm := NewResidentMesh(ctx)
	if !rm.Rebuild(quadFaces, texFn) {
		t.Fatalf("quad Rebuild failed")
	}
	if rm.VertexCount() != 4 || rm.IndexCount() != 6 {
		t.Fatalf("expected 4 deduplicated vertices and 6 indices for quad, got verts=%d indices=%d", rm.VertexCount(), rm.IndexCount())
	}
	idx := rm.IndexData()
	// Triangle 0 uses indices 0,1,2; Triangle 1 must reuse vertices 0 and 2 from f0!
	if idx[0] != 0 || idx[1] != 1 || idx[2] != 2 || idx[3] != 0 || idx[4] != 2 || idx[5] != 3 {
		t.Fatalf("unexpected shared quad indices: %v", idx)
	}
	if len(ctx.uploads) != 2 || ctx.uploads[0].target != api.ARRAY_BUFFER || ctx.uploads[0].size != 4*StaticVertexBytes ||
		ctx.uploads[1].target != api.ELEMENT_ARRAY_BUFFER || ctx.uploads[1].size != 6*4 {
		t.Fatalf("unexpected compressed GPU uploads: %+v", ctx.uploads)
	}

	buildGridFaces := func(zFn func(x, y float64) float64) *volumes.Faces {
		const gridN = 4
		fset := volumes.NewFaces(aabb.NewTree(16, 0.0), false)
		for gy := range gridN {
			for gx := range gridN {
				x0, x1 := float64(gx*10), float64((gx+1)*10)
				y0, y1 := float64(gy*10), float64((gy+1)*10)
				z00, z10, z11, z01 := zFn(x0, y0), zFn(x1, y0), zFn(x1, y1), zFn(x0, y1)
				tA := volumes.NewFace([3]geometry.XYZ{
					{X: x0, Y: y0, Z: z00},
					{X: x1, Y: y0, Z: z10},
					{X: x1, Y: y1, Z: z11},
				}, mat)
				tA.SetUV(float64(gx), float64(gy), float64(gx+1), float64(gy), float64(gx+1), float64(gy+1))
				tB := volumes.NewFace([3]geometry.XYZ{
					{X: x0, Y: y0, Z: z00},
					{X: x1, Y: y1, Z: z11},
					{X: x0, Y: y1, Z: z01},
				}, mat)
				tB.SetUV(float64(gx), float64(gy), float64(gx+1), float64(gy+1), float64(gx), float64(gy+1))
				fset.Add(tA)
				fset.Add(tB)
			}
		}
		fset.Rebuild()
		return fset
	}

	// Case 2a: Sloped 4x4 tiled quad grid (16 quads = 32 tris = 96 corners -> 25 unique vertices, 3.84x reduction).
	planeGridFaces := buildGridFaces(func(x, y float64) float64 { return 0.2*x - 0.1*y })
	if !rm.Rebuild(planeGridFaces, texFn) {
		t.Fatalf("plane grid Rebuild failed")
	}
	if rm.VertexCount() != 25 || rm.IndexCount() != 32*3 {
		t.Fatalf("expected 25 unique grid vertices and %d indices, got verts=%d indices=%d",
			32*3, rm.VertexCount(), rm.IndexCount())
	}

	// Case 2b: Curved 4x4 heightfield (smooth crease normals; distinct twisted-quad tangents kept split).
	gridFaces := buildGridFaces(func(x, y float64) float64 { return math.Sin(x*0.1) + math.Cos(y*0.1) })
	if !rm.Rebuild(gridFaces, texFn) {
		t.Fatalf("smooth grid Rebuild failed")
	}
	if rm.VertexCount() >= rm.IndexCount()/2 || rm.IndexCount() != 32*3 {
		t.Fatalf("expected curved heightfield deduplication (< %d verts) and 96 indices, got verts=%d indices=%d",
			rm.IndexCount()/2, rm.VertexCount(), rm.IndexCount())
	}

	// Steady-state Rebuild of gridFaces must allocate zero heap objects.
	ctx.uploads = ctx.uploads[:0]
	allocs := testing.AllocsPerRun(50, func() {
		if !rm.Rebuild(gridFaces, texFn) {
			t.Fatalf("steady-state grid Rebuild failed")
		}
	})
	if allocs > 0 {
		t.Fatalf("ResidentMesh.Rebuild with deduplication allocated %.2f objects/op, want 0", allocs)
	}

	// Case 3: UV seam splits shared positions into distinct vertices.
	s0 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
	}, mat)
	s0.SetUV(0, 0, 1, 0, 1, 1)
	s0.LockUV(true)
	s1 := volumes.NewFace([3]geometry.XYZ{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: 0},
	}, mat)
	s1.SetUV(2, 0, 3, 1, 2, 1) // discontinuous UV seam at shared edge!
	s1.LockUV(true)
	seamFaces := volumes.NewFaces(aabb.NewTree(4, 0.0), false)
	seamFaces.Add(s0)
	seamFaces.Add(s1)
	seamFaces.Rebuild()
	if !rm.Rebuild(seamFaces, texFn) {
		t.Fatalf("seam Rebuild failed")
	}
	if rm.VertexCount() != 6 {
		t.Fatalf("expected UV seam to preserve 6 distinct vertices, got %d", rm.VertexCount())
	}

	// Case 4: Multi-volume static atlas AppendStaticFaces deduplicates across shared quads without losing SubMesh spans.
	atlas := NewResidentMesh(ctx)
	atlas.ReserveStaticAtlas(quadFaces.Len())
	atlas.BeginStaticAtlas()
	subStart, subCount, _, _, ok := atlas.AppendStaticFaces(quadFaces, texFn)
	if !ok || subCount != 1 {
		t.Fatalf("AppendStaticFaces failed: ok=%v subStart=%d subCount=%d", ok, subStart, subCount)
	}
	atlas.EndStaticAtlas()
	if atlas.VertexCount() != 4 || atlas.IndexCount() != 6 || atlas.SubMeshes[0].IndexCount != 6 {
		t.Fatalf("atlas deduplication mismatch: verts=%d indices=%d sub=%+v", atlas.VertexCount(), atlas.IndexCount(), atlas.SubMeshes[0])
	}
}
