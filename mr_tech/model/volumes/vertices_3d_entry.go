package volumes

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Vertices3DEntryComposed represents an entity composed of multiple sprites linked to a physics entity and volume system.
type Vertices3DEntryComposed struct {
	volume *Volume
	solver *collisions.Solver
}

// NewVertices3DEntryComposed creates a new VerticeMultiSpriteEntity instance using the provided Volume.
func NewVertices3DEntryComposed(volume *Volume) *Vertices3DEntryComposed {
	return &Vertices3DEntryComposed{
		solver: volume.GetSolver(),
		volume: volume,
	}
}

// Setup initializes the composed 3D entry volume's collision solver with object.
func (vm *Vertices3DEntryComposed) Setup(object collisions.IObject) error {
	return vm.solver.Setup(object)
}

// QueryOverlaps checks for overlaps between the entity's volume and the given object, invoking the callback for each match.
func (vm *Vertices3DEntryComposed) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	vm.volume.QueryOverlaps(object, callback)
}

// GetSolver returns the collision solver associated with the composed 3D vertices entry.
func (vm *Vertices3DEntryComposed) GetSolver() *collisions.Solver {
	return vm.solver
}

// GetFaces retrieves the collection of faces associated with the entity's current volume.
func (vm *Vertices3DEntryComposed) GetFaces() *Faces {
	return vm.volume.GetFaces()
}

// SetVolume updates the current volume of the entity.
func (vm *Vertices3DEntryComposed) SetVolume(activeVolume *Volume) {
	vm.volume = activeVolume
}

// VAction defines a named animation action with frame bounds and optional clamping behavior.
type VAction struct {
	Name  string
	Start int
	End   int
	Clamp bool
}

// NewVAction creates a vertex action with the given name and frame interval, with clamping disabled.
func NewVAction(name string, start int, end int) *VAction {
	return &VAction{
		Name:  name,
		Start: start,
		End:   end,
		Clamp: false,
	}
}

// Vertices3DEntry stores 3D volume animation state, frame ranges, actions, and interpolation data.
type Vertices3DEntry struct {
	solver        *collisions.Solver
	ent           *rigid_body.Entity
	cfg           *config.Thing
	materials     *materials.Materials
	comp          *Vertices3DEntryComposed
	volumes       []*Volume
	startFrame    int
	endFrame      int
	startTick     uint64
	currentAction int
	actions       []*VAction
	clampAnim     bool
	tagMap        map[string]int
	volA          *Volume
	volB          *Volume
	lerpT         float64
	singleFrame   bool
	lastVolA      *Volume
	groupSize     float64
	salt          uint64
	tick          uint64
	facesA        *Faces
	facesB        *Faces
	frameCenters  []geometry.XYZ
	frameIdxA     int
	frameIdxB     int
	lastRotA      *articulated_dynamics.TagAxis
	lastRotB      *articulated_dynamics.TagAxis
	rot           *articulated_dynamics.TagAxis
	lastQx        float64
	lastQy        float64
	lastQz        float64
	lastQw        float64
	lastRotVolA   *Volume
	lastRotVolB   *Volume
	minX          float64
	minY          float64
	minZ          float64
	maxX          float64
	maxY          float64
	maxZ          float64
}

// NewVertices3DEntry builds a vertex-based 3D entry from thing configuration, frames, tags, actions, and materials.
// It panics if required frame, action interval, or group size configuration is invalid.
func NewVertices3DEntry(cfg *config.Thing, materials *materials.Materials) *Vertices3DEntry {
	v := &Vertices3DEntry{
		solver:    nil,
		cfg:       cfg,
		materials: materials,
	}

	model := v.cfg.Model3DEntry
	if len(model.Frames) <= 0 {
		panic(fmt.Sprintf("no Model3DEntry frames for thing %s", v.cfg.Id))
	}
	if len(model.ActionDefinitions) != len(model.ActionIntervals) {
		panic(fmt.Sprintf("action definitions and intervals must be the same length"))
	}
	if model.GroupSize <= 0 {
		panic(fmt.Sprintf("groupSize must be greater than 0"))
	}

	actions := make([]*VAction, len(model.ActionDefinitions))
	for idx, name := range model.ActionDefinitions {
		nameC := v3dCleanString(name)
		actions[idx] = NewVAction(nameC, model.ActionIntervals[idx][0], model.ActionIntervals[idx][1])
		for _, z := range model.ActionClamp {
			if strings.Contains(nameC, v3dCleanString(z)) {
				actions[idx].Clamp = true
				break
			}
		}
	}

	tagCounter := 0
	v.tagMap = make(map[string]int)
	for _, cfgFrame := range model.Frames {
		for tagName := range cfgFrame.Tags {
			if _, exists := v.tagMap[tagName]; !exists {
				v.tagMap[tagName] = tagCounter
				tagCounter++
			}
		}
	}

	v.volumes = make([]*Volume, len(model.Frames))
	for frameIdx, cfgFrame := range model.Frames {
		baseId := fmt.Sprintf("%s_md1_frame_%d", v.cfg.Id, frameIdx)
		volume := NewVolumeDynamic(frameIdx, baseId, "thing", v.cfg.Mass, v.cfg.Restitution, v.cfg.Friction, v.cfg.RestitutionCombine, v.cfg.FrictionCombine, v.cfg.GForce)
		for triIdx, tri := range cfgFrame.Triangles {
			label := fmt.Sprintf("%s_%d", baseId, triIdx)
			points := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			var material *textures.Material
			if tri.Material != nil {
				material = v.materials.GetMaterial(tri.Material)
			}
			face := NewFace(points, label, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			volume.AddFace(face)
		}
		// Copy tags to volume
		for tagName, tagData := range cfgFrame.Tags {
			volume.SetVertexTag(v.tagMap[tagName], tagData.Origin, tagData.Axis)
		}
		volume.Rebuild()
		v.volumes[frameIdx] = volume
	}

	maxFaces := 0
	for _, vol := range v.volumes {
		if c := vol.GetFaceCount(); c > maxFaces {
			maxFaces = c
		}
	}
	facesA := NewFaces(maxFaces, true)
	facesB := NewFaces(maxFaces, true)
	for j := 0; j < maxFaces; j++ {
		var mat *textures.Material
		var tu, tv [3]float64
		if len(v.volumes) > 0 && v.volumes[0] != nil && j < v.volumes[0].GetFaceCount() {
			src := v.volumes[0].GetFace(j)
			mat = src.material
			tu = src.u
			tv = src.v
		}
		facesA.Set(j, NewFaceEmpty(mat, tu, tv))
		facesB.Set(j, NewFaceEmpty(mat, tu, tv))
	}

	v.groupSize = float64(model.GroupSize)
	v.actions = actions
	v.startFrame = 0
	v.endFrame = len(model.Frames) - 1
	v.currentAction = -1
	v.volA = v.volumes[0]
	v.volB = v.volumes[0]
	v.comp = NewVertices3DEntryComposed(v.volumes[0])
	v.lerpT = 0.0
	v.lastVolA = nil
	v.salt = uint64(rand.Intn(len(model.Frames)))
	v.facesA = facesA
	v.facesB = facesB
	v.rot = articulated_dynamics.NewTagAxisIdentity()
	v.lastQw = 1.0
	v.singleFrame = v.endFrame == 0

	if len(v.actions) > 0 {
		v.SetAction(0)
	}
	v.Compile(0)
	return v
}

// Setup assigns thing to all volumes and initializes the collision solver for it.
func (v *Vertices3DEntry) Setup(thing IThing) error {
	//for _, f := range v.volumes {
	//	f.SetThing(thing)
	//}
	if err := v.comp.Setup(thing); err != nil {
		return err
	}
	v.solver = v.comp.GetSolver()
	v.ent = v.solver.GetEntity()
	return nil
}

// GetSolver returns the collision solver associated with this vertices entry.
func (v *Vertices3DEntry) GetSolver() *collisions.Solver {
	return v.solver
}

// GetAABB returns the axis-aligned bounding box of the entry's entity.
func (v *Vertices3DEntry) GetAABB() *aabb.AABB {
	ent := v.ent
	b := ent.GetAABB()
	if v.singleFrame {
		qx, qy, qz, qw := ent.GetQRotation().Get()
		if qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
			return b
		}
	}
	cx, cy, cz := ent.GetCenter()
	b.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	return b
}

// QueryOverlaps checks for overlaps between the entry's active volume and object, invoking callback for each match.
func (v *Vertices3DEntry) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.comp.QueryOverlaps(object, callback)
}

// SetAction selects the animation action at idx and resets timing for its frame interval.
func (v *Vertices3DEntry) SetAction(idx int) {
	if idx < 0 || idx >= len(v.actions) {
		fmt.Printf("Invalid action index: %d\n", idx)
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	action := v.actions[idx]
	v.startTick = v.tick
	v.startFrame = action.Start
	v.endFrame = action.End
	v.clampAnim = action.Clamp
	v.singleFrame = v.startFrame == v.endFrame
	if v.startFrame >= 0 && v.startFrame < len(v.volumes) {
		v.frameIdxA = v.startFrame
		v.frameIdxB = v.startFrame
		v.volA = v.volumes[v.startFrame]
		v.volB = v.volumes[v.startFrame]
		v.lerpT = 0.0
		v.comp.SetVolume(v.volA)
		if v.frameCenters == nil {
			v.lastVolA = v.volA
		}
	}
}

// AdvanceAnimation updates volA, volB, and lerpT for tick without transforming render faces or entity AABB.
func (v *Vertices3DEntry) AdvanceAnimation(tick uint64) {
	if v.singleFrame {
		v.lerpT = 0.0
		return
	}
	v.tick = tick + v.salt
	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}
	lerpT := 0.0
	var relativeFrameA int
	var relativeFrameB int
	if v.clampAnim {
		frameFloat := 0.0
		if v.tick > v.startTick {
			frameFloat = float64(v.tick-v.startTick) / v.groupSize
		}
		lastAnim := animLength - 1
		if relativeFrameA = int(frameFloat); relativeFrameA >= lastAnim {
			v.singleFrame = true
			relativeFrameA = lastAnim
			relativeFrameB = lastAnim
			lerpT = 0.0
		} else {
			lerpT = frameFloat - math.Floor(frameFloat)
			if relativeFrameB = relativeFrameA + 1; relativeFrameB >= lastAnim {
				relativeFrameB = lastAnim
			}
		}
	} else {
		frameFloat := float64(v.tick) / v.groupSize
		lerpT = frameFloat - math.Floor(frameFloat)
		relativeFrameA = int(frameFloat) % animLength
		relativeFrameB = (int(frameFloat) + 1) % animLength
	}
	v.frameIdxA = v.startFrame + relativeFrameA
	v.frameIdxB = v.startFrame + relativeFrameB
	v.volA = v.volumes[v.frameIdxA]
	v.volB = v.volumes[v.frameIdxB]
	v.lerpT = lerpT
}

// CenterFrames shifts every frame volume of this entry around its own geometric bounding center
// and rebuilds volume face trees and part solver geometry so each part is 6-DoF ready from creation.
func (v *Vertices3DEntry) CenterFrames(partMass, restitution, friction, gForce float64) {
	v.frameCenters = make([]geometry.XYZ, len(v.volumes))
	for fIdx, vol := range v.volumes {
		if vol == nil {
			continue
		}
		faces := vol.GetFaces()
		minX, minY, minZ := faces.GetMin()
		maxX, maxY, maxZ := faces.GetMax()
		mcx := (minX + maxX) * 0.5
		mcy := (minY + maxY) * 0.5
		mcz := (minZ + maxZ) * 0.5
		v.frameCenters[fIdx] = geometry.XYZ{X: mcx, Y: mcy, Z: mcz}
		if math.Abs(mcx) > 1e-6 || math.Abs(mcy) > 1e-6 || math.Abs(mcz) > 1e-6 {
			for j := 0; j < faces.Len(); j++ {
				f := faces.At(j)
				p0, p1, p2 := f.tri.Points[0], f.tri.Points[1], f.tri.Points[2]
				f.tri.Points[0] = geometry.XYZ{X: p0.X - mcx, Y: p0.Y - mcy, Z: p0.Z - mcz}
				f.tri.Points[1] = geometry.XYZ{X: p1.X - mcx, Y: p1.Y - mcy, Z: p1.Z - mcz}
				f.tri.Points[2] = geometry.XYZ{X: p2.X - mcx, Y: p2.Y - mcy, Z: p2.Z - mcz}
				f.Rebuild()
			}
			vol.Rebuild()
		}
	}

	partSolver := collisions.NewSolver(true, partMass, restitution, friction, v.cfg.RestitutionCombine, v.cfg.FrictionCombine, gForce)
	if v.volA != nil {
		faces := v.volA.GetFaces()
		minX, minY, minZ := faces.GetMin()
		w, h, d := faces.GetSize()
		cx, cy, cz := faces.GetCentroid()
		ixx, ixy, ixz, iyy, iyz, izz := faces.GetInertia()
		partSolver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)
	}
	v.comp.solver = partSolver
	v.comp.volume = v.volA
	v.solver = partSolver
	v.ent = partSolver.GetEntity()
	v.lastVolA = v.volA
	v.lastRotA = articulated_dynamics.NewTagAxisIdentity()
	v.lastRotB = articulated_dynamics.NewTagAxisIdentity()
}

// FrameCenterA returns the local geometric centering offset for the active A frame.
func (v *Vertices3DEntry) FrameCenterA() geometry.XYZ {
	if v.frameIdxA >= 0 && v.frameIdxA < len(v.frameCenters) {
		return v.frameCenters[v.frameIdxA]
	}
	return geometry.XYZ{}
}

// FrameCenterB returns the local geometric centering offset for the active B frame.
func (v *Vertices3DEntry) FrameCenterB() geometry.XYZ {
	if v.frameIdxB >= 0 && v.frameIdxB < len(v.frameCenters) {
		return v.frameCenters[v.frameIdxB]
	}
	return geometry.XYZ{}
}

// SyncSolverToCurrentFrame synchronizes the entry's active collision volume and solver bounds/inertia with volA.
func (v *Vertices3DEntry) SyncSolverToCurrentFrame() {
	if v.volA == nil || v.lastVolA == v.volA {
		return
	}
	v.comp.SetVolume(v.volA)
	faces := v.volA.GetFaces()
	minX, minY, minZ := faces.GetMin()
	w, h, d := faces.GetSize()
	cx, cy, cz := faces.GetCentroid()
	ixx, ixy, ixz, iyy, iyz, izz := faces.GetInertia()
	v.comp.solver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)
	v.lastVolA = v.volA
}

// CompileAttached transforms centered facesA/facesB for attached hierarchy rendering and updates local bounds.
func (v *Vertices3DEntry) CompileAttached(rotA, rotB *articulated_dynamics.TagAxis, dOffX, dOffY, dOffZ float64) {
	v.SyncSolverToCurrentFrame()
	if v.volA == nil {
		return
	}
	isIdentA := rotA.IsIdentity()
	isIdentB := rotB.IsIdentity()
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	countA := v.volA.GetFaceCount()
	for j := 0; j < countA; j++ {
		srcA := v.volA.GetFace(j)
		dstA := v.facesA.At(j)
		dstA.material = srcA.material
		dstA.u = srcA.u
		dstA.v = srcA.v
		if isIdentA {
			dstA.CopyPoints(srcA)
		} else {
			rotA.TransformPointsWithMatrix(dstA.tri, srcA.tri, 0, 0, 0)
		}
		minX, minY, minZ, maxX, maxY, maxZ = dstA.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}
	for j := countA; j < v.facesA.Len(); j++ {
		if faceA := v.facesA.At(j); faceA != nil {
			faceA.Zero()
		}
	}

	volB := v.volB
	if volB == nil {
		volB = v.volA
	}
	countB := volB.GetFaceCount()
	for j := 0; j < countB; j++ {
		srcB := volB.GetFace(j)
		dstB := v.facesB.At(j)
		dstB.material = srcB.material
		dstB.u = srcB.u
		dstB.v = srcB.v
		if isIdentB && dOffX == 0 && dOffY == 0 && dOffZ == 0 {
			dstB.CopyPoints(srcB)
		} else {
			rotB.TransformPointsWithMatrix(dstB.tri, srcB.tri, dOffX, dOffY, dOffZ)
		}
		minX, minY, minZ, maxX, maxY, maxZ = dstB.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}
	for j := countB; j < v.facesB.Len(); j++ {
		if faceB := v.facesB.At(j); faceB != nil {
			faceB.Zero()
		}
	}

	if minX > maxX {
		minX, minY, minZ = 0, 0, 0
		maxX, maxY, maxZ = 0, 0, 0
	}
	v.minX, v.minY, v.minZ = minX, minY, minZ
	v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
}

// CompileDetached rotates the frozen centered volA into facesA using the part entity's 6-DoF quaternion.
func (v *Vertices3DEntry) CompileDetached() {
	if v.volA == nil || v.ent == nil {
		return
	}
	ent := v.ent
	q := ent.GetQRotation()
	qx, qy, qz, qw := q.Get()
	if v.lastRotVolA != v.volA || qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
		v.lastRotVolA = v.volA
		v.lastQx, v.lastQy, v.lastQz, v.lastQw = qx, qy, qz, qw
		isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0
		if !isIdent {
			v.rot.SetFromQuaternion(q)
		}
		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
		count := v.volA.GetFaceCount()
		for j := 0; j < count; j++ {
			src := v.volA.GetFace(j)
			dst := v.facesA.At(j)
			dst.material = src.material
			dst.u = src.u
			dst.v = src.v
			if isIdent {
				dst.CopyPoints(src)
			} else {
				v.rot.TransformPointsWithMatrix(dst.tri, src.tri, 0, 0, 0)
			}
			minX, minY, minZ, maxX, maxY, maxZ = dst.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
		}
		for j := count; j < v.facesA.Len(); j++ {
			if faceA := v.facesA.At(j); faceA != nil {
				faceA.Zero()
			}
		}
		if minX > maxX {
			minX, minY, minZ = 0, 0, 0
			maxX, maxY, maxZ = 0, 0, 0
		}
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
	}
	cx, cy, cz := ent.GetCenter()
	ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
}

// Compile updates the active animation frames and interpolation factor for the given tick.
func (v *Vertices3DEntry) Compile(tick uint64) bool {
	if v.singleFrame {
		v.lerpT = 0.0
	} else {
		v.tick = tick + v.salt
		animLength := v.endFrame - v.startFrame + 1
		if animLength <= 0 {
			animLength = 1
		}
		lerpT := 0.0
		var relativeFrameA int
		var relativeFrameB int
		if v.clampAnim {
			frameFloat := 0.0
			if v.tick > v.startTick {
				frameFloat = float64(v.tick-v.startTick) / v.groupSize
			}
			lastAnim := animLength - 1
			if relativeFrameA = int(frameFloat); relativeFrameA >= lastAnim {
				v.singleFrame = true
				relativeFrameA = lastAnim
				relativeFrameB = lastAnim
				lerpT = 0.0 // Fermo sull'ultimo frame
			} else {
				lerpT = frameFloat - math.Floor(frameFloat)
				if relativeFrameB = relativeFrameA + 1; relativeFrameB >= lastAnim {
					relativeFrameB = lastAnim
				}
			}
		} else {
			// TickGrouped computes the floating-point frame index by dividing the tick by the group size.
			frameFloat := float64(v.tick) / v.groupSize
			lerpT = frameFloat - math.Floor(frameFloat)
			relativeFrameA = int(frameFloat) % animLength
			relativeFrameB = (int(frameFloat) + 1) % animLength
		}
		v.volA = v.volumes[v.startFrame+relativeFrameA]
		v.volB = v.volumes[v.startFrame+relativeFrameB]
		v.lerpT = lerpT
	}

	ent := v.comp.GetSolver().GetEntity()
	q := ent.GetQRotation()
	qx, qy, qz, qw := q.Get()
	isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0

	if v.volA != v.lastRotVolA || v.volB != v.lastRotVolB || qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
		if !isIdent {
			v.rot.SetFromQuaternion(q)
		}
		v.lastQx, v.lastQy, v.lastQz, v.lastQw = qx, qy, qz, qw
		v.lastRotVolA = v.volA
		v.lastRotVolB = v.volB

		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

		countA := v.volA.GetFaceCount()
		for j := 0; j < countA; j++ {
			srcA := v.volA.GetFace(j)
			dstA := v.facesA.At(j)
			dstA.material = srcA.material
			dstA.u = srcA.u
			dstA.v = srcA.v
			if isIdent {
				dstA.CopyPoints(srcA)
			} else {
				v.rot.TransformPointsWithMatrix(dstA.tri, srcA.tri, 0, 0, 0)
			}
			minX, minY, minZ, maxX, maxY, maxZ = dstA.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
		}
		for j := countA; j < v.facesA.Len(); j++ {
			if faceA := v.facesA.At(j); faceA != nil {
				faceA.Zero()
			}
		}

		if v.volB != v.volA {
			countB := v.volB.GetFaceCount()
			for j := 0; j < countB; j++ {
				srcB := v.volB.GetFace(j)
				dstB := v.facesB.At(j)
				dstB.material = srcB.material
				dstB.u = srcB.u
				dstB.v = srcB.v
				if isIdent {
					dstB.CopyPoints(srcB)
				} else {
					v.rot.TransformPointsWithMatrix(dstB.tri, srcB.tri, 0, 0, 0)
				}
				minX, minY, minZ, maxX, maxY, maxZ = dstB.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
			}
			for j := countB; j < v.facesB.Len(); j++ {
				if faceB := v.facesB.At(j); faceB != nil {
					faceB.Zero()
				}
			}
		} else {
			countB := countA
			for j := 0; j < countB; j++ {
				srcB := v.volA.GetFace(j)
				dstB := v.facesB.At(j)
				dstB.material = srcB.material
				dstB.u = srcB.u
				dstB.v = srcB.v
				if isIdent {
					dstB.CopyPoints(srcB)
				} else {
					v.rot.TransformPointsWithMatrix(dstB.tri, srcB.tri, 0, 0, 0)
				}
			}
			for j := countB; j < v.facesB.Len(); j++ {
				if faceB := v.facesB.At(j); faceB != nil {
					faceB.Zero()
				}
			}
		}

		if minX > maxX {
			minX, minY, minZ = 0, 0, 0
			maxX, maxY, maxZ = 0, 0, 0
		}
		prevMinZ := v.minZ
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
		if minZ > prevMinZ+1e-3 && ent.IsSleeping() {
			ent.WakeUp()
		}
	}

	if v.lastVolA != v.volA {
		v.comp.SetVolume(v.volA)
		v.lastVolA = v.volA
	}

	cx, cy, cz := ent.GetCenter()
	ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)

	return true
}

// GetUIFaces returns the two face sets rotated by the entity's orientation quaternion, interpolation factor, and model 3D render mode.
func (v *Vertices3DEntry) GetUIFaces() (*Faces, *Faces, float64, float64) {
	return v.facesA, v.facesB, v.lerpT, RenderModeModel3D
}

// GetVolumes returns the current source and target volumes with the interpolation factor between them.
func (v *Vertices3DEntry) GetVolumes() (*Volume, *Volume, float64) {
	return v.volA, v.volB, v.lerpT
}

// ApplyImpact is a no-op for single-part 3D entries (handled by 6-DoF Entity dynamics).
func (v *Vertices3DEntry) ApplyImpact(fx, fy, fz, px, py, pz float64) {
}

// SetRagdollMode switches the vertices entry to ragdoll behavior.
func (v *Vertices3DEntry) SetRagdollMode() {
}

// IsActionClamped reports whether the action at idx is clamped (such as a death animation).
func (v *Vertices3DEntry) IsActionClamped(idx int) bool {
	if idx < 0 || idx >= len(v.actions) || v.actions[idx] == nil {
		return false
	}
	act := v.actions[idx]
	if act.Clamp {
		return true
	}
	return strings.Contains(act.Name, "death") || strings.Contains(act.Name, "die") || strings.Contains(act.Name, "dead")
}

// FreezeCurrentFrame locks this part on its current animation frame (used when dismembered).
func (v *Vertices3DEntry) FreezeCurrentFrame() {
	v.singleFrame = true
	if v.volA != nil {
		v.volB = v.volA
	}
	v.lerpT = 0.0
}

// GetTagIndex returns the index for the named tag, or -1 if the tag is not found.
func (v *Vertices3DEntry) GetTagIndex(name string) int {
	if idx, ok := v.tagMap[name]; ok {
		return idx
	}
	fmt.Printf("[warning] Vertices3DEntry %s: tag not found\n", name)
	return -1
}

// v3dCleanString normalizes s by trimming surrounding whitespace and converting it to lowercase.
func v3dCleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
