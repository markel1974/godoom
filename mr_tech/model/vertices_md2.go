package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMD2 represents a structured collection of 3D model data, including frames, actions, and volume association.
type VerticesMD2 struct {
	volume        *Volume
	frames        [][]*Face
	actions       [][2]int
	startFrame    int
	endFrame      int
	relativeFrame int
}

// NewVerticesMD2 creates a new VerticesMD2 instance with frames, actions, and volume based on the provided configuration.
func NewVerticesMD2(cfg *config.Thing, pos geometry.XYZ, materials *Materials) *VerticesMD2 {
	x := pos.X - cfg.Radius
	y := pos.Y - cfg.Radius
	z := pos.Z
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	d := cfg.Height
	md2Cfg := cfg.MD2
	volume := NewVolumeDetails3d(0, "md2", "thing", x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	v := &VerticesMD2{
		volume:        volume,
		frames:        make([][]*Face, len(md2Cfg.Frames)),
		actions:       md2Cfg.ActionIntervals,
		startFrame:    0,
		endFrame:      len(md2Cfg.Frames) - 1,
		relativeFrame: -1,
	}
	if v.endFrame < 0 {
		v.endFrame = 0
	}
	if len(v.actions) > 0 {
		v.SetAction(0)
	}
	//v.volume.SetBillboard(2.0)

	for frameIdx, cfgFrame := range md2Cfg.Frames {
		frameFaces := make([]*Face, len(cfgFrame.Triangles))
		for triIdx, tri := range cfgFrame.Triangles {
			tag := fmt.Sprintf("%s_%d_%d", "md2", frameIdx, triIdx)
			points := [3]geometry.XYZ{tri[0].Pos, tri[1].Pos, tri[2].Pos}
			material := materials.GetMaterial(cfg.Material)
			face := NewFace(nil, points, tag, material)
			face.SetUV(float64(tri[0].U), float64(tri[0].V), float64(tri[1].U), float64(tri[1].V), float64(tri[2].U), float64(tri[2].V))
			face.LockUV(true)
			frameFaces[triIdx] = face
		}
		v.frames[frameIdx] = frameFaces
	}
	v.compute(0)
	return v
}

// GetVolume retrieves the Volume instance associated with the VertexMD2.
func (v *VerticesMD2) GetVolume() *Volume {
	return v.volume
}

// SetAction updates the start and end frame of the VertexMD2 based on the action index provided.
func (v *VerticesMD2) SetAction(idx int) {
	if idx < 0 || idx >= len(v.actions) {
		return
	}
	v.startFrame = v.actions[idx][0]
	v.endFrame = v.actions[idx][1]
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *VerticesMD2) GetVertices(tick uint64) ([]*Face, int, []*Face, int, float64) {
	// Se non ci sono frame, restituisce vuoto
	if len(v.frames) == 0 {
		return nil, 0, nil, 0, 0.0
	}
	// Se c'è un solo frame nell'animazione, restituisce lo stesso frame due volte senza lerp
	if v.startFrame == v.endFrame {
		s := v.frames[v.startFrame]
		sCount := len(s)
		return s, sCount, s, sCount, 0.0
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

	if relativeFrameA != v.relativeFrame {
		v.relativeFrame = relativeFrameA
		v.compute(idxA)
	}

	return v.frames[idxA], len(v.frames[idxA]), v.frames[idxB], len(v.frames[idxB]), lerpT
}

// GetDisplacement retrieves the displacement vector (dx, dy, dz) by getting the center position of the associated entity.
func (v *VerticesMD2) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetCenter()
}

// GetBillboard returns a constant value, typically used to represent the billboard distance for the VerticesMD2 instance.
func (v *VerticesMD2) GetBillboard() float64 {
	return 2.0
}

// SetThing sets the IThing instance associated with the VerticesMD2 volume.
func (v *VerticesMD2) SetThing(t IThing) {
	v.volume.SetThing(t)
}

// compute updates the volume by clearing faces, adding a new set of faces from the specified frame index, and rebuilding.
func (v *VerticesMD2) compute(idx int) {
	v.volume.ClearFaces()
	for _, f := range v.frames[idx] {
		v.volume.AddFace(f)
	}
	v.volume.Rebuild()
}
