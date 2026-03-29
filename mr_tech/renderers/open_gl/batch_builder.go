package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// PolyKey represents a key used for uniquely identifying a polygon within a 3D rendered sector's data structure.
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

// CreatePolygonSegment generates a PolyKey based on the provided CompiledPolygon.
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

// CreatePolygonSector creates and returns a PolyKey initialized with the Sector and Kind of the given CompiledPolygon.
func CreatePolygonSector(cp *model.CompiledPolygon) PolyKey {
	key := PolyKey{
		sector: cp.Sector,
		kind:   cp.Kind,
	}
	return key
}

// BatchBuilder is a structure that manages batching of drawing data, including textures, vertices, draw commands, and lights.
type BatchBuilder struct {
	tex          *Textures
	vertices     *FrameVertices
	drawCommands *DrawCommands
	frameLights  *FrameLights
}

// NewBatchBuilder creates and returns a new BatchBuilder configured with preallocated resources for rendering batches.
func NewBatchBuilder(compiler *Textures) *BatchBuilder {
	return &BatchBuilder{
		tex:          compiler,
		vertices:     NewFrameVertices(maxBatchVertices),
		drawCommands: NewDrawCommands(maxFrameCommands),
		frameLights:  NewFrameLights(256),
	}
}

// VerticesStride returns the stride of the vertex buffer in bytes by multiplying the stride in elements by 4.
func (w *BatchBuilder) VerticesStride() int32 {
	return w.vertices.Stride() * 4
}

// LightsStride returns the stride of the vertex buffer in bytes by multiplying the stride in elements by 4.
func (w *BatchBuilder) LightsStride() int32 {
	return w.frameLights.Stride() * 4
}

// GetFrameVertices retrieves the vertex data and indices from the frame's vertex buffer.
func (w *BatchBuilder) GetFrameVertices() ([]float32, int32, []uint32, int32) {
	return w.vertices.GetVertices()
}

// GetFrameLights retrieves the frame light data and its count from the current BatchBuilder instance.
func (w *BatchBuilder) GetFrameLights() ([]float32, int32) {
	fvLen := w.frameLights.Len()
	fv := w.frameLights.Get()
	return fv, int32(fvLen)
}

// GetDrawCommands retrieves the collection of draw commands stored in the BatchBuilder's DrawCommands structure.
func (w *BatchBuilder) GetDrawCommands() []*DrawCommand {
	return w.drawCommands.Get()
}

// Reset clears the state of BatchBuilder by resetting vertices, draw commands, and frame lights to their initial states.
func (w *BatchBuilder) Reset() {
	w.vertices.Reset()
	w.drawCommands.Reset()
	w.frameLights.Reset()
}

// CreateBatch generates a batch of textures based on the given view matrix, compiled sectors, things, and lights.
func (w *BatchBuilder) CreateBatch(vi *model.ViewMatrix, css []*model.CompiledSector, compiled int, things []model.IThing, lights []*model.Light) *textures.Texture {
	var cSky *textures.Texture = nil

	visibleSectors := make(map[*model.Sector]bool)
	processedPolygons := make(map[PolyKey]bool)

	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]

		polygons := current.Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				key := CreatePolygonSegment(cp)
				if processedPolygons[key] {
					continue
				}
				processedPolygons[key] = true
				visibleSectors[cp.Sector] = true
				if cp.Kind == model.IdWall {
					w.pushWall(vi, key, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
				} else if cp.Kind == model.IdUpper {
					w.pushWall(vi, key, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
				} else {
					w.pushWall(vi, key, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
				}
			case model.IdCeil, model.IdCeilTest, model.IdFloor, model.IdFloorTest:
				key := CreatePolygonSector(cp)
				if processedPolygons[key] {
					continue
				}
				processedPolygons[key] = true
				visibleSectors[cp.Sector] = true
				if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
					if sky := w.pushFlat(vi, key, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
						cSky = sky
					}
				} else {
					// IdFloor, IdFloorTest
					if sky := w.pushFlat(vi, key, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
						cSky = sky
					}
				}
			}
		}
	}

	w.pushLights(vi, lights, visibleSectors)
	w.pushThings(vi, things, visibleSectors)
	return cSky
}

// pushWall appends indexed vertices and draw commands to render a wall with given position, texture, and height range.
func (w *BatchBuilder) pushWall(vi *model.ViewMatrix, cp PolyKey, anim *textures.Animation, zBottom, zTop float32) {
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

	sin, cos := vi.GetAngle()
	viX, vizY := vi.GetXY()
	wx1 := (cp.tx1 * float32(sin)) + (cp.tz1 * float32(cos)) + float32(viX)
	wy1 := -(cp.tx1 * float32(cos)) + (cp.tz1 * float32(sin)) + float32(vizY)
	wx2 := (cp.tx2 * float32(sin)) + (cp.tz2 * float32(cos)) + float32(viX)
	wy2 := -(cp.tx2 * float32(cos)) + (cp.tz2 * float32(sin)) + float32(vizY)

	startIndices := w.vertices.GetIndicesLen()

	// Invia SOLO i 4 vertici perimetrali fisici (Stride: X, Y, Z, U, V)
	idx0 := w.vertices.AddVertex(wx1, zTop, -wy1, u0, vTop)
	idx1 := w.vertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom)
	idx2 := w.vertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom)
	idx3 := w.vertices.AddVertex(wx2, zTop, -wy2, u1, vTop)

	// Mesh indicizzata: unione dei due triangoli riutilizzando i vertici
	w.vertices.AddTriangle(idx0, idx1, idx2)
	w.vertices.AddTriangle(idx0, idx2, idx3)

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
}

// pushFlat adds indexed flat polygonal segments to the batch for rendering, applying texture, light, and scale transformations.
func (w *BatchBuilder) pushFlat(vi *model.ViewMatrix, cp PolyKey, anim *textures.Animation, zF float32) *textures.Texture {
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

	// Pre-caricamento vertici unici (Fan Triangle pattern)
	indices := make([]uint32, len(segments))
	for i, seg := range segments {
		v := seg.Start
		u := (float32(v.X) / float32(texW)) * float32(scaleH)
		vV := (float32(-v.Y) / float32(texH)) * float32(scaleH)
		indices[i] = w.vertices.AddVertex(float32(v.X), zF, float32(-v.Y), u, vV)
	}

	// Costruzione della mesh indicizzata a raggiera (Fan) partendo da indices[0]
	for i := 1; i < len(segments)-1; i++ {
		// NOTA: L'ordine dei vertici determina il winding (Cull Face).
		// Essendo state eliminate le normali su CPU, il winding corretto è essenziale per dFdx/dFdy.
		w.vertices.AddTriangle(indices[0], indices[i], indices[i+1])
	}

	currentIndices := w.vertices.GetIndicesLen()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, startIndices, currentIndices)
	return nil
}

// pushLights processes the provided lights and adds them to the frame with their properties transformed for rendering.
func (w *BatchBuilder) pushLights(vi *model.ViewMatrix, lights []*model.Light, sectors map[*model.Sector]bool) {
	if len(lights) == 0 {
		return
	}

	for _, l := range lights {
		//if _, ok := sectors[l.GetSector()]; !ok {
		//	continue
		//}
		r, g, b := float32(1.0), float32(1.0), float32(1.0)
		dirGlX, dirGlY, dirGlZ := float32(0.0), float32(0.0), float32(0.0)
		cutOff := float32(0)
		outerCutOff := float32(0)
		pos := l.GetPos()
		intensity := float32(l.GetIntensity())
		falloff := float32(0.0)
		lightType := float32(-1)

		switch l.GetKind() {
		case model.LightKindOpenAir:
			continue
			pos.Z = 100
			r, g, b = float32(1.0), float32(1.0), float32(1.0)
			lightType = 0
			falloff = 500.0
		case model.LightKindAmbient:
			r, g, b = float32(1.0), float32(1.0), float32(1.0)
			lightType = 0
			falloff = 10.0
		case model.LightKindSpot:
			lightType = 1
			falloff = 100.0
			r, g, b = float32(1.0), float32(1.0), float32(1.0)
			dirGlX, dirGlY, dirGlZ = float32(0.0), float32(-1.0), float32(0.0)
			cutOff = float32(math.Cos(35.0 * math.Pi / 180.0))
			outerCutOff = float32(math.Cos(40 * math.Pi / 180.0))
		case model.LightKindNone:
			continue
		default:
			lightType = 0
		}

		w.frameLights.Add(
			float32(pos.X), float32(pos.Z), float32(-pos.Y), lightType,
			r, g, b, intensity,
			dirGlX, dirGlY, dirGlZ, falloff,
			cutOff, outerCutOff, 0.0, 0.0,
		)
	}
}

// pushThings processes visible entities, applies transformations, performs culling, and updates vertex and draw command buffers.
func (w *BatchBuilder) pushThings(vi *model.ViewMatrix, things []model.IThing, sectors map[*model.Sector]bool) {
	const minDist = 0.0001
	if len(things) == 0 {
		return
	}
	//fv := w.vertices
	camX, camY := vi.GetXY()

	// 1. Culling e calcolo distanza quadrica
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

		// Vettore Right normalizzato e scalato per l'estensione del quad
		halfW := width / 2.0
		rX := -((camY - tPosY) / dist) * halfW
		rY := ((camX - tPosX) / dist) * halfW

		// Coordinate planari dei due spigoli
		v1x := float32(tPosX - rX)
		v1y := float32(tPosY - rY)
		v2x := float32(tPosX + rX)
		v2y := float32(tPosY + rY)

		// Quota verticale
		zBottom := float32(t.GetFloorY())
		zTop := zBottom + float32(height)

		startIndices := w.vertices.GetIndicesLen()

		// A differenza di un muro che ripete la texture, lo sprite mappa l'intera texture (UV 0.0 -> 1.0)
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
