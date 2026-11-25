package open_gl

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

// BuilderVolume coordinates static world volumes, things, lights, occlusion queries, and shadow tiles for OpenGL rendering.
type BuilderVolume struct {
	frustumFront *metrics.Frustum

	bc      *BuilderContext
	bl      *BuilderLights
	bs      *BuilderStatic
	bt      *BuilderThings
	bShadow *BuilderShadow
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance configured for the active rendering context.
func NewBuilderVolume(ctx api.IContext, tex *vramtex.Textures, cal *model.Calibration, player *things.Player) *BuilderVolume {
	return newBuilderVolumeWithOcclusionMax(ctx, tex, cal, player, defaultOcclusionMaxEntities)
}

func newBuilderVolumeWithOcclusionMax(ctx api.IContext, tex *vramtex.Textures, cal *model.Calibration, player *things.Player, occMax int) *BuilderVolume {
	bc := newBuilderContextWithOcclusionMax(ctx, cal, tex, player, occMax)
	bs := NewBuilderStatic(bc)
	bt := NewBuilderThings(bc)
	bl := NewBuilderLights(bc)
	bShadow := NewBuilderShadow(bc, bs, bt)

	ff := metrics.NewFrustum()
	if cal != nil {
		ff.Setup(float32(cal.AspectRatio), float32(cal.ZFar), float32(cal.ZNear), cal.PrjScale, cal.PrjScale)
	}

	return &BuilderVolume{
		frustumFront: ff,
		bc:           bc,
		bl:           bl,
		bs:           bs,
		bt:           bt,
		bShadow:      bShadow,
	}
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame lighting data.
func (w *BuilderVolume) GetShadowLights() ([8]*metrics.Light, int32) {
	return w.bl.GetFrameLights().GetShadowLights()
}

// GetVerticesStride returns the byte stride of streaming vertices.
func (w *BuilderVolume) GetVerticesStride() int32 {
	return w.bc.fv.VerticesStride()
}

// GetLightsStride returns the stride size of the lights buffer in bytes.
func (w *BuilderVolume) GetLightsStride() int32 {
	return w.bl.GetFrameLights().LightsStride()
}

// GetDrawCommandsOpaque returns the compiled opaque draw command list.
func (w *BuilderVolume) GetDrawCommandsOpaque() *draws.DrawCommandsRender {
	return w.bc.dcRenderOpaque
}

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsAdditive() *draws.DrawCommandsRender {
	return w.bc.dcRenderAdditive
}

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsLiquid() *draws.DrawCommandsRender {
	return w.bc.dcRenderLiquid
}

// GetDrawCommandsTransparent returns the sorted transparent draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsTransparent() *draws.DrawCommandsRender {
	return w.bc.dcRenderTransparent
}

// GetVertices retrieves pointers, byte sizes, and counts for streaming vertices and indices.
func (w *BuilderVolume) GetVertices() (unsafe.Pointer, int, int32, unsafe.Pointer, int, int32) {
	return w.bc.fv.GetVertices()
}

// GetLights retrieves the light data pointer, byte stride size, and active light count.
func (w *BuilderVolume) GetLights() (unsafe.Pointer, int32, int32) {
	return w.bl.GetFrameLights().GetLights()
}

// GetSkyMaterial returns the sky material detected among active static volumes.
func (w *BuilderVolume) GetSkyMaterial() *textures.Material {
	return w.bs.GetSkyMaterial()
}

// GetUploadHints returns static prefix byte offsets and generation counters used for incremental GPU buffer uploads.
func (w *BuilderVolume) GetUploadHints() (staticVertBytes, staticIdxBytes int, staticGen, frameGen uint64) {
	sb, si, sg := w.bs.GetUploadHints()
	return sb, si, sg, w.bc.UploadFrameGen()
}

// ComputeShadows updates per-light shadow tile draw lists and instance transforms for the active depth state.
func (w *BuilderVolume) ComputeShadows(en *engine.Engine, d *shaders.Depth) {
	if en == nil || d == nil {
		w.bShadow.DisableAllTiles()
		return
	}
	w.bShadow.ComputeShadows(en.GetVolumes(), en.GetThings(), d)
}

// GetShadowTiles returns the per-light shadow tile descriptors prepared by ComputeShadows.
func (w *BuilderVolume) GetShadowTiles() *[shaders.MaxShadowTiles]shaders.ShadowTileData {
	return w.bShadow.GetShadowTiles()
}

// Compute processes volumes, lights, and things within the camera frustum and compiles rendering draw commands.
func (w *BuilderVolume) Compute(fbw, fbh int32, vi *engine.ViewMatrix, en *engine.Engine) {
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
func (w *BuilderVolume) GetOpaqueInstancedItems() []*draws.DrawItemInstanced {
	return w.bc.instOpaque.Items[:w.bc.instOpaque.Len]
}

// GetResidentVolumeCount returns the number of active resident static volume submesh draws.
func (w *BuilderVolume) GetResidentVolumeCount() int {
	return w.bc.staticOpaque.Len + w.bc.staticLiquid.Len + w.bc.staticAdditive.Len + w.bc.staticTransSrc.Len
}

// GetInstancedDrawCount returns the number of active GPU-instanced submesh draw groups.
func (w *BuilderVolume) GetInstancedDrawCount() int {
	return w.bc.instOpaque.Len + w.bc.instLiquid.Len + w.bc.instAdditive.Len + w.bc.instTransSrc.Len
}

// GetStreamingVertexCount returns the current vertex count stored in the CPU streaming buffer.
func (w *BuilderVolume) GetStreamingVertexCount() int {
	_, _, vCount, _, _, _ := w.bc.fv.GetVertices()
	return int(vCount) / buffers.StreamingVertexFloats
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with this BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *queries.OcclusionHW {
	return w.bc.occlusion
}
