package model

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VertexSprite represents a 2D or 3D graphical object tied to a Volume, used for rendering and spatial interactions.
type VertexSprite struct {
	volume *Volume
}

// NewVertexSprite creates a new VertexSprite with the given sprite, position, dimensions, and physical properties.
func NewVertexSprite(anim *textures.Material, x, y, z, w, h, d, mass, restitution, friction float64) *VertexSprite {
	height := h
	width := w
	halfW := width / 2.0
	if anim != nil {
		tex := anim.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			_, scaleW, scaleH := tex.GetScaleFactor()
			width = float64(texW) * scaleW
			height = float64(texH) * scaleH
			halfW = width / 2.0
		}
	}

	volume := NewVolumeDetails3d(0, "sprite", "thing", x, y, z, width, height, d, mass, restitution, friction)
	f := &VertexSprite{volume: volume}
	f.volume.SetBillboard(1.0)

	// Triangolo 0: Top-Left, Bottom-Left, Bottom-Right
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(nil, t0, "", anim)
	// Passiamo V=0 per il top e V=-1 per il bottom (diventerà 1 nel renderer)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	f.volume.AddFace(f0)

	// Triangolo 1: Top-Left, Bottom-Right, Top-Right
	t1 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
		{X: halfW, Y: 0.0, Z: height},  // TR
	}
	f1 := NewFace(nil, t1, "", anim)
	// TL: (0,0), BR: (1,-1), TR: (1,0)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	f.volume.AddFace(f1)

	f.volume.Rebuild()

	return f
}

// GetVolume returns the Volume instance associated with the VertexSprite.
func (v *VertexSprite) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves the collection of visible faces for the specified simulation tick.
// The returned faces represent the geometry of the vertex sprite at the given moment in time.
func (v *VertexSprite) GetVertices(tick uint64) ([]*Face, []*Face, float64) {
	f := v.volume.GetFaces()
	return f, f, 0.0
}
