package builder

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/queries"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Builder coordinates static world volumes, things, lights, occlusion queries, and shadow tiles for OpenGL rendering.
type Builder struct {
	frustumFront *metrics.Frustum

	bc      *Context
	bl      *Lights
	bs      *Static
	bt      *Things
	bShadow *Shadows
}

// NewBuilder initializes and returns a new Builder instance configured for the active rendering context.
func NewBuilder(ctx api.IContext, tex *vramtex.Textures, cal *model.Calibration, player *things.Player) *Builder {
	return newBuilderWithOcclusionMax(ctx, tex, cal, player, defaultOcclusionMaxEntities)
}

func newBuilderWithOcclusionMax(ctx api.IContext, tex *vramtex.Textures, cal *model.Calibration, player *things.Player, occMax int) *Builder {
	bc := NewContextWithOcclusionMax(ctx, cal, tex, player, occMax)
	bs := NewStatic(bc)
	bt := NewThings(bc)
	bl := NewLights(bc)
	bShadow := NewShadows(bc, bs, bt)

	ff := metrics.NewFrustum()
	if cal != nil {
		ff.Setup(float32(cal.AspectRatio), float32(cal.ZFar), float32(cal.ZNear), cal.PrjScale, cal.PrjScale)
	}

	return &Builder{
		frustumFront: ff,
		bc:           bc,
		bl:           bl,
		bs:           bs,
		bt:           bt,
		bShadow:      bShadow,
	}
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame lighting data.
func (w *Builder) GetShadowLights() ([8]*metrics.Light, int32) {
	return w.bl.GetFrameLights().GetShadowLights()
}

// GetSceneDirectional returns the active global directional light collected this frame, if present.
func (w *Builder) GetSceneDirectional() ([3]float32, [4]float32, bool) {
	return w.bl.GetFrameLights().GetSceneDirectional()
}

// SetSuppressDirectionalForCSM controls whether directional scene lights are omitted from the UBO when SunCSM shades them.
func (w *Builder) SetSuppressDirectionalForCSM(suppress bool) {
	w.bl.GetFrameLights().SetSuppressDirectionalForCSM(suppress)
}

// GetVerticesStride returns the byte stride of streaming vertices.
func (w *Builder) GetVerticesStride() int32 {
	return w.bc.fv.VerticesStride()
}

// GetLightsStride returns the stride size of the lights buffer in bytes.
func (w *Builder) GetLightsStride() int32 {
	return w.bl.GetFrameLights().LightsStride()
}

// GetDrawCommandsOpaque returns the compiled opaque draw command list.
func (w *Builder) GetDrawCommandsOpaque() *draws.DrawCommandsRender {
	return w.bc.dcRenderOpaque
}

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *Builder) GetDrawCommandsAdditive() *draws.DrawCommandsRender {
	return w.bc.dcRenderAdditive
}

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *Builder) GetDrawCommandsLiquid() *draws.DrawCommandsRender {
	return w.bc.dcRenderLiquid
}

// GetDrawCommandsTransparent returns the sorted transparent draw commands prepared for rendering.
func (w *Builder) GetDrawCommandsTransparent() *draws.DrawCommandsRender {
	return w.bc.dcRenderTransparent
}

// GetVertices retrieves pointers, byte sizes, and counts for streaming vertices and indices.
func (w *Builder) GetVertices() (unsafe.Pointer, int, int32, unsafe.Pointer, int, int32) {
	return w.bc.fv.GetVertices()
}

// GetLights retrieves the light data pointer, byte stride size, and active light count.
func (w *Builder) GetLights() (unsafe.Pointer, int32, int32) {
	return w.bl.GetFrameLights().GetLights()
}

// GetSkyMaterial returns the sky material detected among active static volumes.
func (w *Builder) GetSkyMaterial() *textures.Material {
	return w.bs.GetSkyMaterial()
}

// GetUploadHints returns static prefix byte offsets and generation counters used for incremental GPU buffer uploads.
func (w *Builder) GetUploadHints() (staticVertBytes, staticIdxBytes int, staticGen, frameGen uint64) {
	sb, si, sg := w.bs.GetUploadHints()
	return sb, si, sg, w.bc.UploadFrameGen()
}

// ComputeShadows updates per-light shadow tile draw lists and instance transforms for the active depth state.
func (w *Builder) ComputeShadows(en *engine.Engine, d *shaders.Depth) {
	if en == nil || d == nil {
		w.bShadow.DisableAllTiles()
		return
	}
	w.bShadow.Process(en.GetVolumes(), en.GetThings(), d)
}

// GetShadowTiles returns the per-light shadow tile descriptors prepared by ComputeShadows.
func (w *Builder) GetShadowTiles() *[shaders.MaxShadowTiles]shaders.ShadowsTileData {
	return w.bShadow.GetTiles()
}

// Compute processes volumes, lights, and things within the camera frustum and compiles rendering draw commands.
func (w *Builder) Compute(fbw, fbh int32, vi *engine.ViewMatrix, en *engine.Engine) {
	if en == nil || vi == nil {
		return
	}
	w.bc.meshFrame++
	px, py, pz := vi.GetView()
	angle, pitch, roll := vi.GetAngle(), -vi.GetPitch(), vi.GetRoll()
	w.frustumFront.Update(float32(px), float32(py), float32(pz), angle, pitch, roll)

	frustumFront := vi.GetFrontFrustum(w.frustumFront.Get())
	sizeZLight := 0.0
	if w.bc.cal != nil {
		sizeZLight = w.bc.cal.ZLightFar
	}

	if w.bc.occlusion != nil {
		w.bc.occlusion.Reset()
	}

	staticChanged := w.bs.Process(en.GetVolumes(), frustumFront, px, py, pz)
	w.bl.Process(en.GetLights(), frustumFront, px, py, pz, sizeZLight, sizeZLight, sizeZLight)
	thingsChanged := w.bt.Process(en.GetThings(), frustumFront)

	w.bc.Commit(px, py, pz, staticChanged, thingsChanged)
}

// GetOpaqueInstancedItems returns the active rigid instanced opaque draw items for inspection.
func (w *Builder) GetOpaqueInstancedItems() []*draws.DrawItemInstanced {
	return w.bc.instOpaque.Items[:w.bc.instOpaque.Len]
}

// GetResidentVolumeCount returns the number of active resident static volume submesh draws.
func (w *Builder) GetResidentVolumeCount() int {
	return w.bc.staticOpaque.Len + w.bc.staticLiquid.Len + w.bc.staticAdditive.Len + w.bc.staticTransSrc.Len
}

// GetInstancedDrawCount returns the number of active GPU-instanced submesh draw groups.
func (w *Builder) GetInstancedDrawCount() int {
	return w.bc.instOpaque.Len + w.bc.instLiquid.Len + w.bc.instAdditive.Len + w.bc.instTransSrc.Len
}

// GetStreamingVertexCount returns the current vertex count stored in the CPU streaming buffer.
func (w *Builder) GetStreamingVertexCount() int {
	_, _, vCount, _, _, _ := w.bc.fv.GetVertices()
	return int(vCount) / buffers.StreamingVertexFloats
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with this Builder.
func (w *Builder) GetHWOcclusion() *queries.OcclusionHW {
	return w.bc.occlusion
}
