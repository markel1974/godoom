package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// PolyKey represents a key used for identifying and differentiating polygonal geometry in a 3D simulation space.
// It comprises references to a sector, specific texture coordinates, and unique identifiers for polygonal boundaries.
type PolyKey struct {
	sector *model.Sector
	kind   int
	tx1    float32
	tz1    float32
	tx2    float32
	tz2    float32
	u0     float32
	u1     float32
}

// CreatePolygonSegment converts a CompiledPolygon into a PolyKey for identification and processing.
func CreatePolygonSegment(cp *model.CompiledPolygon) PolyKey {
	key := PolyKey{
		sector: cp.Sector,
		kind:   cp.Kind,
		tx1:    float32(cp.Tx1),
		tz1:    float32(cp.Tz1),
		tx2:    float32(cp.Tx2),
		tz2:    float32(cp.Tz2),
		u0:     float32(cp.U0),
		u1:     float32(cp.U1),
	}
	return key
}

// CreatePolygonSector generates a PolyKey from a given CompiledPolygon by extracting its Sector and Kind properties.
func CreatePolygonSector(cp *model.CompiledPolygon) PolyKey {
	key := PolyKey{
		sector: cp.Sector,
		kind:   cp.Kind,
	}
	return key
}

// BuilderTraverse is a utility structure for efficiently handling batched rendering operations.
// It organizes textures, vertex data, draw commands, lighting, and visibility checks for a frame.
type BuilderTraverse struct {
	dcRender          *DrawCommandsRender
	tex               *Textures
	fv                *FrameVertices
	dc                *DrawCommands
	fl                *FrameLights
	visibleSectors    map[*model.Sector]bool
	processedPolygons map[PolyKey]bool
	cSky              *textures.Texture
}

// NewBuilderTraverse creates and initializes a new BuilderTraverse with preallocated memory for vertices, commands, and lights.
func NewBuilderTraverse(tex *Textures) *BuilderTraverse {
	return &BuilderTraverse{
		tex:               tex,
		fv:                NewFrameVertices(maxBatchVertices),
		dc:                NewDrawCommands(maxFrameCommands),
		fl:                NewFrameLights(256),
		dcRender:          NewDrawCommandsRender(),
		visibleSectors:    make(map[*model.Sector]bool, 256),
		processedPolygons: make(map[PolyKey]bool, 2048),
		cSky:              nil,
	}
}

// GetVerticesStride returns the byte stride of vertex data in the buffer, computed as the attribute group size multiplied by 4.
func (w *BuilderTraverse) GetVerticesStride() int32 {
	return w.fv.VerticesStride()
}

// GetLightsStride returns the stride value for lights in the current frame, scaled by 4.
func (w *BuilderTraverse) GetLightsStride() int32 {
	return w.fl.LightsStride()
}

// GetSkyTexture retrieves the current sky texture used in the rendering pipeline.
func (w *BuilderTraverse) GetSkyTexture() *textures.Texture {
	return w.cSky
}

// GetDrawCommands returns a slice of active draw commands for rendering the current frame.
func (w *BuilderTraverse) GetDrawCommands() *DrawCommandsRender {
	return w.dcRender
}

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the frame vertices.
func (w *BuilderTraverse) GetVertices() ([]float32, int32, []uint32, int32) {
	return w.fv.GetVertices()
}

// GetLights retrieves the light data and count from the current frame, returning them as a float32 slice and an int32.
func (w *BuilderTraverse) GetLights() ([]float32, int32) {
	return w.fl.GetLights()
}

// Compute generates a batch for rendering by processing sectors, walls, floors, ceilings, things, and lights.
// It updates visible sectors and processed polygons, and returns a sky texture if one is found.
func (w *BuilderTraverse) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	w.fv.Reset()
	w.dc.Reset()
	w.fl.Reset()
	for k := range w.visibleSectors {
		delete(w.visibleSectors, k)
	}
	for k := range w.processedPolygons {
		delete(w.processedPolygons, k)
	}

	css, compiled := engine.Traverse(fbw, fbh, vi)
	things := engine.GetThings()
	lights := engine.GetLights()

	w.cSky = nil

	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]
		polygons := current.Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				key := CreatePolygonSegment(cp)
				if w.processedPolygons[key] {
					continue
				}
				w.processedPolygons[key] = true
				w.visibleSectors[cp.Sector] = true
				if cp.Kind == model.IdWall {
					w.pushWall(w.fv, w.dc, vi, key, cp.Animation, float32(cp.Sector.GetFloorY()), float32(cp.Sector.GetCeilY()))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(w.fv, w.dc, vi, key, cp.Animation, float32(cp.Neighbor.GetCeilY()), float32(cp.Sector.GetCeilY()))
				} else {
					w.pushWall(w.fv, w.dc, vi, key, cp.Animation, float32(cp.Sector.GetFloorY()), float32(cp.Neighbor.GetFloorY()))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				key := CreatePolygonSector(cp)
				if w.processedPolygons[key] {
					continue
				}
				w.processedPolygons[key] = true
				w.visibleSectors[cp.Sector] = true
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(w.fv, w.dc, key, cp.AnimationCeil, float32(cp.Sector.GetCeilY())); sky != nil {
						w.cSky = sky
					}
				} else {
					// IdFloor, IdFloorTest
					if sky := w.pushFlat(w.fv, w.dc, key, cp.AnimationFloor, float32(cp.Sector.GetFloorY())); sky != nil {
						w.cSky = sky
					}
				}
			}
		}
	}

	w.pushLights(w.fl, lights, w.visibleSectors)

	w.pushThings(w.fv, w.dc, vi, things, w.visibleSectors)

	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushWall adds a textured wall polygon to the batch using the specified view matrix, polygon key, animation, and height range.
func (w *BuilderTraverse) pushWall(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, cp PolyKey, anim *textures.Animation, zBottom, zTop float32) {
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	layer, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	scaleW, scaleH := anim.ScaleFactor()

	u0 := cp.u0 / (float32(texW) * float32(scaleW))
	u1 := cp.u1 / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)

	// Back-transformation: View-Space -> World-Space
	sin, cos := vi.GetAngle()
	viX, vizY := vi.GetXY()
	wx1 := (cp.tx1 * float32(sin)) + (cp.tz1 * float32(cos)) + float32(viX)
	wy1 := -(cp.tx1 * float32(cos)) + (cp.tz1 * float32(sin)) + float32(vizY)
	wx2 := (cp.tx2 * float32(sin)) + (cp.tz2 * float32(cos)) + float32(viX)
	wy2 := -(cp.tx2 * float32(cos)) + (cp.tz2 * float32(sin)) + float32(vizY)

	startIndices := w.fv.GetIndicesLen()

	idx0 := fv.AddVertex(wx1, zTop, -wy1, u0, vTop, layer)
	idx1 := fv.AddVertex(wx1, zBottom, -wy1, u0, vBottom, layer)
	idx2 := fv.AddVertex(wx2, zBottom, -wy2, u1, vBottom, layer)
	idx3 := fv.AddVertex(wx2, zTop, -wy2, u1, vTop, layer)

	fv.AddTriangle(idx0, idx1, idx2)
	fv.AddTriangle(idx0, idx2, idx3)

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
}

// pushFlat processes a flat surface for rendering, computes its vertices and indices, and adds draw commands.
func (w *BuilderTraverse) pushFlat(fv *FrameVertices, dc *DrawCommands, cp PolyKey, anim *textures.Animation, zF float32) *textures.Texture {
	if anim.Kind() == int(config.AnimationKindSky) {
		return anim.CurrentFrame()
	}

	tex := anim.CurrentFrame()
	if tex == nil {
		return nil
	}
	segments := cp.sector.Segments
	if len(segments) < 3 {
		return nil
	}

	layer, ok := w.tex.Get(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	_, scaleH := anim.ScaleFactor()

	startIndices := fv.GetIndicesLen()

	indices := make([]uint32, len(segments))
	for i, seg := range segments {
		v := seg.Start
		u := (float32(v.X) / float32(texW)) * float32(scaleH)
		vV := (float32(-v.Y) / float32(texH)) * float32(scaleH)
		// Coordinate assolute
		indices[i] = fv.AddVertex(float32(v.X), zF, float32(-v.Y), u, vV, layer)
	}

	for i := 1; i < len(segments)-1; i++ {
		fv.AddTriangle(indices[0], indices[i], indices[i+1])
	}

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
	return nil
}

// pushThings processes and adds things to the frame rendering pipeline based on their position, texture, and visibility.
func (w *BuilderTraverse) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing, sectors map[*model.Sector]bool) {
	const minDist = 0.0001
	if len(things) == 0 {
		return
	}
	camX, camY := vi.GetXY()

	for _, t := range things {
		if !sectors[t.GetSector()] {
			continue
		}
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

		// Vettori spigoli in spazio assoluto
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

// pushLights adds the specified lights to the frame based on their type, position, and characteristics, filtering by sector.
func (w *BuilderTraverse) pushLights(fl *FrameLights, lights []*model.Light, sectors map[*model.Sector]bool) {
	if len(lights) == 0 {
		return
	}

	for _, l := range lights {
		//if _, ok := sectors[l.GetSector()]; !ok {
		//	continue
		//}
		fl.Create(l)
	}
}
