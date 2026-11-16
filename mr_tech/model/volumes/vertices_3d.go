package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Vertices3D represents a composite animated 3D model with linked parts, volumes, faces, and 11-DoF ragdoll/dismemberment physics state.
type Vertices3D struct {
	solver        *collisions.Solver
	ent           *rigid_body.Entity
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
	compiled      bool
	compiledTick  uint64
	compiledQX    float64
	compiledQY    float64
	compiledQZ    float64
	compiledQW    float64
}

// NewVertices3D creates a Vertices3D instance from a thing configuration and its materials.
// It validates model parts, links, action maps, and initializes face and transform buffers.
func NewVertices3D(cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	v := &Vertices3D{
		solver:        nil,
		links:         articulated_dynamics.NewLinks(),
		joints:        articulated_dynamics.NewJoints(),
		elements:      NewVertices3DElements(cfg, materials),
		actions:       NewVertices3DActions(),
		currentAction: -1,
	}
	configLinks, cErr := v.createConfigLinks(cfg.Model3D)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.links.Setup(configLinks); err != nil {
		panic(err)
	}
	configJoints, cErr := v.createConfigJoints(cfg.Model3D, v.elements)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.joints.Setup(v.elements.vVolume.solver, configJoints, v.links); err != nil {
		panic(err)
	}
	if err := v.actions.Setup(cfg.Model3D, v.elements); err != nil {
		panic(err)
	}

	v.Compile(0)
	if v.minZ <= v.maxZ {
		if baseZ := -(v.minZ + v.maxZ) * 0.5; math.Abs(baseZ) > 1e-6 {
			v.joints.SetBaseOffsetZ(baseZ)
			v.compiled = false
			v.Compile(0)
		}
	}

	v.elements.UpdateEntityBounds()
	return v
}

// Setup initializes model elements, links, joints, and actions, compiles bounds, and returns any setup error.
func (v *Vertices3D) Setup(thing IThing) error {
	if err := v.elements.Setup(thing); err != nil {
		return err
	}
	if err := v.elements.vVolume.Setup(thing); err != nil {
		return err
	}
	v.solver = v.elements.GetSolver()
	v.ent = v.solver.GetEntity()
	return nil
}

// GetSolver returns the collision solver associated with the model.
func (v *Vertices3D) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps checks for overlaps between the entry's active volume and object, invoking callback for each match.
func (v *Vertices3D) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.elements.vVolume.QueryOverlaps(object, callback)
}

// GetAABB returns the bounding box of the associated entity, or nil if no entity is available.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	ent := v.ent
	cx, cy, cz := ent.GetCenter()
	b := ent.GetAABB()
	if v.minX <= v.maxX {
		b.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}
	return b
}

// SetRagdollMode freezes the model and activates ragdoll collapse if it is not already ragdoll.
func (v *Vertices3D) SetRagdollMode() {
	if v.joints.IsRagdoll() {
		return
	}
	ent := v.ent
	v.compiled = false
	ent.LockUpright()
	v.elements.Freeze()
	v.joints.SetRagdollCollapse()
	v.joints.ActivateRagdoll()
}

// SetAction selects action idx and propagates mapped sub-actions to parts in execution order.
func (v *Vertices3D) SetAction(idx int) {
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
	v.compiled = false
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
			el.SetAction(targetAction)
			if el.IsActionClamped(targetAction) {
				enteringRagdoll = true
			}
		} else {
			fmt.Printf("[warning] Vertices3D: part %d has no mapping for action %d, falling back to action 0\n", i, idx)
			el.SetAction(0)
		}
	}
	if enteringRagdoll {
		ent := v.ent
		ent.LockUpright()
		v.joints.ActivateRagdoll()
	}
}

// ApplyImpact routes a localized 3D impact force (fx, fy, fz) at world point (px, py, pz)
// to the closest articulated part, exciting the 5+ joint DoFs along the kinematic chain, jolting ragdoll bodies,
// and triggering dismemberment on critical or post-mortem hits.
func (v *Vertices3D) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	v.compiled = false
	v.joints.ApplyImpact(fx, fy, fz, px, py, pz)
}

// Compile updates all model parts for the tick and writes transformed frame data into render buffers.
// It resolves parent tag links, integrates 11-DoF joint articulations and dismembered limbs, and clears stale faces.
func (v *Vertices3D) Compile(tick uint64) bool {
	ent := v.elements.vVolume.GetSolver().GetEntity()
	cx, cy, cz := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	if v.compiled && v.compiledTick == tick && qx == v.compiledQX && qy == v.compiledQY && qz == v.compiledQZ && qw == v.compiledQW {
		if v.minX <= v.maxX {
			ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
		}
		return false
	}
	v.compiled = true
	v.compiledTick = tick
	v.compiledQX, v.compiledQY, v.compiledQZ, v.compiledQW = qx, qy, qz, qw

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
		prevMinZ := v.minZ
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
		ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
		if v.joints.IsRagdoll() && minZ > prevMinZ+1e-3 && ent.IsSleeping() {
			ent.WakeUp()
		}
	}

	return false
}

// GetUIFaces returns the current face sets, interpolation factor, and render mode.
func (v *Vertices3D) GetUIFaces() (*Faces, *Faces, float64, float64) {
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
