package volumes

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesSprite represents sprite data backed by a source volume and a view-specific volume.
type VerticesSprite struct {
	volume *Volume
}

// NewVerticesSprite creates a billboard-rendered vertex sprite for thing from cfg and optional sprite material.
func NewVerticesSprite(thing IThing, cfg *config.Thing, materials *materials.Materials) *VerticesSprite {
	w := cfg.Radius * 2
	d := cfg.Height

	var material *textures.Material
	if cfg.Sprite != nil {
		material = materials.GetMaterial(cfg.Sprite.Material)
		if material != nil {
			tex := material.GetCurrentFrame()
			if tex != nil {
				w, d = tex.GetSize()
			}
		}
	}

	halfX := w * 0.5
	halfY := w * 0.5

	volume := NewVolume(0, "material", "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesSprite{
		volume: volume,
	}

	// Costruzione degli 8 vertici del Box (Bottom e Top)
	halfD := d * 0.5

	v000 := geometry.XYZ{X: -halfX, Y: -halfY, Z: -halfD}
	v100 := geometry.XYZ{X: halfX, Y: -halfY, Z: -halfD}
	v110 := geometry.XYZ{X: halfX, Y: halfY, Z: -halfD}
	v010 := geometry.XYZ{X: -halfX, Y: halfY, Z: -halfD}

	v001 := geometry.XYZ{X: -halfX, Y: -halfY, Z: halfD}
	v101 := geometry.XYZ{X: halfX, Y: -halfY, Z: halfD}
	v111 := geometry.XYZ{X: halfX, Y: halfY, Z: halfD}
	v011 := geometry.XYZ{X: -halfX, Y: halfY, Z: halfD}

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

	f.volume.SetThing(thing)

	return f
}

// VerticesGetVolume returns the volume associated with the vertices sprite.
func (v *VerticesSprite) VerticesGetVolume() IVolume {
	return v.volume
}

// VerticesGetEntity returns the physics entity associated with the sprite's volume.
func (v *VerticesSprite) VerticesGetEntity() *physics.Entity {
	return v.volume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box of the sprite's underlying entity.
func (v *VerticesSprite) GetAABB() *aabb.AABB {
	return v.volume.GetEntity().GetAABB()
}

// VerticesCompile updates the sprite's view volume from its current volume.
// It ignores tick and returns false because no geometry is rebuilt.
func (v *VerticesSprite) VerticesCompile(tick uint64) bool {
	return false
}

// VerticesGet returns the view volume faces for both face sets, zero offset, and the render mode.
func (v *VerticesSprite) VerticesGet() (*Faces, *Faces, float64, float64) {
	f := v.volume.GetFaces()
	return f, f, 0.0, RenderModeBillboard
}

// VerticesSetAction applies the vertex action identified by idx.
func (v *VerticesSprite) VerticesSetAction(idx int) {
	//TODO IMPLEMENT
	return
}

// VerticesApplyImpact is a no-op for billboard sprites without articulated sub-parts.
func (v *VerticesSprite) VerticesApplyImpact(fx, fy, fz, px, py, pz float64) {
}

func (v *VerticesSprite) VerticesSetRagdoll() {
}
