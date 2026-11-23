package shaders

import (
	"fmt"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// DrawCommand represents a rendering command with a starting index and the number of indices to be drawn.
type DrawCommand struct {
	firstIndex int32
	indexCount int32
	material   *textures.Material
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

// Reset resets the RenderBatch, clearing its length and setting depthWrite, polygonOffset, and cutout mode.
func (rb *RenderBatch) Reset(depthWrite int, polygonOffset int, hasCutout int) {
	rb.len = 0
	rb.depthWrite = depthWrite
	rb.polygonOffset = polygonOffset
	rb.hasCutout = hasCutout
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
}

// NewDrawCommandsRender creates and initializes a new DrawCommandsRender with specified context and batch type.
func NewDrawCommandsRender(ctx api.IContext, name string, isAdditive bool) *DrawCommandsRender {
	const startSize = 16
	dcr := &DrawCommandsRender{
		ctx:        ctx,
		name:       name,
		batches:    make([]*RenderBatch, startSize),
		isAdditive: isAdditive,
	}
	for idx := range dcr.batches {
		dcr.batches[idx] = NewRenderBatch()
	}
	return dcr
}

// SetProfiler sets the GpuProfiler instance to be used for profiling GPU performance in DrawCommandsRender.
func (w *DrawCommandsRender) SetProfiler(profiler *GpuProfiler) {
	w.profiler = profiler
}

// HasCommands returns true if there are render batches available to process; otherwise, it returns false.
func (w *DrawCommandsRender) HasCommands() bool {
	return w.batchesLen > 0
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

// Prepare organizes and processes an array of DrawCommand instances into renderable batches for efficient rendering.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batchesLen = 0
	w.hasSolidPrepass = false
	w.hasCutoutPrepass = false
	w.hasSolid = false
	w.hasCutout = false

	if len(dc) == 0 {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1
	currentHasCutout := -1

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

		if depthWrite != currentDepthWrite || polygonOffset != currentPolygonOffset || hasCutout != currentHasCutout || currentBatch == nil {
			if w.batchesLen >= len(w.batches) {
				w.grow()
			}
			currentBatch = w.batches[w.batchesLen]
			currentBatch.Reset(depthWrite, polygonOffset, hasCutout)
			w.batchesLen++
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
			currentHasCutout = hasCutout
		}
		currentBatch.Add(cmd.indexCount, cmd.firstIndex*4)
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

func (w *DrawCommandsRender) renderFiltered(cutoutFilter int) {
	if w.batchesLen == 0 {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1

	for i := 0; i < w.batchesLen; i++ {
		b2 := w.batches[i]
		if cutoutFilter >= 0 && b2.hasCutout != cutoutFilter {
			continue
		}
		w.applyDepthState(&currentDepthWrite, &currentPolygonOffset, b2.depthWrite, b2.polygonOffset)
		w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
	}

	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}

	if currentPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
}

// Render executes the rendering process for all batched draw commands, managing depth writes and polygon offset states.
func (w *DrawCommandsRender) Render() {
	w.renderFiltered(-1)
}

// RenderCutoutFiltered renders only solid (cutout=false) or alpha-cutout (cutout=true) batches.
func (w *DrawCommandsRender) RenderCutoutFiltered(cutout bool) {
	if cutout {
		w.renderFiltered(1)
		return
	}
	w.renderFiltered(0)
}

// RenderPrepass draws prepass-eligible (depth-writing, non-offset) batches filtered by cutout mode.
// Caller must set ColorMask(false, false, false, false), DepthMask(true), DepthFunc(LESS), and disable POLYGON_OFFSET_FILL.
func (w *DrawCommandsRender) RenderPrepass(cutout bool) {
	if w.batchesLen == 0 {
		return
	}
	wantCutout := 0
	if cutout {
		wantCutout = 1
	}
	for i := 0; i < w.batchesLen; i++ {
		b2 := w.batches[i]
		if b2.depthWrite == 1 && b2.polygonOffset == 0 && b2.hasCutout == wantCutout {
			w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
		}
	}
}

// RenderShade draws the color shading pass for either solid (cutout=false) or alpha-cutout (cutout=true) batches
// after Z-Prepass has populated depth for depth-writing non-offset batches.
func (w *DrawCommandsRender) RenderShade(cutout bool) {
	if w.batchesLen == 0 {
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

	for i := 0; i < w.batchesLen; i++ {
		b2 := w.batches[i]
		if b2.hasCutout != wantCutout {
			continue
		}
		targetDepthFunc := uint32(api.LEQUAL)
		targetDepthWrite := b2.depthWrite
		targetPolygonOffset := b2.polygonOffset
		if b2.depthWrite == 1 && b2.polygonOffset == 0 {
			targetDepthFunc = prepassDepthFunc
			targetDepthWrite = 0
		}
		if targetDepthFunc != curDepthFunc {
			w.ctx.DepthFunc(targetDepthFunc)
			curDepthFunc = targetDepthFunc
		}
		w.applyDepthState(&curDepthWrite, &curPolygonOffset, targetDepthWrite, targetPolygonOffset)
		w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
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
