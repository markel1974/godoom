package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderVolume is a True-3D scene builder. It directly ingests Volumes and Faces (B-Rep)
// instead of 2.5D extruded walls and flats.
type BuilderVolume struct {
	tex         *Textures
	fv          *FrameVertices
	dc          *DrawCommands
	fl          *FrameLights
	dcRender    *DrawCommandsRender
	mapBuilt    bool
	cSky        *textures.Texture
	faceIndices []uint32
}

func NewBuilderVolume(tex *Textures) *BuilderVolume {
	return &BuilderVolume{
		tex:         tex,
		dcRender:    NewDrawCommandsRender(),
		fv:          NewFrameVertices(startBatchVertices),
		dc:          NewDrawCommands(startFrameCommands),
		fl:          NewFrameLights(256),
		faceIndices: make([]uint32, 0, 128),
		mapBuilt:    false,
		cSky:        nil,
	}
}

func (w *BuilderVolume) GetVerticesStride() int32                         { return w.fv.VerticesStride() }
func (w *BuilderVolume) GetLightsStride() int32                           { return w.fl.LightsStride() }
func (w *BuilderVolume) GetDrawCommands() *DrawCommandsRender             { return w.dcRender }
func (w *BuilderVolume) GetVertices() ([]float32, int32, []uint32, int32) { return w.fv.GetVertices() }
func (w *BuilderVolume) GetLights() ([]float32, int32)                    { return w.fl.GetLights() }
func (w *BuilderVolume) GetSkyTexture() *textures.Texture                 { return w.cSky }

func (w *BuilderVolume) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	w.fv.Reset()
	w.dc.Reset()
	w.fl.Reset()

	if !w.mapBuilt {
		w.fv.DeepReset()
		w.dc.DeepReset()
		w.fl.DeepReset()
		w.cSky = nil

		volumes, _ := engine.Build()

		for _, vol := range volumes {
			for _, face := range vol.Volume.GetFaces() {
				w.pushFace(w.fv, w.dc, face)
			}
		}

		w.pushLights(w.fl, engine.GetLights())

		w.fv.Freeze()
		w.dc.Freeze()
		w.fl.Freeze()
		w.mapBuilt = true
	}

	w.pushThings(w.fv, w.dc, vi, engine.GetThings())
	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushFace generates vertices and triangle fans for an arbitrary 3D face, applying Triplanar UV mapping.
func (w *BuilderVolume) pushFace(fv *FrameVertices, dc *DrawCommands, face *model.Face) {
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

	startIndices := fv.GetIndicesLen()
	w.faceIndices = w.faceIndices[:0]

	for _, p := range points {
		var u, v float32

		// Triplanar Mapping: Proietta le UV in base all'orientamento spaziale della faccia
		if absZ >= absX && absZ >= absY {
			// Dominante Z (Soffitti, Pavimenti o Slopes orizzontali)
			u = float32(p.X) / fTexW
			v = float32(-p.Y) / fTexH
		} else if absY >= absX && absY >= absZ {
			// Dominante Y (Muri verticali lungo l'asse X)
			u = float32(p.X) / fTexW
			v = float32(p.Z) / fTexH
		} else {
			// Dominante X (Muri verticali lungo l'asse Y)
			u = float32(p.Y) / fTexW
			v = float32(p.Z) / fTexH
		}

		// OpenGL Mapping: Invertiamo l'asse Y in Z per allinearci alla telecamera
		w.faceIndices = append(w.faceIndices, fv.AddVertex(float32(p.X), float32(p.Z), float32(-p.Y), u, v, layer))
	}

	// Triangolazione a ventaglio (Triangle Fan) per i poligoni convessi 3D
	for i := 1; i < len(w.faceIndices)-1; i++ {
		fv.AddTriangle(w.faceIndices[0], w.faceIndices[i], w.faceIndices[i+1])
	}

	currentIndices := fv.GetIndicesLen()
	if currentIndices > startIndices {
		dc.Compute(startIndices, currentIndices)
	}
}

// pushThings processes and pushes billboarded objects (things) into the rendering pipeline.
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

func (w *BuilderVolume) pushLights(fl *FrameLights, lights []*model.Light) {
	if len(lights) == 0 {
		return
	}
	for _, l := range lights {
		fl.Create(l)
	}
}
