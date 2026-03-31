package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
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
	tex               *Textures
	vertices          *FrameVertices
	drawCommands      *DrawCommands
	frameLights       *FrameLights
	visibleSectors    map[*model.Sector]bool
	processedPolygons map[PolyKey]bool
}

// NewBuilderTraverse creates and initializes a new BuilderTraverse with preallocated memory for vertices, commands, and lights.
func NewBuilderTraverse(vertices *FrameVertices, commands *DrawCommands, lights *FrameLights, tex *Textures) *BuilderTraverse {
	return &BuilderTraverse{
		tex:               tex,
		vertices:          vertices,
		drawCommands:      commands,
		frameLights:       lights,
		visibleSectors:    make(map[*model.Sector]bool, 256),
		processedPolygons: make(map[PolyKey]bool, 2048),
	}
}

// GetFrameVertices retrieves the current vertex and index buffers along with their respective counts.
//func (w *BuilderTraverse) GetFrameVertices() ([]float32, int32, []uint32, int32) {
//	return w.vertices.GetVertices()
//}

// Reset clears all data stored in the BuilderTraverse, resetting its vertices, draw commands, lights, and sector data maps.
func (w *BuilderTraverse) Reset() {
	w.vertices.Reset()
	w.drawCommands.Reset()
	w.frameLights.Reset()
	for k := range w.visibleSectors {
		delete(w.visibleSectors, k)
	}
	for k := range w.processedPolygons {
		delete(w.processedPolygons, k)
	}
}

// Compute generates a batch for rendering by processing sectors, walls, floors, ceilings, things, and lights.
// It updates visible sectors and processed polygons, and returns a sky texture if one is found.
func (w *BuilderTraverse) Compute(vi *model.ViewMatrix, css []*model.CompiledSector, compiled int, things []model.IThing, lights []*model.Light) *textures.Texture {
	var cSky *textures.Texture = nil

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
					w.pushWall(vi, key, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(vi, key, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
				} else {
					w.pushWall(vi, key, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				key := CreatePolygonSector(cp)
				if w.processedPolygons[key] {
					continue
				}
				w.processedPolygons[key] = true
				w.visibleSectors[cp.Sector] = true
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(key, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
						cSky = sky
					}
				} else {
					// IdFloor, IdFloorTest
					if sky := w.pushFlat(key, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
						cSky = sky
					}
				}
			}
		}
	}

	w.pushLights(lights, w.visibleSectors)
	w.pushThings(vi, things, w.visibleSectors)
	return cSky
}

// pushWall adds a textured wall polygon to the batch using the specified view matrix, polygon key, animation, and height range.
func (w *BuilderTraverse) pushWall(vi *model.ViewMatrix, cp PolyKey, anim *textures.Animation, zBottom, zTop float32) {
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

	startIndices := w.vertices.GetIndicesLen()

	idx0 := w.vertices.AddVertex(wx1, zTop, -wy1, u0, vTop)
	idx1 := w.vertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom)
	idx2 := w.vertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom)
	idx3 := w.vertices.AddVertex(wx2, zTop, -wy2, u1, vTop)

	w.vertices.AddTriangle(idx0, idx1, idx2)
	w.vertices.AddTriangle(idx0, idx2, idx3)

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
}

// pushFlat processes a flat surface for rendering, computes its vertices and indices, and adds draw commands.
func (w *BuilderTraverse) pushFlat(cp PolyKey, anim *textures.Animation, zF float32) *textures.Texture {
	if anim.Kind() == int(model.AnimationKindSky) {
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
		// Coordinate assolute
		indices[i] = w.vertices.AddVertex(float32(v.X), zF, float32(-v.Y), u, vV)
	}

	for i := 1; i < len(segments)-1; i++ {
		w.vertices.AddTriangle(indices[0], indices[i], indices[i+1])
	}

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
	return nil
}

// pushThings processes and adds things to the frame rendering pipeline based on their position, texture, and visibility.
func (w *BuilderTraverse) pushThings(vi *model.ViewMatrix, things []model.IThing, sectors map[*model.Sector]bool) {
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

// pushLights adds the specified lights to the frame based on their type, position, and characteristics, filtering by sector.
func (w *BuilderTraverse) pushLights(lights []*model.Light, sectors map[*model.Sector]bool) {
	if len(lights) == 0 {
		return
	}

	for _, l := range lights {
		//if _, ok := sectors[l.GetSector()]; !ok {
		//	continue
		//}
		w.frameLights.Create(l)
	}
}
