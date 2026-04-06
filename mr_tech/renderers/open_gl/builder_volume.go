package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/config"
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
	return &BuilderVolume{
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

		volumes, _ := engine.Build()

		for _, vol := range volumes {
			startIdx := w.fv.GetIndicesLen()
			for _, face := range vol.Volume.GetFaces() {
				w.pushFace(w.fv, face)
			}
			endIdx := w.fv.GetIndicesLen()

			if endIdx > startIdx {
				w.volRanges[vol.Volume] = VolumeRange{start: startIdx, end: endIdx}
			}
		}

		// Congeliamo SOLO Geometria e DrawCommands
		w.fv.Freeze()
		w.dc.Freeze()
		w.mapBuilt = true
	}

	// 1. Estrazione Frustum Dinamico
	frustum := vi.GetFrustum(fbw, fbh, w.calibration.ZFarRoom)

	// 2. Frustum Culling sulla Geometria Statica (AABB Tree)
	engine.QueryFrustum(frustum, func(object physics.IAABB) bool {
		if vol, ok := object.(*model.Volume); ok {
			if vr, exists := w.volRanges[vol]; exists {
				w.dc.Compute(vr.start, vr.end)
			}
		}
		return false
	})

	// 3. Frustum Culling sulle Luci
	w.pushLights(w.fl, engine.GetLights(), frustum)

	// 4. Entità Dinamiche
	w.pushThings(w.fv, w.dc, vi, engine.GetThings().Get())

	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushFace generates vertices and triangle fans for an arbitrary 3D face, applying Triplanar UV mapping.
func (w *BuilderVolume) pushFace(fv *FrameVertices, face *model.Face) {
	anim := face.GetMaterial()
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
	if len(points) < 3 {
		return
	}
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

		w.faceIndices = append(w.faceIndices, fv.AddVertex(float32(p.X), float32(p.Z), float32(-p.Y), u, v, layer))
	}

	for i := 1; i < len(w.faceIndices)-1; i++ {
		fv.AddTriangle(w.faceIndices[0], w.faceIndices[i], w.faceIndices[i+1])
	}
}

// pushThings processes and adds the given list of things to the frame vertices and draw commands for rendering.
func (w *BuilderVolume) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing) {
	const minDist = 0.0001
	if len(things) == 0 {
		return
	}
	camX, camY := vi.GetXY()

	for _, t := range things {
		if t.GetAnimation() == nil {
			continue
		}
		tPosX, tPosY := t.GetPosition()
		dx := tPosX - camX
		dy := tPosY - camY
		distSq := dx*dx + dy*dy

		tex := t.GetAnimation().CurrentFrame()
		if tex == nil {
			continue
		}
		layer, ok := w.tex.Get(tex)
		if !ok {
			continue
		}

		texW, texH := tex.Size()
		scaleW, scaleH := t.GetAnimation().ScaleFactor()
		width := float64(texW) * scaleW
		height := float64(texH) * scaleH

		dist := math.Sqrt(distSq)
		if dist < minDist {
			dist = minDist
		}

		halfW := width / 2.0
		rX := -((camY - tPosY) / dist) * halfW
		rY := ((camX - tPosX) / dist) * halfW

		v1x := float32(tPosX - rX)
		v1y := float32(tPosY - rY)
		v2x := float32(tPosX + rX)
		v2y := float32(tPosY + rY)

		zBottom := float32(t.GetFloorY())
		zTop := zBottom + float32(height)

		startIndices := fv.GetIndicesLen()
		u0, u1 := float32(0.0), float32(1.0)
		vTop, vBottom := float32(0.0), float32(1.0)

		idx0 := fv.AddVertex(v1x, zTop, -v1y, u0, vTop, layer)
		idx1 := fv.AddVertex(v1x, zBottom, -v1y, u0, vBottom, layer)
		idx2 := fv.AddVertex(v2x, zBottom, -v2y, u1, vBottom, layer)
		idx3 := fv.AddVertex(v2x, zTop, -v2y, u1, vTop, layer)

		fv.AddTriangle(idx0, idx1, idx2)
		fv.AddTriangle(idx0, idx2, idx3)

		currentIndices := fv.GetIndicesLen()
		dc.Compute(startIndices, currentIndices)
	}
}

// pushLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushLights(fl *FrameLights, lights *model.Lights, frustum *physics.Frustum) {
	lights.QueryFrustum(frustum, func(object physics.IAABB) bool {
		if l, ok := object.(*model.Light); ok {
			fl.Create(l)
		}
		return false
	})
}
