package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMultiSprite represents a sprite with multiple views composed of 3D volumes, enabling dynamic angle adjustments.
type VerticesMultiSprite struct {
	currentAction int
	currentAngle  int
	volumes       []*Volume
	viewVolume    *Volume
}

// NewVerticesMultiSprite initializes and returns a VerticesMultiSprite with volumes and faces based on the given configuration.
func NewVerticesMultiSprite(cfg *config.Thing, materials *Materials) *VerticesMultiSprite {
	f := &VerticesMultiSprite{
		volumes:       make([]*Volume, len(cfg.MultiSprite.Materials)),
		currentAction: 0,
		currentAngle:  0,
	}
	for viewIdx, view := range cfg.MultiSprite.Materials {
		baseId := fmt.Sprintf("%s_multisprite_frame_%d", cfg.Id, viewIdx)
		volume := NewVolume(viewIdx, baseId, "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		f.volumes[viewIdx] = volume
		if view != nil && len(view.Frames) > 0 {
			material := materials.GetMaterial(view)
			//cfg.Radius * 2
			faces := f.createFaces(material)
			for _, face := range faces {
				volume.AddFace(face)
			}
		}
		volume.Rebuild()
	}
	f.compute()
	return f
}

// GetVolume retrieves the current active volume associated with the VerticesMultiSprite instance.
func (v *VerticesMultiSprite) GetVolume() *Volume {
	return v.viewVolume
}

// GetEntity returns the physics entity associated with the current view volume of the VerticesMultiSprite.
func (v *VerticesMultiSprite) GetEntity() *physics.Entity {
	return v.viewVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the current view volume's associated physics entity.
func (v *VerticesMultiSprite) GetAABB() *physics.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// GetVertices retrieves the vertices and face count of the current view volume and duplicates, along with a default value.
func (v *VerticesMultiSprite) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	f, c := v.viewVolume.GetFaces()
	return f, c, f, c, 0.0, v.GetBillboard()
}

// SetAction updates the current action index for the sprite if the provided index is within valid bounds.
func (v *VerticesMultiSprite) SetAction(idx int) {
	if idx < 0 || idx >= len(v.volumes) {
		return
	}
	v.currentAction = idx
	v.compute()
}

// GetDisplacement retrieves the bottom-left coordinates (x, y, z) of the entity associated with the current view volume.
func (v *VerticesMultiSprite) GetDisplacement() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetBottomLeft()
}

// GetBillboard returns a constant value of 1.0, typically used to represent a uniform scaling factor for billboards.
func (v *VerticesMultiSprite) GetBillboard() float64 {
	return 1.0
}

// SetThing assigns the specified IThing instance to all volumes in the VerticesMultiSprite object.
func (v *VerticesMultiSprite) SetThing(t IThing) {
	for _, volume := range v.volumes {
		volume.SetThing(t)
	}
}

// compute updates the current view volume based on the current angle, ensuring it is valid or falls back to a default volume.
func (v *VerticesMultiSprite) compute() {
	viewVolume := v.volumes[v.currentAngle]
	if viewVolume == v.viewVolume {
		return
	}
	v.viewVolume = viewVolume
	if v.viewVolume == nil {
		v.viewVolume = v.volumes[0]
	}
}

// createFaces constructs 3D box geometry with optional visual material and returns its set of face objects for rendering and physics.
func (v *VerticesMultiSprite) createFaces(material *textures.Material) []*Face {
	if material == nil {
		return nil
	}
	tex := material.CurrentFrame()
	if tex == nil {
		return nil
	}
	texW, texH := tex.Size()
	scaleW, scaleH := tex.GetScaleFactor()
	width := float64(texW) * scaleW
	height := float64(texH) * scaleH

	halfW := width * 0.5
	halfY := halfW // Profondità dell'AABB fisico

	// Vertici del Box 3D (Fisica)
	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: 0.0}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: 0.0}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: 0.0}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: 0.0}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: height}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: height}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: height}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: height}

	var faces []*Face

	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ, mat *textures.Material) {
		f0 := NewFace([3]geometry.XYZ{vA, vB, vC}, "", mat) // NIL MATERIAL
		f1 := NewFace([3]geometry.XYZ{vA, vC, vD}, "", mat) // NIL MATERIAL
		faces = append(faces, f0, f1)
	}

	addPhysicsQuad(v001, v000, v100, v101, nil) // Front
	addPhysicsQuad(v111, v110, v010, v011, nil) // Back
	addPhysicsQuad(v011, v010, v000, v001, nil) // Left
	addPhysicsQuad(v101, v100, v110, v111, nil) // Right
	addPhysicsQuad(v011, v001, v101, v111, nil) // Top
	addPhysicsQuad(v000, v010, v110, v100, nil) // Bottom

	// 2. PIANO VISIVO (Render)
	// Posizionato ESATTAMENTE al centro (Y=0) con il materiale visibile.
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(t0, "", material)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)

	t1 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
		{X: halfW, Y: 0.0, Z: height},  // TR
	}
	f1 := NewFace(t1, "", material)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)

	faces = append(faces, f0, f1)

	return faces
}
