package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const lightScaleFactor = 5.0

// BatchBuilder is a utility for constructing GPU-ready batches of vertices and draw commands.
type BatchBuilder struct {
	tex           *Textures
	frameVertices *FrameVertices
	drawCommands  *DrawCommands
}

// NewBatchBuilder initializes and returns a new BatchBuilder using the provided Compiler to manage rendering resources.
func NewBatchBuilder(compiler *Textures) *BatchBuilder {
	return &BatchBuilder{
		tex:           compiler,
		frameVertices: NewFrameVertices(maxBatchVertices),
		drawCommands:  NewDrawCommands(maxFrameCommands),
	}
}

// Stride calculates and returns the vertex stride size in bytes, based on the alignment of frame vertices.
func (w *BatchBuilder) Stride() int32 {
	return w.frameVertices.Alignment() * 4
}

// GetFrameVertices retrieves the current frame's vertex data as a slice of float32 and the number of vertices stored.
func (w *BatchBuilder) GetFrameVertices() ([]float32, int) {
	fvLen := w.frameVertices.Len()
	fv := w.frameVertices.Get()
	return fv, fvLen
}

// GetDrawCommands retrieves the list of draw commands that represent rendering instructions for the current batch.
func (w *BatchBuilder) GetDrawCommands() []*DrawCommand {
	return w.drawCommands.Get()
}

func (w *BatchBuilder) Reset() {
	w.frameVertices.Reset()
	w.drawCommands.Reset()
}

// CreateBatch generates a batch of rendering data by processing compiled sectors and objects with the provided ViewMatrix.
func (w *BatchBuilder) CreateBatch(vi *model.ViewMatrix, css []*model.CompiledSector, compiled int, things []model.IThing, lights []*model.Light) *textures.Texture {
	var cSky *textures.Texture = nil

	//TODO BETTER IMPLEMENTATION
	sectors := make(map[*model.Sector]bool, len(css))

	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]

		polygons := current.Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]

			sectors[cp.Sector] = true

			switch cp.Kind {
			case model.IdWall:
				w.pushWall(vi, cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
			case model.IdUpper:
				w.pushWall(vi, cp, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
			case model.IdLower:
				w.pushWall(vi, cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
			case model.IdCeil, model.IdCeilTest:
				if sky := w.pushFlat(vi, cp, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
					cSky = sky
				}
			case model.IdFloor, model.IdFloorTest:
				if sky := w.pushFlat(vi, cp, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
					cSky = sky
				}
			}
		}
	}

	//TODO
	w.pushThings(vi, things, sectors)
	return cSky
}

// pushWall adds vertices for a wall segment to the frame, computing texture coordinates, normals, and lighting.
func (w *BatchBuilder) pushWall(vi *model.ViewMatrix, cp *model.CompiledPolygon, anim *textures.Animation, zBottom, zTop float32) {
	//prepare
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	texId, normTexId, emissiveTexId, ok := w.tex.Get(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	startLen := w.frameVertices.Len()
	scaleW, scaleH := anim.ScaleFactor()

	u0 := float32(cp.U0) / (float32(texW) * float32(scaleW))
	u1 := float32(cp.U1) / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)

	sin, cos := vi.GetAngle()
	viX, vizY := vi.GetXY()
	wx1 := float32((cp.Tx1 * sin) + (cp.Tz1 * cos) + viX)
	wy1 := float32(-(cp.Tx1 * cos) + (cp.Tz1 * sin) + vizY)
	wx2 := float32((cp.Tx2 * sin) + (cp.Tz2 * cos) + viX)
	wy2 := float32(-(cp.Tx2 * cos) + (cp.Tz2 * sin) + vizY)

	dx := float64(wx2 - wx1)
	dz := float64((-wy2) - (-wy1))
	length := math.Hypot(dx, dz)

	nX := float32(-dz / length)
	nY := float32(0.0)
	nZ := float32(dx / length)

	light, lcX, lcY, lcZ := w.createLight(vi, cp.Sector.Light, lightScaleFactor)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zTop, -wy2, u1, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)

	//apply
	currentLen := w.frameVertices.Len()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())
}

// pushFlat processes a flat polygon, computes its light and texture mapping, and adds its vertices to the frame buffer.
func (w *BatchBuilder) pushFlat(vi *model.ViewMatrix, cp *model.CompiledPolygon, anim *textures.Animation, zF float32) *textures.Texture {
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
	//prepare
	texId, normTexId, emissiveTexId, ok := w.tex.Get(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	startLen := w.frameVertices.Len()
	_, scaleH := anim.ScaleFactor()
	v0 := segments[0].Start

	u0 := (float32(v0.X) / float32(texW)) * float32(scaleH)
	v0V := (float32(-v0.Y) / float32(texH)) * float32(scaleH)

	nY := float32(1.0)
	if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
		nY = -1.0
	}

	light, lcX, lcY, lcZ := w.createLight(vi, cp.Sector.Light, lightScaleFactor)

	for i := 1; i < len(segments)-1; i++ {
		v1, v2 := segments[i].Start, segments[i+1].Start

		u1 := (float32(v1.X) / float32(texW)) * float32(scaleH)
		v1V := (float32(-v1.Y) / float32(texH)) * float32(scaleH)
		u2 := (float32(v2.X) / float32(texW)) * float32(scaleH)
		v2V := (float32(-v2.Y) / float32(texH)) * float32(scaleH)

		w.frameVertices.AddVertex(float32(v0.X), zF, float32(-v0.Y), u0, v0V, light, lcX, lcY, lcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v1.X), zF, float32(-v1.Y), u1, v1V, light, lcX, lcY, lcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v2.X), zF, float32(-v2.Y), u2, v2V, light, lcX, lcY, lcZ, 0, nY, 0)
	}

	//apply
	currentLen := w.frameVertices.Len()
	w.drawCommands.Compute(texId, normTexId, emissiveTexId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())

	return nil
}

// pushThings processes and batches a list of things into the frame buffer using depth sorting and cylindrical billboarding.
func (w *BatchBuilder) pushThings(vi *model.ViewMatrix, things []model.IThing, sectors map[*model.Sector]bool) {
	const minDist = 0.0001
	if len(things) == 0 {
		return
	}
	fv := w.frameVertices
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

		light, vLcX, vLcY, vLcZ := w.createLight(vi, t.GetLight(), lightScaleFactor)

		// --- CALCOLO NORMALE IDENTICO A PUSH WALL ---
		dxNorm := float64(v2x - v1x)
		dzNorm := float64((-v2y) - (-v1y))
		length := math.Hypot(dxNorm, dzNorm)

		nX := float32(-dzNorm / length)
		nY := float32(0.0)
		nZ := float32(dxNorm / length)

		startLen := int32(fv.Len())

		// --- BATCHING NEL VBO ---
		// A differenza di un muro che ripete la texture, lo sprite mappa l'intera texture (UV 0.0 -> 1.0)
		u0, u1 := float32(0.0), float32(1.0)
		vTop, vBottom := float32(0.0), float32(1.0)

		// Triangolo 1 (Top-Left -> Bottom-Left -> Bottom-Right)
		w.frameVertices.AddVertex(v1x, zTop, -v1y, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v1x, zBottom, -v1y, u0, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zBottom, -v2y, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

		// Triangolo 2 (Top-Left -> Bottom-Right -> Top-Right)
		w.frameVertices.AddVertex(v1x, zTop, -v1y, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zBottom, -v2y, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zTop, -v2y, u1, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

		w.drawCommands.Compute(texId, normTexId, emissiveTexId, startLen, int32(fv.Len()), fv.Alignment())
	}
}

// createLight calculates light level and transformed position for rendering, returning intensity and adjusted position values.
func (w *BatchBuilder) createLight(vi *model.ViewMatrix, mLight *model.Light, lightFactor float64) (float32, float32, float32, float32) {
	lightIntensity := (1.0 - mLight.GetIntensity()) * lightFactor
	lightPos := mLight.GetPos()

	// Trasmissione diretta delle coordinate World (X, Z, -Y)
	return float32(lightIntensity), float32(lightPos.X), float32(lightPos.Z), float32(-lightPos.Y)
}
