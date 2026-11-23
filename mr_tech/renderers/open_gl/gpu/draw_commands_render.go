package gpu

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// DrawCommandsRender organizes and manages render batches for executing draw commands in a graphics context.
type DrawCommandsRender struct {
	ctx              api.IContext
	batches          []*DrawBatch
	batchesLen       int
	streamSolid      int
	streamCutout     int
	isAdditive       bool
	name             string
	profiler         *GpuProfiler
	totalDraws       uint64
	totalIndices     uint64
	hasSolidPrepass  bool
	hasCutoutPrepass bool
	hasSolid         bool
	hasCutout        bool

	staticItems      []*DrawItemStatic
	staticLen        int
	instancedItems   []*DrawItemInstanced
	instancedLen     int
	transparentItems *DrawItemsTransparent

	staticScratch []*DrawItemStatic

	shadowSolidStatic     []*DrawBatchStaticMulti
	shadowSolidStaticLen  int
	shadowCutoutStatic    []*DrawBatchStaticMulti
	shadowCutoutStaticLen int

	prepassSolidStatic     []*DrawBatchStaticMulti
	prepassSolidStaticLen  int
	prepassCutoutStatic    []*DrawBatchStaticMulti
	prepassCutoutStaticLen int

	shadeSolidStatic     []*DrawBatchStaticMulti
	shadeSolidStaticLen  int
	shadeCutoutStatic    []*DrawBatchStaticMulti
	shadeCutoutStaticLen int

	renderAllStatic    []*DrawBatchStaticMulti
	renderAllStaticLen int

	shadowSolidInst     []InstShadowCommand
	shadowSolidInstLen  int
	shadowCutoutInst    []InstShadowCommand
	shadowCutoutInstLen int

	streamingVAO    uint32
	locInstanceMode int32
	curInstanceMode int32
}

// NewDrawCommandsRender creates and initializes a new DrawCommandsRender with specified context and batch type.
func NewDrawCommandsRender(ctx api.IContext, name string, isAdditive bool) *DrawCommandsRender {
	const startSize = 16
	dcr := &DrawCommandsRender{
		ctx:             ctx,
		name:            name,
		batches:         make([]*DrawBatch, startSize),
		isAdditive:      isAdditive,
		locInstanceMode: -1,
		curInstanceMode: -1,
	}
	for idx := range dcr.batches {
		dcr.batches[idx] = NewDrawBatch()
	}
	return dcr
}

// SetPassState records the streaming VAO ID and u_instanceMode uniform location for the active shader program.
func (w *DrawCommandsRender) SetPassState(streamingVAO uint32, locInstanceMode int32) {
	w.streamingVAO = streamingVAO
	w.locInstanceMode = locInstanceMode
	w.curInstanceMode = -1
}

// SetTransparentDraws stores depth-sorted translucent items across streaming, static, and instanced sources.
func (w *DrawCommandsRender) SetTransparentDraws(items *DrawItemsTransparent) {
	w.transparentItems = items
}

// HasCommands returns true if there are streaming or resident render commands available to process.
func (w *DrawCommandsRender) HasCommands() bool {
	return w.batchesLen > 0 || w.staticLen > 0 || w.instancedLen > 0 || (w.transparentItems != nil && w.transparentItems.Len() > 0)
}

// GetName retrieves the name of the DrawCommandsRender instance.
func (w *DrawCommandsRender) GetName() string {
	return w.name
}

// HasSolidPrepass returns true if at least one solid depth-writing batch qualifies for the Z-prepass.
func (w *DrawCommandsRender) HasSolidPrepass() bool { return w.hasSolidPrepass }

// HasCutoutPrepass returns true if at least one alpha-cutout depth-writing batch qualifies for the Z-prepass.
func (w *DrawCommandsRender) HasCutoutPrepass() bool { return w.hasCutoutPrepass }

// HasSolid returns true if at least one non-cutout batch exists.
func (w *DrawCommandsRender) HasSolid() bool { return w.hasSolid }

// HasCutout returns true if at least one alpha-cutout batch exists.
func (w *DrawCommandsRender) HasCutout() bool { return w.hasCutout }

// SetProfiler sets the GpuProfiler instance to be used for profiling GPU performance in DrawCommandsRender.
func (w *DrawCommandsRender) SetProfiler(profiler *GpuProfiler) {
	w.profiler = profiler
}

func (w *DrawCommandsRender) setInstanceMode(mode int32) {
	if w.locInstanceMode != -1 && w.curInstanceMode != mode {
		w.ctx.Uniform1i(w.locInstanceMode, mode)
		w.curInstanceMode = mode
	}
}

func (w *DrawCommandsRender) applyDepthState(curDepthWrite, curPolygonOffset *int, depthWrite, polygonOffset int) {
	if depthWrite != *curDepthWrite {
		w.ctx.DepthMask(depthWrite == 1)
		*curDepthWrite = depthWrite
	}

	if polygonOffset != *curPolygonOffset {
		if polygonOffset == 1 {
			w.ctx.Enable(api.POLYGON_OFFSET_FILL)
			w.ctx.PolygonOffset(-1.0, -1.0)
		} else {
			w.ctx.Disable(api.POLYGON_OFFSET_FILL)
		}
		*curPolygonOffset = polygonOffset
	}
}

// Render executes the rendering process for all batched and resident draw commands, managing depth writes, polygon offset, and culling states.
func (w *DrawCommandsRender) Render() {
	if !w.HasCommands() {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1
	curCull := CullStateUnset
	var boundVAO uint32

	// Resident static meshes (u_instanceMode = 0)
	if w.renderAllStaticLen > 0 {
		w.setInstanceMode(InstanceModeStatic)
		for i := 0; i < w.renderAllStaticLen; i++ {
			b := w.renderAllStatic[i]
			w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, b.depthWrite, b.polygonOffset)
			curCull = ApplyMaterialCullState(w.ctx, curCull, b.cullMode, b.hasCutout, false)
			if boundVAO != b.vao {
				w.ctx.BindVertexArray(b.vao)
				boundVAO = b.vao
			}
			b.Draw(w.ctx)
		}
	} else if w.staticLen > 0 {
		w.setInstanceMode(InstanceModeStatic)
		for i := 0; i < w.staticLen; i++ {
			item := w.staticItems[i]
			sub := item.Sub
			if sub.IndexCount == 0 {
				continue
			}
			dw := 0
			if !w.isAdditive && sub.DepthWrite {
				dw = 1
			}
			po := 0
			if sub.PolygonOffset {
				po = 1
			}
			w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, dw, po)
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, false)
			if boundVAO != item.Mesh.VAO {
				w.ctx.BindVertexArray(item.Mesh.VAO)
				boundVAO = item.Mesh.VAO
			}
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		}
	}

	// Resident instanced rigid meshes (u_instanceMode = 1)
	if w.instancedLen > 0 {
		w.setInstanceMode(InstanceModeInstanced)
		for i := 0; i < w.instancedLen; i++ {
			item := w.instancedItems[i]
			sub := item.Sub
			dw := 0
			if !w.isAdditive && sub.DepthWrite {
				dw = 1
			}
			po := 0
			if sub.PolygonOffset {
				po = 1
			}
			w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, dw, po)
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, false)
			boundVAO = item.Mesh.BindInstancedWithCache(item.InstanceVBO, item.BaseByteOffset, boundVAO)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	// Streaming CPU batches (morphing models, billboards, UI; u_instanceMode = 2)
	if w.batchesLen > 0 {
		if w.streamingVAO != 0 && boundVAO != w.streamingVAO {
			w.ctx.BindVertexArray(w.streamingVAO)
		}
		w.setInstanceMode(InstanceModeStreaming)
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, b2.depthWrite, b2.polygonOffset)
			curCull = ApplyMaterialCullState(w.ctx, curCull, b2.cullMode, b2.hasCutout == 1, false)
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}

	if curCull > CullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
	if currentPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
}

// RenderCutoutFiltered renders only solid (cutout=false) or alpha-cutout (cutout=true) batches without culling (shadow atlas pass).
func (w *DrawCommandsRender) RenderCutoutFiltered(cutout bool) {
	if !w.HasCommands() {
		return
	}

	currentDepthWrite := 1
	wantCutout := 0
	streamCount := w.streamSolid
	staticBatches := w.shadowSolidStatic
	staticBatchLen := w.shadowSolidStaticLen
	instCmds := w.shadowSolidInst
	instLen := w.shadowSolidInstLen
	if cutout {
		wantCutout = 1
		streamCount = w.streamCutout
		staticBatches = w.shadowCutoutStatic
		staticBatchLen = w.shadowCutoutStaticLen
		instCmds = w.shadowCutoutInst
		instLen = w.shadowCutoutInstLen
	}

	var boundVAO uint32
	if staticBatchLen > 0 {
		w.setInstanceMode(InstanceModeStatic)
		for i := 0; i < staticBatchLen; i++ {
			b := staticBatches[i]
			if b.depthWrite != currentDepthWrite {
				w.ctx.DepthMask(b.depthWrite == 1)
				currentDepthWrite = b.depthWrite
			}
			if boundVAO != b.vao {
				w.ctx.BindVertexArray(b.vao)
				boundVAO = b.vao
			}
			b.Draw(w.ctx)
		}
	}

	if instLen > 0 {
		w.setInstanceMode(InstanceModeInstanced)
		for i := 0; i < instLen; i++ {
			cmd := &instCmds[i]
			if cmd.depthWrite != currentDepthWrite {
				w.ctx.DepthMask(cmd.depthWrite == 1)
				currentDepthWrite = cmd.depthWrite
			}
			boundVAO = cmd.mesh.BindInstancedWithCache(cmd.instanceVBO, cmd.baseByteOffset, boundVAO)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, cmd.indexCount, api.UNSIGNED_INT, cmd.indexOffsetPtr, cmd.instanceCount)
		}
	}

	if streamCount > 0 {
		if w.streamingVAO != 0 && boundVAO != w.streamingVAO {
			w.ctx.BindVertexArray(w.streamingVAO)
		}
		w.setInstanceMode(InstanceModeStreaming)
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if b2.hasCutout != wantCutout {
				continue
			}
			if b2.depthWrite != currentDepthWrite {
				w.ctx.DepthMask(b2.depthWrite == 1)
				currentDepthWrite = b2.depthWrite
			}
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}

	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
}

// RenderPrepass draws prepass-eligible (depth-writing, non-offset) batches across resident and streaming geometry.
// Caller must set ColorMask(false, false, false, false), DepthMask(true), DepthFunc(LESS), and disable POLYGON_OFFSET_FILL.
func (w *DrawCommandsRender) RenderPrepass(cutout bool) {
	wantCutout := 0
	streamCount := w.streamSolid
	staticBatches := w.prepassSolidStatic
	staticBatchLen := w.prepassSolidStaticLen
	if cutout {
		wantCutout = 1
		streamCount = w.streamCutout
		staticBatches = w.prepassCutoutStatic
		staticBatchLen = w.prepassCutoutStaticLen
	}
	curCull := CullStateUnset
	var boundVAO uint32

	if staticBatchLen > 0 {
		w.setInstanceMode(InstanceModeStatic)
		for i := 0; i < staticBatchLen; i++ {
			b := staticBatches[i]
			curCull = ApplyMaterialCullState(w.ctx, curCull, b.cullMode, b.hasCutout, false)
			if boundVAO != b.vao {
				w.ctx.BindVertexArray(b.vao)
				boundVAO = b.vao
			}
			b.Draw(w.ctx)
		}
	}

	if w.instancedLen > 0 {
		didMode := false
		for i := 0; i < w.instancedLen; i++ {
			item := w.instancedItems[i]
			sub := item.Sub
			if !sub.DepthWrite || sub.PolygonOffset || sub.HasCutout != cutout {
				continue
			}
			if !didMode {
				w.setInstanceMode(InstanceModeInstanced)
				didMode = true
			}
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, false)
			boundVAO = item.Mesh.BindInstancedWithCache(item.InstanceVBO, item.BaseByteOffset, boundVAO)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	if streamCount > 0 {
		didStream := false
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if b2.depthWrite == 1 && b2.polygonOffset == 0 && b2.hasCutout == wantCutout {
				if !didStream {
					if w.streamingVAO != 0 && boundVAO != w.streamingVAO {
						w.ctx.BindVertexArray(w.streamingVAO)
					}
					w.setInstanceMode(InstanceModeStreaming)
					didStream = true
				}
				curCull = ApplyMaterialCullState(w.ctx, curCull, b2.cullMode, b2.hasCutout == 1, false)
				w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
			}
		}
	}

	if curCull > CullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
}

func (w *DrawCommandsRender) applyShadeState(prepassDepthFunc uint32, curDepthFunc *uint32, curDepthWrite, curPolygonOffset, curCull *int, depthWrite, polygonOffset, cullMode int, hasCutout bool) {
	targetDepthFunc := uint32(api.LEQUAL)
	targetDepthWrite := depthWrite
	if depthWrite == 1 && polygonOffset == 0 {
		targetDepthFunc = prepassDepthFunc
		targetDepthWrite = 0
	}
	if targetDepthFunc != *curDepthFunc {
		w.ctx.DepthFunc(targetDepthFunc)
		*curDepthFunc = targetDepthFunc
	}
	w.applyDepthState(curDepthWrite, curPolygonOffset, targetDepthWrite, polygonOffset)
	*curCull = ApplyMaterialCullState(w.ctx, *curCull, cullMode, hasCutout, false)
}

// RenderShade draws the color shading pass for solid (cutout=false) or alpha-cutout (cutout=true) geometry
// across resident static, instanced rigid, and streaming sources with Early-Z prepass parity.
func (w *DrawCommandsRender) RenderShade(cutout bool) {
	if !w.HasCommands() {
		return
	}
	prepassDepthFunc := uint32(api.EQUAL)
	wantCutout := 0
	streamCount := w.streamSolid
	staticBatches := w.shadeSolidStatic
	staticBatchLen := w.shadeSolidStaticLen
	if cutout {
		wantCutout = 1
		streamCount = w.streamCutout
		prepassDepthFunc = api.LEQUAL
		staticBatches = w.shadeCutoutStatic
		staticBatchLen = w.shadeCutoutStaticLen
	}

	curDepthFunc := uint32(0)
	curDepthWrite := -1
	curPolygonOffset := -1
	curCull := CullStateUnset
	var boundVAO uint32

	if staticBatchLen > 0 {
		w.setInstanceMode(InstanceModeStatic)
		for i := 0; i < staticBatchLen; i++ {
			b := staticBatches[i]
			w.applyShadeState(prepassDepthFunc, &curDepthFunc, &curDepthWrite, &curPolygonOffset, &curCull, b.depthWrite, b.polygonOffset, b.cullMode, b.hasCutout)
			if boundVAO != b.vao {
				w.ctx.BindVertexArray(b.vao)
				boundVAO = b.vao
			}
			b.Draw(w.ctx)
		}
	}

	if w.instancedLen > 0 {
		didMode := false
		for i := 0; i < w.instancedLen; i++ {
			item := w.instancedItems[i]
			sub := item.Sub
			if sub.HasCutout != cutout {
				continue
			}
			if !didMode {
				w.setInstanceMode(InstanceModeInstanced)
				didMode = true
			}
			dw := 0
			if sub.DepthWrite {
				dw = 1
			}
			po := 0
			if sub.PolygonOffset {
				po = 1
			}
			w.applyShadeState(prepassDepthFunc, &curDepthFunc, &curDepthWrite, &curPolygonOffset, &curCull, dw, po, sub.CullMode, sub.HasCutout)
			boundVAO = item.Mesh.BindInstancedWithCache(item.InstanceVBO, item.BaseByteOffset, boundVAO)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	if streamCount > 0 {
		if w.streamingVAO != 0 && boundVAO != w.streamingVAO {
			w.ctx.BindVertexArray(w.streamingVAO)
		}
		w.setInstanceMode(InstanceModeStreaming)
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if b2.hasCutout != wantCutout {
				continue
			}
			w.applyShadeState(prepassDepthFunc, &curDepthFunc, &curDepthWrite, &curPolygonOffset, &curCull, b2.depthWrite, b2.polygonOffset, b2.cullMode, b2.hasCutout == 1)
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}

	if curCull > CullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
	if curPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
	if curDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
	if curDepthFunc != 0 {
		w.ctx.DepthFunc(api.LESS)
	}
}

// RenderTransparent executes depth-sorted translucent items strictly back-to-front.
func (w *DrawCommandsRender) RenderTransparent() {
	if w.transparentItems == nil {
		return
	}
	nTrans := w.transparentItems.Len()
	if nTrans == 0 {
		return
	}
	curCull := CullStateUnset
	curPoly := -1
	var boundVAO uint32

	for i := 0; i < nTrans; i++ {
		it := w.transparentItems.data[i]
		po := 0
		if it.PolygonOffset {
			po = 1
		}
		if po != curPoly {
			if po == 1 {
				w.ctx.Enable(api.POLYGON_OFFSET_FILL)
				w.ctx.PolygonOffset(-1.0, -1.0)
			} else {
				w.ctx.Disable(api.POLYGON_OFFSET_FILL)
			}
			curPoly = po
		}
		curCull = ApplyMaterialCullState(w.ctx, curCull, it.CullMode, false, false)

		switch it.Kind {
		case TransparentStatic:
			w.setInstanceMode(InstanceModeStatic)
			if boundVAO != it.Mesh.VAO {
				w.ctx.BindVertexArray(it.Mesh.VAO)
				boundVAO = it.Mesh.VAO
			}
			sub := it.Sub
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		case TransparentInstanced:
			w.setInstanceMode(InstanceModeInstanced)
			boundVAO = it.Mesh.BindInstancedWithCache(it.InstanceVBO, it.BaseByteOffset, boundVAO)
			sub := it.Sub
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), it.InstanceCount)
		default:
			w.setInstanceMode(InstanceModeStreaming)
			if w.streamingVAO != 0 && boundVAO != w.streamingVAO {
				w.ctx.BindVertexArray(w.streamingVAO)
				boundVAO = w.streamingVAO
			}
			w.ctx.DrawElements(api.TRIANGLES, it.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(it.FirstIndex)*4))
		}
	}

	if curPoly == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
	if curCull > CullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
}

func (w *DrawCommandsRender) RenderProfiled() {
	w.profiler.MeasureGPU("DrawCommandsRender.MultiDraw", w.Render)
	if w.profiler.EndFrame() {
		fmt.Printf("[%s] batches=%d draws=%d indices=%d triangles~=%d\n", w.name, w.batchesLen, w.totalDraws, w.totalIndices, w.totalIndices/3)
		w.totalDraws = 0
		w.totalIndices = 0
	}
}
