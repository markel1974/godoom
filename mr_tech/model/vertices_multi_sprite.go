package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMultiSprite represents a sprite with multiple views composed of 3D volumes, enabling dynamic angle adjustments.
type VerticesMultiSprite struct {
	currentAction int
	currentAngle  int
	volumes       []*Volume
	viewVolume    *Volume
	renderMode    float64
	debug         bool
}

// NewVerticesMultiSprite initializes and returns a VerticesMultiSprite with volumes and faces based on the given configuration.
func NewVerticesMultiSprite(cfg *config.Thing, materials *Materials) *VerticesMultiSprite {
	f := &VerticesMultiSprite{
		volumes:       make([]*Volume, len(cfg.MultiSprite.Materials)),
		currentAction: 0,
		currentAngle:  0,
		renderMode:    RenderModeBillboard,
		debug:         false,
	}

	if f.debug {
		f.renderMode = RenderModeModel3D
	}

	counter := 0
	for viewIdx, view := range cfg.MultiSprite.Materials {
		baseId := fmt.Sprintf("%s_multisprite_frame_%d", cfg.Id, viewIdx)
		volume := NewVolume(viewIdx, baseId, "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		if view != nil && len(view.Frames) > 0 {
			material := materials.GetMaterial(view)
			faces, err := f.createFaces(material)
			if err != nil {
				fmt.Println("[warning] VerticesMultiSprite createFaces: ", err.Error())
			}
			for _, face := range faces {
				volume.AddFace(face)
			}
			counter++
		}
		volume.Rebuild()
		f.volumes[viewIdx] = volume
	}

	if counter == 0 {
		panic("[error] VerticesMultiSprite: no valid materials found")
	}

	//TODO current angle
	//viewVolume := v.volumes[v.currentAngle]
	f.viewVolume = f.volumes[f.currentAction]

	//fmt.Println("[info] VerticesMultiSprite: created", cfg.Id, counter, "valid volumes")
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
func (v *VerticesMultiSprite) GetAABB() *aabb.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// GetVertices retrieves the vertices and face count of the current view volume and duplicates, along with a default value.
func (v *VerticesMultiSprite) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	f, c := v.viewVolume.GetFaces()
	return f, c, f, c, 0.0, v.GetRenderMode()
}

// SetAction updates the current action index for the sprite if the provided index is within valid bounds.
func (v *VerticesMultiSprite) SetAction(idx int) {
	if v.currentAction == idx {
		return
	}
	if idx < 0 || idx >= len(v.volumes) {
		fmt.Println("[warning] VerticesMultiSprite SetAction index out of bounds:", idx)
		return
	}
	v.currentAction = idx
	//TODO current angle
	//viewVolume := v.volumes[v.currentAngle]
	v.viewVolume = v.volumes[v.currentAction]
	faces, count := v.viewVolume.GetFaces()
	if count == 0 {
		fmt.Println("[warning] VerticesMultiSprite GetVertices empty faces")
		return
	}

	// Restart animation for all faces in the new action volume
	for i := 0; i < count; i++ {
		f := (*faces)[i]
		if f == nil {
			continue
		}
		mat := f.GetMaterial()
		if mat == nil {
			continue
			//fmt.Println("[warning] VerticesMultiSprite Material is nil for face at index:", i)
		}
		mat.RestartAnim()
		//TODO CLAMP FROM CONFIG
		// Action 3 is death and 4 is extreme death
		//if idx == 3 || idx == 4 {
		//	mat.SetClampAnim(true)
		//} else {
		//	mat.SetClampAnim(false)
		//}
	}
}

// GetVerticesCenter retrieves the bottom-left coordinates (x, y, z) of the entity associated with the current view volume.
func (v *VerticesMultiSprite) GetVerticesCenter() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetCenter()
}

// GetRenderMode returns a constant value of 1.0, typically used to represent a uniform scaling factor for renderModes.
func (v *VerticesMultiSprite) GetRenderMode() float64 {
	return v.renderMode
}

// SetThing assigns the specified IThing instance to all volumes in the VerticesMultiSprite object.
func (v *VerticesMultiSprite) SetThing(t IThing) {
	for _, volume := range v.volumes {
		volume.SetThing(t)
	}
}

// createFaces generates 3D faces for the VerticesMultiSprite based on the given material, returning an array of faces or an error.
func (v *VerticesMultiSprite) createFaces(material *textures.Material) ([]*Face, error) {
	if material == nil {
		return nil, fmt.Errorf("material is nil")
	}
	tex := material.CurrentFrame()
	if tex == nil {
		return nil, fmt.Errorf("texture is nil")
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

	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ, tag string, mat *textures.Material) {
		f0 := NewFace([3]geometry.XYZ{vA, vB, vC}, tag+"_A", mat)
		f1 := NewFace([3]geometry.XYZ{vA, vC, vD}, tag+"_B", mat)
		faces = append(faces, f0, f1)
	}

	var debug *textures.Material
	if v.debug {
		debug = material
	}

	addPhysicsQuad(v001, v000, v100, v101, "Front", debug)  // Front
	addPhysicsQuad(v111, v110, v010, v011, "Back", debug)   // Back
	addPhysicsQuad(v011, v010, v000, v001, "Left", debug)   // Left
	addPhysicsQuad(v101, v100, v110, v111, "Right", debug)  // Right
	addPhysicsQuad(v011, v001, v101, v111, "Top", debug)    // Top
	addPhysicsQuad(v000, v010, v110, v100, "Bottom", debug) // Bottom

	// PIANO VISIVO (Render) Posizionato ESATTAMENTE al centro (Y=0) con il materiale visibile.
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

	return faces, nil
}
