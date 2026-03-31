package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderScene represents the state and resources used to construct a 3D scene, including textures, vertices, commands, and lighting.
type BuilderScene struct {
	tex          *Textures
	vertices     *FrameVertices
	drawCommands *DrawCommands
	frameLights  *FrameLights
	mapBuilt     bool
	cSkyCached   *textures.Texture
}

// NewBuilderScene initializes and returns a new BuilderScene instance with given vertices, commands, lights, and textures.
func NewBuilderScene(vertices *FrameVertices, commands *DrawCommands, lights *FrameLights, tex *Textures) *BuilderScene {
	return &BuilderScene{
		tex:          tex,
		vertices:     vertices,
		drawCommands: commands,
		frameLights:  lights,
		mapBuilt:     false,
		cSkyCached:   nil,
	}
}

// reset clears all frame-related data structures, restoring the BuilderScene to its initial state for reuse.
func (w *BuilderScene) reset() {
	w.vertices.Reset()
	w.drawCommands.Reset()
	w.frameLights.Reset()
}

func (w *BuilderScene) Compute(vi *model.ViewMatrix, engine *engine.Engine) *textures.Texture {
	w.reset()

	css, compiled := engine.Build()
	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]

		polygons := current.Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				if cp.Kind == model.IdWall {
					w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(cp, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
				} else {
					w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(cp, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
						w.cSkyCached = sky
					}
				} else {
					if sky := w.pushFlat(cp, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
						w.cSkyCached = sky
					}
				}
			}
		}
	}

	lights := engine.GetLights()
	w.pushLights(lights)

	things := engine.GetThings()
	w.pushThings(vi, things)

	return w.cSkyCached
}

// ComputeNew processes visible sectors, walls, floors, ceilings, lights, and things to prepare the render state. Returns the sky texture.
func (w *BuilderScene) ComputeNew(vi *model.ViewMatrix, engine *engine.Engine) *textures.Texture {
	//const dirty = false //TODO From engine
	if w.mapBuilt {
		w.vertices.RestoreCheckpoint()
		w.drawCommands.RestoreCheckpoint()
		w.frameLights.RestoreCheckpoint()
		w.pushThings(vi, engine.GetThings())
		return w.cSkyCached
	}

	w.reset()

	css, compiled := engine.Build()
	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]
		polygons := current.Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				if cp.Kind == model.IdWall {
					w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(cp, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
				} else {
					w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(cp, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
						w.cSkyCached = sky
					}
				} else {
					if sky := w.pushFlat(cp, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
						w.cSkyCached = sky
					}
				}
			}
		}
	}
	w.pushLights(engine.GetLights())
	// Salviamo lo stato esatto dei buffer (Geometria + Luci statiche finite)
	w.vertices.SaveCheckpoint()
	w.drawCommands.SaveCheckpoint()
	w.frameLights.SaveCheckpoint()
	w.mapBuilt = true

	w.pushThings(vi, engine.GetThings())

	return w.cSkyCached
}

// pushWall processes polygonal wall data, calculates texture mapping, and adds the wall's vertices and triangles to the scene.
func (w *BuilderScene) pushWall(cp *model.CompiledPolygon, anim *textures.Animation, zBottom, zTop float32) {
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	texId, normTexId, emissiveTexId, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	scaleW, scaleH := anim.ScaleFactor()

	u0 := float32(cp.U0) / (float32(texW) * float32(scaleW))
	u1 := float32(cp.U1) / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)

	wx1 := float32(cp.Tx1)
	wy1 := float32(cp.Tz1)
	wx2 := float32(cp.Tx2)
	wy2 := float32(cp.Tz2)

	startIndices := w.vertices.GetIndicesLen()

	// Invertiamo l'asse Z, OpenGL guarda in -Z
	idx0 := w.vertices.AddVertex(wx1, zTop, -wy1, u0, vTop)
	idx1 := w.vertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom)
	idx2 := w.vertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom)
	idx3 := w.vertices.AddVertex(wx2, zTop, -wy2, u1, vTop)

	w.vertices.AddTriangle(idx0, idx1, idx2)
	w.vertices.AddTriangle(idx0, idx2, idx3)

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
}

// pushFlat processes and renders a flat surface using the given polygon key, animation, and Z-coordinate.
// It returns a texture if the animation is of type sky or a nil value in other cases.
func (w *BuilderScene) pushFlat(cp *model.CompiledPolygon, anim *textures.Animation, zF float32) *textures.Texture {
	if anim.Kind() == int(model.AnimationKindSky) {
		return anim.CurrentFrame()
	}

	tex := anim.CurrentFrame()
	if tex == nil {
		return nil
	}
	segments := cp.Sector.Segments
	if len(segments) < 3 {
		return nil
	}

	texId, normTexId, emissiveTexId, ok := w.tex.Get(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	_, scaleH := anim.ScaleFactor()

	startIndices := w.vertices.GetIndicesLen()

	indices := make([]uint32, len(segments))
	for i, seg := range segments {
		v := seg.Start
		u := (float32(v.X) / float32(texW)) * float32(scaleH)
		vV := (float32(-v.Y) / float32(texH)) * float32(scaleH)
		// Coordinate assolute (nessuna rotazione)
		indices[i] = w.vertices.AddVertex(float32(v.X), zF, float32(-v.Y), u, vV)
	}

	for i := 1; i < len(segments)-1; i++ {
		w.vertices.AddTriangle(indices[0], indices[i], indices[i+1])
	}

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
	return nil
}

// pushThings processes and pushes visible objects (things) into the rendering pipeline for the current frame.
func (w *BuilderScene) pushThings(vi *model.ViewMatrix, things []model.IThing) {
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
		texId, normTexId, emissiveTexId, ok := w.tex.Get(tex)
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

		// Calcolo del piano billboad perpendicolare alla camera
		halfW := width / 2.0
		rX := -((camY - tPosY) / dist) * halfW
		rY := ((camX - tPosX) / dist) * halfW

		v1x := float32(tPosX - rX)
		v1y := float32(tPosY - rY)
		v2x := float32(tPosX + rX)
		v2y := float32(tPosY + rY)

		zBottom := float32(t.GetFloorY())
		zTop := zBottom + float32(height)

		startIndices := w.vertices.GetIndicesLen()
		u0, u1 := float32(0.0), float32(1.0)
		vTop, vBottom := float32(0.0), float32(1.0)

		idx0 := w.vertices.AddVertex(v1x, zTop, -v1y, u0, vTop)
		idx1 := w.vertices.AddVertex(v1x, zBottom, -v1y, u0, vBottom)
		idx2 := w.vertices.AddVertex(v2x, zBottom, -v2y, u1, vBottom)
		idx3 := w.vertices.AddVertex(v2x, zTop, -v2y, u1, vTop)

		w.vertices.AddTriangle(idx0, idx1, idx2)
		w.vertices.AddTriangle(idx0, idx2, idx3)

		currentIndices := w.vertices.GetIndicesLen()
		w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
	}
}

// pushLights adds the provided lights to the frameLights while optionally filtering by sectors. Skips if no lights are given.
func (w *BuilderScene) pushLights(lights []*model.Light) {
	if len(lights) == 0 {
		return
	}
	for _, l := range lights {
		w.frameLights.Create(l)
	}
}
