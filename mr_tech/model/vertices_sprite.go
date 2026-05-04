package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// VerticesSprite represents a 2D or 3D graphical object tied to a Volume, used for rendering and spatial interactions.
type VerticesSprite struct {
	volume *Volume
}

// NewVerticesSprite creates a new VerticesSprite with the given material, position, dimensions, and physical properties.
func NewVerticesSprite(cfg *config.Thing, pos geometry.XYZ, materials *Materials) *VerticesSprite {
	x := pos.X - cfg.Radius
	y := pos.Y - cfg.Radius
	z := pos.Z
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	d := cfg.Height

	height := h
	width := w
	halfW := width / 2.0
	material := materials.GetMaterial(cfg.Material)
	if material != nil {
		tex := material.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			scaleW, scaleH := tex.GetScaleFactor()
			width = float64(texW) * scaleW
			height = float64(texH) * scaleH
			halfW = width / 2.0
		}
	}

	volume := NewVolumeDetails3d(0, "material", "thing", x, y, z, width, height, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesSprite{volume: volume}

	// Triangolo 0: Top-Left, Bottom-Left, Bottom-Right
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(nil, t0, "", material)
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
	f1 := NewFace(nil, t1, "", material)
	// TL: (0,0), BR: (1,-1), TR: (1,0)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	f.volume.AddFace(f1)

	f.volume.Rebuild()

	return f
}

// GetVolume returns the Volume instance associated with the VerticesSprite.
func (v *VerticesSprite) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves the collection of visible faces for the specified simulation tick.
// The returned faces represent the geometry of the vertex material at the given moment in time.
func (v *VerticesSprite) GetVertices(tick uint64) ([]*Face, int, []*Face, int, float64) {
	f, c := v.volume.GetFaces()
	return f, c, f, c, 0.0
}

// SetAction sets the action index for the VerticesSprite, modifying its behavior or state based on the specified index.
func (v *VerticesSprite) SetAction(idx int) {
	//TODO IMPLEMENT
	return
}

// GetDisplacement retrieves the bottom-left coordinates of the entity associated with the VerticesSprite's Volume.
func (v *VerticesSprite) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetBottomLeft()
}

// GetBillboard retrieves the billboard value associated with the Face instance.
func (v *VerticesSprite) GetBillboard() float64 {
	return 1.0
}

// SetThing assigns an IThing instance to the underlying Volume of the VerticesSprite.
func (v *VerticesSprite) SetThing(t IThing) {
	v.volume.SetThing(t)
}
