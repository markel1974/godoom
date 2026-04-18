package model

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VertexSprite represents a 2D or 3D graphical object tied to a Volume, used for rendering and spatial interactions.
type VertexSprite struct {
	volume *Volume
}

// NewVertexSprite creates a new VertexSprite with the given animation, position, dimensions, and physical properties.
func NewVertexSprite(anim *textures.Animation, x, y, z, w, h, d, mass, restitution, friction float64) *VertexSprite {
	volume := NewVolumeDetails3d(0, "sprite", "thing", x, y, z, w, h, d, mass, restitution, friction)
	f := &VertexSprite{
		volume: volume,
	}
	f.volume.SetBillboard(1.0)
	height := 0.0
	halfW := 0.0
	if anim != nil {
		tex := anim.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			scaleW, scaleH := anim.ScaleFactor()
			width := float64(texW) * scaleW
			height = float64(texH) * scaleH
			halfW = width / 2.0
		}
	}

	t1 := [3]geometry.XYZ{{X: -halfW, Y: height, Z: 0.0}, {X: -halfW, Y: 0.0, Z: 0.0}, {X: halfW, Y: 0.0, Z: 0.0}}
	face0 := NewFace(nil, t1, "", anim)
	face0.SetUV(0.0, 0.0, 0.0, 1.0, 1.0, 1.0)
	face0.LockUV(true)

	t2 := [3]geometry.XYZ{{X: -halfW, Y: height, Z: 0.0}, {X: halfW, Y: 0.0, Z: 0.0}, {X: halfW, Y: height, Z: 0.0}}
	face1 := NewFace(nil, t2, "", anim)
	face1.SetUV(0.0, 0.0, 1.0, 1.0, 1.0, 0.0)
	face1.LockUV(true)

	f.volume.AddFace(face0)
	f.volume.AddFace(face1)
	f.volume.Rebuild()

	return f
}

// GetVolume returns the Volume instance associated with the VertexSprite.
func (v *VertexSprite) GetVolume() *Volume {
	return v.volume
}

// SetAngle modifies the angle of the VertexSprite based on the provided directional changes dx and dy.
func (v *VertexSprite) SetAngle(angle float64) {

}

// GetVertices retrieves the collection of visible faces for the specified simulation tick.
// The returned faces represent the geometry of the vertex sprite at the given moment in time.
func (v *VertexSprite) GetVertices(tick uint64) []*Face {
	return v.volume.GetFaces()
}
