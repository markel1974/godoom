package vramtex

import (
	"math"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

type texParamRecord struct {
	target uint32
	pname  uint32
	param  int32
}

type texImageRecord struct {
	target uint32
	level  int32
	w, h   int32
}

type vramtexMockCtx struct {
	api.IContext
	nextID       uint32
	activeUnit   uint32
	boundCube    [16]uint32
	boundArr2D   [16]uint32
	texParams    []texParamRecord
	texImages    []texImageRecord
	mipGenerated int
}

func newVramtexMockCtx() *vramtexMockCtx {
	return &vramtexMockCtx{nextID: 1}
}

func (c *vramtexMockCtx) ActiveTexture(tex uint32) {
	if tex >= api.TEXTURE0 {
		c.activeUnit = tex - api.TEXTURE0
	}
}

func (c *vramtexMockCtx) GenTextures(n int32, textures *uint32) {
	s := unsafe.Slice(textures, int(n))
	for i := range s {
		s[i] = c.nextID
		c.nextID++
	}
}

func (c *vramtexMockCtx) DeleteTextures(_ int32, _ *uint32) {}

func (c *vramtexMockCtx) BindTexture(target uint32, tex uint32) {
	if int(c.activeUnit) < len(c.boundCube) {
		switch target {
		case api.TEXTURE_CUBE_MAP:
			c.boundCube[c.activeUnit] = tex
		case api.TEXTURE_2D_ARRAY:
			c.boundArr2D[c.activeUnit] = tex
		}
	}
}

func (c *vramtexMockCtx) TexParameteri(target uint32, pname uint32, param int32) {
	c.texParams = append(c.texParams, texParamRecord{target: target, pname: pname, param: param})
}

func (c *vramtexMockCtx) TexParameterf(_ uint32, _ uint32, _ float32) {}

func (c *vramtexMockCtx) TexImage2D(target uint32, level int32, _ int32, width int32, height int32, _ int32, _ uint32, _ uint32, _ unsafe.Pointer) {
	c.texImages = append(c.texImages, texImageRecord{target: target, level: level, w: width, h: height})
}

func (c *vramtexMockCtx) TexImage3D(_ uint32, _ int32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}

func (c *vramtexMockCtx) TexSubImage3D(_ uint32, _ int32, _, _, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}

func (c *vramtexMockCtx) GenerateMipmap(_ uint32) {
	c.mipGenerated++
}

func (c *vramtexMockCtx) GetFloatv(_ uint32, data *float32) {
	*data = 4.0
}

func (c *vramtexMockCtx) Ptr(data interface{}) unsafe.Pointer {
	if b, ok := data.([]byte); ok && len(b) > 0 {
		return unsafe.Pointer(&b[0])
	}
	return nil
}

func TestBucketsStayWithin16UnitsAndEnvMapUploadsPrefilteredMips(t *testing.T) {
	ctx := newVramtexMockCtx()
	tx := NewTextures(ctx)

	if tx.GetBucketsLen() != 3 {
		t.Fatalf("expected 3 texture size buckets, got %d", tx.GetBucketsLen())
	}

	tx.UploadEnvironment(nil)
	if tx.GetEnvMapTexture() == 0 {
		t.Fatalf("expected non-zero procedural environment cubemap texture")
	}
	if tx.GetEnvMapMaxLod() != 5.0 {
		t.Fatalf("expected max mip LOD 5.0 for 64x64 procedural cubemap, got %v", tx.GetEnvMapMaxLod())
	}

	// 6 mip levels (0..5) x 6 cube faces = 36 TexImage2D uploads.
	if len(ctx.texImages) != 36 {
		t.Fatalf("expected 36 cubemap mip face uploads (6 levels x 6 faces), got %d", len(ctx.texImages))
	}
	for lvl := int32(0); lvl <= 5; lvl++ {
		wantDim := int32(64 >> lvl)
		for f := 0; f < 6; f++ {
			rec := ctx.texImages[int(lvl)*6+f]
			if rec.level != lvl || rec.w != wantDim || rec.h != wantDim {
				t.Fatalf("mip %d face %d: got level=%d size=%dx%d, want %dx%d",
					lvl, f, rec.level, rec.w, rec.h, wantDim, wantDim)
			}
		}
	}

	hasTrilinear := false
	hasMaxLevel5 := false
	for _, p := range ctx.texParams {
		if p.target == api.TEXTURE_CUBE_MAP && p.pname == api.TEXTURE_MIN_FILTER && p.param == api.LINEAR_MIPMAP_LINEAR {
			hasTrilinear = true
		}
		if p.target == api.TEXTURE_CUBE_MAP && p.pname == api.TEXTURE_MAX_LEVEL && p.param == 5 {
			hasMaxLevel5 = true
		}
	}
	if !hasTrilinear || !hasMaxLevel5 {
		t.Fatalf("missing trilinear mipmap or max-level params on env cubemap (trilinear=%v maxLevel5=%v)", hasTrilinear, hasMaxLevel5)
	}

	// Verify custom power-of-2 envmap sizing and lightmap unit placement.
	ctxCustom := newVramtexMockCtx()
	txCustom := NewTextures(ctxCustom)
	const customDim = 16
	var customFaces [6][]byte
	for f := 0; f < 6; f++ {
		face := make([]byte, customDim*customDim*4)
		for i := 0; i < len(face); i += 4 {
			face[i], face[i+1], face[i+2], face[i+3] = 200, 120, 40, 255
		}
		customFaces[f] = face
	}
	txCustom.UploadEnvironment(map[string][6][]byte{"env": customFaces})
	if txCustom.GetEnvMapMaxLod() != 4.0 {
		t.Fatalf("expected max mip LOD 4.0 for 16x16 custom cubemap, got %v", txCustom.GetEnvMapMaxLod())
	}

	txCustom.UploadLightmaps([]*config.Lightmap{{Width: 2, Height: 2, Data: make([]byte, 16)}})
	txCustom.BindBuckets()
	if ctxCustom.boundCube[api.TexUnitEnvMap] != txCustom.GetEnvMapTexture() {
		t.Fatalf("envMap not bound on TexUnitEnvMap (%d): %+v", api.TexUnitEnvMap, ctxCustom.boundCube)
	}
	if ctxCustom.boundArr2D[api.TexUnitLightmap] != txCustom.GetLightmapTexture() {
		t.Fatalf("lightmap not bound on TexUnitLightmap (%d): %+v", api.TexUnitLightmap, ctxCustom.boundArr2D)
	}
}

func TestCubeUVRoundTripAndPrefilterMonotonicity(t *testing.T) {
	// Exact round-trip on all 6 cube faces.
	for f := 0; f < 6; f++ {
		for _, uv := range [][2]float64{{-0.75, -0.5}, {0.0, 0.25}, {0.85, -0.8}} {
			nx, ny, nz := cubeUVToDir(f, uv[0], uv[1])
			rf, ru, rv := dirToCubeUV(nx, ny, nz)
			if rf != f || math.Abs(ru-uv[0]) > 1e-9 || math.Abs(rv-uv[1]) > 1e-9 {
				t.Fatalf("face %d uv %v round-trip mismatch: got face=%d uv=(%v,%v)", f, uv, rf, ru, rv)
			}
		}
	}

	dim, base := buildBaseEnvFaces(nil)
	mip1 := prefilterEnvMip(base, dim, dim>>1, 0.2, 24)
	mip5 := prefilterEnvMip(base, dim, dim>>5, 1.0, 24)

	maxCh := func(faces [6][]float32) float32 {
		m := float32(0)
		for f := 0; f < 6; f++ {
			for _, v := range faces[f] {
				if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
					t.Fatalf("NaN/Inf in filtered cubemap face %d", f)
				}
				if v > m {
					m = v
				}
			}
		}
		return m
	}

	peak0 := maxCh(base)
	peak1 := maxCh(mip1)
	peak5 := maxCh(mip5)
	if !(peak0 >= peak1 && peak1 > peak5 && peak5 > 0) {
		t.Fatalf("expected sun peak to smooth monotonically across roughness mips: mip0=%v mip1=%v mip5=%v", peak0, peak1, peak5)
	}
}
