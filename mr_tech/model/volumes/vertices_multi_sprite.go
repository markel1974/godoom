package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticeMultiSpriteComposed represents an entity composed of multiple sprites linked to a physics entity and volume system.
type VerticeMultiSpriteComposed struct {
	volume *Volume
	entity *rigid_body.Entity
}

// NewVerticeMultiSpriteComposed creates a new VerticeMultiSpriteEntity instance using the provided Volume.
func NewVerticeMultiSpriteComposed(volume *Volume) *VerticeMultiSpriteComposed {
	return &VerticeMultiSpriteComposed{
		entity: volume.GetEntity(),
		volume: volume,
	}
}

// QueryOverlaps checks for overlaps between the entity's volume and the given object, invoking the callback for each match.
func (vm *VerticeMultiSpriteComposed) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	vm.volume.QueryOverlaps(object, callback)
}

// GetEntity returns the physics.Entity associated with the VerticeMultiSpriteEntity.
func (vm *VerticeMultiSpriteComposed) GetEntity() *rigid_body.Entity {
	return vm.entity
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the entity encapsulated by this instance.
func (vm *VerticeMultiSpriteComposed) GetAABB() *aabb.AABB {
	return vm.entity.GetAABB()
}

// GetFaces retrieves the collection of faces associated with the entity's current volume.
func (vm *VerticeMultiSpriteComposed) GetFaces() *Faces {
	return vm.volume.GetFaces()
}

// SetVolume updates the current volume of the entity.
func (vm *VerticeMultiSpriteComposed) SetVolume(activeVolume *Volume) {
	if activeVolume == vm.volume {
		return
	}
	vm.volume = activeVolume
}

// VerticesMultiSprite manages multiple sprite volumes for view selection, physics, actions, and rendering state.
type VerticesMultiSprite struct {
	//currentAngle  int
	volumes       []*Volume
	comp          *VerticeMultiSpriteComposed
	renderMode    float64
	debug         bool
	clampAnim     []bool
	standingHalfH float64
	minX          float64
	minY          float64
	minZ          float64
	maxX          float64
	maxY          float64
	maxZ          float64
}

// NewVerticesMultiSprite creates a multi-sprite vertex set for thing using the configured views and materials.
// It initializes per-view volumes, rendering mode, animation clamps, and binds each volume to thing.
// It panics if a view is nil, has no textures, or face creation fails
func NewVerticesMultiSprite(thing IThing, cfg *config.Thing, materials *materials.Materials) *VerticesMultiSprite {
	f := &VerticesMultiSprite{
		volumes: make([]*Volume, len(cfg.MultiSprite.Views)),
		//currentAngle:  0,
		renderMode: RenderModeBillboard,
		clampAnim:  make([]bool, len(cfg.MultiSprite.Views)),
		debug:      false,
	}
	f.standingHalfH = cfg.Height * 0.5
	if f.standingHalfH <= 0 && len(cfg.MultiSprite.Views) > 0 {
		if mat0 := materials.GetMaterial(cfg.MultiSprite.Views[0]); mat0 != nil {
			if head0 := mat0.GetHeadFrame(); head0 != nil {
				_, h0 := head0.GetSize()
				f.standingHalfH = h0 * 0.5
			}
		}
	}
	if f.standingHalfH <= 0 {
		f.standingHalfH = 28.0
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
		volume := NewVolume(viewIdx, baseId, thingId, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.RestitutionCombine, cfg.FrictionCombine, cfg.GForce)
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

	f.comp = NewVerticeMultiSpriteComposed(f.volumes[0])
	//f.ent = f.volumes[0].entity
	//f.actionVolume = f.volumes[0]

	for _, viewIdx := range cfg.MultiSprite.Clamps {
		if viewIdx < len(f.volumes) {
			f.clampAnim[viewIdx] = true
		}
	}

	for _, volume := range f.volumes {
		volume.SetThing(thing)
		if f.renderMode == RenderModeBillboard {
			volume.GetEntity().LockUpright()
		}
	}

	f.updateBounds()
	f.VerticesCompile(0)
	return f
}

// updateBounds recalculates the active volume's tight local bounds anchored to the standing base.
func (v *VerticesMultiSprite) updateBounds() {
	if v.comp == nil || v.comp.volume == nil {
		return
	}
	w, _, d := v.comp.volume.GetEntity().GetSize()
	halfW := w * 0.5
	v.minX = -halfW
	v.maxX = halfW
	v.minY = -halfW
	v.maxY = halfW
	v.minZ = -v.standingHalfH
	v.maxZ = -v.standingHalfH + d
}

// VerticesGetVolume returns the currently active action volume.
func (v *VerticesMultiSprite) VerticesGetVolume() IVolume {
	return v.comp
}

// GetAABB returns the axis-aligned bounding box of the physics volume's entity.
func (v *VerticesMultiSprite) GetAABB() *aabb.AABB {
	cx, cy, cz := v.comp.GetEntity().GetCenter()
	b := v.comp.GetEntity().GetAABB()
	b.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	return b
}

// VerticesCompile updates the view volume to the current action volume and refreshes the enclosing AABB.
func (v *VerticesMultiSprite) VerticesCompile(tick uint64) bool {
	v.updateBounds()
	cx, cy, cz := v.comp.GetEntity().GetCenter()
	v.comp.GetEntity().GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	return true
}

// VerticesGet returns the current view volume faces twice, a zero value, and the active render mode.
func (v *VerticesMultiSprite) VerticesGet() (*Faces, *Faces, float64, float64) {
	f := v.comp.GetFaces()
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
	if faces := v.comp.GetFaces(); faces != nil {
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
	v.comp.SetVolume(activeVolume)
	v.updateBounds()
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
	zMin := -v.standingHalfH
	zMax := -v.standingHalfH + height

	var faces []*Face

	// PIANO VISIVO (Render) Posizionato ESATTAMENTE ancorato a zMin/zMax con il materiale visibile.
	// TL BL BR
	t0 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: zMax}, {X: -halfW, Y: 0.0, Z: zMin}, {X: halfW, Y: 0.0, Z: zMin}}
	f0 := NewFace(t0, "", material)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	faces = append(faces, f0)
	// TL BR TR
	t1 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: zMax}, {X: halfW, Y: 0.0, Z: zMin}, {X: halfW, Y: 0.0, Z: zMax}}
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

	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: zMin}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: zMin}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: zMin}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: zMin}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: zMax}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: zMax}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: zMax}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: zMax}

	addPhysicsQuad(v001, v000, v100, v101, "Front", debug)  // Front
	addPhysicsQuad(v111, v110, v010, v011, "Back", debug)   // Back
	addPhysicsQuad(v011, v010, v000, v001, "Left", debug)   // Left
	addPhysicsQuad(v101, v100, v110, v111, "Right", debug)  // Right
	addPhysicsQuad(v011, v001, v101, v111, "Top", debug)    // Top
	addPhysicsQuad(v000, v010, v110, v100, "Bottom", debug) // Bottom

	return faces, nil
}
