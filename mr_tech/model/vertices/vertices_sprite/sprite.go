package vertices_sprite

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesSprite represents sprite data backed by a source volume and a view-specific volume.
type VerticesSprite struct {
	solver    *collisions.Solver
	ent       *rigid_body.Entity
	cfg       *config.Thing
	materials *materials.Materials
	volume    *volumes.Volume
}

// NewVerticesSprite creates a billboard-rendered vertex sprite for thing from cfg and optional sprite material.
func NewVerticesSprite(cfg *config.Thing, materials *materials.Materials) *VerticesSprite {
	v := &VerticesSprite{
		solver:    nil,
		ent:       nil,
		cfg:       cfg,
		materials: materials,
	}

	w := v.cfg.Radius * 2
	d := v.cfg.Height

	var material *textures.Material
	if v.cfg.Sprite != nil {
		material = v.materials.GetMaterial(v.cfg.Sprite.Material)
		if material != nil {
			tex := material.GetCurrentFrame()
			if tex != nil {
				w, d = tex.GetSize()
			}
		}
	}

	halfX := w * 0.5
	halfY := w * 0.5

	v.volume = volumes.NewVolumeDynamic(0, "material", "thing", v.cfg.Mass, v.cfg.Restitution, v.cfg.Friction, v.cfg.RestitutionCombine, v.cfg.FrictionCombine, v.cfg.GForce)

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

	v.volume.ClearFaces()

	// Helper per aggiungere una faccia (2 triangoli)
	addQuad := func(vA, vB, vC, vD geometry.XYZ) {
		t0 := [3]geometry.XYZ{vA, vB, vC}
		t1 := [3]geometry.XYZ{vA, vC, vD}

		f0 := volumes.NewFace(t0, "", material)
		f1 := volumes.NewFace(t1, "", material)

		f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
		f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)

		f0.LockUV(true)
		f1.LockUV(true)

		v.volume.AddFace(f0)
		v.volume.AddFace(f1)
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

	return v
}

// Setup initializes the sprite volume geometry, material, physics properties, UVs, and entity state.
func (v *VerticesSprite) Setup(thing ithing.IThing) error {
	vol := v.volume
	if err := vol.Setup(thing); err != nil {
		return err
	}
	v.solver = vol.GetSolver()
	v.ent = v.solver.GetEntity()
	vol.Rebuild()
	//vol.SetThing(thing)
	v.ent.LockUpright()
	return nil
}

// GetSolver returns the collision solver associated with the vertices sprite.
func (v *VerticesSprite) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps searches the sprite volume for objects whose AABB overlaps the given object's AABB.
func (v *VerticesSprite) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.volume.QueryOverlaps(object, callback)
}

// GetAABB returns the axis-aligned bounding box of the sprite's underlying entity.
func (v *VerticesSprite) GetAABB() *aabb.AABB {
	return v.ent.GetAABB()
}

// Translate shifts the sprite AABB by the specified displacement.
func (v *VerticesSprite) Translate(dx, dy, dz float64) {
}

// Compile updates the sprite's view volume from its current volume.
// It ignores tick and returns false because no geometry is rebuilt.
func (v *VerticesSprite) Compile(tick uint64) bool {
	return false
}

// GetUIFaces returns the view volume faces for both face sets, zero offset, and the render mode.
func (v *VerticesSprite) GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64) {
	f := v.volume.GetFaces()
	return f, f, 0.0, vertices.RenderModeBillboard
}

// SetAction applies the vertex action identified by idx.
func (v *VerticesSprite) SetAction(idx int) {
	//TODO IMPLEMENT
	return
}

// SetRagdollMode switches the vertices sprite to ragdoll behavior.
func (v *VerticesSprite) SetRagdollMode() {
}

func (v *VerticesSprite) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	return false
}

func (v *VerticesSprite) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	return false
}
