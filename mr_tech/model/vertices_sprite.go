package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesSprite represents a 2D or 3D graphical object tied to a Volume, used for rendering and spatial interactions.
type VerticesSprite struct {
	volume *Volume
}

// NewVerticesSprite creates a new VerticesSprite with 3D physical geometry.
func NewVerticesSprite(cfg *config.Thing, materials *Materials) *VerticesSprite {
	w := cfg.Radius * 2
	d := cfg.Height

	var material *textures.Material
	if cfg.Sprite != nil {
		material = materials.GetMaterial(cfg.Sprite.Material)
		if material != nil {
			tex := material.CurrentFrame()
			if tex != nil {
				texW, texH := tex.Size()
				scaleW, scaleH := tex.GetScaleFactor()
				w = float64(texW) * scaleW
				d = float64(texH) * scaleH
			}
		}
	}

	halfW := w * 0.5
	halfY := cfg.Radius // Profondità 3D dell'entità sull'asse Y

	volume := NewVolume(0, "material", "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesSprite{volume: volume}

	// Costruzione degli 8 vertici del Box (Bottom e Top)
	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: 0.0}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: 0.0}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: 0.0}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: 0.0}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: d}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: d}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: d}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: d}

	f.volume.ClearFaces()

	// Helper per aggiungere una faccia (2 triangoli)
	addQuad := func(vA, vB, vC, vD geometry.XYZ) {
		t0 := [3]geometry.XYZ{vA, vB, vC}
		t1 := [3]geometry.XYZ{vA, vC, vD}

		f0 := NewFace(t0, "", material)
		f1 := NewFace(t1, "", material)

		f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
		f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)

		f0.LockUV(true)
		f1.LockUV(true)

		f.volume.AddFace(f0)
		f.volume.AddFace(f1)
	}

	// Front (Y = -halfY)
	addQuad(v001, v000, v100, v101)
	// Back (Y = halfY)
	addQuad(v111, v110, v010, v011)
	// Left (X = -halfW)
	addQuad(v011, v010, v000, v001)
	// Right (X = halfW)
	addQuad(v101, v100, v110, v111)
	// Top (Z = d)
	addQuad(v011, v001, v101, v111)
	// Bottom (Z = 0)
	addQuad(v000, v010, v110, v100)

	// Ora Rebuild itera su un volume 3D reale. minY e maxY non sono più 0.
	f.volume.Rebuild()

	return f
}

// GetVolume returns the Volume instance associated with the VerticesSprite.
func (v *VerticesSprite) GetVolume() *Volume {
	return v.volume
}

// GetEntity retrieves the physics.Entity instance associated with the underlying Volume of the VerticesSprite.
func (v *VerticesSprite) GetEntity() *physics.Entity {
	return v.volume.GetEntity()
}

// GetVertices retrieves the collection of visible faces for the specified simulation tick.
// The returned faces represent the geometry of the vertex material at the given moment in time.
func (v *VerticesSprite) GetVertices(tick uint64) ([]*Face, int, []*Face, int, float64) {
	f, c := v.volume.GetFaces()
	return f[:], c, f[:], c, 0.0
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
