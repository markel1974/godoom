package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type VolumeRange struct {
	start int32
	end   int32
}

type BuilderVolume struct {
	tex         *Textures
	fv          *FrameVertices
	dc          *DrawCommands
	fl          *FrameLights
	dcRender    *DrawCommandsRender
	mapBuilt    bool
	cSky        *textures.Texture
	faceIndices []uint32
	volRanges   map[*model.Volume]VolumeRange // CACHE DI CULLING
	calibration *model.Calibration
}

func NewBuilderVolume(tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		tex:         tex,
		dcRender:    NewDrawCommandsRender(),
		fv:          NewFrameVertices(startBatchVertices),
		dc:          NewDrawCommands(startFrameCommands),
		fl:          NewFrameLights(256),
		faceIndices: make([]uint32, 0, 128),
		volRanges:   make(map[*model.Volume]VolumeRange), // Inizializzazione
		mapBuilt:    false,
		cSky:        nil,
		calibration: calibration,
	}

	return bv
}

// GetVerticesStride returns the byte stride of the vertex data as an int32 by delegating to the underlying FrameVertices.
func (w *BuilderVolume) GetVerticesStride() int32 { return w.fv.VerticesStride() }

// GetLightsStride returns the stride value for light data, representing the size in bytes of a single light entry.
func (w *BuilderVolume) GetLightsStride() int32 { return w.fl.LightsStride() }

// GetDrawCommands returns the prepared draw commands for rendering stored in the DrawCommandsRender instance.
func (w *BuilderVolume) GetDrawCommands() *DrawCommandsRender { return w.dcRender }

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the BuilderVolume instance.
func (w *BuilderVolume) GetVertices() ([]float32, int32, []uint32, int32) { return w.fv.GetVertices() }

// GetLights retrieves the current set of light data and the number of lights in the frame as a slice and an integer.
func (w *BuilderVolume) GetLights() ([]float32, int32) { return w.fl.GetLights() }

// GetSkyTexture retrieves the current sky texture associated with the BuilderVolume. Returns nil if no texture is set.
func (w *BuilderVolume) GetSkyTexture() *textures.Texture { return w.cSky }

func (w *BuilderVolume) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	// Ripristina VBO e Comandi allo stato congelato
	w.fv.Reset()
	w.dc.Reset()

	// Reset TOTALE del buffer luci ogni frame (le calcoliamo dinamicamente)
	w.fl.DeepReset()

	if !w.mapBuilt {
		w.fv.DeepReset()
		w.dc.DeepReset()
		w.cSky = nil
		volumes := engine.GetVolumes()
		for _, vol := range volumes.GetVolumes() {
			startIdx := w.fv.GetIndicesLen()
			for _, face := range vol.GetFaces() {
				w.pushFace(w.fv, face)
			}
			endIdx := w.fv.GetIndicesLen()
			if endIdx > startIdx {
				w.volRanges[vol] = VolumeRange{start: startIdx, end: endIdx}
			}
		}
		// Congeliamo SOLO Geometria e DrawCommands
		w.fv.Freeze()
		w.dc.Freeze()
		w.mapBuilt = true
	}

	const usFrustum = false

	if usFrustum {
		queryGeom := func(object physics.IAABB) bool {
			if vol, ok := object.(*model.Volume); ok {
				if vr, exists := w.volRanges[vol]; exists {
					w.dc.Compute(vr.start, vr.end)
				}
			}
			return false
		}
		frustumFront, frustumRear := vi.GetFrustum(fbw, fbh, w.calibration.ZFarRoom)
		engine.QueryMultiFrustum(frustumFront, frustumRear, queryGeom)
		w.pushLights(w.fl, engine.GetLights(), frustumFront, frustumRear)
	} else {
		for _, vr := range w.volRanges {
			w.dc.Compute(vr.start, vr.end)
		}
		for _, vl := range engine.GetLights().Get() {
			w.fl.Create(vl)
		}
	}
	// 4. Entità Dinamiche
	tA, tC := engine.GetThings().GetActive()
	w.pushThings(w.fv, w.dc, vi, tA, tC)
	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushFace generates vertices and triangle fans for an arbitrary 3D face, applying Triplanar UV mapping.
func (w *BuilderVolume) pushFace(fv *FrameVertices, face *model.Face) {
	anim := face.GetRootMaterial()
	if anim == nil {
		return
	}
	if anim.Kind() == int(config.AnimationKindSky) {
		w.cSky = anim.CurrentFrame()
		return
	}
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	layer, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	points := face.GetPoints()
	texW, texH := tex.Size()
	scaleW, scaleH := anim.ScaleFactor()
	fTexW := float32(texW) * float32(scaleW)
	fTexH := float32(texH) * float32(scaleH)
	normal := face.GetNormal()
	absX := math.Abs(normal.X)
	absY := math.Abs(normal.Y)
	absZ := math.Abs(normal.Z)
	w.faceIndices = w.faceIndices[:0]
	for _, p := range points {
		var u, v float32
		if absZ >= absX && absZ >= absY {
			u = float32(p.X) / fTexW
			v = float32(-p.Y) / fTexH
		} else if absY >= absX && absY >= absZ {
			u = float32(p.X) / fTexW
			v = float32(p.Z) / fTexH
		} else {
			u = float32(p.Y) / fTexW
			v = float32(p.Z) / fTexH
		}
		w.faceIndices = append(w.faceIndices, fv.AddVertex(float32(p.X), float32(p.Z), float32(-p.Y), u, v, layer, 0, 0, 0, 0))
	}
	for i := 1; i < len(w.faceIndices)-1; i++ {
		fv.AddTriangle(w.faceIndices[0], w.faceIndices[i], w.faceIndices[i+1])
	}
}

// pushThings processes and adds the given list of things to the frame vertices and draw commands for rendering.
func (w *BuilderVolume) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing, thingsCount int) {
	if len(things) == 0 {
		return
	}
	for idx := 0; idx < thingsCount; idx++ {
		thing := things[idx]
		vertices := thing.GetVertices()
		if vertices == nil {
			continue
		}
		startIndices := fv.GetIndicesLen()
		for _, tri := range vertices {
			layer, ok := w.tex.Get(tri[0].Material)
			if !ok {
				continue
			}
			p0, p1, p2 := tri[0], tri[1], tri[2]
			// AddVertex ora accetta 10 parametri (Pos[3], Tex[3], Origin[3], IsBB[1])
			id0 := fv.AddVertex(
				float32(p0.X), float32(p0.Y), float32(p0.Z),
				float32(p0.U), float32(p0.V), layer,
				float32(p0.Origin.X), float32(p0.Origin.Y), float32(p0.Origin.Z),
				float32(p0.IsBillboard),
			)
			id1 := fv.AddVertex(
				float32(p1.X), float32(p1.Y), float32(p1.Z),
				float32(p1.U), float32(p1.V), layer,
				float32(p1.Origin.X), float32(p1.Origin.Y), float32(p1.Origin.Z),
				float32(p1.IsBillboard),
			)
			id2 := fv.AddVertex(
				float32(p2.X), float32(p2.Y), float32(p2.Z),
				float32(p2.U), float32(p2.V), layer,
				float32(p2.Origin.X), float32(p2.Origin.Y), float32(p2.Origin.Z),
				float32(p2.IsBillboard),
			)
			fv.AddTriangle(id0, id1, id2)
		}
		currentIndices := fv.GetIndicesLen()
		dc.Compute(startIndices, currentIndices)
	}
}

// pushLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushLights(fl *FrameLights, lights *model.Lights, frustumFront, frustumRear *physics.Frustum) {
	queryLights := func(object physics.IAABB) bool {
		if l, ok := object.(*model.Light); ok {
			fl.Create(l)
		}
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}
