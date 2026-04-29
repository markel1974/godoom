package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderScene represents the state and resources used to construct a 3D scene, including textures, vertices, commands, and lighting.
type BuilderScene struct {
	tex         *Textures
	fv          *FrameVertices
	dc          *DrawCommands
	fl          *FrameLights
	dcRender    *DrawCommandsRender
	mapBuilt    bool
	cSky        *textures.Texture
	flatIndices []uint32
}

// NewBuilderScene initializes and returns a new BuilderScene instance with given vertices, commands, lights, and textures.
func NewBuilderScene(tex *Textures) *BuilderScene {
	return &BuilderScene{
		tex:         tex,
		dcRender:    NewDrawCommandsRender(),
		fv:          NewFrameVertices(startBatchVertices),
		dc:          NewDrawCommands(startFrameCommands),
		fl:          NewFrameLights(256),
		flatIndices: make([]uint32, 0, 128),
		mapBuilt:    false,
		cSky:        nil,
	}
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame lighting configuration.
func (w *BuilderScene) GetShadowLights() ([8]*Light, int32) {
	return w.fl.GetShadowLights()
}

// GetVerticesStride returns the byte stride of vertex data in the buffer, calculated as the vertex attribute group size multiplied by 4.
func (w *BuilderScene) GetVerticesStride() int32 {
	return w.fv.VerticesStride()
}

// GetLightsStride retrieves the stride of light data, measured as the number of float32 values per light attribute set.
func (w *BuilderScene) GetLightsStride() int32 {
	return w.fl.LightsStride()
}

// GetDrawCommands returns a slice of active draw commands for rendering the current frame.
func (w *BuilderScene) GetDrawCommands() *DrawCommandsRender {
	return w.dcRender
}

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the frame vertices.
func (w *BuilderScene) GetVertices() ([]float32, int32, []uint32, int32) {
	return w.fv.GetVertices()
}

// GetLights retrieves the light data and count from the current frame, returning them as a float32 slice and an int32.
func (w *BuilderScene) GetLights() ([]float32, int32) {
	return w.fl.GetLights()
}

// GetSkyTexture returns the cached sky texture associated with the BuilderScene.
func (w *BuilderScene) GetSkyTexture() *textures.Texture {
	return w.cSky
}

// Compute generates the final rendered texture for the current scene based on the provided view matrix and engine state.
func (w *BuilderScene) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	// 1. Reset al Checkpoint Statico (azzeramento totale solo se mapBuilt è false)
	w.fv.Reset()
	w.dc.Reset()
	w.fl.Reset()

	// 2. Compilazione Mappa Statica (One-Off)
	if !w.mapBuilt {
		w.fv.DeepReset()
		w.dc.DeepReset()
		w.fl.DeepReset()
		w.cSky = nil
		css, compiled := engine.Build()
		for idx := compiled - 1; idx >= 0; idx-- {
			current := css[idx]

			polygons := current.Get()
			for k := len(polygons) - 1; k >= 0; k-- {
				cp := polygons[k]
				switch cp.Kind {
				case model.IdWall, model.IdUpper, model.IdLower:
					if cp.Kind == model.IdWall {
						w.pushWall(w.fv, w.dc, cp, cp.Material, float32(cp.Volume.GetMinZ()), float32(cp.Volume.GetMaxZ()))
					} else if cp.Kind == model.IdUpper {
						w.pushWall(w.fv, w.dc, cp, cp.Material, float32(cp.Neighbor.GetMaxZ()), float32(cp.Volume.GetMaxZ()))
					} else {
						w.pushWall(w.fv, w.dc, cp, cp.Material, float32(cp.Volume.GetMinZ()), float32(cp.Neighbor.GetMinZ()))
					}
				case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
					if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
						if sky := w.pushFlat(w.fv, w.dc, cp, cp.MaterialCeil, float32(cp.Volume.GetMaxZ())); sky != nil {
							w.cSky = sky
						}
					} else {
						if sky := w.pushFlat(w.fv, w.dc, cp, cp.MaterialFloor, float32(cp.Volume.GetMinZ())); sky != nil {
							w.cSky = sky
						}
					}
				}
			}
		}

		w.pushLights(w.fl, engine.GetLights().Get())

		// Salvataggio dello stato statico!
		w.fv.Freeze()
		w.dc.Freeze()
		w.fl.Freeze()
		w.mapBuilt = true
	}

	// 3. Geometria Dinamica (Ogni Frame)
	tA, tC := engine.GetThings().GetActive()
	w.pushThings(w.fv, w.dc, vi, tA, tC)

	// 4. Sort Globale e Batching
	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushWall processes polygonal wall data, calculates texture mapping, and adds the wall's vertices and triangles to the scene.
func (w *BuilderScene) pushWall(fv *FrameVertices, dc *DrawCommands, cp *model.CompiledPolygon, anim *textures.Material, zBottom, zTop float32) {
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	layer, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	_, scaleW, scaleH := tex.GetScaleFactor()

	u0 := float32(cp.U0) / (float32(texW) * float32(scaleW))
	u1 := float32(cp.U1) / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)

	wx1 := float32(cp.Tx1)
	wy1 := float32(cp.Tz1)
	wx2 := float32(cp.Tx2)
	wy2 := float32(cp.Tz2)

	startIndices := fv.GetIndicesLen()

	// Invertiamo l'asse Z, OpenGL guarda in -Z
	idx0 := fv.AddVertex6(wx1, zTop, -wy1, u0, vTop, layer)
	idx1 := fv.AddVertex6(wx1, zBottom, -wy1, u0, vBottom, layer)
	idx2 := fv.AddVertex6(wx2, zBottom, -wy2, u1, vBottom, layer)
	idx3 := fv.AddVertex6(wx2, zTop, -wy2, u1, vTop, layer)

	fv.AddTriangle(idx0, idx1, idx2)
	fv.AddTriangle(idx0, idx2, idx3)

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
}

// pushFlat processes and renders a flat surface using the given polygon key, animation, and Z-coordinate.
// It returns a texture if the animation is of type sky or a nil value in other cases.
func (w *BuilderScene) pushFlat(fv *FrameVertices, dc *DrawCommands, cp *model.CompiledPolygon, anim *textures.Material, zF float32) *textures.Texture {
	if anim.Kind() == int(config.MaterialKindSky) {
		return anim.CurrentFrame()
	}

	tex := anim.CurrentFrame()
	if tex == nil {
		return nil
	}
	faces := cp.Volume.GetFaces()
	if len(faces) < 3 {
		return nil
	}

	layer, ok := w.tex.Get(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	scaleH := tex.GetScaleFactorH()

	startIndices := fv.GetIndicesLen()

	w.flatIndices = w.flatIndices[:0]
	for _, seg := range faces {
		v := seg.GetStart()
		u := (float32(v.X) / float32(texW)) * float32(scaleH)
		vV := (float32(-v.Y) / float32(texH)) * float32(scaleH)
		w.flatIndices = append(w.flatIndices, fv.AddVertex6(float32(v.X), zF, float32(-v.Y), u, vV, layer))
	}

	for i := 1; i < len(faces)-1; i++ {
		fv.AddTriangle(w.flatIndices[0], w.flatIndices[i], w.flatIndices[i+1])
	}

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
	return nil
}

// pushThings processes and pushes visible objects (things) into the rendering pipeline for the current frame.
func (w *BuilderScene) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing, thingsCount int) {
	if len(things) == 0 {
		return
	}
	for idx := 0; idx < thingsCount; idx++ {
		thing := things[idx]
		faces, nextFaces, lp, billBoard := thing.GetVertices()
		if faces == nil {
			continue
		}
		lerp := float32(lp)
		yaw := float32(thing.GetAngle())
		tPosX, tPosY, zBot := thing.GetPosition()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
		b := float32(billBoard)
		startIndices := fv.GetIndicesLen()
		for fx, f := range faces {
			mat := f.GetMaterial()
			if mat == nil {
				continue
			}
			l, ok := w.tex.Get(mat)
			if !ok {
				continue
			}
			p := f.GetPoints()
			u, v := f.GetUV()
			np := nextFaces[fx].GetPoints()
			id0 := w.fv.AddVertex15(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), l, oX, oY, oZ, b, float32(np[0].X), float32(np[0].Z), float32(-np[0].Y), lerp, yaw)
			id1 := w.fv.AddVertex15(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), l, oX, oY, oZ, b, float32(np[1].X), float32(np[1].Z), float32(-np[1].Y), lerp, yaw)
			id2 := w.fv.AddVertex15(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), l, oX, oY, oZ, b, float32(np[2].X), float32(np[2].Z), float32(-np[2].Y), lerp, yaw)
			fv.AddTriangle(id0, id1, id2)
		}
		currentIndices := fv.GetIndicesLen()
		dc.Compute(startIndices, currentIndices)
	}
}

// pushLights adds the provided lights to the frameLights while optionally filtering by sectors. Skips if no lights are given.
func (w *BuilderScene) pushLights(fl *FrameLights, lights []*model.Light) {
	if len(lights) == 0 {
		return
	}
	for _, l := range lights {
		fl.Create(l)
	}
}
