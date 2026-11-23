package gpu

import (
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type drawStateCtx struct {
	api.IContext
	nextID       uint32
	boundVAO     uint32
	boundProgram uint32
	enableLog    []uint32
	disableLog   []uint32
	cullFaceLog  []uint32
	drawStarts   []uintptr
}

func newDrawStateCtx() *drawStateCtx {
	return &drawStateCtx{nextID: 1}
}

func (c *drawStateCtx) BindVertexArray(vao uint32)                             { c.boundVAO = vao }
func (c *drawStateCtx) BindBuffer(_ uint32, _ uint32)                          {}
func (c *drawStateCtx) DepthFunc(_ uint32)                                     {}
func (c *drawStateCtx) DepthMask(_ bool)                                       {}
func (c *drawStateCtx) Enable(cap uint32)                                      { c.enableLog = append(c.enableLog, cap) }
func (c *drawStateCtx) Disable(cap uint32)                                     { c.disableLog = append(c.disableLog, cap) }
func (c *drawStateCtx) CullFace(mode uint32)                                   { c.cullFaceLog = append(c.cullFaceLog, mode) }
func (c *drawStateCtx) PolygonOffset(_, _ float32)                             {}
func (c *drawStateCtx) Uniform1i(_ int32, _ int32)                             {}
func (c *drawStateCtx) Ptr(_ interface{}) unsafe.Pointer                       { return nil }
func (c *drawStateCtx) PtrOffset(offset int) unsafe.Pointer                    { return unsafe.Pointer(uintptr(offset)) }
func (c *drawStateCtx) EnableVertexAttribArray(_ uint32)                       {}
func (c *drawStateCtx) VertexAttribDivisor(_ uint32, _ uint32)                 {}
func (c *drawStateCtx) GenBuffers(n int32, b *uint32)                          { c.allocIDs(n, b) }
func (c *drawStateCtx) GenVertexArrays(n int32, a *uint32)                     { c.allocIDs(n, a) }
func (c *drawStateCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (c *drawStateCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}

func (c *drawStateCtx) allocIDs(n int32, out *uint32) {
	if out == nil || n <= 0 {
		return
	}
	s := unsafe.Slice(out, int(n))
	for i := range s {
		s[i] = c.nextID
		c.nextID++
	}
}

func (c *drawStateCtx) DrawElements(_ uint32, _ int32, _ uint32, indices unsafe.Pointer) {
	c.drawStarts = append(c.drawStarts, uintptr(indices))
}

func (c *drawStateCtx) MultiDrawElements(_ uint32, _ *int32, _ uint32, _ *unsafe.Pointer, _ int32) {
}

func (c *drawStateCtx) DrawElementsInstanced(_ uint32, _ int32, _ uint32, indices unsafe.Pointer, _ int32) {
	c.drawStarts = append(c.drawStarts, uintptr(indices))
}

func TestPerMaterialCullStateAndShadowDoubleSidedAndTransparentSort(t *testing.T) {
	ctx := newDrawStateCtx()
	dcr := NewDrawCommandsRender(ctx, "cull_test", false)

	tex := textures.NewTexture("T", 1, 32, 32, false)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			tex.SetDiffuse(x, y, int(0xFFFFFFFF))
		}
	}
	frames := []*textures.Texture{tex}
	texAlpha := textures.NewTexture("T_CUTOUT", 1, 32, 32, true)
	alphaFrames := []*textures.Texture{texAlpha}

	matFront := textures.NewMaterial("front", frames, int(config.MaterialKindSolid))
	matFront.SetCullMode(config.CullFront)
	matFront.SetDepthWrite(true)

	matBack := textures.NewMaterial("back", frames, int(config.MaterialKindSolid))
	matBack.SetCullMode(config.CullBack)
	matBack.SetDepthWrite(true)

	matTwoSided := textures.NewMaterial("two", frames, int(config.MaterialKindSolid))
	matTwoSided.SetCullMode(config.CullNone)
	matTwoSided.SetDepthWrite(true)

	matCutout := textures.NewMaterial("cutout", alphaFrames, int(config.MaterialKindSolid))
	matCutout.SetCullMode(config.CullFront)
	matCutout.SetDepthWrite(true)

	cmds := make([]*DrawCommand, 4)
	for i := range cmds {
		cmds[i] = NewDrawCommand()
		cmds[i].SetFirstIndex(int32(i * 6))
		cmds[i].SetIndexCount(6)
	}
	cmds[0].SetMaterial(matFront)
	cmds[1].SetMaterial(matBack)
	cmds[2].SetMaterial(matTwoSided)
	cmds[3].SetMaterial(matCutout)

	dcr.Prepare(cmds)
	dcr.Render()

	if len(ctx.cullFaceLog) < 2 || ctx.cullFaceLog[0] != api.BACK || ctx.cullFaceLog[1] != api.FRONT {
		t.Fatalf("expected CullFront->BACK then CullBack->FRONT transitions, got %v", ctx.cullFaceLog)
	}
	// Last disabled state should include CULL_FACE when entering two-sided / cutout and at end of pass.
	sawDisableCull := false
	for _, cap := range ctx.disableLog {
		if cap == api.CULL_FACE {
			sawDisableCull = true
			break
		}
	}
	if !sawDisableCull {
		t.Fatalf("expected CULL_FACE disabled for two-sided/cutout materials, disableLog=%v", ctx.disableLog)
	}

	// Shadow atlas pass (RenderCutoutFiltered) must keep CULL_FACE disabled even for CullFront/CullBack.
	ctx.cullFaceLog = ctx.cullFaceLog[:0]
	ctx.enableLog = ctx.enableLog[:0]
	dcr.RenderCutoutFiltered(false)
	for _, cap := range ctx.enableLog {
		if cap == api.CULL_FACE {
			t.Fatalf("shadow atlas RenderCutoutFiltered must not enable CULL_FACE")
		}
	}
	if len(ctx.cullFaceLog) != 0 {
		t.Fatalf("shadow atlas RenderCutoutFiltered must not switch CullFace, got %v", ctx.cullFaceLog)
	}

	// Verify back-to-front transparent sorting (far before near, SortOrder ascending tie-breaker).
	mesh := NewResidentMesh(ctx)
	mesh.VAO = 77
	mesh.SubMeshes = []*ResidentSubMesh{
		{FirstIndex: 30, IndexCount: 6, SortOrder: 2, CullMode: config.CullNone},
	}
	mesh.SubMeshesLen = 1

	transItems := NewDrawItemsTransparent(4)
	transItems.Add(&DrawItemTransparent{Kind: TransparentStreaming, DistSq: 100.0, SortOrder: 1, CullMode: config.CullNone, FirstIndex: 10, IndexCount: 6})
	transItems.Add(&DrawItemTransparent{Kind: TransparentStreaming, DistSq: 900.0, SortOrder: 2, CullMode: config.CullNone, FirstIndex: 40, IndexCount: 6})
	transItems.Add(&DrawItemTransparent{Kind: TransparentStatic, DistSq: 900.0, SortOrder: 0, CullMode: config.CullNone, Mesh: mesh, Sub: mesh.SubMeshes[0]})
	transItems.SortTransparentDrawItems()
	dcrTrans := NewDrawCommandsRender(ctx, "trans_test", false)
	dcrTrans.SetTransparentDraws(transItems)
	ctx.drawStarts = ctx.drawStarts[:0]
	dcrTrans.RenderTransparent()

	if len(ctx.drawStarts) != 3 {
		t.Fatalf("expected 3 sorted transparent draws, got %d", len(ctx.drawStarts))
	}
	// Expected draw index byte offsets: 30*4 (DistSq=900, SortOrder=0) -> 40*4 (DistSq=900, SortOrder=2) -> 10*4 (DistSq=100).
	if ctx.drawStarts[0] != 30*4 || ctx.drawStarts[1] != 40*4 || ctx.drawStarts[2] != 10*4 {
		t.Fatalf("unexpected transparent back-to-front order (byte offsets): %v", ctx.drawStarts)
	}

	allocs := testing.AllocsPerRun(50, func() {
		transItems.SortTransparentDrawItems()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocs in SortTransparentDrawItems, got %.2f", allocs)
	}

	// Verify Begin/AddStatic/AddInstanced/AddCommands/End lifecycle, stale trimming, Clear, and 0 allocs.
	var staticSrc PtrPool[DrawItemTransparent]
	st := staticSrc.Acquire()
	st.Kind = TransparentStatic
	st.Mesh = mesh
	st.Sub = &ResidentSubMesh{FirstIndex: 30, IndexCount: 6, CentroidX: 10, CentroidY: 0, CentroidZ: 0, SortOrder: 1, CullMode: config.CullNone}
	transCmd := NewDrawCommand()
	transCmd.SetFirstIndex(10)
	transCmd.SetIndexCount(6)
	transCmd.SetCentroid(3, 0, 0)
	transCmds := []*DrawCommand{transCmd}
	subInst := &ResidentSubMesh{FirstIndex: 60, IndexCount: 6, SortOrder: 0, CullMode: config.CullNone}

	runCycle := func(withInst bool) {
		transItems.Begin()
		transItems.AddStatic(&staticSrc, 0, 0, 0)
		if withInst {
			transItems.AddInstanced(mesh, subInst, 9, 16, 20, 0, 0, 0, 0, 0)
		}
		transItems.AddCommands(transCmds, 0, 0, 0)
		transItems.End()
	}
	runCycle(true)
	if transItems.Len() != 3 || transItems.At(0).Kind != TransparentInstanced || transItems.At(1).Kind != TransparentStatic || transItems.At(2).Kind != TransparentStreaming {
		t.Fatalf("unexpected lifecycle order or count=%d", transItems.Len())
	}
	// Shrink from 3 to 2 items and verify stale slot 2 is nilled cleanly.
	runCycle(false)
	if transItems.Len() != 2 || transItems.data[2] != nil {
		t.Fatalf("expected Trim on End to shrink to 2 and clear stale data[2], got len=%d ptr=%v", transItems.Len(), transItems.data[2])
	}
	cycleAllocs := testing.AllocsPerRun(50, func() {
		runCycle(true)
	})
	if cycleAllocs != 0 {
		t.Fatalf("expected 0 allocs in TransparentDrawItems Begin/Add/End cycle, got %.2f", cycleAllocs)
	}
	transItems.Clear()
	if transItems.Len() != 0 || transItems.data[0] != nil {
		t.Fatalf("expected Clear to reset Len=0 and clear stale entries, got len=%d", transItems.Len())
	}
}
