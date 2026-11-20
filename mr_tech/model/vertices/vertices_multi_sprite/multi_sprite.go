package vertices_multi_sprite

import (
	"fmt"

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

// MultiSprite manages multiple sprite volumes for view selection, physics, actions, and rendering state.
type MultiSprite struct {
	solver        *collisions.Solver
	ent           *rigid_body.Entity
	cfg           *config.Thing
	materials     *materials.Materials
	volumes       []*volumes.Volume
	comp          *Composer
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
	//currentAngle  int
}

// NewMultiSprite creates a multi-sprite vertex set for thing using the configured views and materials.
// It initializes per-view volumes, rendering mode, animation clamps, and binds each volume to thing.
// It panics if a view is nil, has no textures, or face creation fails
func NewMultiSprite(cfg *config.Thing, materials *materials.Materials) *MultiSprite {
	v := &MultiSprite{
		cfg:       cfg,
		materials: materials,
		solver:    nil,
		volumes:   make([]*volumes.Volume, len(cfg.MultiSprite.Views)),
		//currentAngle:  0,
		renderMode: vertices.RenderModeBillboard,
		clampAnim:  make([]bool, len(cfg.MultiSprite.Views)),
		debug:      false,
	}

	v.standingHalfH = 0
	if len(v.cfg.MultiSprite.Views) > 0 {
		if mat0 := v.materials.GetMaterial(v.cfg.MultiSprite.Views[0]); mat0 != nil {
			if head0 := mat0.GetHeadFrame(); head0 != nil {
				_, h0 := head0.GetSize()
				v.standingHalfH = h0 * 0.5
			}
		}
	}
	if v.standingHalfH <= 0 {
		v.standingHalfH = v.cfg.Height * 0.5
	}
	if v.standingHalfH <= 0 {
		v.standingHalfH = 28.0
	}
	if v.debug {
		v.renderMode = vertices.RenderModeModel3D
	}
	for viewIdx, view := range v.cfg.MultiSprite.Views {
		if view == nil {
			panic(fmt.Sprintf("view is nil for viewIdx %d", viewIdx))
		}
		if len(view.Textures) == 0 {
			panic(fmt.Sprintf("view Frames is empty for viewIdx %d", viewIdx))
		}
		baseId := fmt.Sprintf("%s_multisprite_anim_%d", v.cfg.Id, viewIdx)
		thingId := fmt.Sprintf("thing_%s_%d", v.cfg.Id, viewIdx)
		volume := volumes.NewVolumeDynamic(viewIdx, baseId, thingId, v.cfg.Mass, v.cfg.Restitution, v.cfg.Friction, v.cfg.RestitutionCombine, v.cfg.FrictionCombine, v.cfg.GForce)
		mat := v.materials.GetMaterial(view)
		faces, err := v.createFaces(mat)
		if err != nil {
			panic(fmt.Sprintf("[error] NewVerticesMultiSprite createFaces error: %s", err.Error()))
		}
		for _, face := range faces {
			volume.AddFace(face)
		}
		volume.Rebuild()
		v.volumes[viewIdx] = volume
	}

	v.comp = NewComposer(v.volumes[0])
	v.solver = v.comp.GetSolver()
	v.ent = v.solver.GetEntity()

	for _, viewIdx := range v.cfg.MultiSprite.Clamps {
		if viewIdx < len(v.volumes) {
			v.clampAnim[viewIdx] = true
		}
	}

	return v
}

// Setup assigns the thing to all volumes, refreshes bounds and compiled state, and initializes collision solving.
func (v *MultiSprite) Setup(thing ithing.IThing) error {
	for _, volume := range v.volumes {
		//volume.SetThing(thing)
		if v.renderMode == vertices.RenderModeBillboard {
			volume.GetSolver().GetEntity().LockUpright()
		}
	}

	if err := v.comp.Setup(thing); err != nil {
		return err
	}
	v.solver = v.comp.GetSolver()
	v.ent = v.solver.GetEntity()
	v.Compile(0)
	return nil
}

// GetSolver returns the collision solver associated with the multi-sprite.
func (v *MultiSprite) GetSolver() *collisions.Solver {
	return v.solver
}

// updateBounds recalculates the active volume's tight local bounds anchored to the standing base.
func (v *MultiSprite) updateBounds() {
	ent := v.ent
	width, _, height := ent.GetSize()
	halfW := width * 0.5
	v.minX = -halfW
	v.maxX = halfW
	v.minY = -halfW
	v.maxY = halfW
	v.minZ = -v.standingHalfH
	v.maxZ = -v.standingHalfH + height
}

// QueryOverlaps checks for overlaps between the entry's active volume and object, invoking callback for each match.
func (v *MultiSprite) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.comp.QueryOverlaps(object, callback)
}

// GetAABB returns the axis-aligned bounding box of the physics volume's entity.
func (v *MultiSprite) GetAABB() *aabb.AABB {
	if v.ent == nil {
		return nil
	}
	return v.ent.GetAABB()
}

// Compile updates the view volume to the current action volume and refreshes the enclosing AABB.
func (v *MultiSprite) Compile(tick uint64) bool {
	ent := v.ent
	v.updateBounds()
	cx, cy, cz := ent.GetCenter()
	ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	return true
}

// GetUIFaces returns the current view volume faces twice, a zero value, and the active render mode.
func (v *MultiSprite) GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64) {
	f := v.comp.GetFaces()
	return f, f, 0.0, v.renderMode
}

// SetAction switches to the action volume at idx and restarts its face material animations.
func (v *MultiSprite) SetAction(idx int) {
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

// SetRagdollMode is a no-op for multi-sprite volumes without ragdoll state.
func (v *MultiSprite) SetRagdollMode() {
}

// computeAction selects the action volume for action and reports whether the active volume changed.
// Invalid actions fall back to the default volume; angle is currently unused.
func (v *MultiSprite) computeAction(action int, angle int) bool {
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
func (v *MultiSprite) createFaces(material *textures.Material) ([]*volumes.Face, error) {
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

	var faces []*volumes.Face

	// PIANO VISIVO (Render) Posizionato ESATTAMENTE ancorato a zMin/zMax con il materiale visibile.
	// TL BL BR
	t0 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: zMax}, {X: -halfW, Y: 0.0, Z: zMin}, {X: halfW, Y: 0.0, Z: zMin}}
	f0 := volumes.NewFace(t0, "", material)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	faces = append(faces, f0)
	// TL BR TR
	t1 := [3]geometry.XYZ{{X: -halfW, Y: 0.0, Z: zMax}, {X: halfW, Y: 0.0, Z: zMin}, {X: halfW, Y: 0.0, Z: zMax}}
	f1 := volumes.NewFace(t1, "", material)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	faces = append(faces, f1)

	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ, tag string, mat *textures.Material) {
		f0Phys := volumes.NewFace([3]geometry.XYZ{vA, vB, vC}, tag+"_A", mat) // NIL MATERIAL
		f1Phys := volumes.NewFace([3]geometry.XYZ{vA, vC, vD}, tag+"_B", mat) // NIL MATERIAL
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

func (v *MultiSprite) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	return false
}

func (v *MultiSprite) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	return false
}
