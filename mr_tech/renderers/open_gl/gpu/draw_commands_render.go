package gpu

import (
	"cmp"
	"fmt"
	"slices"
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

type staticMultiDrawBatch struct {
	vao            uint32
	depthWrite     int
	polygonOffset  int
	cullMode       int
	hasCutout      bool
	mc             []int32
	mi             []uintptr
	len            int32
	mcPtr          *int32
	miPtr          *unsafe.Pointer
	firstOffsetPtr unsafe.Pointer
	lastEndIndex   int32
}

func newStaticMultiDrawBatch() *staticMultiDrawBatch {
	const initCap = 64
	b := &staticMultiDrawBatch{
		mc: make([]int32, initCap),
		mi: make([]uintptr, initCap),
	}
	b.mcPtr = &b.mc[0]
	b.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&b.mi[0]))
	return b
}

func (b *staticMultiDrawBatch) reset(vao uint32, depthWrite, polygonOffset, cullMode int, hasCutout bool) {
	b.vao = vao
	b.depthWrite = depthWrite
	b.polygonOffset = polygonOffset
	b.cullMode = cullMode
	b.hasCutout = hasCutout
	b.len = 0
	b.lastEndIndex = -1
	b.firstOffsetPtr = nil
}

func (b *staticMultiDrawBatch) appendRange(ctx api.IContext, firstIndex, indexCount int32) {
	if b.len > 0 && b.lastEndIndex == firstIndex {
		b.mc[b.len-1] += indexCount
		b.lastEndIndex += indexCount
		return
	}
	if int(b.len) >= len(b.mc) {
		newLen := len(b.mc) * 2
		if newLen < 64 {
			newLen = 64
		}
		newMC := make([]int32, newLen)
		copy(newMC, b.mc[:b.len])
		newMI := make([]uintptr, newLen)
		copy(newMI, b.mi[:b.len])
		b.mc = newMC
		b.mi = newMI
		b.mcPtr = &b.mc[0]
		b.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&b.mi[0]))
	}
	if b.len == 0 {
		b.firstOffsetPtr = ctx.PtrOffset(int(firstIndex) * 4)
	}
	b.mc[b.len] = indexCount
	b.mi[b.len] = uintptr(firstIndex) * 4
	b.len++
	b.lastEndIndex = firstIndex + indexCount
}

func (b *staticMultiDrawBatch) draw(ctx api.IContext) {
	if b.len == 1 {
		ctx.DrawElements(api.TRIANGLES, b.mc[0], api.UNSIGNED_INT, b.firstOffsetPtr)
		return
	}
	ctx.MultiDrawElements(api.TRIANGLES, b.mcPtr, api.UNSIGNED_INT, b.miPtr, b.len)
}

type instShadowCmd struct {
	mesh           *ResidentMesh
	instanceVBO    uint32
	baseByteOffset int32
	instanceCount  int32
	indexCount     int32
	indexOffsetPtr unsafe.Pointer
	depthWrite     int
	polygonOffset  int
	cullMode       int
	hasCutout      bool
}

// DrawCommandsRender organizes and manages render batches for executing draw commands in a graphics context.
type DrawCommandsRender struct {
	ctx              api.IContext
	batches          []*RenderBatch
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

	staticItems      []*StaticDrawItem
	staticLen        int
	instancedItems   []*InstancedDrawItem
	instancedLen     int
	transparentItems *TransparentDrawItems

	staticScratch []*StaticDrawItem

	shadowSolidStatic     []*staticMultiDrawBatch
	shadowSolidStaticLen  int
	shadowCutoutStatic    []*staticMultiDrawBatch
	shadowCutoutStaticLen int

	prepassSolidStatic     []*staticMultiDrawBatch
	prepassSolidStaticLen  int
	prepassCutoutStatic    []*staticMultiDrawBatch
	prepassCutoutStaticLen int

	shadeSolidStatic     []*staticMultiDrawBatch
	shadeSolidStaticLen  int
	shadeCutoutStatic    []*staticMultiDrawBatch
	shadeCutoutStaticLen int

	renderAllStatic    []*staticMultiDrawBatch
	renderAllStaticLen int

	shadowSolidInst     []instShadowCmd
	shadowSolidInstLen  int
	shadowCutoutInst    []instShadowCmd
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

func (w *DrawCommandsRender) acquireStaticBatch(pool *[]*staticMultiDrawBatch, count *int) *staticMultiDrawBatch {
	idx := *count
	if idx >= len(*pool) {
		oldLen := len(*pool)
		newLen := max(oldLen*2, 16)
		grown := make([]*staticMultiDrawBatch, newLen)
		copy(grown, *pool)
		for i := oldLen; i < newLen; i++ {
			grown[i] = newStaticMultiDrawBatch()
		}
		*pool = grown
	}
	b := (*pool)[idx]
	*count = idx + 1
	return b
}

func (w *DrawCommandsRender) compileShadowStaticBatches(cutout bool, dst *[]*staticMultiDrawBatch, dstLen *int) {
	*dstLen = 0
	if w.staticLen == 0 {
		return
	}
	if len(w.staticScratch) < w.staticLen {
		w.staticScratch = make([]*StaticDrawItem, max(len(w.staticScratch)*2, w.staticLen))
	}
	n := 0
	for i := 0; i < w.staticLen; i++ {
		item := w.staticItems[i]
		sub := item.Sub
		if sub.IndexCount > 0 && sub.HasCutout == cutout {
			w.staticScratch[n] = item
			n++
		}
	}
	if n == 0 {
		return
	}
	scratch := w.staticScratch[:n]
	if n > 1 {
		slices.SortFunc(scratch, func(a, b *StaticDrawItem) int {
			if a.Mesh.VAO != b.Mesh.VAO {
				return cmp.Compare(a.Mesh.VAO, b.Mesh.VAO)
			}
			dwa := 0
			if !w.isAdditive && a.Sub.DepthWrite {
				dwa = 1
			}
			dwb := 0
			if !w.isAdditive && b.Sub.DepthWrite {
				dwb = 1
			}
			if dwa != dwb {
				// Draw DepthWrite=1 first so Depth.Render's initial DepthMask(true) avoids redundant toggles.
				return cmp.Compare(dwb, dwa)
			}
			return cmp.Compare(a.Sub.FirstIndex, b.Sub.FirstIndex)
		})
	}
	var curBatch *staticMultiDrawBatch
	curVAO := uint32(0)
	curDW := -1
	for i := 0; i < n; i++ {
		item := scratch[i]
		sub := item.Sub
		dw := 0
		if !w.isAdditive && sub.DepthWrite {
			dw = 1
		}
		if curBatch == nil || item.Mesh.VAO != curVAO || dw != curDW {
			curVAO = item.Mesh.VAO
			curDW = dw
			curBatch = w.acquireStaticBatch(dst, dstLen)
			curBatch.reset(curVAO, dw, 0, config.CullNone, cutout)
		}
		curBatch.appendRange(w.ctx, sub.FirstIndex, sub.IndexCount)
	}
}

func (w *DrawCommandsRender) compileOrderedStaticBatches(cutoutFilter int, prepassOnly bool, dst *[]*staticMultiDrawBatch, dstLen *int) {
	*dstLen = 0
	if w.staticLen == 0 {
		return
	}
	var curBatch *staticMultiDrawBatch
	curVAO := uint32(0)
	curDW := -1
	curPO := -1
	curCull := -1
	for i := 0; i < w.staticLen; i++ {
		item := w.staticItems[i]
		sub := item.Sub
		if sub.IndexCount == 0 {
			continue
		}
		if cutoutFilter >= 0 && ((cutoutFilter == 1) != sub.HasCutout) {
			continue
		}
		if prepassOnly && (!sub.DepthWrite || sub.PolygonOffset) {
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
		if curBatch == nil || item.Mesh.VAO != curVAO || dw != curDW || po != curPO || sub.CullMode != curCull || curBatch.hasCutout != sub.HasCutout {
			curVAO = item.Mesh.VAO
			curDW = dw
			curPO = po
			curCull = sub.CullMode
			curBatch = w.acquireStaticBatch(dst, dstLen)
			curBatch.reset(curVAO, dw, po, sub.CullMode, sub.HasCutout)
		}
		curBatch.appendRange(w.ctx, sub.FirstIndex, sub.IndexCount)
	}
}

func (w *DrawCommandsRender) compileShadowInstanced(cutout bool, dst *[]instShadowCmd, dstLen *int) {
	*dstLen = 0
	if w.instancedLen == 0 {
		return
	}
	if len(*dst) < w.instancedLen {
		*dst = make([]instShadowCmd, max(len(*dst)*2, w.instancedLen))
	}
	out := *dst
	n := 0
	// Two passes put DepthWrite=1 before DepthWrite=0 without allocation.
	for passDW := 1; passDW >= 0; passDW-- {
		for i := 0; i < w.instancedLen; i++ {
			item := w.instancedItems[i]
			sub := item.Sub
			if sub.IndexCount == 0 || sub.HasCutout != cutout {
				continue
			}
			dw := 0
			if !w.isAdditive && sub.DepthWrite {
				dw = 1
			}
			if dw != passDW {
				continue
			}
			po := 0
			if sub.PolygonOffset {
				po = 1
			}
			out[n] = instShadowCmd{
				mesh:           item.Mesh,
				instanceVBO:    item.InstanceVBO,
				baseByteOffset: item.BaseByteOffset,
				instanceCount:  item.InstanceCount,
				indexCount:     sub.IndexCount,
				indexOffsetPtr: w.ctx.PtrOffset(int(sub.FirstIndex) * 4),
				depthWrite:     dw,
				polygonOffset:  po,
				cullMode:       sub.CullMode,
				hasCutout:      sub.HasCutout,
			}
			n++
		}
	}
	*dstLen = n
}

// SetResidentDraws attaches GPU-resident static and instanced draw slices built for the current frame.
func (w *DrawCommandsRender) SetResidentDraws(staticItems []*StaticDrawItem, staticLen int, instancedItems []*InstancedDrawItem, instancedLen int) {
	w.staticItems = staticItems
	w.staticLen = staticLen
	w.instancedItems = instancedItems
	w.instancedLen = instancedLen
	for i := 0; i < staticLen; i++ {
		sub := staticItems[i].Sub
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
	for i := 0; i < instancedLen; i++ {
		sub := instancedItems[i].Sub
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

	w.compileShadowStaticBatches(false, &w.shadowSolidStatic, &w.shadowSolidStaticLen)
	w.compileShadowStaticBatches(true, &w.shadowCutoutStatic, &w.shadowCutoutStaticLen)
	w.compileShadowInstanced(false, &w.shadowSolidInst, &w.shadowSolidInstLen)
	w.compileShadowInstanced(true, &w.shadowCutoutInst, &w.shadowCutoutInstLen)

	if w.isAdditive {
		w.prepassSolidStaticLen = 0
		w.prepassCutoutStaticLen = 0
		w.shadeSolidStaticLen = 0
		w.shadeCutoutStaticLen = 0
		w.compileOrderedStaticBatches(-1, false, &w.renderAllStatic, &w.renderAllStaticLen)
		return
	}
	w.renderAllStaticLen = 0
	w.compileOrderedStaticBatches(0, true, &w.prepassSolidStatic, &w.prepassSolidStaticLen)
	w.compileOrderedStaticBatches(1, true, &w.prepassCutoutStatic, &w.prepassCutoutStaticLen)
	w.compileOrderedStaticBatches(0, false, &w.shadeSolidStatic, &w.shadeSolidStaticLen)
	w.compileOrderedStaticBatches(1, false, &w.shadeCutoutStatic, &w.shadeCutoutStaticLen)
}

// SetTransparentDraws stores depth-sorted translucent items across streaming, static, and instanced sources.
func (w *DrawCommandsRender) SetTransparentDraws(items *TransparentDrawItems) {
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

// Prepare organizes and processes an array of DrawCommand instances into renderable batches for efficient rendering.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batchesLen = 0
	w.streamSolid = 0
	w.streamCutout = 0
	w.hasSolidPrepass = false
	w.hasCutoutPrepass = false
	w.hasSolid = false
	w.hasCutout = false
	w.staticItems = nil
	w.staticLen = 0
	w.instancedItems = nil
	w.instancedLen = 0
	w.transparentItems = nil
	w.shadowSolidStaticLen = 0
	w.shadowCutoutStaticLen = 0
	w.shadowSolidInstLen = 0
	w.shadowCutoutInstLen = 0
	w.prepassSolidStaticLen = 0
	w.prepassCutoutStaticLen = 0
	w.shadeSolidStaticLen = 0
	w.shadeCutoutStaticLen = 0
	w.renderAllStaticLen = 0

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
			if hasCutout == 1 {
				w.streamCutout++
			} else {
				w.streamSolid++
			}
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
			b.draw(w.ctx)
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
			b.draw(w.ctx)
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
			b.draw(w.ctx)
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
			b.draw(w.ctx)
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
