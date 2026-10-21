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
	//currentAngle  int
	volumes       []*Volume
	actionVolume  *Volume
	physicsVolume *Volume
	renderMode    float64
	debug         bool
}

// NewVerticesMultiSprite initializes and returns a VerticesMultiSprite with volumes and faces based on the given configuration.
func NewVerticesMultiSprite(cfg *config.Thing, materials *Materials) *VerticesMultiSprite {
	f := &VerticesMultiSprite{
		volumes: make([]*Volume, len(cfg.MultiSprite.Views)),
		//currentAngle:  0,
		renderMode: RenderModeBillboard,
		debug:      false,
	}
	if f.debug {
		f.renderMode = RenderModeModel3D
	}

	//if len(cfg.MultiSprite.Views) > 1 {
	//	fmt.Println("len(cfg.MultiSprite.Views) > 1")
	//}
	for viewIdx, view := range cfg.MultiSprite.Views {
		if view == nil {
			panic(fmt.Sprintf("view is nil for viewIdx %d", viewIdx))
		}
		if len(view.Frames) == 0 {
			panic(fmt.Sprintf("view Frames is empty for viewIdx %d", viewIdx))
		}
		baseId := fmt.Sprintf("%s_multisprite_anim_%d", cfg.Id, viewIdx)
		thingId := fmt.Sprintf("thing_%s_%d", cfg.Id, viewIdx)
		volume := NewVolume(viewIdx, baseId, thingId, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		material := materials.GetMaterial(view)
		faces, err := f.createFaces(material)
		if err != nil {
			panic(fmt.Sprintf("[error] NewVerticesMultiSprite createFaces error: %s", err.Error()))
		}
		for _, face := range faces {
			volume.AddFace(face)
		}
		volume.Rebuild()
		f.volumes[viewIdx] = volume
		//TEST
		//f.volumes[0] = volume
	}

	f.actionVolume = f.volumes[0]
	f.physicsVolume = f.volumes[0]
	//TEST
	//if len(f.volumes) > 3 {
	//	f.activeVolume = f.volumes[3]
	//}
	return f
}

// GetVolume retrieves the current active volume associated with the VerticesMultiSprite instance.
func (v *VerticesMultiSprite) GetVolume() *Volume {
	return v.physicsVolume
}

// GetEntity returns the physics entity associated with the current view volume of the VerticesMultiSprite.
func (v *VerticesMultiSprite) GetEntity() *physics.Entity {
	return v.physicsVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the current view volume's associated physics entity.
func (v *VerticesMultiSprite) GetAABB() *aabb.AABB {
	return v.physicsVolume.GetEntity().GetAABB()
}

// GetVertices retrieves the vertices and face count of the current view volume and duplicates, along with a default value.
func (v *VerticesMultiSprite) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	f, c := v.actionVolume.GetFaces()
	return f, c, f, c, 0.0, v.GetRenderMode()
}

// SetAction updates the current action index for the sprite if the provided index is within valid bounds.
func (v *VerticesMultiSprite) SetAction(idx int) {
	//return
	v.computeAction(idx, 0)

	// Restart animation for all faces in the new action volume
	if faces, count := v.actionVolume.GetFaces(); faces != nil {
		for i := 0; i < count; i++ {
			f := (*faces)[i]
			if f == nil {
				continue
			}
			if mat := f.GetMaterial(); mat != nil {
				mat.RestartAnim()
				//TODO
				/*
					// Action 3 is death and 4 is extreme death
					if idx == 3 || idx == 4 {
						mat.SetClampAnim(true)
					} else {
						mat.SetClampAnim(false)
					}

				*/
			}
		}
	}
}

// GetVerticesCenter retrieves the bottom-left coordinates (x, y, z) of the entity associated with the current view volume.
func (v *VerticesMultiSprite) GetVerticesCenter() (float64, float64, float64) {
	return v.volumes[0].GetEntity().GetCenter()
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

// compute updates the current view volume based on the current angle, ensuring it is valid or falls back to a default volume.
func (v *VerticesMultiSprite) computeAction(action int, angle int) bool {
	if action < 0 || action >= len(v.volumes) {
		action = 0
	}
	activeVolume := v.volumes[action]
	//activeVolume := v.volumes[v.currentAngle]
	if activeVolume == v.actionVolume {
		return false
	}
	v.actionVolume = activeVolume
	if v.actionVolume == nil {
		v.actionVolume = v.volumes[0]
	}
	return true
}

// createFaces constructs 3D box geometry with optional visual material and returns its set of face objects for rendering and physics.
func (v *VerticesMultiSprite) createFaces(material *textures.Material) ([]*Face, error) {
	if material == nil {
		return nil, fmt.Errorf("material is nil")
	}
	tex := material.HeadFrame()
	if tex == nil {
		return nil, fmt.Errorf("texture is nil")
	}
	texW, texH := tex.Size()
	//scaleW, scaleH := tex.GetScaleFactor()
	width := float64(texW)  // * scaleW
	height := float64(texH) // * scaleH

	halfW := width * 0.5
	halfY := halfW
	halfH := height * 0.5

	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: -halfH}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: -halfH}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: -halfH}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: -halfH}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: halfH}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: halfH}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: halfH}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: halfH}

	var faces []*Face

	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ, tag string, mat *textures.Material) {
		f0 := NewFace([3]geometry.XYZ{vA, vB, vC}, tag+"_A", mat) // NIL MATERIAL
		f1 := NewFace([3]geometry.XYZ{vA, vC, vD}, tag+"_B", mat) // NIL MATERIAL
		faces = append(faces, f0, f1)
	}

	var debug *textures.Material = nil
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
	// TL BL BR
	t0 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: halfH}, {X: -halfW, Y: 0.0, Z: -halfH}, {X: halfW, Y: 0.0, Z: -halfH}}
	f0 := NewFace(t0, "", material)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	faces = append(faces, f0)
	// TL BR TR
	t1 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: halfH}, {X: halfW, Y: 0.0, Z: -halfH}, {X: halfW, Y: 0.0, Z: halfH}}
	f1 := NewFace(t1, "", material)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	faces = append(faces, f1)

	return faces, nil
}
