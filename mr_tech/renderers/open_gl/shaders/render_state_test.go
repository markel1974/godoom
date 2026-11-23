package shaders

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type subDataCall struct {
	target uint32
	offset int
	size   int
	vao    uint32
}

type recordingCtx struct {
	api.IContext

	nextID             uint32
	boundVAO           uint32
	boundProgram       uint32
	vaoHistory         []uint32
	eboBindVAO         []uint32
	clearCalls         int
	bufferDataCalls    int
	subDataCalls       []subDataCall
	getUniformLocCalls int
	strCalls           int
	uniform1ivCalls    int
	uniform1iCalls     int
	blitCalls          int
	lastBlitMask       uint32
	shaderSources      map[uint32]string
	programShaders     map[uint32][]uint32
	programHistory     []uint32
	colorMaskCalls     [][4]bool
	depthMaskCalls     []bool
	depthFuncCalls     []uint32
	enabledHistory     []uint32
	disabledHistory    []uint32
	multiDrawPrograms  []uint32
	lastUBOData        []float32
	scissorCalls       [][4]int32
	viewportCalls      [][4]int32
	polygonOffsetCalls [][2]float32
	polyOffsetActive   bool
	depthClampActive   bool
	curScissor         [4]int32
	drawPolyOffsets    []bool
	drawDepthClamps    []bool
	drawScissors       [][4]int32
	activeTexUnit      uint32
	boundTex2D         [16]uint32
}

func newRecordingCtx() *recordingCtx {
	return &recordingCtx{
		nextID:         1,
		shaderSources:  make(map[uint32]string),
		programShaders: make(map[uint32][]uint32),
	}
}

func (c *recordingCtx) resetLogs() {
	c.vaoHistory = c.vaoHistory[:0]
	c.eboBindVAO = c.eboBindVAO[:0]
	c.subDataCalls = c.subDataCalls[:0]
	c.programHistory = c.programHistory[:0]
	c.colorMaskCalls = c.colorMaskCalls[:0]
	c.depthMaskCalls = c.depthMaskCalls[:0]
	c.depthFuncCalls = c.depthFuncCalls[:0]
	c.enabledHistory = c.enabledHistory[:0]
	c.disabledHistory = c.disabledHistory[:0]
	c.multiDrawPrograms = c.multiDrawPrograms[:0]
	c.scissorCalls = c.scissorCalls[:0]
	c.viewportCalls = c.viewportCalls[:0]
	c.polygonOffsetCalls = c.polygonOffsetCalls[:0]
	c.drawPolyOffsets = c.drawPolyOffsets[:0]
	c.drawDepthClamps = c.drawDepthClamps[:0]
	c.drawScissors = c.drawScissors[:0]
}

func (c *recordingCtx) allocIDs(n int32, out *uint32) {
	if out == nil || n <= 0 {
		return
	}
	s := unsafe.Slice(out, int(n))
	for i := range s {
		s[i] = c.nextID
		c.nextID++
	}
}

func (c *recordingCtx) ActiveTexture(tex uint32) {
	if tex >= api.TEXTURE0 {
		c.activeTexUnit = tex - api.TEXTURE0
	}
}
func (c *recordingCtx) AttachShader(prg, sh uint32) {
	c.programShaders[prg] = append(c.programShaders[prg], sh)
}
func (c *recordingCtx) BindBufferBase(_ uint32, _ uint32, _ uint32) {}
func (c *recordingCtx) BindFramebuffer(_ uint32, _ uint32)          {}
func (c *recordingCtx) BindRenderbuffer(_ uint32, _ uint32)         {}
func (c *recordingCtx) BindTexture(target uint32, tex uint32) {
	if target == api.TEXTURE_2D && int(c.activeTexUnit) < len(c.boundTex2D) {
		c.boundTex2D[c.activeTexUnit] = tex
	}
}
func (c *recordingCtx) BlendEquation(_ uint32) {}
func (c *recordingCtx) BlendFunc(_, _ uint32)  {}
func (c *recordingCtx) BlitFramebuffer(_, _, _, _, _, _, _, _ int32, mask, _ uint32) {
	c.blitCalls++
	c.lastBlitMask = mask
}
func (c *recordingCtx) CheckFramebufferStatus(_ uint32) uint32 { return api.FRAMEBUFFER_COMPLETE }
func (c *recordingCtx) ColorMask(r, g, b, a bool) {
	c.colorMaskCalls = append(c.colorMaskCalls, [4]bool{r, g, b, a})
}
func (c *recordingCtx) CompileShader(_ uint32)                 {}
func (c *recordingCtx) CreateProgram() uint32                  { id := c.nextID; c.nextID++; return id }
func (c *recordingCtx) CreateShader(_ uint32) uint32           { id := c.nextID; c.nextID++; return id }
func (c *recordingCtx) DeleteBuffers(_ int32, _ *uint32)       {}
func (c *recordingCtx) DeleteFramebuffers(_ int32, _ *uint32)  {}
func (c *recordingCtx) DeleteRenderbuffers(_ int32, _ *uint32) {}
func (c *recordingCtx) DeleteShader(_ uint32)                  {}
func (c *recordingCtx) DeleteTextures(_ int32, _ *uint32)      {}
func (c *recordingCtx) DeleteVertexArrays(_ int32, _ *uint32)  {}
func (c *recordingCtx) DepthFunc(fn uint32) {
	c.depthFuncCalls = append(c.depthFuncCalls, fn)
}
func (c *recordingCtx) DepthMask(flag bool) {
	c.depthMaskCalls = append(c.depthMaskCalls, flag)
}
func (c *recordingCtx) Disable(cap uint32) {
	c.disabledHistory = append(c.disabledHistory, cap)
	if cap == api.POLYGON_OFFSET_FILL {
		c.polyOffsetActive = false
	} else if cap == api.DEPTH_CLAMP {
		c.depthClampActive = false
	}
}
func (c *recordingCtx) DrawArrays(_ uint32, _ int32, _ int32) {}
func (c *recordingCtx) DrawBuffer(_ uint32)                   {}
func (c *recordingCtx) DrawBuffers(_ int32, _ *uint32)        {}
func (c *recordingCtx) Enable(cap uint32) {
	c.enabledHistory = append(c.enabledHistory, cap)
	if cap == api.POLYGON_OFFSET_FILL {
		c.polyOffsetActive = true
	} else if cap == api.DEPTH_CLAMP {
		c.depthClampActive = true
	}
}
func (c *recordingCtx) EnableVertexAttribArray(_ uint32)                {}
func (c *recordingCtx) FramebufferRenderbuffer(_, _, _, _ uint32)       {}
func (c *recordingCtx) FramebufferTexture2D(_, _, _, _ uint32, _ int32) {}
func (c *recordingCtx) GenBuffers(n int32, buffers *uint32)             { c.allocIDs(n, buffers) }
func (c *recordingCtx) GenFramebuffers(n int32, fbos *uint32)           { c.allocIDs(n, fbos) }
func (c *recordingCtx) GenRenderbuffers(n int32, rbos *uint32)          { c.allocIDs(n, rbos) }
func (c *recordingCtx) GenTextures(n int32, textures *uint32)           { c.allocIDs(n, textures) }
func (c *recordingCtx) GenVertexArrays(n int32, arrays *uint32)         { c.allocIDs(n, arrays) }
func (c *recordingCtx) GetShaderiv(_ uint32, _ uint32, params *int32)   { *params = 1 }
func (c *recordingCtx) GetProgramiv(_ uint32, _ uint32, params *int32)  { *params = 1 }
func (c *recordingCtx) LinkProgram(_ uint32)                            {}
func (c *recordingCtx) DrawElements(_ uint32, _ int32, _ uint32, _ unsafe.Pointer) {
	c.multiDrawPrograms = append(c.multiDrawPrograms, c.boundProgram)
	c.drawPolyOffsets = append(c.drawPolyOffsets, c.polyOffsetActive)
	c.drawDepthClamps = append(c.drawDepthClamps, c.depthClampActive)
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *recordingCtx) MultiDrawElements(_ uint32, _ *int32, _ uint32, _ *unsafe.Pointer, _ int32) {
	c.multiDrawPrograms = append(c.multiDrawPrograms, c.boundProgram)
	c.drawPolyOffsets = append(c.drawPolyOffsets, c.polyOffsetActive)
	c.drawDepthClamps = append(c.drawDepthClamps, c.depthClampActive)
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *recordingCtx) DrawElementsInstanced(_ uint32, _ int32, _ uint32, _ unsafe.Pointer, _ int32) {
	c.multiDrawPrograms = append(c.multiDrawPrograms, c.boundProgram)
	c.drawPolyOffsets = append(c.drawPolyOffsets, c.polyOffsetActive)
	c.drawDepthClamps = append(c.drawDepthClamps, c.depthClampActive)
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *recordingCtx) CullFace(_ uint32) {}
func (c *recordingCtx) PolygonOffset(factor, units float32) {
	c.polygonOffsetCalls = append(c.polygonOffsetCalls, [2]float32{factor, units})
}
func (c *recordingCtx) Ptr(data interface{}) unsafe.Pointer {
	if s, ok := data.([]float32); ok && len(s) > 0 {
		return unsafe.Pointer(&s[0])
	}
	return nil
}
func (c *recordingCtx) PtrOffset(offset int) unsafe.Pointer                                    { return unsafe.Pointer(uintptr(offset)) }
func (c *recordingCtx) ReadBuffer(_ uint32)                                                    {}
func (c *recordingCtx) RenderbufferStorage(_, _ uint32, _, _ int32)                            {}
func (c *recordingCtx) RenderbufferStorageMultisample(_ uint32, _ int32, _ uint32, _, _ int32) {}
func (c *recordingCtx) Scissor(x, y, width, height int32) {
	c.curScissor = [4]int32{x, y, width, height}
	c.scissorCalls = append(c.scissorCalls, c.curScissor)
}
func (c *recordingCtx) ShaderSource(shader uint32, _ int32, xstring **uint8, _ *int32) {
	if xstring != nil && *xstring != nil {
		var buf []byte
		ptr := unsafe.Pointer(*xstring)
		for i := uintptr(0); i < 65536; i++ {
			b := *(*byte)(unsafe.Add(ptr, i))
			if b == 0 {
				break
			}
			buf = append(buf, b)
		}
		c.shaderSources[shader] = string(buf)
	}
}
func (c *recordingCtx) TexImage2D(_ uint32, _ int32, _ int32, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *recordingCtx) TexSubImage2D(_ uint32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *recordingCtx) TexParameteri(_, _ uint32, _ int32)                    {}
func (c *recordingCtx) TexParameterfv(_, _ uint32, _ *float32)                {}
func (c *recordingCtx) Uniform1f(_ int32, _ float32)                          {}
func (c *recordingCtx) Uniform2f(_ int32, _, _ float32)                       {}
func (c *recordingCtx) Uniform3f(_ int32, _, _, _ float32)                    {}
func (c *recordingCtx) Uniform4f(_ int32, _, _, _, _ float32)                 {}
func (c *recordingCtx) UniformBlockBinding(_, _, _ uint32)                    {}
func (c *recordingCtx) UniformMatrix4fv(_ int32, _ int32, _ bool, _ *float32) {}
func (c *recordingCtx) UseProgram(prg uint32) {
	c.boundProgram = prg
	c.programHistory = append(c.programHistory, prg)
}
func (c *recordingCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}
func (c *recordingCtx) VertexAttribDivisor(_, _ uint32) {}
func (c *recordingCtx) Viewport(x, y, width, height int32) {
	c.viewportCalls = append(c.viewportCalls, [4]int32{x, y, width, height})
}

func (c *recordingCtx) BindVertexArray(array uint32) {
	c.boundVAO = array
	c.vaoHistory = append(c.vaoHistory, array)
}

func (c *recordingCtx) BindBuffer(target uint32, _ uint32) {
	if target == api.ELEMENT_ARRAY_BUFFER {
		c.eboBindVAO = append(c.eboBindVAO, c.boundVAO)
	}
}

func (c *recordingCtx) Clear(_ uint32) {
	c.clearCalls++
}

func (c *recordingCtx) ClearColor(_, _, _, _ float32) {}

func (c *recordingCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {
	c.bufferDataCalls++
}

func (c *recordingCtx) BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	c.subDataCalls = append(c.subDataCalls, subDataCall{
		target: target,
		offset: offset,
		size:   size,
		vao:    c.boundVAO,
	})
	if target == api.UNIFORM_BUFFER && data != nil && size >= 4 {
		floats := unsafe.Slice((*float32)(data), size/4)
		c.lastUBOData = append(c.lastUBOData[:0], floats...)
	}
}

func (c *recordingCtx) Str(_ string) *uint8 {
	c.strCalls++
	var b byte
	return &b
}

func (c *recordingCtx) Strs(strs ...string) (**uint8, func()) {
	if len(strs) > 0 {
		raw := append([]byte(strs[0]), 0)
		p := &raw[0]
		return &p, func() { _ = raw }
	}
	var b *uint8
	return &b, func() {}
}

func (c *recordingCtx) GetUniformLocation(_ uint32, _ *uint8) int32 {
	c.getUniformLocCalls++
	return 1
}

func (c *recordingCtx) GetUniformBlockIndex(_ uint32, _ *uint8) uint32 { return 0 }

func (c *recordingCtx) Uniform1i(_ int32, _ int32) {
	c.uniform1iCalls++
}

func (c *recordingCtx) Uniform1iv(_ int32, _ int32, _ *int32) {
	c.uniform1ivCalls++
}

type mockAssets struct{}

func (mockAssets) BasePath(vPath string) string { return vPath }
func (mockAssets) Read(_ string) ([]byte, error) {
	return []byte(""), nil
}
func (mockAssets) ReadMulti(_, _ string) ([]byte, []byte, error) {
	return []byte(""), []byte(""), nil
}

func newTestHandler(t *testing.T, ctx *recordingCtx) *Handler {
	t.Helper()
	calCfg := config.NewConfigCalibration()
	cal := model.NewCalibration(calCfg, nil)
	cal.MSAASamples = 0
	flashCfg := config.NewFlash()
	flashCfg.FovDeg = 80.0
	flashCfg.ShadowFovFactor = 1.75
	flashCfg.ZNear = 0.1
	flashCfg.ZFar = 2048.0
	flash := things.NewFlash(flashCfg)
	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, flash, 640, 480); err != nil {
		t.Fatalf("Handler.Setup failed: %v", err)
	}
	return h
}

func TestMainPrepareFrameVAOBindingNoClearAndPartialUploads(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandler(t, ctx)
	m := h.GetMain()
	if err := m.Init(); err != nil {
		t.Fatalf("Main.Init failed: %v", err)
	}

	verts := make([]float32, 4096)
	idxs := make([]uint32, 4096)
	vPtr := unsafe.Pointer(&verts[0])
	iPtr := unsafe.Pointer(&idxs[0])

	// Simulate post.vao still being bound from previous frame.
	ctx.BindVertexArray(999)
	ctx.clearCalls = 0
	ctx.eboBindVAO = nil
	ctx.subDataCalls = nil

	// Frame 1 -> slot 1, staticGen=1, frameGen=1 -> full upload from offset 0.
	m.PrepareFrame(vPtr, 1000, 800, iPtr, 400, 300, 1, 1)
	if ctx.clearCalls != 0 {
		t.Fatalf("PrepareFrame must not call Clear on unbound FBO, got clearCalls=%d", ctx.clearCalls)
	}
	if len(ctx.eboBindVAO) != 1 || ctx.eboBindVAO[0] != m.GetVAO() {
		t.Fatalf("ELEMENT_ARRAY_BUFFER bound under VAO %v, want mainVAO %d", ctx.eboBindVAO, m.GetVAO())
	}
	if ctx.boundVAO != 0 {
		t.Fatalf("expected VAO 0 after PrepareFrame, got %d", ctx.boundVAO)
	}
	if len(ctx.subDataCalls) != 2 || ctx.subDataCalls[0].offset != 0 || ctx.subDataCalls[0].size != 1000 || ctx.subDataCalls[1].offset != 0 || ctx.subDataCalls[1].size != 400 {
		t.Fatalf("unexpected frame 1 uploads: %+v", ctx.subDataCalls)
	}

	// Frame 2 -> slot 0, staticGen=1, frameGen=1 -> slot 0 also initializes cleanly from 0.
	ctx.subDataCalls = nil
	m.PrepareFrame(vPtr, 1000, 800, iPtr, 400, 300, 1, 1)
	if len(ctx.subDataCalls) != 2 || ctx.subDataCalls[0].offset != 0 || ctx.subDataCalls[1].offset != 0 {
		t.Fatalf("unexpected frame 2 uploads: %+v", ctx.subDataCalls)
	}

	// Frame 3 -> slot 1, identical frameGen=1 -> zero BufferSubData calls.
	ctx.subDataCalls = nil
	m.PrepareFrame(vPtr, 1000, 800, iPtr, 400, 300, 1, 1)
	if len(ctx.subDataCalls) != 0 {
		t.Fatalf("expected 0 BufferSubData calls for unchanged frame, got %+v", ctx.subDataCalls)
	}

	// Frame 4 -> slot 0, staticGen=1 unchanged, frameGen=2 -> upload only dynamic tail [staticBytes..size).
	ctx.subDataCalls = nil
	m.PrepareFrame(vPtr, 1080, 800, iPtr, 448, 300, 1, 2)
	if len(ctx.subDataCalls) != 2 ||
		ctx.subDataCalls[0].offset != 800 || ctx.subDataCalls[0].size != 280 ||
		ctx.subDataCalls[1].offset != 300 || ctx.subDataCalls[1].size != 148 {
		t.Fatalf("expected dynamic-tail-only uploads, got %+v", ctx.subDataCalls)
	}

	// Frame 5 -> slot 1, oversized buffer forcing BufferData reallocation -> must upload full [0..vSize).
	oversized := m.vboBytesCap[1] + 256
	ctx.subDataCalls = nil
	ctx.bufferDataCalls = 0
	m.PrepareFrame(vPtr, oversized, 800, iPtr, 448, 300, 1, 3)
	if ctx.bufferDataCalls != 1 {
		t.Fatalf("expected 1 BufferData resize call, got %d", ctx.bufferDataCalls)
	}
	if len(ctx.subDataCalls) != 2 || ctx.subDataCalls[0].offset != 0 || ctx.subDataCalls[0].size != oversized {
		t.Fatalf("expected full VBO re-upload after BufferData resize, got %+v", ctx.subDataCalls)
	}
}

func TestPostEnsureAllocatedCreatesAndResizesFBOBeforeRender(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandler(t, ctx)
	p := h.GetPost()

	if p.GetFBO() != 0 {
		t.Fatalf("expected zero FBO before EnsureAllocated, got %d", p.GetFBO())
	}
	p.EnsureAllocated()
	fbo1 := p.GetFBO()
	if fbo1 == 0 {
		t.Fatalf("expected non-zero FBO after EnsureAllocated on frame 0")
	}

	// Calling EnsureAllocated again at same size must be a no-op.
	p.EnsureAllocated()
	if p.GetFBO() != fbo1 {
		t.Fatalf("unexpected FBO reallocation at unchanged size: %d -> %d", fbo1, p.GetFBO())
	}

	// Changing handler size must reallocate FBO before Main.Render draws.
	h.SetSize(800, 600)
	p.EnsureAllocated()
	if p.GetFBO() == fbo1 || p.GetFBO() == 0 {
		t.Fatalf("expected FBO reallocation after resize, got %d", p.GetFBO())
	}
}

func TestLiquidRenderDoesNotResetupSamplersPerFrame(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandler(t, ctx)
	l := h.GetLiquid()
	if err := l.Compile(mockAssets{}); err != nil {
		t.Fatalf("Liquid.Compile failed: %v", err)
	}
	if err := l.SetupSamplers(); err != nil {
		t.Fatalf("Liquid.SetupSamplers failed: %v", err)
	}

	ctx.getUniformLocCalls = 0
	ctx.strCalls = 0
	ctx.uniform1ivCalls = 0
	ctx.uniform1iCalls = 0

	drawn := 0
	for range 5 {
		l.Render(func() { drawn++ })
	}

	if drawn != 5 {
		t.Fatalf("expected 5 liquid draws, got %d", drawn)
	}
	if ctx.getUniformLocCalls != 0 || ctx.strCalls != 0 || ctx.uniform1ivCalls != 0 || ctx.uniform1iCalls != 0 {
		t.Fatalf("Liquid.Render must not call SetupSamplers per frame (getUniformLoc=%d str=%d 1iv=%d 1i=%d)",
			ctx.getUniformLocCalls, ctx.strCalls, ctx.uniform1ivCalls, ctx.uniform1iCalls)
	}
}

type diskAssets struct{}

func (diskAssets) BasePath(vPath string) string { return vPath }
func (diskAssets) Read(p string) ([]byte, error) {
	return os.ReadFile(filepath.Join("..", "assets", p))
}
func (diskAssets) ReadMulti(a, b string) ([]byte, []byte, error) {
	da, err := os.ReadFile(filepath.Join("..", "assets", a))
	if err != nil {
		return nil, nil, err
	}
	db, err := os.ReadFile(filepath.Join("..", "assets", b))
	if err != nil {
		return nil, nil, err
	}
	return da, db, nil
}

func newTestHandlerWithMSAA(t *testing.T, ctx *recordingCtx, samples int) *Handler {
	t.Helper()
	calCfg := config.NewConfigCalibration()
	cal := model.NewCalibration(calCfg, nil)
	cal.MSAASamples = samples
	flashCfg := config.NewFlash()
	flashCfg.FovDeg = 80.0
	flashCfg.ShadowFovFactor = 1.75
	flashCfg.ZNear = 0.1
	flashCfg.ZFar = 2048.0
	flash := things.NewFlash(flashCfg)
	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, flash, 640, 480); err != nil {
		t.Fatalf("Handler.Setup failed: %v", err)
	}
	return h
}

func makeSolidAndCutoutMaterials(t *testing.T) (solidMat *textures.Material, cutoutMat *textures.Material) {
	t.Helper()
	solidTex := textures.NewTexture("solid_wall", 1, 2, 2, false)
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			solidTex.SetDiffuse(x, y, 0x112233ff)
		}
	}
	solidMat = textures.NewMaterial("solid_mat", []*textures.Texture{solidTex}, int(config.MaterialKindSolid))
	solidMat.SetDepthWrite(true)

	cutoutTex := textures.NewTexture("fence_cutout", 2, 2, 2, false)
	cutoutTex.SetDiffuse(0, 0, 0x112233ff)
	cutoutTex.SetDiffuse(0, 1, 0x11223300) // transparent cutout texel
	cutoutTex.SetDiffuse(1, 0, 0x112233ff)
	cutoutTex.SetDiffuse(1, 1, 0x112233ff)
	cutoutMat = textures.NewMaterial("cutout_mat", []*textures.Texture{cutoutTex}, int(config.MaterialKindSolid))
	cutoutMat.SetDepthWrite(true)
	return solidMat, cutoutMat
}

func TestPostZeroBlitWhenMSAADisabledAndLazyRefraction(t *testing.T) {
	// Case 1: MSAA == 0 renders directly into resolve FBO with zero frame blits.
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(t, ctx, 0)
	p := h.GetPost()
	p.EnsureAllocated()

	if p.GetSamples() != 0 {
		t.Fatalf("expected 0 MSAA samples, got %d", p.GetSamples())
	}
	if p.msaaFbo != 0 {
		t.Fatalf("expected no MSAA FBO when samples == 0, got %d", p.msaaFbo)
	}
	if p.GetFBO() == 0 || p.GetFBO() != p.fbo {
		t.Fatalf("expected GetFBO() to return resolve FBO %d directly when samples == 0, got %d", p.fbo, p.GetFBO())
	}

	ctx.blitCalls = 0
	p.Prepare()
	if ctx.blitCalls != 0 {
		t.Fatalf("expected 0 BlitFramebuffer calls in Post.Prepare when samples == 0, got %d", ctx.blitCalls)
	}
	if p.refractFbo != 0 {
		t.Fatalf("expected lazy refractFbo to stay unallocated when liquid is absent, got %d", p.refractFbo)
	}

	// Liquid pass lazily creates refractFbo and snapshots color+depth cleanly without touching normal Prepare.
	p.ResolveMSAA(640, 480, ResolveColor|ResolveDepth)
	if p.refractFbo == 0 || p.refractColorBuffer == 0 || p.refractDepthBuffer == 0 {
		t.Fatalf("expected lazy refraction FBO and textures to allocate on ResolveMSAA")
	}
	if ctx.blitCalls != 2 {
		t.Fatalf("expected 2 blits (color + depth) on liquid refraction resolve, got %d", ctx.blitCalls)
	}
	if p.GetRefractionColorBuffer() != p.refractColorBuffer || p.GetRefractionDepthBuffer() != p.refractDepthBuffer {
		t.Fatalf("expected GetRefraction* to return refraction buffers when samples == 0")
	}
	if p.GetRefractionColorBuffer() == p.GetColorBuffer() {
		t.Fatalf("refraction color buffer must differ from live scene color attachment to avoid feedback loops")
	}

	// Subsequent frame without liquid must again perform 0 blits.
	ctx.blitCalls = 0
	p.Prepare()
	if ctx.blitCalls != 0 {
		t.Fatalf("expected 0 blits on non-liquid frame when samples == 0, got %d", ctx.blitCalls)
	}

	// Case 2: MSAA == 4 uses dedicated msaaFbo and resolves via BlitFramebuffer.
	ctxMSAA := newRecordingCtx()
	hMSAA := newTestHandlerWithMSAA(t, ctxMSAA, 4)
	pMSAA := hMSAA.GetPost()
	pMSAA.EnsureAllocated()
	if pMSAA.msaaFbo == 0 || pMSAA.GetFBO() != pMSAA.msaaFbo || pMSAA.GetFBO() == pMSAA.fbo {
		t.Fatalf("expected separate MSAA FBO when samples == 4 (fbo=%d msaaFbo=%d)", pMSAA.fbo, pMSAA.msaaFbo)
	}
	ctxMSAA.blitCalls = 0
	pMSAA.Prepare()
	if ctxMSAA.blitCalls != 2 {
		t.Fatalf("expected 2 MSAA resolve blits in Post.Prepare when samples == 4, got %d", ctxMSAA.blitCalls)
	}
	if pMSAA.GetRefractionColorBuffer() != pMSAA.GetColorBuffer() || pMSAA.GetRefractionDepthBuffer() != pMSAA.GetDepthBuffer() {
		t.Fatalf("expected GetRefraction* to return resolved buffers directly when samples == 4")
	}
}

func TestZPrepassSolidVsCutoutEarlyZAndShaderDirectives(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(t, ctx, 0)
	m := h.GetMain()
	d := h.GetDepth()
	h.GetPost().EnsureAllocated()

	if err := m.Init(); err != nil {
		t.Fatalf("Main.Init failed: %v", err)
	}
	if err := m.Compile(diskAssets{}); err != nil {
		t.Fatalf("Main.Compile(diskAssets) failed: %v", err)
	}
	if err := m.SetupSamplers(); err != nil {
		t.Fatalf("Main.SetupSamplers failed: %v", err)
	}
	if err := d.Compile(diskAssets{}); err != nil {
		t.Fatalf("Depth.Compile(diskAssets) failed: %v", err)
	}

	// Verify real GLSL sources linked into solid vs cutout vs z-prepass programs.
	solidShaders := ctx.programShaders[m.GetProgramOpaque()]
	if len(solidShaders) != 2 {
		t.Fatalf("expected 2 shaders attached to main_opaque, got %d", len(solidShaders))
	}
	vertSrc := ctx.shaderSources[solidShaders[0]]
	solidFragSrc := ctx.shaderSources[solidShaders[1]]
	if !strings.Contains(vertSrc, "invariant gl_Position;") {
		t.Fatalf("main.vert must declare invariant gl_Position for Z-prepass parity")
	}
	if strings.Contains(solidFragSrc, "#define ALPHA_CUTOUT") {
		t.Fatalf("solid main_opaque shader must not define ALPHA_CUTOUT")
	}
	if !strings.Contains(solidFragSrc, "kShadowAtlasTexel") || strings.Contains(solidFragSrc, "textureSize(u_shadowAtlas") {
		t.Fatalf("main_opaque.frag must use constant kShadowAtlasTexel instead of textureSize(u_shadowAtlas, 0)")
	}
	if strings.Contains(solidFragSrc, "u_view * vec4") || strings.Contains(solidFragSrc, "mat3(u_view) * u_shadowLights") || strings.Contains(solidFragSrc, "mat3(u_view) * u_lights") {
		t.Fatalf("main_opaque.frag must consume CPU-transformed view-space lights without per-pixel u_view matrix multiplies")
	}

	cutoutShaders := ctx.programShaders[m.GetProgramCutout()]
	cutoutFragSrc := ctx.shaderSources[cutoutShaders[1]]
	if !strings.Contains(cutoutFragSrc, "#define ALPHA_CUTOUT 1") {
		t.Fatalf("cutout main shader must inject #define ALPHA_CUTOUT 1")
	}

	solidMat, cutoutMat := makeSolidAndCutoutMaterials(t)
	if solidMat.HasCutout() {
		t.Fatalf("all-opaque texture material must report HasCutout() == false")
	}
	if !cutoutMat.HasCutout() {
		t.Fatalf("texture with alpha < 255 must report HasCutout() == true")
	}

	dcr := gpu.NewDrawCommandsRender(ctx, "test_opaque", false)
	cmdSolid := gpu.NewDrawCommand()
	cmdSolid.SetFirstIndex(0)
	cmdSolid.SetIndexCount(6)
	cmdSolid.SetMaterial(solidMat)
	cmdCutout := gpu.NewDrawCommand()
	cmdCutout.SetFirstIndex(6)
	cmdCutout.SetIndexCount(6)
	cmdCutout.SetMaterial(cutoutMat)
	dcr.Prepare([]*gpu.DrawCommand{cmdSolid, cmdCutout})

	ctx.colorMaskCalls = nil
	ctx.depthMaskCalls = nil
	ctx.depthFuncCalls = nil
	ctx.enabledHistory = nil
	ctx.multiDrawPrograms = nil

	m.Render(dcr, 0.25, 0, 0, 1)

	// Z-Prepass must draw both zprepass_solid and zprepass_cutout before forward shading.
	wantPrograms := []uint32{
		m.GetProgramZPrepassSolid(),
		m.GetProgramZPrepassCutout(),
		m.GetProgramOpaque(),
		m.GetProgramCutout(),
	}
	if len(ctx.multiDrawPrograms) != len(wantPrograms) {
		t.Fatalf("expected %d draw program passes, got %v (want %v)", len(wantPrograms), ctx.multiDrawPrograms, wantPrograms)
	}
	for i, got := range ctx.multiDrawPrograms {
		if got != wantPrograms[i] {
			t.Fatalf("draw pass %d used program %d, want %d (all=%v)", i, got, wantPrograms[i], ctx.multiDrawPrograms)
		}
	}

	// Solid pass must set DepthFunc(EQUAL) and DepthMask(false), and must never enable SAMPLE_ALPHA_TO_COVERAGE when MSAA == 0.
	hasEqual := false
	for _, fn := range ctx.depthFuncCalls {
		if fn == api.EQUAL {
			hasEqual = true
			break
		}
	}
	if !hasEqual {
		t.Fatalf("expected DepthFunc(EQUAL) during Early-Z solid shading, got %v", ctx.depthFuncCalls)
	}
	for _, cap := range ctx.enabledHistory {
		if cap == api.SAMPLE_ALPHA_TO_COVERAGE {
			t.Fatalf("SAMPLE_ALPHA_TO_COVERAGE must not be enabled when MSAA samples == 0")
		}
	}

	// Solid-only shadow atlas pass must skip cutout depth shader completely.
	d.SetShadowsEnabled(true)
	dcrSolidOnly := gpu.NewDrawCommandsRender(ctx, "test_solid_only", false)
	dcrSolidOnly.Prepare([]*gpu.DrawCommand{cmdSolid})
	ctx.multiDrawPrograms = nil
	d.Render(dcrSolidOnly)
	for _, prg := range ctx.multiDrawPrograms {
		if prg != d.prgSolid {
			t.Fatalf("solid-only shadow pass must use prgSolid (%d), got %d", d.prgSolid, prg)
		}
	}
}

type stubTexHandler struct{ tex *textures.Texture }

func (m *stubTexHandler) GetNames() []string { return []string{"WALL"} }
func (m *stubTexHandler) GetFrames(_ []string) []*textures.Texture {
	return []*textures.Texture{m.tex}
}

func makeRotatedViewMatrix(t *testing.T) *engine.ViewMatrix {
	t.Helper()
	wallTex := textures.NewTexture("WALL", 1, 4, 4, false)
	texH := &stubTexHandler{tex: wallTex}
	cal := config.NewConfigCalibration()
	mat := config.NewConfigMaterial("m", []string{"WALL"}, config.MaterialKindSolid)
	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = mat
	sec.Ceil = mat
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: -500}, geometry.XY{X: 500, Y: -500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: -500}, geometry.XY{X: 500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: 500}, geometry.XY{X: -500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: 500}, geometry.XY{X: -500, Y: -500})
			s.Middle = mat
			return s
		}(),
	}
	p := config.NewConfigPlayer(geometry.XYZ{X: 15, Y: 25, Z: 40}, 0.75, 40, 1800, 2.5, 66.0)
	_ = common.NewPlayer(p)
	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p, nil, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}
	vi := engine.NewViewMatrix()
	pl := en.GetPlayer()
	vi.Update(pl)
	return vi
}

func TestLightsCPUViewSpaceTransformAndAttenuation(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(t, ctx, 0)
	l := h.GetLights()
	m := h.GetMapMetrics()

	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}

	vi := makeRotatedViewMatrix(t)
	m.SetCenter(15, 25, 0)
	m.Update(vi, 1.2, 1.5)

	// Layout per light: [posX,posY,posZ,type, r,g,b,intensity, dirX,dirY,dirZ,falloff, cutOff,outerCutOff, pad1,pad2]
	raw := make([]float32, 32)
	// Light 0: Spotlight
	raw[0], raw[1], raw[2], raw[3] = 100, 50, -40, 1
	raw[4], raw[5], raw[6], raw[7] = 1.0, 0.8, 0.6, 2.5
	raw[8], raw[9], raw[10], raw[11] = 2.0, 0.0, 0.0, 80.0
	raw[12], raw[13] = 0.9, 0.75
	// Light 1: Small point light
	raw[16], raw[17], raw[18], raw[19] = -30, 10, 75, 0
	raw[20], raw[21], raw[22], raw[23] = 0.2, 0.4, 1.0, 0.05 // < 0.1 clamps to 0.1
	raw[24], raw[25], raw[26], raw[27] = 0.0, 1.0, 0.0, 0.5  // < 1.0 clamps to 1.0
	raw[28], raw[29] = 0.0, 0.0

	l.Prepare(unsafe.Pointer(&raw[0]), int32(len(raw)*4), 2)

	if len(ctx.lastUBOData) != 32 {
		t.Fatalf("expected 32 floats uploaded to LightsBlock UBO, got %d", len(ctx.lastUBOData))
	}

	wx, wy, wz := m.TransformPointToView(100, 50, -40)
	dx, dy, dz := m.TransformDirectionNormalizedToView(2.0, 0.0, 0.0)
	wantRadSq0, wantInvScale0 := metrics.ComputeLightAttenuation(2.5, 80.0)

	if math.Abs(float64(ctx.lastUBOData[0]-wx)) > 1e-5 ||
		math.Abs(float64(ctx.lastUBOData[1]-wy)) > 1e-5 ||
		math.Abs(float64(ctx.lastUBOData[2]-wz)) > 1e-5 {
		t.Fatalf("light 0 view pos mismatch: got (%v,%v,%v), want (%v,%v,%v)",
			ctx.lastUBOData[0], ctx.lastUBOData[1], ctx.lastUBOData[2], wx, wy, wz)
	}
	dirLen := math.Sqrt(float64(ctx.lastUBOData[8]*ctx.lastUBOData[8] + ctx.lastUBOData[9]*ctx.lastUBOData[9] + ctx.lastUBOData[10]*ctx.lastUBOData[10]))
	if math.Abs(dirLen-1.0) > 1e-5 {
		t.Fatalf("light 0 view dir not normalized: len=%v", dirLen)
	}
	if math.Abs(float64(ctx.lastUBOData[8]-dx)) > 1e-5 ||
		math.Abs(float64(ctx.lastUBOData[9]-dy)) > 1e-5 ||
		math.Abs(float64(ctx.lastUBOData[10]-dz)) > 1e-5 {
		t.Fatalf("light 0 view dir mismatch: got (%v,%v,%v), want (%v,%v,%v)",
			ctx.lastUBOData[8], ctx.lastUBOData[9], ctx.lastUBOData[10], dx, dy, dz)
	}
	if math.Abs(float64(ctx.lastUBOData[14]-wantRadSq0)) > 1e-2 ||
		math.Abs(float64(ctx.lastUBOData[15]-wantInvScale0)) > 1e-6 {
		t.Fatalf("light 0 attenuation mismatch: got (%v,%v), want (%v,%v)",
			ctx.lastUBOData[14], ctx.lastUBOData[15], wantRadSq0, wantInvScale0)
	}

	// Verify minimum clamping on light 1 (effIntensity=0.1, falloff=1.0 -> effRadius = 0.4605 -> sq ~ 0.21206025).
	wantRadSq1, wantInvScale1 := metrics.ComputeLightAttenuation(0.05, 0.5)
	if math.Abs(float64(ctx.lastUBOData[30]-wantRadSq1)) > 1e-5 ||
		math.Abs(float64(ctx.lastUBOData[31]-wantInvScale1)) > 1e-5 {
		t.Fatalf("light 1 clamped attenuation mismatch: got (%v,%v), want (%v,%v)",
			ctx.lastUBOData[30], ctx.lastUBOData[31], wantRadSq1, wantInvScale1)
	}

	// Steady-state Lights.Prepare must make 0 heap allocations per call.
	allocs := testing.AllocsPerRun(20, func() {
		l.Prepare(unsafe.Pointer(&raw[0]), int32(len(raw)*4), 2)
	})
	if allocs > 0 {
		t.Fatalf("Lights.Prepare allocated %.2f objects/op, want 0", allocs)
	}
}

func TestDepthRenderShadowAtlasStateIsolationAndPolygonOffset(t *testing.T) {
	ctx := newRecordingCtx()
	calCfg := config.NewConfigCalibration()
	cal := model.NewCalibration(calCfg, nil)
	cal.MSAASamples = 0
	flashCfg := config.NewFlash()
	flashCfg.FovDeg = 80.0
	flashCfg.ShadowFovFactor = 1.35
	flashCfg.ZNear = 0.1
	flashCfg.ZFar = 8192.0
	flashCfg.Intensity = 0.02
	flashCfg.Falloff = 2000.0
	flash := things.NewFlash(flashCfg)
	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, flash, 640, 480); err != nil {
		t.Fatalf("Handler.Setup failed: %v", err)
	}
	m := h.GetMain()
	d := h.GetDepth()
	h.GetPost().EnsureAllocated()
	if err := m.Init(); err != nil {
		t.Fatalf("Main.Init failed: %v", err)
	}
	if err := d.Compile(diskAssets{}); err != nil {
		t.Fatalf("Depth.Compile failed: %v", err)
	}

	solidMat, cutoutMat := makeSolidAndCutoutMaterials(t)
	decalMat := textures.NewMaterial("decal_mat", []*textures.Texture{solidMat.GetCurrentFrame()}, int(config.MaterialKindSolid))
	decalMat.SetDepthWrite(true)
	decalMat.SetPolygonOffset(true)

	cmdSolid := gpu.NewDrawCommand()
	cmdSolid.SetFirstIndex(0)
	cmdSolid.SetIndexCount(6)
	cmdSolid.SetMaterial(solidMat)
	cmdDecal := gpu.NewDrawCommand()
	cmdDecal.SetFirstIndex(6)
	cmdDecal.SetIndexCount(6)
	cmdDecal.SetMaterial(decalMat)
	cmdCutout := gpu.NewDrawCommand()
	cmdCutout.SetFirstIndex(12)
	cmdCutout.SetIndexCount(6)
	cmdCutout.SetMaterial(cutoutMat)

	dcr := gpu.NewDrawCommandsRender(ctx, "shadow_state", false)
	dcr.Prepare([]*gpu.DrawCommand{cmdSolid, cmdDecal, cmdCutout})

	d.SetShadowsEnabled(true)
	d.Prepare()
	ctx.scissorCalls = nil
	ctx.polygonOffsetCalls = nil
	ctx.drawPolyOffsets = nil
	ctx.drawDepthClamps = nil
	ctx.drawScissors = nil
	ctx.enabledHistory = nil
	ctx.disabledHistory = nil

	d.Render(dcr)

	// 4 draw batches per tile (Solid non-decal + Solid decal across Room & Flashlight, then Cutout across Room & Flashlight -> wait: Room/Flash solid = 2 batches each, Room/Flash cutout = 1 batch each => 6 draws total).
	if len(ctx.drawScissors) != 6 {
		t.Fatalf("expected 6 draw calls across Room + Flashlight solid/cutout passes, got %d", len(ctx.drawScissors))
	}

	// Verify every draw call inside Depth.Render keeps POLYGON_OFFSET_FILL active and never resets to (-1, -1).
	for i, poActive := range ctx.drawPolyOffsets {
		if !poActive {
			t.Fatalf("draw %d had POLYGON_OFFSET_FILL disabled during shadow atlas pass", i)
		}
	}
	for _, po := range ctx.polygonOffsetCalls {
		if po[0] <= 0 || po[1] <= 0 {
			t.Fatalf("shadow atlas pass set negative/zero PolygonOffset(%v, %v)", po[0], po[1])
		}
	}

	// Room passes (draw 0, 1 solid; draw 4 cutout) must have DEPTH_CLAMP=true & Scissor(0,1024,1024,1024).
	// Flashlight passes (draw 2, 3 solid; draw 5 cutout) must have DEPTH_CLAMP=false & Scissor(1024,1024,1024,1024).
	wantScissor := [6][4]int32{
		{0, 1024, 1024, 1024},
		{0, 1024, 1024, 1024},
		{1024, 1024, 1024, 1024},
		{1024, 1024, 1024, 1024},
		{0, 1024, 1024, 1024},
		{1024, 1024, 1024, 1024},
	}
	wantClamp := [6]bool{true, true, false, false, true, false}
	for i := 0; i < 6; i++ {
		if ctx.drawScissors[i] != wantScissor[i] {
			t.Fatalf("draw %d scissor=%v, want %v", i, ctx.drawScissors[i], wantScissor[i])
		}
		if ctx.drawDepthClamps[i] != wantClamp[i] {
			t.Fatalf("draw %d depthClamp=%v, want %v", i, ctx.drawDepthClamps[i], wantClamp[i])
		}
	}
	// SCISSOR_TEST must be disabled at exit so later passes are not clipped.
	if ctx.disabledHistory[0] != api.CULL_FACE || ctx.polyOffsetActive || ctx.depthClampActive {
		t.Fatalf("expected clean state restore after Depth.Render (polyActive=%v, clampActive=%v)", ctx.polyOffsetActive, ctx.depthClampActive)
	}
	disabledScissor := false
	for _, cap := range ctx.disabledHistory {
		if cap == api.SCISSOR_TEST {
			disabledScissor = true
			break
		}
	}
	if !disabledScissor {
		t.Fatalf("expected SCISSOR_TEST to be disabled on exit from Depth.Render")
	}
}

func TestFlashlightShadowAlignmentAndDepthPrecision(t *testing.T) {
	flashCfg := config.NewFlash()
	flashCfg.FovDeg = 80.0
	flashCfg.ShadowFovFactor = 1.35
	flashCfg.ZNear = 0.1
	flashCfg.ZFar = 8192.0
	flashCfg.Intensity = 0.02
	flashCfg.Falloff = 2000.0
	flashCfg.OffsetX = 0.2
	flashCfg.OffsetY = 0.1
	flash := things.NewFlash(flashCfg)

	flM := metrics.NewFlashlight(flash)
	identityView := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}

	const flashX = float32(5.0)
	const flashY = float32(-0.9)
	const sens = float32(0.01)

	flM.Update(&identityView[0], flashX, flashY, sens)
	dirX, dirY, dirZ := flM.ComputeAim(flashX, flashY, sens)

	// Spot direction matches main_opaque.frag: normalize(vec3(dirX,dirY,dirZ)*512 - vec3(flashX,flashY,0))
	tx, ty, tz := dirX*512.0-flashX, dirY*512.0-flashY, dirZ*512.0
	tLen := float32(math.Sqrt(float64(tx*tx + ty*ty + tz*tz)))
	spotX, spotY, spotZ := tx/tLen, ty/tLen, tz/tLen

	// A point 200 units ahead along the spotlight center ray MUST project to shadow UV (0.5, 0.5)
	// AND preserve at least 20x higher z resolution than ZNear=0.1 (pz < 0.992, not >= 0.9995).
	var fSpace [16]float32
	copy(fSpace[:], (*[16]float32)(unsafe.Pointer(flM.GetShadowSpacePtr()))[:])
	const dist = float32(200.0)
	wx, wy, wz := flashX+spotX*dist, flashY+spotY*dist, spotZ*dist
	cx := fSpace[0]*wx + fSpace[4]*wy + fSpace[8]*wz + fSpace[12]
	cy := fSpace[1]*wx + fSpace[5]*wy + fSpace[9]*wz + fSpace[13]
	cz := fSpace[2]*wx + fSpace[6]*wy + fSpace[10]*wz + fSpace[14]
	cw := fSpace[3]*wx + fSpace[7]*wy + fSpace[11]*wz + fSpace[15]
	if cw <= 0 {
		t.Fatalf("expected positive clip w along spotlight axis, got %v", cw)
	}
	px := (cx/cw)*0.5 + 0.5
	py := (cy/cw)*0.5 + 0.5
	pz := (cz/cw)*0.5 + 0.5
	if math.Abs(float64(px-0.5)) > 1e-3 || math.Abs(float64(py-0.5)) > 1e-3 {
		t.Fatalf("flashlight center ray did not project to shadow map center (0.5,0.5): got (%.4f,%.4f)", px, py)
	}
	if pz >= 0.992 || pz <= 0.9 {
		t.Fatalf("unexpected depth value pz=%.6f at d=200 (want tighter near-plane resolution <0.992)", pz)
	}

	// Verify UpdateWithAim aligns shadow camera from wall-clamped origin (14, -0.9, 0) toward raw SwayOffsetX=50 target.
	flM.UpdateWithAim(&identityView[0], 14.0, flashY, 50.0, flashY, sens)
	ax, ay, az := flM.ResolveAim(14.0, flashY, sens)
	wantAimX := float32(flashCfg.OffsetX) - 50.0*sens
	if math.Abs(float64(ax-wantAimX)) > 1e-5 {
		t.Fatalf("ResolveAim did not preserve configured raw sway direction: got %v, want %v", ax, wantAimX)
	}
	atx, aty, atz := ax*512.0-14.0, ay*512.0-flashY, az*512.0
	atLen := float32(math.Sqrt(float64(atx*atx + aty*aty + atz*atz)))
	wx2, wy2, wz2 := 14.0+(atx/atLen)*dist, flashY+(aty/atLen)*dist, (atz/atLen)*dist
	copy(fSpace[:], (*[16]float32)(unsafe.Pointer(flM.GetShadowSpacePtr()))[:])
	cx2 := fSpace[0]*wx2 + fSpace[4]*wy2 + fSpace[8]*wz2 + fSpace[12]
	cy2 := fSpace[1]*wx2 + fSpace[5]*wy2 + fSpace[9]*wz2 + fSpace[13]
	cw2 := fSpace[3]*wx2 + fSpace[7]*wy2 + fSpace[11]*wz2 + fSpace[15]
	px2 := (cx2/cw2)*0.5 + 0.5
	py2 := (cy2/cw2)*0.5 + 0.5
	if cw2 <= 0 || math.Abs(float64(px2-0.5)) > 1e-3 || math.Abs(float64(py2-0.5)) > 1e-3 {
		t.Fatalf("UpdateWithAim shadow center misaligned: got uv=(%.4f,%.4f), w=%v", px2, py2, cw2)
	}
}

func TestMatrixInverse4x4ExactIdentity(t *testing.T) {
	// Include rotation + translation like Quake/Doom ViewMatrix so inv[2], inv[6] are exercised cleanly.
	ang := 0.785398
	c := float32(math.Cos(ang))
	s := float32(math.Sin(ang))
	m := [16]float32{
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		-495, -128, -337, 1,
	}
	inv, ok := metrics.MatrixInverse4x4(m)
	if !ok {
		t.Fatal("MatrixInverse4x4 returned !ok on invertible matrix")
	}
	prod := metrics.MatrixMultiply4x4(m, inv)
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			want := float32(0)
			if col == row {
				want = 1
			}
			got := prod[col*4+row]
			if math.Abs(float64(got-want)) > 1e-3 {
				t.Fatalf("M * M^-1 mismatch at (%d,%d): got=%v want=%v (inv=%v)", row, col, got, want, inv)
			}
		}
	}
}

func TestDepthDoubleBufferShadowAtlasAndUnbinds(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(t, ctx, 0)
	if err := h.GetMain().Compile(diskAssets{}); err != nil {
		t.Fatalf("Main.Compile failed: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		t.Fatalf("Main.Init failed: %v", err)
	}
	if err := h.GetLights().Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(diskAssets{}); err != nil {
		t.Fatalf("Depth.Compile failed: %v", err)
	}
	p := h.GetPost()
	if err := p.Compile(diskAssets{}); err != nil {
		t.Fatalf("Post.Compile failed: %v", err)
	}
	_ = p.SetupSamplers()
	b := h.GetBloom()
	if err := b.Compile(diskAssets{}); err != nil {
		t.Fatalf("Bloom.Compile failed: %v", err)
	}
	_ = b.SetupSamplers()
	p.EnsureAllocated()

	// Verify solid depth program omits ALPHA_CUTOUT while cutout depth program defines it.
	for _, sh := range ctx.programShaders[d.prgSolid] {
		if strings.Contains(ctx.shaderSources[sh], "#define ALPHA_CUTOUT 1") {
			t.Fatalf("prgSolid must not define ALPHA_CUTOUT")
		}
	}
	hasAlphaCutout := false
	for _, sh := range ctx.programShaders[d.prg] {
		if strings.Contains(ctx.shaderSources[sh], "#define ALPHA_CUTOUT 1") {
			hasAlphaCutout = true
			break
		}
	}
	if !hasAlphaCutout {
		t.Fatalf("cutout depth prg must inject #define ALPHA_CUTOUT 1")
	}

	solidMat, _ := makeSolidAndCutoutMaterials(t)
	cmdSolid := gpu.NewDrawCommand()
	cmdSolid.SetFirstIndex(0)
	cmdSolid.SetIndexCount(6)
	cmdSolid.SetMaterial(solidMat)
	dcr := gpu.NewDrawCommandsRender(ctx, "atlas_dbl", false)
	dcr.Prepare([]*gpu.DrawCommand{cmdSolid})

	// Simulate Textures.BindBuckets() leaving unit 11/13 active before Depth.Render creates atlas.
	ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitEmissiveBase + 3)
	d.SetShadowsEnabled(true)
	h.GetMain().SetShadowsEnabled(true)
	d.Prepare()

	d.Render(dcr)
	tex1 := d.GetShadowAtlasTexture()
	if tex1 == 0 {
		t.Fatalf("expected non-zero shadow atlas texture on frame 1")
	}
	if ctx.boundTex2D[api.TexUnitShadowAtlas] != 0 || ctx.boundTex2D[api.TexUnitEmissiveBase+3] != 0 {
		t.Fatalf("DepthMap.Update leaked TEXTURE_2D binding: unit14=%d unit11=%d",
			ctx.boundTex2D[api.TexUnitShadowAtlas], ctx.boundTex2D[api.TexUnitEmissiveBase+3])
	}

	h.GetMain().Render(dcr, 0.25, 0, 0, 1)
	if ctx.boundTex2D[api.TexUnitShadowAtlas] != 0 {
		t.Fatalf("Main.Render left shadow atlas texture bound on unit %d: %d",
			api.TexUnitShadowAtlas, ctx.boundTex2D[api.TexUnitShadowAtlas])
	}
	b.Render()
	if ctx.boundTex2D[0] != 0 {
		t.Fatalf("Bloom.Render left TEXTURE_2D bound on unit 0: %d", ctx.boundTex2D[0])
	}
	p.Render(640, 480)
	if ctx.boundTex2D[0] != 0 || ctx.boundTex2D[1] != 0 {
		t.Fatalf("Post.Render left TEXTURE_2D bound on units 0/1: %d, %d", ctx.boundTex2D[0], ctx.boundTex2D[1])
	}

	d.Render(dcr)
	tex2 := d.GetShadowAtlasTexture()
	if tex2 == 0 || tex2 == tex1 {
		t.Fatalf("expected double-buffered shadow atlas to alternate textures: tex1=%d tex2=%d", tex1, tex2)
	}
	d.Render(dcr)
	tex3 := d.GetShadowAtlasTexture()
	if tex3 != tex1 {
		t.Fatalf("expected frame 3 shadow atlas to reuse tex1 (%d), got %d", tex1, tex3)
	}

	allocs := testing.AllocsPerRun(50, func() {
		d.Render(dcr)
	})
	if allocs > 0 {
		t.Fatalf("Depth.Render allocated %.2f objects/op, want 0", allocs)
	}
}
