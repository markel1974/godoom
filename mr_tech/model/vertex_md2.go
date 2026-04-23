package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VertexMD2 represents a structure containing 3D volume data and associated animation frame face mappings.
type VertexMD2 struct {
	volume *Volume
	frames [][]*Face
}

// NewVertexMD2 creates a new VertexMD2 object with the given configuration, animation, dimensions, mass, restitution, and friction.
func NewVertexMD2(cfg *config.Model3d, anim *textures.Animation, x, y, z, w, h, d, mass, restitution, friction float64) *VertexMD2 {
	volume := NewVolumeDetails3d(0, "md2", "thing", x, y, z, w, h, d, mass, restitution, friction)
	v := &VertexMD2{
		volume: volume,
		frames: make([][]*Face, len(cfg.Frames)),
	}
	v.volume.SetBillboard(2.0)

	for frameIdx, cfgFrame := range cfg.Frames {
		frameFaces := make([]*Face, len(cfgFrame.Triangles))
		for triIdx, tri := range cfgFrame.Triangles {
			tag := fmt.Sprintf("%s_%d_%d", "md2", frameIdx, triIdx)
			points := [3]geometry.XYZ{tri[0].Pos, tri[1].Pos, tri[2].Pos}
			face := NewFace(nil, points, tag, anim)
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

// GetVolume returns the Volume instance associated with the VertexMD2 object.
func (v *VertexMD2) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves the vertices corresponding to the current animation frame, determined by the specified tick value.
func (v *VertexMD2) GetVertices(tick uint64) ([]*Face, []*Face, float64) {
	const groupSize = 6.0
	frameFloat := textures.TickGrouped(tick, int(groupSize))
	idxA := int(frameFloat) % len(v.frames)
	idxB := (idxA + 1) % len(v.frames)
	// Parte frazionaria per l'interpolazione fluida
	lerpT := frameFloat - math.Floor(frameFloat)
	return v.frames[idxA], v.frames[idxB], lerpT
}
