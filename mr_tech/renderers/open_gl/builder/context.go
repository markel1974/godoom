package builder

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/queries"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const defaultOcclusionMaxEntities = 4096

// Context owns shared GPU staging buffers, draw pools, occlusion queries, and frame generation state
// coordinated across static, thing, light, and shadow sub-builders.
type Context struct {
	ctx           api.IContext
	cal           *model.Calibration
	tex           *vramtex.Textures
	player        *things.Player
	occlusion     *queries.OcclusionHW
	fv            *draws.FrameVertices
	dcOpaque      *draws.DrawCommands
	dcAdditive    *draws.DrawCommands
	dcLiquid      *draws.DrawCommands
	dcTransparent *draws.DrawCommands
	modes         []*draws.DrawCommands

	staticOpaque   buffers.PtrPool[draws.DrawItemStatic]
	staticLiquid   buffers.PtrPool[draws.DrawItemStatic]
	staticAdditive buffers.PtrPool[draws.DrawItemStatic]
	staticTransSrc buffers.PtrPool[draws.DrawItemTransparent]

	instOpaque   buffers.PtrPool[draws.DrawItemInstanced]
	instLiquid   buffers.PtrPool[draws.DrawItemInstanced]
	instAdditive buffers.PtrPool[draws.DrawItemInstanced]
	instTransSrc buffers.PtrPool[RigidTransItem]

	dcRenderOpaque      *draws.DrawCommandsRender
	dcRenderAdditive    *draws.DrawCommandsRender
	dcRenderLiquid      *draws.DrawCommandsRender
	dcRenderTransparent *draws.DrawCommandsRender

	transparentItems *draws.DrawItemsTransparent

	lastInstVBO uint32

	meshFrame       uint64
	frameGen        uint64
	shadowExtraGen  uint64
	lastPreparedGen uint64

	lastCamGLX float32
	lastCamGLY float32
	lastCamGLZ float32
}

// NewContext allocates and initializes a Context with default hardware occlusion capacity.
func NewContext(ctx api.IContext, cal *model.Calibration, tex *vramtex.Textures, player *things.Player) *Context {
	return NewContextWithOcclusionMax(ctx, cal, tex, player, defaultOcclusionMaxEntities)
}

func NewContextWithOcclusionMax(ctx api.IContext, cal *model.Calibration, tex *vramtex.Textures, player *things.Player, occMax int) *Context {
	var occ *queries.OcclusionHW
	if ctx != nil && occMax > 0 {
		occ = queries.NewOcclusionHW(ctx, occMax)
	}
	bc := &Context{
		ctx:                 ctx,
		cal:                 cal,
		tex:                 tex,
		player:              player,
		occlusion:           occ,
		fv:                  draws.NewFrameVertices(ctx, 1048576),
		dcOpaque:            draws.NewDrawCommands(config.BlendModeOpaque, 32768),
		dcAdditive:          draws.NewDrawCommands(config.BlendModeAdditive, 4096),
		dcLiquid:            draws.NewDrawCommands(config.BlendModeLiquid, 4096),
		dcTransparent:       draws.NewDrawCommands(config.BlendModeTransparent, 4096),
		modes:               make([]*draws.DrawCommands, config.BlendModeLatest),
		dcRenderOpaque:      draws.NewDrawCommandsRender(ctx, "opaque", false),
		dcRenderAdditive:    draws.NewDrawCommandsRender(ctx, "additive", true),
		dcRenderLiquid:      draws.NewDrawCommandsRender(ctx, "liquid", true),
		dcRenderTransparent: draws.NewDrawCommandsRender(ctx, "transparent", false),
		transparentItems:    draws.NewDrawItemsTransparent(64),
	}

	bc.modes[bc.dcOpaque.GetBlendMode()] = bc.dcOpaque
	bc.modes[bc.dcAdditive.GetBlendMode()] = bc.dcAdditive
	bc.modes[bc.dcLiquid.GetBlendMode()] = bc.dcLiquid
	bc.modes[bc.dcTransparent.GetBlendMode()] = bc.dcTransparent

	return bc
}

// LookupTexLayer returns the texture array layer identifier for tex, caching it on the texture on first resolve.
func (w *Context) LookupTexLayer(tex *textures.Texture) (float32, bool) {
	if tex == nil {
		return 0, false
	}
	tId := tex.GetIdentifier()
	if tId < 0 {
		if w.tex == nil {
			return 0, false
		}
		var ok bool
		tId, ok = w.tex.Get(tex)
		if !ok {
			return 0, false
		}
		tex.SetIdentifier(tId)
	}
	return tId, true
}

// ResetDynamicDraws rewinds dynamic per-frame vertex/index commands back to the frozen static prefix.
func (w *Context) ResetDynamicDraws() {
	w.fv.Reset()
	w.dcOpaque.Reset()
	w.dcAdditive.Reset()
	w.dcLiquid.Reset()
	if w.dcTransparent != nil {
		w.dcTransparent.Reset()
	}
}

// DeepResetDynamicDraws clears all buffered vertices, indices, and draw command batches including any frozen static prefix.
func (w *Context) DeepResetDynamicDraws() {
	w.fv.DeepReset()
	w.dcOpaque.DeepReset()
	w.dcAdditive.DeepReset()
	w.dcLiquid.DeepReset()
	if w.dcTransparent != nil {
		w.dcTransparent.DeepReset()
	}
}

// FreezeDynamicDraws marks the end of the static prefix so subsequent dynamic thing batches preserve it across frames.
func (w *Context) FreezeDynamicDraws() {
	w.fv.Freeze()
	w.dcOpaque.Freeze()
	w.dcAdditive.Freeze()
	w.dcLiquid.Freeze()
	if w.dcTransparent != nil {
		w.dcTransparent.Freeze()
	}
}

// ResetStaticPools clears pooled resident static draw items before rebuilding static visibility.
func (w *Context) ResetStaticPools() {
	w.staticOpaque.Reset()
	w.staticLiquid.Reset()
	w.staticAdditive.Reset()
	w.staticTransSrc.Reset()
}

// UploadFrameGen returns the combined GPU upload generation key across main frame state and off-camera shadow streams.
func (w *Context) UploadFrameGen() uint64 {
	return w.frameGen + w.shadowExtraGen
}

// Commit compiles updated opaque/additive/liquid draw lists and updates back-to-front translucent sorting when needed.
func (w *Context) Commit(pX, pY, pZ float64, staticChanged bool, thingsChanged bool) {
	if staticChanged || thingsChanged || w.frameGen == 0 {
		w.frameGen++
	}
	if w.lastPreparedGen != w.frameGen {
		w.dcRenderOpaque.Prepare(w.dcOpaque.GetDrawCommands())
		w.dcRenderOpaque.SetResidentDraws(w.staticOpaque.Items, w.staticOpaque.Len, w.instOpaque.Items, w.instOpaque.Len)

		w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
		w.dcRenderAdditive.SetResidentDraws(w.staticAdditive.Items, w.staticAdditive.Len, w.instAdditive.Items, w.instAdditive.Len)

		w.dcRenderLiquid.Prepare(w.dcLiquid.GetDrawCommands())
		w.dcRenderLiquid.SetResidentDraws(w.staticLiquid.Items, w.staticLiquid.Len, w.instLiquid.Items, w.instLiquid.Len)
		w.lastPreparedGen = w.frameGen
	}

	camGLX, camGLY, camGLZ := float32(pX), float32(pZ), float32(-pY)
	var transCmds []*draws.DrawCommand
	if w.dcTransparent != nil {
		transCmds = w.dcTransparent.GetDrawCommands()
	}
	hasTrans := w.staticTransSrc.Len > 0 || w.instTransSrc.Len > 0 || len(transCmds) > 0
	camMoved := camGLX != w.lastCamGLX || camGLY != w.lastCamGLY || camGLZ != w.lastCamGLZ
	if hasTrans {
		if staticChanged || thingsChanged || camMoved || w.transparentItems.Len() == 0 {
			w.transparentItems.Begin()
			w.transparentItems.AddStatic(&w.staticTransSrc, camGLX, camGLY, camGLZ)
			for i := 0; i < w.instTransSrc.Len; i++ {
				rt := w.instTransSrc.Items[i]
				w.transparentItems.AddInstanced(rt.mesh, rt.sub, w.lastInstVBO, rt.baseByteOffset, rt.cx, rt.cy, rt.cz, camGLX, camGLY, camGLZ)
			}
			w.transparentItems.AddCommands(transCmds, camGLX, camGLY, camGLZ)
			w.transparentItems.End()
			w.dcRenderTransparent.SetTransparentDraws(w.transparentItems)
		}
	} else if w.transparentItems.Len() > 0 {
		w.transparentItems.Clear()
		w.dcRenderTransparent.SetTransparentDraws(nil)
	}
	w.lastCamGLX, w.lastCamGLY, w.lastCamGLZ = camGLX, camGLY, camGLZ
}
