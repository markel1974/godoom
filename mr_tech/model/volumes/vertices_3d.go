package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Vertices3D represents a composite animated 3D model with linked parts, volumes, faces, and 11-DoF ragdoll/dismemberment physics state.
type Vertices3D struct {
	actions       *Vertices3DActions
	currentAction int
	links         *Vertices3DLinks
	elements      *Vertices3DElements
	joints        *Vertices3DJoints
}

// NewVertices3D creates a Vertices3D instance from a thing configuration and its materials.
// It validates model parts, links, action maps, and initializes face and transform buffers.
func NewVertices3D(thing IThing, cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	v := &Vertices3D{
		elements:      NewVertices3DElements(),
		links:         NewVertices3DLinks(),
		joints:        NewVertices3DJoints(),
		actions:       NewVertices3DActions(),
		currentAction: -1,
	}
	if err := v.elements.Setup(thing, cfg, materials); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}
	if err := v.links.Setup(cfg.Model3D, v.elements); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}
	if err := v.joints.Setup(cfg.Model3D, v.elements, v.links); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}
	if err := v.actions.Setup(cfg.Model3D, v.elements); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}
	v.VerticesCompile(0)

	numParts := v.elements.Len()
	if numParts > 1 {
		v.elements.Rebuild(thing, cfg)
	}

	return v
}

// VerticesGetVolume returns the composite view volume of the model, or nil if unavailable.
func (v *Vertices3D) VerticesGetVolume() *Volume {
	return v.elements.viewVolume
}

// VerticesGetEntity returns the physics entity for the composite model, or the first element's entity if unset.
func (v *Vertices3D) VerticesGetEntity() *physics.Entity {
	return v.elements.entity
}

// GetAABB returns the bounding box of the associated entity, or nil if no entity is available.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	if e := v.VerticesGetEntity(); e != nil {
		return e.GetAABB()
	}
	return nil
}

// VerticesSetAction selects action idx and propagates mapped sub-actions to parts in execution order.
// Passing idx == -2 triggers procedural ragdoll death and collapse when no baked death animation exists.
func (v *Vertices3D) VerticesSetAction(idx int) {
	if idx == -2 {
		if v.currentAction == idx && v.joints.IsRagdoll() {
			return
		}
		v.currentAction = idx
		v.elements.Freeze()
		v.joints.SetRagdollCollapse()
		v.joints.ActivateRagdoll()
		return
	}
	if idx < 0 {
		return
	}
	if v.currentAction == idx {
		return
	}
	if v.joints.IsRagdoll() {
		return
	}
	v.currentAction = idx
	enteringRagdoll := false
	for _, i := range v.links.executionOrder {
		el := v.elements.GetEntry(i)
		if el == nil {
			continue
		}
		if i > 0 && v.joints.IsPartDetached(i) {
			continue
		}
		if targetAction, ok := v.actions.Get(i, idx); ok {
			el.VerticesSetAction(targetAction)
			if el.IsActionClamped(targetAction) {
				enteringRagdoll = true
			}
		} else {
			fmt.Printf("[warning] Vertices3D: part %d has no mapping for action %d, falling back to action 0\n", i, idx)
			el.VerticesSetAction(0)
		}
	}
	if enteringRagdoll {
		v.joints.ActivateRagdoll()
	}
}

// VerticesApplyImpact routes a localized 3D impact force (fx, fy, fz) at world point (px, py, pz)
// to the closest articulated part, exciting the 5+ joint DoFs along the kinematic chain, jolting ragdoll bodies,
// and triggering dismemberment on critical or post-mortem hits.
func (v *Vertices3D) VerticesApplyImpact(fx, fy, fz, px, py, pz float64) {
	v.joints.ApplyImpact(fx, fy, fz, px, py, pz)
}

// VerticesGetCenter returns the center coordinates of the composite entity, or zero coordinates if none is available.
func (v *Vertices3D) VerticesGetCenter() (float64, float64, float64) {
	return v.elements.GetCenter()
}

// VerticesCompile updates all model parts for the tick and writes transformed frame data into render buffers.
// It resolves parent tag links, integrates 11-DoF joint articulations and dismembered limbs, and clears stale faces.
func (v *Vertices3D) VerticesCompile(tick uint64) bool {
	v.joints.StepJoints(tick)
	cx, cy, cz := v.VerticesGetCenter()

	for _, i := range v.links.executionOrder {
		v.elements.Bind(i, tick)

		v.joints.Compile(i, cx, cy, cz)
		origAX, origAY, origAZ := v.joints.GetOriginsA(i)
		origBX, origBY, origBZ := v.joints.GetOriginsB(i)
		rotA := v.joints.GetRotationA(i)
		rotB := v.joints.GetRotationB(i)

		v.elements.Compile(i, origAX, origAY, origAZ, origBX, origBY, origBZ, rotA, rotB)
	}
	return false
}

// VerticesGet returns the current face sets, interpolation factor, and render mode.
func (v *Vertices3D) VerticesGet() (*Faces, *Faces, float64, float64) {
	return v.elements.facesA, v.elements.facesB, v.elements.lerpT, RenderModeModel3D
}
