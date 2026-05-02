package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// PolyKey represents a key used for identifying and differentiating polygonal geometry in a 3D simulation space.
// It comprises references to a sector, specific texture coordinates, and unique identifiers for polygonal boundaries.
type PolyKey struct {
	volume *model.Volume
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
		volume: cp.Volume,
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
		volume: cp.Volume,
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
	visibleVolumes    map[*model.Volume]bool
	processedPolygons map[PolyKey]bool
	cSky              *textures.Texture
	cal               *model.Calibration
}

// NewBuilderTraverse creates and initializes a new BuilderTraverse with preallocated memory for vertices, commands, and lights.
func NewBuilderTraverse(tex *Textures, calibration *model.Calibration) *BuilderTraverse {
	return &BuilderTraverse{
		tex:               tex,
		cal:               calibration,
		fv:                NewFrameVertices(startBatchVertices),
		dc:                NewDrawCommands(startFrameCommands),
		fl:                NewFrameLights(256),
		dcRender:          NewDrawCommandsRender(),
		visibleVolumes:    make(map[*model.Volume]bool, 256),
		processedPolygons: make(map[PolyKey]bool, 2048),
		cSky:              nil,
	}
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame's light data.
func (w *BuilderTraverse) GetShadowLights() ([8]*Light, int32) {
	return w.fl.GetShadowLights()
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
	for k := range w.visibleVolumes {
		delete(w.visibleVolumes, k)
	}
	for k := range w.processedPolygons {
		delete(w.processedPolygons, k)
	}

	css, compiled := engine.Traverse(fbw, fbh, vi)
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
				w.visibleVolumes[cp.Volume] = true
				if cp.Kind == model.IdWall {
					w.pushWall(w.fv, w.dc, vi, key, cp.Material, float32(cp.Volume.GetMinZ()), float32(cp.Volume.GetMaxZ()))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(w.fv, w.dc, vi, key, cp.Material, float32(cp.Neighbor.GetMaxZ()), float32(cp.Volume.GetMaxZ()))
				} else {
					w.pushWall(w.fv, w.dc, vi, key, cp.Material, float32(cp.Volume.GetMinZ()), float32(cp.Neighbor.GetMinZ()))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				key := CreatePolygonSector(cp)
				if w.processedPolygons[key] {
					continue
				}
				w.processedPolygons[key] = true
				w.visibleVolumes[cp.Volume] = true
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(w.fv, w.dc, key, cp.MaterialCeil, float32(cp.Volume.GetMaxZ())); sky != nil {
						w.cSky = sky
					}
				} else {
					// IdFloor, IdFloorTest
					if sky := w.pushFlat(w.fv, w.dc, key, cp.MaterialFloor, float32(cp.Volume.GetMinZ())); sky != nil {
						w.cSky = sky
					}
				}
			}
		}
	}

	px, py, pz := vi.GetXYZ()
	angle, pitch, roll := vi.GetAngle(), vi.GetPitch(), vi.GetRoll()
	fm, fr := CreateFrontRearFrustum(float32(w.cal.AspectRatio), float32(w.cal.ZFarRoom), float32(px), float32(py), float32(pz), angle, pitch, roll)
	frustumFront, frustumRear := vi.GetFrustum(fm, fr)

	w.pushLights(w.fl, lights, frustumFront, frustumRear)

	tA, tC := engine.GetThings().GetActive()

	w.pushThings(w.fv, w.dc, vi, tA, tC, w.visibleVolumes)

	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushWall adds a textured wall polygon to the batch using the specified view matrix, polygon key, animation, and height range.
func (w *BuilderTraverse) pushWall(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, cp PolyKey, anim *textures.Material, zBottom, zTop float32) {
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	layer, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	scaleW, scaleH := tex.GetScaleFactor()

	u0 := cp.u0 / (float32(texW) * float32(scaleW))
	u1 := cp.u1 / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)

	// Back-transformation: View-Space -> World-Space
	sin, cos := vi.GetAngleFull()
	viX, vizY := vi.GetXY()
	wx1 := (cp.tx1 * float32(sin)) + (cp.tz1 * float32(cos)) + float32(viX)
	wy1 := -(cp.tx1 * float32(cos)) + (cp.tz1 * float32(sin)) + float32(vizY)
	wx2 := (cp.tx2 * float32(sin)) + (cp.tz2 * float32(cos)) + float32(viX)
	wy2 := -(cp.tx2 * float32(cos)) + (cp.tz2 * float32(sin)) + float32(vizY)

	startIndices := w.fv.GetIndicesLen()

	idx0 := fv.AddVertex6(wx1, zTop, -wy1, u0, vTop, layer)
	idx1 := fv.AddVertex6(wx1, zBottom, -wy1, u0, vBottom, layer)
	idx2 := fv.AddVertex6(wx2, zBottom, -wy2, u1, vBottom, layer)
	idx3 := fv.AddVertex6(wx2, zTop, -wy2, u1, vTop, layer)

	fv.AddTriangle(idx0, idx1, idx2)
	fv.AddTriangle(idx0, idx2, idx3)

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
}

// pushFlat processes a flat surface for rendering, computes its vertices and indices, and adds draw commands.
func (w *BuilderTraverse) pushFlat(fv *FrameVertices, dc *DrawCommands, cp PolyKey, anim *textures.Material, zF float32) *textures.Texture {
	if anim.Kind() == int(config.MaterialKindSky) {
		return anim.CurrentFrame()
	}

	tex := anim.CurrentFrame()
	if tex == nil {
		return nil
	}
	faces := cp.volume.GetFaces()
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

	indices := make([]uint32, len(faces))
	for i, seg := range faces {
		v := seg.GetStart()
		u := (float32(v.X) / float32(texW)) * float32(scaleH)
		vV := (float32(-v.Y) / float32(texH)) * float32(scaleH)
		// Coordinate assolute
		indices[i] = fv.AddVertex6(float32(v.X), zF, float32(-v.Y), u, vV, layer)
	}

	for i := 1; i < len(faces)-1; i++ {
		fv.AddTriangle(indices[0], indices[i], indices[i+1])
	}

	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
	return nil
}

// pushThings processes and adds things to the frame rendering pipeline based on their position, texture, and visibility.
func (w *BuilderTraverse) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing, thingsCount int, volumes map[*model.Volume]bool) {
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
		tPosX, tPosY, zBot := thing.GetBottomLeft()
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

// pushLights adds the specified lights to the frame based on their type, position, and characteristics, filtering by sector.
// pushLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderTraverse) pushLights(fl *FrameLights, lights *model.Lights, frustumFront, frustumRear *physics.Frustum) {
	queryLights := func(object physics.IAABB) bool {
		if l, ok := object.(*model.Light); ok {

			fl.Create(l)
		}
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}
