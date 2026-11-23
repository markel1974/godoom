package shaders

import (
	"fmt"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// DrawCommand represents a rendering command with a starting index and the number of indices to be drawn.
type DrawCommand struct {
	firstIndex int32
	indexCount int32
	material   *textures.Material
	centroidX  float32
	centroidY  float32
	centroidZ  float32
}

// NewDrawCommand initializes and returns a pointer to a new DrawCommand instance with default values.
func NewDrawCommand() *DrawCommand {
	return &DrawCommand{}
}

// SetMaterial assigns a new material to the DrawCommand for rendering.
func (w *DrawCommand) SetMaterial(material *textures.Material) {
	w.material = material
}

// SetFirstIndex sets the starting index for the draw command.
func (w *DrawCommand) SetFirstIndex(firstIndex int32) {
	w.firstIndex = firstIndex
}

// SetIndexCount sets the count of indices to be drawn for the draw command.
func (w *DrawCommand) SetIndexCount(indexCount int32) {
	w.indexCount = indexCount
}

// SetCentroid sets the approximate GL-space centroid for transparent depth sorting.
func (w *DrawCommand) SetCentroid(cx, cy, cz float32) {
	w.centroidX = cx
	w.centroidY = cy
	w.centroidZ = cz
}

// GetCentroid returns the stored GL-space centroid.
func (w *DrawCommand) GetCentroid() (float32, float32, float32) {
	return w.centroidX, w.centroidY, w.centroidZ
}

// IncIndexCount increments the index count of the draw command by the specified amount.
func (w *DrawCommand) IncIndexCount(count int32) {
	w.indexCount += count
}

// GetMaterial retrieves the material associated with the DrawCommand.
func (w *DrawCommand) GetMaterial() *textures.Material {
	return w.material
}

// GetFirstIndex returns the starting index of the draw command used for rendering.
func (w *DrawCommand) GetFirstIndex() int32 {
	return w.firstIndex
}

// GetIndexCount returns the total number of indices to be drawn in this draw command.
func (w *DrawCommand) GetIndexCount() int32 {
	return w.indexCount
}

// RenderBatch is a structure to hold batched rendering data such as index counts and memory pointers.
// It is used to optimize rendering by grouping draw calls with similar states together.
type RenderBatch struct {
	mc            []int32
	mi            []uintptr
	depthWrite    int
	polygonOffset int
	hasCutout     int
	cullMode      int
	len           int
	mcPtr         *int32
	miPtr         *unsafe.Pointer
}

// NewRenderBatch creates and initializes a new RenderBatch with default starting size for internal slices.
func NewRenderBatch() *RenderBatch {
	const startSize = 4096
	rb := &RenderBatch{
		mc:  nil,
		mi:  nil,
		len: 0,
	}
	rb.rebuild(startSize)
	return rb
}

// Reset resets the RenderBatch, clearing its length and setting depthWrite, polygonOffset, cutout, and cull mode.
func (rb *RenderBatch) Reset(depthWrite int, polygonOffset int, hasCutout int, cullMode int) {
	rb.len = 0
	rb.depthWrite = depthWrite
	rb.polygonOffset = polygonOffset
	rb.hasCutout = hasCutout
	rb.cullMode = cullMode
}

// Add appends a new draw command to the RenderBatch with specified index count and EBO byte offset.
func (rb *RenderBatch) Add(indexCount int32, byteOffset int32) {
	if rb.len >= len(rb.mc) {
		rb.grow()
	}
	rb.mc[rb.len] = indexCount
	rb.mi[rb.len] = uintptr(byteOffset)
	rb.len++
}

// rebuild recreates and resizes the internal slices mc and mi to the specified length, ensuring a minimum size of 1.
func (rb *RenderBatch) rebuild(newLen int) {
	if newLen < 1 {
		newLen = 1
	}
	rb.mc = make([]int32, newLen)
	rb.mi = make([]uintptr, newLen)
	rb.mcPtr = &rb.mc[0]
	rb.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&rb.mi[0]))
}

// grow increases the capacity of the RenderBatch's internal buffers to accommodate additional draw commands.
func (rb *RenderBatch) grow() {
	mc, mi := rb.mc, rb.mi
	newLen := rb.len * 2
	rb.rebuild(newLen)
	copy(rb.mc, mc)
	copy(rb.mi, mi)
}

// DrawCommandsRender organizes and manages render batches for executing draw commands in a graphics context.
type DrawCommandsRender struct {
	ctx              api.IContext
	batches          []*RenderBatch
	batchesLen       int
	isAdditive       bool
	name             string
	profiler         *GpuProfiler
	totalDraws       uint64
	totalIndices     uint64
	hasSolidPrepass  bool
	hasCutoutPrepass bool
	hasSolid         bool
	hasCutout        bool

	staticItems      []StaticDrawItem
	instancedItems   []InstancedDrawItem
	transparentItems []TransparentDrawItem

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
		batches:         make([]*RenderBatch, startSize),
		isAdditive:      isAdditive,
		locInstanceMode: -1,
		curInstanceMode: -1,
	}
	for idx := range dcr.batches {
		dcr.batches[idx] = NewRenderBatch()
	}
	return dcr
}

// SetPassState records the streaming VAO ID and u_instanceMode uniform location for the active shader program.
func (w *DrawCommandsRender) SetPassState(streamingVAO uint32, locInstanceMode int32) {
	w.streamingVAO = streamingVAO
	w.locInstanceMode = locInstanceMode
	w.curInstanceMode = -1
}

// SetResidentDraws attaches GPU-resident static and instanced draw slices built for the current frame.
func (w *DrawCommandsRender) SetResidentDraws(staticItems []StaticDrawItem, instancedItems []InstancedDrawItem) {
	w.staticItems = append(w.staticItems[:0], staticItems...)
	w.instancedItems = append(w.instancedItems[:0], instancedItems...)
	for i := range w.staticItems {
		sub := &w.staticItems[i].Mesh.SubMeshes[w.staticItems[i].SubIdx]
		if sub.HasCutout {
			w.hasCutout = true
			if sub.DepthWrite && !sub.PolygonOffset {
				w.hasCutoutPrepass = true
			}
		} else {
			w.hasSolid = true
			if sub.DepthWrite && !sub.PolygonOffset {
				w.hasSolidPrepass = true
			}
		}
	}
	for i := range w.instancedItems {
		sub := &w.instancedItems[i].Mesh.SubMeshes[w.instancedItems[i].SubIdx]
		if sub.HasCutout {
			w.hasCutout = true
			if sub.DepthWrite && !sub.PolygonOffset {
				w.hasCutoutPrepass = true
			}
		} else {
			w.hasSolid = true
			if sub.DepthWrite && !sub.PolygonOffset {
				w.hasSolidPrepass = true
			}
		}
	}
}

// SetTransparentDraws stores depth-sorted translucent items across streaming, static, and instanced sources.
func (w *DrawCommandsRender) SetTransparentDraws(items []TransparentDrawItem) {
	w.transparentItems = append(w.transparentItems[:0], items...)
}

// HasCommands returns true if there are streaming or resident render commands available to process.
func (w *DrawCommandsRender) HasCommands() bool {
	return w.batchesLen > 0 || len(w.staticItems) > 0 || len(w.instancedItems) > 0 || len(w.transparentItems) > 0
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

// Prepare organizes and processes an array of DrawCommand instances into renderable batches for efficient rendering.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batchesLen = 0
	w.hasSolidPrepass = false
	w.hasCutoutPrepass = false
	w.hasSolid = false
	w.hasCutout = false
	w.staticItems = w.staticItems[:0]
	w.instancedItems = w.instancedItems[:0]
	w.transparentItems = w.transparentItems[:0]

	if len(dc) == 0 {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1
	currentHasCutout := -1
	currentCullMode := -1

	var currentBatch *RenderBatch

	for _, cmd := range dc {
		if cmd.indexCount == 0 {
			continue
		}

		w.totalDraws++
		w.totalIndices += uint64(cmd.indexCount)

		depthWrite := 0
		polygonOffset := 0
		hasCutout := 0
		cullMode := config.CullFront

		if cmd.material != nil {
			if !w.isAdditive && cmd.material.GetDepthWrite() {
				depthWrite = 1
			}
			if cmd.material.GetPolygonOffset() {
				polygonOffset = 1
			}
			if !w.isAdditive && cmd.material.HasCutout() {
				hasCutout = 1
			}
			cullMode = cmd.material.GetCullMode()
		} else {
			if !w.isAdditive {
				depthWrite = 1
			}
		}

		if hasCutout == 1 {
			w.hasCutout = true
			if depthWrite == 1 && polygonOffset == 0 {
				w.hasCutoutPrepass = true
			}
		} else {
			w.hasSolid = true
			if depthWrite == 1 && polygonOffset == 0 {
				w.hasSolidPrepass = true
			}
		}

		if depthWrite != currentDepthWrite || polygonOffset != currentPolygonOffset || hasCutout != currentHasCutout || cullMode != currentCullMode || currentBatch == nil {
			if w.batchesLen >= len(w.batches) {
				w.grow()
			}
			currentBatch = w.batches[w.batchesLen]
			currentBatch.Reset(depthWrite, polygonOffset, hasCutout, cullMode)
			w.batchesLen++
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
			currentHasCutout = hasCutout
			currentCullMode = cullMode
		}
		currentBatch.Add(cmd.indexCount, cmd.firstIndex*4)
	}
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

func (w *DrawCommandsRender) renderFilteredImpl(cutoutFilter int, shadowPass bool) {
	if !w.HasCommands() {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1
	curCull := cullStateUnset

	// Shadow passes manage POLYGON_OFFSET_FILL per atlas tile; keep decal overrides disabled there.
	applyShadowOrColorState := func(dw, po int) {
		if shadowPass {
			if dw != currentDepthWrite {
				w.ctx.DepthMask(dw == 1)
				currentDepthWrite = dw
			}
			return
		}
		w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, dw, po)
	}

	// 1. Resident static meshes (u_instanceMode = 0)
	if len(w.staticItems) > 0 {
		w.setInstanceMode(InstanceModeStatic)
		var boundVAO uint32
		for i := range w.staticItems {
			item := &w.staticItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
			if cutoutFilter >= 0 && ((cutoutFilter == 1) != sub.HasCutout) {
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
			applyShadowOrColorState(dw, po)
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, shadowPass)
			if boundVAO != item.Mesh.VAO {
				w.ctx.BindVertexArray(item.Mesh.VAO)
				boundVAO = item.Mesh.VAO
			}
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		}
	}

	// 2. Resident instanced rigid meshes (u_instanceMode = 1)
	if len(w.instancedItems) > 0 {
		w.setInstanceMode(InstanceModeInstanced)
		for i := range w.instancedItems {
			item := &w.instancedItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
			if cutoutFilter >= 0 && ((cutoutFilter == 1) != sub.HasCutout) {
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
			applyShadowOrColorState(dw, po)
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, shadowPass)
			item.Mesh.BindInstanced(item.InstanceVBO, item.BaseByteOffset)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	// 3. Streaming CPU batches (morphing models, billboards, UI; u_instanceMode = 2)
	if w.batchesLen > 0 {
		didBindStreaming := false
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if cutoutFilter >= 0 && b2.hasCutout != cutoutFilter {
				continue
			}
			if !didBindStreaming {
				if w.streamingVAO != 0 {
					w.ctx.BindVertexArray(w.streamingVAO)
				}
				w.setInstanceMode(InstanceModeStreaming)
				didBindStreaming = true
			}
			applyShadowOrColorState(b2.depthWrite, b2.polygonOffset)
			curCull = ApplyMaterialCullState(w.ctx, curCull, b2.cullMode, b2.hasCutout == 1, shadowPass)
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}

	if curCull > cullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
	if !shadowPass && currentPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
}

// Render executes the rendering process for all batched and resident draw commands, managing depth writes, polygon offset, and culling states.
func (w *DrawCommandsRender) Render() {
	w.renderFilteredImpl(-1, false)
}

// RenderCutoutFiltered renders only solid (cutout=false) or alpha-cutout (cutout=true) batches without culling (shadow atlas pass).
func (w *DrawCommandsRender) RenderCutoutFiltered(cutout bool) {
	if cutout {
		w.renderFilteredImpl(1, true)
		return
	}
	w.renderFilteredImpl(0, true)
}

// RenderPrepass draws prepass-eligible (depth-writing, non-offset) batches across resident and streaming geometry.
// Caller must set ColorMask(false, false, false, false), DepthMask(true), DepthFunc(LESS), and disable POLYGON_OFFSET_FILL.
func (w *DrawCommandsRender) RenderPrepass(cutout bool) {
	wantCutout := 0
	if cutout {
		wantCutout = 1
	}
	curCull := cullStateUnset

	if len(w.staticItems) > 0 {
		var boundVAO uint32
		didMode := false
		for i := range w.staticItems {
			item := &w.staticItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
			if !sub.DepthWrite || sub.PolygonOffset || sub.HasCutout != cutout {
				continue
			}
			if !didMode {
				w.setInstanceMode(InstanceModeStatic)
				didMode = true
			}
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, false)
			if boundVAO != item.Mesh.VAO {
				w.ctx.BindVertexArray(item.Mesh.VAO)
				boundVAO = item.Mesh.VAO
			}
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		}
	}

	if len(w.instancedItems) > 0 {
		didMode := false
		for i := range w.instancedItems {
			item := &w.instancedItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
			if !sub.DepthWrite || sub.PolygonOffset || sub.HasCutout != cutout {
				continue
			}
			if !didMode {
				w.setInstanceMode(InstanceModeInstanced)
				didMode = true
			}
			curCull = ApplyMaterialCullState(w.ctx, curCull, sub.CullMode, sub.HasCutout, false)
			item.Mesh.BindInstanced(item.InstanceVBO, item.BaseByteOffset)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	if w.batchesLen > 0 {
		didStream := false
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if b2.depthWrite == 1 && b2.polygonOffset == 0 && b2.hasCutout == wantCutout {
				if !didStream {
					if w.streamingVAO != 0 {
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

	if curCull > cullStateOff {
		w.ctx.Disable(api.CULL_FACE)
	}
}

// RenderShade draws the color shading pass for solid (cutout=false) or alpha-cutout (cutout=true) geometry
// across resident static, instanced rigid, and streaming sources with Early-Z prepass parity.
func (w *DrawCommandsRender) RenderShade(cutout bool) {
	if !w.HasCommands() {
		return
	}
	wantCutout := 0
	if cutout {
		wantCutout = 1
	}
	prepassDepthFunc := uint32(api.EQUAL)
	if cutout {
		prepassDepthFunc = api.LEQUAL
	}

	curDepthFunc := uint32(0)
	curDepthWrite := -1
	curPolygonOffset := -1
	curCull := cullStateUnset

	setState := func(depthWrite, polygonOffset, cullMode int, hasCutout bool) {
		targetDepthFunc := uint32(api.LEQUAL)
		targetDepthWrite := depthWrite
		if depthWrite == 1 && polygonOffset == 0 {
			targetDepthFunc = prepassDepthFunc
			targetDepthWrite = 0
		}
		if targetDepthFunc != curDepthFunc {
			w.ctx.DepthFunc(targetDepthFunc)
			curDepthFunc = targetDepthFunc
		}
		w.applyDepthState(&curDepthWrite, &curPolygonOffset, targetDepthWrite, polygonOffset)
		curCull = ApplyMaterialCullState(w.ctx, curCull, cullMode, hasCutout, false)
	}

	if len(w.staticItems) > 0 {
		var boundVAO uint32
		didMode := false
		for i := range w.staticItems {
			item := &w.staticItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
			if sub.HasCutout != cutout {
				continue
			}
			if !didMode {
				w.setInstanceMode(InstanceModeStatic)
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
			setState(dw, po, sub.CullMode, sub.HasCutout)
			if boundVAO != item.Mesh.VAO {
				w.ctx.BindVertexArray(item.Mesh.VAO)
				boundVAO = item.Mesh.VAO
			}
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		}
	}

	if len(w.instancedItems) > 0 {
		didMode := false
		for i := range w.instancedItems {
			item := &w.instancedItems[i]
			sub := &item.Mesh.SubMeshes[item.SubIdx]
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
			setState(dw, po, sub.CullMode, sub.HasCutout)
			item.Mesh.BindInstanced(item.InstanceVBO, item.BaseByteOffset)
			w.ctx.DrawElementsInstanced(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4), item.InstanceCount)
		}
	}

	if w.batchesLen > 0 {
		didStream := false
		for i := 0; i < w.batchesLen; i++ {
			b2 := w.batches[i]
			if b2.hasCutout != wantCutout {
				continue
			}
			if !didStream {
				if w.streamingVAO != 0 {
					w.ctx.BindVertexArray(w.streamingVAO)
				}
				w.setInstanceMode(InstanceModeStreaming)
				didStream = true
			}
			setState(b2.depthWrite, b2.polygonOffset, b2.cullMode, b2.hasCutout == 1)
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}

	if curCull > cullStateOff {
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
	if len(w.transparentItems) == 0 {
		return
	}
	curCull := cullStateUnset
	curPoly := -1
	var boundVAO uint32

	for i := range w.transparentItems {
		it := &w.transparentItems[i]
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
			sub := &it.Mesh.SubMeshes[it.SubIdx]
			w.ctx.DrawElements(api.TRIANGLES, sub.IndexCount, api.UNSIGNED_INT, w.ctx.PtrOffset(int(sub.FirstIndex)*4))
		case TransparentInstanced:
			w.setInstanceMode(InstanceModeInstanced)
			it.Mesh.BindInstanced(it.InstanceVBO, it.BaseByteOffset)
			boundVAO = it.Mesh.VAOInstanced
			sub := &it.Mesh.SubMeshes[it.SubIdx]
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
	if curCull > cullStateOff {
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

// grow doubles the capacity of the render batches slice and initializes new RenderBatch instances for added capacity.
func (w *DrawCommandsRender) grow() {
	batches := w.batches
	newLen := len(w.batches) * 2
	w.batches = make([]*RenderBatch, newLen)
	copy(w.batches, batches)
	for i := len(batches); i < newLen; i++ {
		w.batches[i] = NewRenderBatch()
	}
}
