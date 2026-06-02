package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMD1 represents a structured collection of 3D model data, including frames, actions, and volume association.
type VerticesMD1 struct {
	viewVolume *Volume
	//rootEntity *physics.Entity
	volumes    []*Volume
	actions    [][2]int
	startFrame int
	endFrame   int
	idxA       int
}

// NewVerticesMD2 creates a new VerticesMD1 instance with frames, actions, and volume based on the provided configuration.
func NewVerticesMD2(cfg *config.Thing, materials *Materials) *VerticesMD1 {
	if len(cfg.MD1.Frames) == 0 {
		panic(fmt.Sprintf("no MD1 frames for thing %s", cfg.Id))
	}

	v := &VerticesMD1{
		volumes:    make([]*Volume, len(cfg.MD1.Frames)),
		actions:    cfg.MD1.ActionIntervals,
		startFrame: 0,
		endFrame:   len(cfg.MD1.Frames) - 1,
		idxA:       -1,
	}
	if v.endFrame < 0 {
		v.endFrame = 0
	}
	if len(v.actions) > 0 {
		v.SetAction(0)
	}
	//entity := physics.NewEntity(x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	for frameIdx, cfgFrame := range cfg.MD1.Frames {
		baseId := fmt.Sprintf("%s_md1_frame_%d", cfg.Id, frameIdx)
		volume := NewVolume(frameIdx, baseId, "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		for triIdx, tri := range cfgFrame.Triangles {
			tag := fmt.Sprintf("%s_%d", baseId, triIdx)
			points := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			material := materials.GetMaterial(tri.Material)
			face := NewFace(points, tag, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			volume.AddFace(face)
		}
		volume.Rebuild()
		v.volumes[frameIdx] = volume
	}
	v.viewVolume = v.volumes[0]
	//v.rootEntity = v.viewVolume.GetEntity()
	return v
}

// GetVolume retrieves the Volume instance associated with the VertexMD2.
func (v *VerticesMD1) GetVolume() *Volume {
	return v.viewVolume
}

// GetEntity returns the physics.Entity instance associated with the VerticesMD1 viewVolume.
func (v *VerticesMD1) GetEntity() *physics.Entity {
	return v.viewVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the associated entity in the view volume.
func (v *VerticesMD1) GetAABB() *physics.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// SetAction updates the start and end frame of the VertexMD2 based on the action index provided.
func (v *VerticesMD1) SetAction(idx int) {
	if idx < 0 || idx >= len(v.actions) {
		return
	}
	v.startFrame = v.actions[idx][0]
	v.endFrame = v.actions[idx][1]
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *VerticesMD1) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	// Se non ci sono frame, restituisce vuoto
	if len(v.volumes) == 0 {
		return nil, 0, nil, 0, 0.0, v.GetBillboard()
	}
	// Se c'è un solo frame nell'animazione, restituisce lo stesso frame due volte senza lerp
	if v.startFrame == v.endFrame {
		s := v.volumes[v.startFrame]
		faces, faceCount := s.GetFaces()
		return faces, faceCount, faces, faceCount, 0.0, v.GetBillboard()
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

	curr := v.volumes[idxA]
	next := v.volumes[idxB]

	if v.idxA != idxA {
		v.idxA = idxA
		// TODO TERMINATE IMPLEMENTATION
		//v.rootEntity.SetSize(curr.GetEntity().GetSize())
		//curr.entity = v.rootEntity
		//v.rootVolume = curr
	}

	facesA, faceCountA := curr.GetFaces()
	facesB, faceCountB := next.GetFaces()

	return facesA, faceCountA, facesB, faceCountB, lerpT, v.GetBillboard()
}

// GetDisplacement retrieves the displacement vector (dx, dy, dz) by getting the center position of the associated entity.
func (v *VerticesMD1) GetDisplacement() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetCenter()
}

// GetBillboard returns a constant value, typically used to represent the billboard distance for the VerticesMD1 instance.
func (v *VerticesMD1) GetBillboard() float64 {
	return 2.0
}

// SetThing sets the IThing instance associated with the VerticesMD1 volume.
func (v *VerticesMD1) SetThing(t IThing) {
	for _, f := range v.volumes {
		f.SetThing(t)
	}
}
