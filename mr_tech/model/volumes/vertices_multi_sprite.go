package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMultiSprite manages multiple sprite volumes for view selection, physics, actions, and rendering state.
type VerticesMultiSprite struct {
	//currentAngle  int
	volumes       []*Volume
	viewVolume    *Volume
	actionVolume  *Volume
	physicsVolume *Volume
	renderMode    float64
	debug         bool
	clampAnim     []bool
}

// NewVerticesMultiSprite creates a multi-sprite vertex set for thing using the configured views and materials.
// It initializes per-view volumes, rendering mode, animation clamps, and binds each volume to thing.
// It panics if a view is nil, has no textures, or face creation fails.
func NewVerticesMultiSprite(thing IThing, cfg *config.Thing, materials *materials.Materials) *VerticesMultiSprite {
	f := &VerticesMultiSprite{
		volumes: make([]*Volume, len(cfg.MultiSprite.Views)),
		//currentAngle:  0,
		renderMode: RenderModeBillboard,
		clampAnim:  make([]bool, len(cfg.MultiSprite.Views)),
		debug:      false,
	}
	if f.debug {
		f.renderMode = RenderModeModel3D
	}
	for viewIdx, view := range cfg.MultiSprite.Views {
		if view == nil {
			panic(fmt.Sprintf("view is nil for viewIdx %d", viewIdx))
		}
		if len(view.Textures) == 0 {
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
	}

	f.actionVolume = f.volumes[0]
	f.physicsVolume = f.volumes[0]
	f.viewVolume = f.actionVolume

	for _, viewIdx := range cfg.MultiSprite.Clamps {
		if viewIdx < len(f.volumes) {
			f.clampAnim[viewIdx] = true
		}
	}

	for _, volume := range f.volumes {
		volume.SetThing(thing)
	}
	return f
}

// VerticesGetVolume returns the currently active action volume.
func (v *VerticesMultiSprite) VerticesGetVolume() *Volume {
	return v.actionVolume
}

// VerticesGetEntity returns the entity associated with the current action volume.
func (v *VerticesMultiSprite) VerticesGetEntity() *physics.Entity {
	return v.actionVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box of the physics volume's entity.
func (v *VerticesMultiSprite) GetAABB() *aabb.AABB {
	//TODO BUG UNA VOLTA INSERITO L'AABB DEVE RIMANERE COSTANTE, DOBBIAMO NOTIFICARE IL CAMBIAMENTO
	return v.physicsVolume.GetEntity().GetAABB()
}

// VerticesCompile updates the view volume to the current action volume.
// It returns false because no rebuild is performed.
func (v *VerticesMultiSprite) VerticesCompile(tick uint64) bool {
	v.viewVolume = v.actionVolume
	return false
}

// VerticesGet returns the current view volume faces twice, a zero value, and the active render mode.
func (v *VerticesMultiSprite) VerticesGet() (*Faces, *Faces, float64, float64) {
	f := v.viewVolume.GetFaces()
	return f, f, 0.0, v.renderMode
}

// VerticesSetAction switches to the action volume at idx and restarts its face material animations.
func (v *VerticesMultiSprite) VerticesSetAction(idx int) {
	if idx < 0 {
		return
	}
	if !v.computeAction(idx, 0) {
		return
	}

	// Restart animation for all faces in the new action volume
	if faces := v.actionVolume.GetFaces(); faces != nil {
		for i := 0; i < faces.Len(); i++ {
			f := faces.At(i)
			if f == nil {
				continue
			}
			if mat := f.GetMaterial(); mat != nil {
				mat.RestartAnim()
				if v.clampAnim[idx] {
					mat.SetClampAnim(true)
				}
			}
		}
	}
}

// VerticesGetCenter returns the x, y, and z coordinates of the center of the first volume's entity.
func (v *VerticesMultiSprite) VerticesGetCenter() (float64, float64, float64) {
	return v.volumes[0].GetEntity().GetCenter()
}

// VerticesApplyImpact is a no-op for multi-sprite views without articulated 3D sub-parts.
func (v *VerticesMultiSprite) VerticesApplyImpact(fx, fy, fz, px, py, pz float64) {
}

// VerticesSetRagdoll is a no-op for multi-sprite volumes without ragdoll state.
func (v *VerticesMultiSprite) VerticesSetRagdoll() {
}

// computeAction selects the action volume for action and reports whether the active volume changed.
// Invalid actions fall back to the default volume; angle is currently unused.
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
	//TODO BUG UNA VOLTA INSERITO L'AABB DEVE RIMANERE COSTANTE, DOBBIAMO NOTIFICARE IL CAMBIAMENTO
	return true
}

// createFaces builds render and physics faces from the material's head frame texture.
// It returns an error when the material or texture is nil.
func (v *VerticesMultiSprite) createFaces(material *textures.Material) ([]*Face, error) {
	if material == nil {
		return nil, fmt.Errorf("material is nil")
	}
	tex := material.GetHeadFrame()
	if tex == nil {
		return nil, fmt.Errorf("texture is nil")
	}
	width, height := tex.GetSize()
	halfW := width * 0.5
	halfY := halfW
	halfH := height * 0.5

	var faces []*Face

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

	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ, tag string, mat *textures.Material) {
		f0Phys := NewFace([3]geometry.XYZ{vA, vB, vC}, tag+"_A", mat) // NIL MATERIAL
		f1Phys := NewFace([3]geometry.XYZ{vA, vC, vD}, tag+"_B", mat) // NIL MATERIAL
		faces = append(faces, f0Phys, f1Phys)
	}

	var debug *textures.Material = nil
	if v.debug {
		debug = material
	}

	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: -halfH}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: -halfH}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: -halfH}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: -halfH}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: halfH}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: halfH}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: halfH}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: halfH}

	addPhysicsQuad(v001, v000, v100, v101, "Front", debug)  // Front
	addPhysicsQuad(v111, v110, v010, v011, "Back", debug)   // Back
	addPhysicsQuad(v011, v010, v000, v001, "Left", debug)   // Left
	addPhysicsQuad(v101, v100, v110, v111, "Right", debug)  // Right
	addPhysicsQuad(v011, v001, v101, v111, "Top", debug)    // Top
	addPhysicsQuad(v000, v010, v110, v100, "Bottom", debug) // Bottom

	return faces, nil
}
