package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Vertices3D represents a composite animated 3D model with linked parts, volumes, faces, and 11-DoF ragdoll/dismemberment physics state.
type Vertices3D struct {
	actions       *Vertices3DActions
	currentAction int
	links         *articulated_dynamics.Links
	elements      *Vertices3DElements
	joints        *articulated_dynamics.Joints
	minX          float64
	minY          float64
	minZ          float64
	maxX          float64
	maxY          float64
	maxZ          float64
}

// NewVertices3D creates a Vertices3D instance from a thing configuration and its materials.
// It validates model parts, links, action maps, and initializes face and transform buffers.
func NewVertices3D(thing IThing, cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	v := &Vertices3D{
		links:         articulated_dynamics.NewLinks(),
		joints:        articulated_dynamics.NewJoints(),
		elements:      NewVertices3DElements(),
		actions:       NewVertices3DActions(),
		currentAction: -1,
	}
	if err := v.elements.Setup(thing, cfg, materials); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}

	configLinks, cErr := v.createConfigLinks(cfg.Model3D)
	if cErr != nil {
		panic("Error setting up config links elements: " + cErr.Error())
	}
	if err := v.links.Setup(configLinks); err != nil {
		panic("Error setting up links elements: " + err.Error())
	}

	configJoints, cErr := v.createConfigJoints(cfg.Model3D, v.elements)
	if cErr != nil {
		panic("Error setting up config joints elements: " + cErr.Error())
	}
	if err := v.joints.Setup(v.elements.vVolume.entity, configJoints, v.links); err != nil {
		panic("Error setting up joints elements: " + err.Error())
	}
	if err := v.actions.Setup(cfg.Model3D, v.elements); err != nil {
		panic("Error setting up vertices elements: " + err.Error())
	}

	v.VerticesCompile(0)

	v.elements.UpdateEntityBounds()

	return v
}

// VerticesGetVolume returns the composite view volume of the model, or nil if unavailable.
func (v *Vertices3D) VerticesGetVolume() IVolume {
	return v.elements.vVolume
}

// VerticesGetEntity returns the physics entity for the composite model, or the first element's entity if unset.
func (v *Vertices3D) VerticesGetEntity() *rigid_body.Entity {
	return v.elements.vVolume.entity
}

// GetAABB returns the bounding box of the associated entity, or nil if no entity is available.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	ent := v.elements.vVolume.entity
	cx, cy, cz := ent.GetCenter()
	b := ent.GetAABB()
	if v.minX <= v.maxX {
		b.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}
	return b
}

// VerticesSetRagdoll freezes the model and activates ragdoll collapse if it is not already ragdoll.
func (v *Vertices3D) VerticesSetRagdoll() {
	if v.joints.IsRagdoll() {
		return
	}
	v.elements.Freeze()
	v.joints.SetRagdollCollapse()
	v.joints.ActivateRagdoll()
}

// VerticesSetAction selects action idx and propagates mapped sub-actions to parts in execution order.
func (v *Vertices3D) VerticesSetAction(idx int) {
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
	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
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

// VerticesCompile updates all model parts for the tick and writes transformed frame data into render buffers.
// It resolves parent tag links, integrates 11-DoF joint articulations and dismembered limbs, and clears stale faces.
func (v *Vertices3D) VerticesCompile(tick uint64) bool {
	ent := v.elements.vVolume.entity

	cx, cy, cz := ent.GetCenter()
	v.joints.StepJoints(tick)

	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)

		v.elements.Bind(i, tick)

		if i > 0 && v.joints.IsDetached(i) {
			v.joints.CompileDetached(i)
		} else {
			pIdx := v.links.GetParent(i)
			if pIdx < 0 || pIdx == i {
				v.joints.Restore(i, ent.GetQRotation())
			} else {
				volA := v.elements.volsA[pIdx]
				volB := v.elements.volsB[pIdx]
				if volA == nil || volB == nil {
					v.joints.Restore(i, ent.GetQRotation())
				} else {
					tagIdx := v.links.GetTagIndex(i)
					if tagIdx < 0 {
						fmt.Printf("[warning] Vertices3D: missing tag %q on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links.GetTag(i), pIdx, i)
						v.joints.Reset(i, pIdx)
					} else {
						tagAxisA := volA.GetVertexTag(tagIdx)
						tagAxisB := volB.GetVertexTag(tagIdx)
						v.joints.CompileNormal(i, pIdx, tagAxisA, tagAxisB)
					}
				}
			}
		}

		origAX, origAY, origAZ := v.joints.GetOriginsA(i)
		origBX, origBY, origBZ := v.joints.GetOriginsB(i)
		rotA := v.joints.GetRotationA(i)
		rotB := v.joints.GetRotationB(i)

		pMinX, pMinY, pMinZ, pMaxX, pMaxY, pMaxZ := v.elements.Compile(i, origAX, origAY, origAZ, origBX, origBY, origBZ, rotA, rotB)
		if !v.joints.IsPartDetached(i) {
			if pMinX < minX {
				minX = pMinX
			}
			if pMinY < minY {
				minY = pMinY
			}
			if pMinZ < minZ {
				minZ = pMinZ
			}
			if pMaxX > maxX {
				maxX = pMaxX
			}
			if pMaxY > maxY {
				maxY = pMaxY
			}
			if pMaxZ > maxZ {
				maxZ = pMaxZ
			}
		}
	}

	if minX <= maxX {
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
		ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}

	return false
}

// VerticesGet returns the current face sets, interpolation factor, and render mode.
func (v *Vertices3D) VerticesGet() (*Faces, *Faces, float64, float64) {
	return v.elements.facesA, v.elements.facesB, v.elements.lerpT, RenderModeModel3D
}

func (v *Vertices3D) createConfigJoints(cfg *config.Model3D, elements *Vertices3DElements) ([]*articulated_dynamics.Joint, error) {
	numParts := len(cfg.Parts)
	joints := make([]*articulated_dynamics.Joint, numParts)
	for i := 0; i < numParts; i++ {
		element := elements.GetEntry(i)
		if element == nil {
			return nil, fmt.Errorf("entry %d is nil", i)
		}
		// TODO FROM CONFIG default partRadius
		j := articulated_dynamics.NewJoint2(element, 8.0)
		joints[i] = j

		if vol := element.volumes[0]; vol != nil {
			if fc := vol.GetFaceCount(); fc > 0 {
				minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
				maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
				for fIdx := 0; fIdx < fc; fIdx++ {
					f := vol.GetFace(fIdx)
					minX, minY, minZ, maxX, maxY, maxZ = f.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
				}
				j.SetLocalCenter(minX, minY, minZ, maxX, maxY, maxZ)
			}
		}
		if i == 0 || v.links.GetParent(i) < 0 {
			j.SetDofScale(0, 0, 0)
			j.SetMaxAngleAlive(0)
			j.SetMaxAngleRagdoll(0)
			j.SetStiffness(80.0)
			j.SetDamping(10.0)
			j.SetCanDetach(false)
		} else {
			tag := v.links.GetTag(i)
			z := cfg.Joints.Get(tag)
			j.SetDofScale(z.DofScale.X, z.DofScale.Y, z.DofScale.Z)
			j.SetMaxAngleAlive(z.MaxAngleAlive)
			j.SetMaxAngleRagdoll(z.MaxAngleRagdoll)
			j.SetStiffness(z.Stiffness)
			j.SetDamping(z.Damping)
			j.SetDetachThreshold(z.DetachThreshold)
			j.SetCanDetach(z.CanDetach)
		}
	}
	return joints, nil
}

// createConfigLinks creates and initializes physics.Link objects for each part in the given config.Model3D structure.
// Returns the created links or an error if initialization fails.
func (v *Vertices3D) createConfigLinks(cfg *config.Model3D) ([]*articulated_dynamics.Link, error) {
	numParts := len(cfg.Parts)
	links := make([]*articulated_dynamics.Link, numParts)
	for i := 0; i < numParts; i++ {
		parent, tag := articulated_dynamics.LinkInvalidIndex, ""
		if i < len(cfg.Links) {
			parent, tag = cfg.Links[i].Parent, cfg.Links[i].Tag
		}
		link := articulated_dynamics.NewLink(parent, tag)
		links[i] = link
		lParent := link.GetParent()
		if lParent < 0 {
			continue
		}
		element := v.elements.GetEntry(lParent)
		if element == nil {
			fmt.Printf("[warning] Vertices3D: parent %d not found for part %d, inheriting parent origin and rotation without tag offset\n", lParent, i)
			continue
		}
		lTag := link.GetTag()
		tagIdx := element.GetTagIndex(lTag)
		if tagIdx < 0 {
			fmt.Printf("[warning] Vertices3D: tag %q not found on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", lTag, lParent, i)
			continue
		}
		link.SetTagIndex(tagIdx)
		for frameIdx, vol := range element.volumes {
			if vol == nil {
				fmt.Printf("[warning] Vertices3D: volume not found in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", frameIdx, lParent, i)
				continue
			}
			if !vol.HasVertexTag(tagIdx) {
				fmt.Printf("[warning] Vertices3D: tag %q (idx %d) missing in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", lTag, tagIdx, frameIdx, lParent, i)
				continue
			}
		}
	}
	return links, nil
}
