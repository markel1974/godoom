package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VertexMD2 represents a data structure used for handling 3D vertex animations with multiple frames and actions.
type VertexMD2 struct {
	volume     *Volume
	frames     [][]*Face
	actions    [][2]int
	startFrame int
	endFrame   int
}

// NewVertexMD2 creates a new VertexMD2 object with volume, frames, actions, and initial configurations for animation rendering.
func NewVertexMD2(cfg *config.MD2, material *textures.Material, x, y, z, w, h, d, mass, restitution, friction, gForce float64) *VertexMD2 {
	volume := NewVolumeDetails3d(0, "md2", "thing", x, y, z, w, h, d, mass, restitution, friction, gForce)
	v := &VertexMD2{
		volume:     volume,
		frames:     make([][]*Face, len(cfg.Frames)),
		actions:    cfg.ActionIntervals,
		startFrame: 0,
		endFrame:   len(cfg.Frames) - 1,
	}
	if v.endFrame < 0 {
		v.endFrame = 0
	}
	if len(v.actions) > 0 {
		v.SetAction(0)
	}
	//v.volume.SetBillboard(2.0)

	for frameIdx, cfgFrame := range cfg.Frames {
		frameFaces := make([]*Face, len(cfgFrame.Triangles))
		for triIdx, tri := range cfgFrame.Triangles {
			tag := fmt.Sprintf("%s_%d_%d", "md2", frameIdx, triIdx)
			points := [3]geometry.XYZ{tri[0].Pos, tri[1].Pos, tri[2].Pos}
			face := NewFace(nil, points, tag, material)
			face.SetUV(float64(tri[0].U), float64(tri[0].V), float64(tri[1].U), float64(tri[1].V), float64(tri[2].U), float64(tri[2].V))
			face.LockUV(true)
			frameFaces[triIdx] = face
		}
		v.frames[frameIdx] = frameFaces
	}
	if len(v.frames[0]) > 0 {
		for _, f := range v.frames[0] {
			v.volume.AddFace(f)
		}
	}
	v.volume.Rebuild()
	return v
}

// GetVolume retrieves the associated Volume instance of the VertexMD2.
func (v *VertexMD2) GetVolume() *Volume {
	return v.volume
}

// SetAction updates the start and end frame indices based on the action at the specified index if it exists within bounds.
func (v *VertexMD2) SetAction(idx int) {
	if idx < 0 || idx >= len(v.actions) {
		return
	}
	v.startFrame = v.actions[idx][0]
	v.endFrame = v.actions[idx][1]
}

// GetVertices returns two sets of interpolated faces and a lerp factor for rendering based on the provided tick value.
// GetVertices returns two sets of interpolated faces and a lerp factor for rendering based on the provided tick value.
func (v *VertexMD2) GetVertices(tick uint64) ([]*Face, []*Face, float64) {
	// Se non ci sono frame, restituisce vuoto
	if len(v.frames) == 0 {
		return nil, nil, 0.0
	}
	// Se c'è un solo frame nell'animazione, restituisce lo stesso frame due volte senza lerp
	if v.startFrame == v.endFrame {
		return v.frames[v.startFrame], v.frames[v.startFrame], 0.0
	}
	const groupSize = 6.0
	frameFloat := textures.TickGrouped(tick, int(groupSize))
	// Calcoliamo la durata dell'animazione corrente in termini di numero di frame
	// Aggiungiamo 1 perché l'intervallo è inclusivo (es. frame 0-5 sono 6 frame)
	animLength := v.endFrame - v.startFrame + 1
	// Assicuriamoci che animLength sia valido (prevenzione crash in caso di configurazione errata)
	if animLength <= 0 {
		animLength = 1
	}
	// Troviamo l'indice relativo all'interno dell'animazione corrente
	relativeFrameA := int(frameFloat) % animLength
	// L'indice B è il frame successivo relativo, e se supera la lunghezza dell'animazione, torna a 0
	relativeFrameB := (relativeFrameA + 1) % animLength
	// Mappiamo l'indice relativo sull'indice assoluto dell'array v.frames
	idxA := v.startFrame + relativeFrameA
	idxB := v.startFrame + relativeFrameB
	// Parte frazionaria per l'interpolazione fluida tra i due frame calcolati
	lerpT := frameFloat - math.Floor(frameFloat)
	return v.frames[idxA], v.frames[idxB], lerpT
}

// GetPosition returns the center position of the associated entity as a tuple of three float64 values.
func (v *VertexMD2) GetPosition() (float64, float64, float64) {
	return v.volume.entity.GetCenter()
}

// GetBillboard retrieves the billboard value associated with the Face instance.
func (v *VertexMD2) GetBillboard() float64 {
	return 2.0
}

// SetThing assigns an IThing instance to the underlying Volume of the VertexSprite.
func (v *VertexMD2) SetThing(t IThing) {
	v.volume.SetThing(t)
}
