package vertices_element_3d

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type frameSlot struct {
	frameIdx int
	seq      uint64
	vol      *volumes.Volume
}

// Element3D stores runtime state for a 3D vertex-based entity with animation, volumes, and physics data.
type Element3D struct {
	solver             *collisions.Solver
	ent                *rigid_body.Entity
	restitutionCombine int
	frictionCombine    int
	materials          *materials.Materials
	comp               *Composer
	vols               *volumes.Volumes
	tpl                *ElementTemplate
	sharedTree         *aabb.Tree
	slots              []frameSlot
	slotSeq            uint64
	numFrames          int
	materializedAll    bool
	startFrame         int
	endFrame           int
	startTick          uint64
	currentAction      int
	actions            []*Action
	clampAnim          bool
	tagMap             map[string]int
	volA               *volumes.Volume
	volB               *volumes.Volume
	lerpT              float64
	singleFrame        bool
	lastVolA           *volumes.Volume
	lastSolverFrameA   int
	groupSize          float64
	salt               uint64
	tick               uint64
	facesA             *volumes.Faces
	facesB             *volumes.Faces
	frameIdxA          int
	frameIdxB          int
	rot                *articulated_dynamics.TagAxis
	lastQx             float64
	lastQy             float64
	lastQz             float64
	lastQw             float64
	lastRotVolA        *volumes.Volume
	lastRotVolB        *volumes.Volume
	lastRotFrameA      int
	lastRotFrameB      int
	minX               float64
	minY               float64
	minZ               float64
	maxX               float64
	maxY               float64
	maxZ               float64
	treeVol            *volumes.Volume
	treeFrameIdx       int
	treeVolVer         uint64
	treeRebuilds       uint64
	lastPoseGen        uint64
}

const maxCachedElementSlots = 4

// NewElement3D creates and initializes a 3D vertex entry from thing model frames, actions, tags, and materials.
// NewElement3D panics if required frame, action, or group size configuration is invalid.
func NewElement3D(cfg *config.Thing, materials *materials.Materials) *Element3D {
	model := cfg.Element3D
	if len(model.Frames) <= 0 {
		panic(fmt.Sprintf("no Model3DEntry frames for thing %s", cfg.Id))
	}
	if len(model.ActionDefinitions) != len(model.ActionIntervals) {
		panic(fmt.Sprintf("action definitions and intervals must be the same length"))
	}
	if model.GroupSize <= 0 {
		panic(fmt.Sprintf("groupSize must be greater than 0"))
	}

	actions := make([]*Action, len(model.ActionDefinitions))
	for idx, name := range model.ActionDefinitions {
		nameC := CleanString(name)
		actions[idx] = NewAction(nameC, model.ActionIntervals[idx][0], model.ActionIntervals[idx][1])
		for _, z := range model.ActionClamp {
			if strings.Contains(nameC, CleanString(z)) {
				actions[idx].Clamp = true
				break
			}
		}
	}

	maxFaces := 0
	for _, cfgFrame := range model.Frames {
		if len(cfgFrame.Triangles) > maxFaces {
			maxFaces = len(cfgFrame.Triangles)
		}
	}
	treeCap := uint(1)
	if maxFaces > 0 {
		// Pre-size a single shared AABB tree for 2*maxFaces nodes so frame rebuilds avoid heap growth.
		treeCap = uint(maxFaces * 2)
	}
	sharedTree := aabb.NewTree(treeCap, 0.0)
	sharedSolver := collisions.NewSolver(true, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.RestitutionCombine, cfg.FrictionCombine, cfg.GForce)

	numFrames := len(model.Frames)
	v := &Element3D{
		solver:             sharedSolver,
		restitutionCombine: cfg.RestitutionCombine,
		frictionCombine:    cfg.FrictionCombine,
		materials:          materials,
		sharedTree:         sharedTree,
		numFrames:          numFrames,
		lastSolverFrameA:   -1,
		lastRotFrameA:      -1,
		lastRotFrameB:      -1,
		treeFrameIdx:       -1,
		actions:            actions,
	}

	var refVol *volumes.Volume
	if numFrames == 1 {
		cfgFrame := model.Frames[0]
		v.tagMap = model.BuildTagMap()
		v.vols = volumes.NewVolumes(1)
		v.materializedAll = true
		volume := volumes.NewVolumeDynamicSharedSized(0, "", "thing", sharedTree, sharedSolver, maxFaces)
		for _, tri := range cfgFrame.Triangles {
			points := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			var material *textures.Material
			if tri.Material != nil {
				material = v.materials.GetMaterial(tri.Material)
			}
			face := volumes.NewFace(points, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			if tri.Vertices[0].HasNormal || tri.Vertices[1].HasNormal || tri.Vertices[2].HasNormal {
				n0X, n0Y, n0Z := tri.Vertices[0].Normal.ToFloat64()
				n1X, n1Y, n1Z := tri.Vertices[1].Normal.ToFloat64()
				n2X, n2Y, n2Z := tri.Vertices[2].Normal.ToFloat64()
				face.SetVertexNormals(n0X, n0Y, n0Z, n1X, n1Y, n1Z, n2X, n2Y, n2Z)
			}
			volume.AddFace(face)
		}
		for tagName, tagData := range cfgFrame.Tags {
			volume.SetVertexTag(v.tagMap[tagName], tagData.Origin, tagData.Axis)
		}
		volume.RebuildGeometry()
		v.vols.Set(0, volume)
		v.vols.SetAllowTessellate(true)
		refVol = volume
	} else {
		tpl := v.materials.GetOrBuildTemplate(model, func() any {
			return NewElementTemplate(model, v.materials)
		}).(*ElementTemplate)
		v.tpl = tpl
		v.tagMap = tpl.tagMap
		v.vols = volumes.NewVolumes(0)
		v.materializedAll = false

		slotCount := min(numFrames, maxCachedElementSlots)
		v.slots = make([]frameSlot, slotCount)
		for i := range slotCount {
			v.slots[i] = frameSlot{
				frameIdx: -1,
				vol:      tpl.AllocateSlotVolume(-1, sharedTree, sharedSolver, false),
			}
		}

		refIdx := v.findReferenceFrameIndex()
		if refSolverVol := v.getFrameVolume(refIdx, -1); refSolverVol != nil {
			// Restore shared solver extents to the standing/reference pose rather than leaving the last frame (often death).
			rebuildSolverFromVolume(sharedSolver, refSolverVol)
		}
		refVol = v.getFrameVolume(0, -1)
	}

	facesA := volumes.NewFacesSized(nil, maxFaces, true)
	facesB := volumes.NewFacesSized(nil, maxFaces, true)
	facesA.SetAllowTessellate(false)
	facesB.SetAllowTessellate(false)

	for j := range maxFaces {
		var mat *textures.Material
		var tu, tv [3]float64
		if refVol != nil && j < refVol.GetFaceCount() {
			src := refVol.GetFace(j)
			mat = src.GetMaterial()
			tu, tv = src.GetUV()
		}
		facesA.Set(j, volumes.NewFaceEmpty(mat, tu, tv))
		facesB.Set(j, volumes.NewFaceEmpty(mat, tu, tv))
	}

	v.groupSize = float64(model.GroupSize)
	v.startFrame = 0
	v.endFrame = numFrames - 1
	v.currentAction = -1

	v.volA = refVol
	v.volB = refVol
	v.comp = NewComposer(refVol)
	v.solver = v.comp.GetSolver()
	v.ent = v.solver.GetEntity()
	v.lerpT = 0.0
	v.lastVolA = nil
	v.salt = uint64(rand.Intn(numFrames))
	v.facesA = facesA
	v.facesB = facesB
	v.rot = articulated_dynamics.NewTagAxisIdentity()
	v.lastQw = 1.0
	v.singleFrame = v.endFrame == 0

	if len(v.actions) > 0 {
		v.setActionState(0)
	}

	v.Compile(0)
	return v
}

// Setup initializes the entry with the given thing and stores its solver and entity references.
func (v *Element3D) Setup(thing ithing.IThing) error {
	//for _, f := range v.volumes {
	//	f.SetThing(thing)
	//}
	if err := v.comp.Setup(thing); err != nil {
		return err
	}
	v.solver = v.comp.GetSolver()
	v.ent = v.solver.GetEntity()
	v.lastPoseGen = 0
	return nil
}

// GetSalt returns the salt value associated with the entry.
func (v *Element3D) GetSalt() uint64 {
	return v.salt
}

// SetSalt sets the salt value for the Element3D.
func (v *Element3D) SetSalt(salt uint64) {
	v.salt = salt
}

// GetFacesA returns the faces collection associated with side A.
func (v *Element3D) GetFacesA() *volumes.Faces {
	return v.facesA
}

// GetFacesB returns the secondary Faces collection associated with the entry.
func (v *Element3D) GetFacesB() *volumes.Faces {
	return v.facesB
}

// GetLerpT returns the interpolation parameter for this 3D vertices entry.
func (v *Element3D) GetLerpT() float64 {
	return v.lerpT
}

// GetLastRolVolA returns the last recorded rotation volume A.
func (v *Element3D) GetLastRolVolA() *volumes.Volume {
	return v.lastRotVolA
}

// GetLastRolVolB returns the last rotated volume B associated with the entry.
func (v *Element3D) GetLastRolVolB() *volumes.Volume {
	return v.lastRotVolB
}

// UnsetLastRolVolA clears the stored last rotation volume A value.
func (v *Element3D) UnsetLastRolVolA() {
	v.lastRotVolA = nil
	v.lastRotFrameA = -1
}

// UnsetLastRolVolB clears the stored last rotational volume B reference.
func (v *Element3D) UnsetLastRolVolB() {
	v.lastRotVolB = nil
	v.lastRotFrameB = -1
}

// GetMinMax returns the local minimum and maximum bounds as minX, minY, minZ, maxX, maxY, and maxZ.
func (v *Element3D) GetMinMax() (float64, float64, float64, float64, float64, float64) {
	return v.minX, v.minY, v.minZ, v.maxX, v.maxY, v.maxZ
}

// getFrameVolume returns the active Volume for frameIdx, loading it from the template LRU pool on demand
// without evicting keepFrame or currently active animation frames unless necessary.
func (v *Element3D) getFrameVolume(frameIdx int, keepFrame int) *volumes.Volume {
	if frameIdx < 0 || frameIdx >= v.numFrames {
		return nil
	}
	if v.materializedAll {
		return v.vols.Get(frameIdx)
	}
	for i := range v.slots {
		if v.slots[i].frameIdx == frameIdx {
			v.slotSeq++
			v.slots[i].seq = v.slotSeq
			return v.slots[i].vol
		}
	}
	victim := -1
	bestSeq := uint64(math.MaxUint64)
	for i := range v.slots {
		sFrame := v.slots[i].frameIdx
		if sFrame >= 0 && (sFrame == keepFrame || sFrame == v.frameIdxA || sFrame == v.frameIdxB) {
			continue
		}
		if v.slots[i].seq < bestSeq {
			bestSeq = v.slots[i].seq
			victim = i
		}
	}
	if victim < 0 {
		for i := range v.slots {
			sFrame := v.slots[i].frameIdx
			if sFrame >= 0 && sFrame == keepFrame {
				continue
			}
			if v.slots[i].seq < bestSeq {
				bestSeq = v.slots[i].seq
				victim = i
			}
		}
	}
	if victim < 0 {
		victim = 0
	}
	slot := &v.slots[victim]
	v.tpl.ApplyTemplateFrameToVolume(frameIdx, slot.vol)
	v.slotSeq++
	slot.frameIdx = frameIdx
	slot.seq = v.slotSeq
	return slot.vol
}

// ensureMaterialized allocates full per-frame volumes when operations such as Deform or CenterFrames
// require independent persistent modifications across all animation frames.
func (v *Element3D) ensureMaterialized() {
	if v.materializedAll {
		return
	}
	vols := volumes.NewVolumes(v.numFrames)
	for i := range v.slots {
		sf := v.slots[i].frameIdx
		if sf >= 0 && sf < v.numFrames && vols.Get(sf) == nil {
			vols.Set(sf, v.slots[i].vol)
		}
	}
	for i := 0; i < v.numFrames; i++ {
		if vols.Get(i) != nil {
			continue
		}
		vol := v.tpl.AllocateSlotVolume(i, v.sharedTree, v.solver, false)
		v.tpl.ApplyTemplateFrameToVolume(i, vol)
		vols.Set(i, vol)
	}
	v.vols = vols
	v.slots = nil
	v.materializedAll = true
	if v.frameIdxA >= 0 && v.frameIdxA < v.numFrames {
		v.volA = vols.Get(v.frameIdxA)
		v.comp.SetVolume(v.volA)
	}
	if v.frameIdxB >= 0 && v.frameIdxB < v.numFrames {
		v.volB = vols.Get(v.frameIdxB)
	}
}

// GetVolume returns the volume at the specified index.
func (v *Element3D) GetVolume(idx int) *volumes.Volume {
	return v.getFrameVolume(idx, -1)
}

// GetVolumesLen returns the number of volumes associated with the entry.
func (v *Element3D) GetVolumesLen() int {
	return v.numFrames
}

// GetActionsLen returns the number of actions associated with the entry.
func (v *Element3D) GetActionsLen() int {
	return len(v.actions)
}

// GetSolver returns the collision solver associated with the vertices entry.
func (v *Element3D) GetSolver() *collisions.Solver {
	return v.solver
}

// GetAABB returns the entity bounding box updated from cached vertex bounds when applicable.
func (v *Element3D) GetAABB() *aabb.AABB {
	if v.ent == nil {
		return nil
	}
	return v.ent.GetAABB()
}

// QueryOverlaps invokes callback for each object whose bounds overlap object, stopping when callback returns false.
func (v *Element3D) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.comp.QueryOverlaps(object, callback)
}

// ensureActiveTree rebuilds the shared AABB tree from v.volA only when the active frame
// volume, frame index, or its geometry revision changes.
func (v *Element3D) ensureActiveTree() {
	if v.volA == nil {
		return
	}
	faces := v.volA.GetFaces()
	if faces == nil {
		return
	}
	ver := faces.GetVersion()
	if v.treeVol == v.volA && v.treeFrameIdx == v.frameIdxA && v.treeVolVer == ver {
		return
	}
	v.volA.RebuildTree()
	v.treeVol = v.volA
	v.treeFrameIdx = v.frameIdxA
	v.treeVolVer = ver
	v.treeRebuilds++
}

// SetAction switches to the action at idx and initializes its animation frame, volume, and interpolation state.
func (v *Element3D) SetAction(idx int) {
	if v.setActionState(idx) {
		v.ensureActiveTree()
	}
}

func (v *Element3D) setActionState(idx int) bool {
	if idx < 0 || idx >= len(v.actions) {
		fmt.Printf("Invalid action index: %d\n", idx)
		return false
	}
	if v.currentAction == idx {
		return false
	}
	v.currentAction = idx
	action := v.actions[idx]
	v.startTick = v.tick
	v.startFrame = action.Start
	v.endFrame = action.End
	v.clampAnim = action.Clamp
	v.singleFrame = v.startFrame == v.endFrame
	if v.startFrame >= 0 {
		v.frameIdxA = v.startFrame
		v.frameIdxB = v.startFrame
		refVol := v.getFrameVolume(v.startFrame, -1)
		v.volA = refVol
		v.volB = refVol
		v.lerpT = 0.0
		v.comp.SetVolume(v.volA)
		if !v.vols.HasCenters() {
			v.lastVolA = v.volA
			v.lastSolverFrameA = v.frameIdxA
		}
	}
	v.lastPoseGen = 0
	vertices.BumpStateEpoch()
	if v.ent != nil {
		v.ent.BumpPoseGen()
	}
	return true
}

// AdvanceAnimation advances the animation state to tick, selecting frame volumes and interpolation factor.
func (v *Element3D) AdvanceAnimation(tick uint64) {
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
	v.volA = v.getFrameVolume(v.frameIdxA, v.frameIdxB)
	if v.frameIdxB == v.frameIdxA {
		v.volB = v.volA
	} else {
		v.volB = v.getFrameVolume(v.frameIdxB, v.frameIdxA)
	}
	v.lerpT = lerpT
}

// CenterFrames recenters the entry volumes and rebuilds the collision solver using the given physics parameters.
func (v *Element3D) CenterFrames(partMass, restitution, friction, gForce float64) {
	v.ensureMaterialized()
	v.vols.BuildCenters()

	partSolver := collisions.NewSolver(true, partMass, restitution, friction, v.restitutionCombine, v.frictionCombine, gForce)
	for i := 0; i < v.vols.Len(); i++ {
		if vol := v.vols.Get(i); vol != nil {
			vol.SetSolver(partSolver)
		}
	}
	rebuildSolverFromVolume(partSolver, v.volA)
	v.comp.solver = partSolver
	v.comp.volume = v.volA
	v.solver = partSolver
	v.ent = partSolver.GetEntity()
	v.lastVolA = v.volA
	v.lastSolverFrameA = v.frameIdxA
	v.treeVol = nil
	v.treeFrameIdx = -1
	v.lastPoseGen = 0
	v.ensureActiveTree()
	if v.ent != nil {
		v.ent.BumpPoseGen()
	}
}

// IntersectRay tests a world-space ray against the thing's mesh geometry and returns the hit distance and face normal.
func (v *Element3D) IntersectRay(oX, oY, oZ, dirX, dirY, dirZ, maxDist float64) (float64, geometry.XYZ, bool) {
	ent := v.ent
	cx, cy, cz := ent.GetCenter()
	dx := oX - cx
	dy := oY - cy
	dz := oZ - cz
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := ent.GetAxes()
	locOx := dx*ax0 + dy*ay0 + dz*az0
	locOy := dx*ax1 + dy*ay1 + dz*az1
	locOz := dx*ax2 + dy*ay2 + dz*az2
	locDirX := dirX*ax0 + dirY*ay0 + dirZ*az0
	locDirY := dirX*ax1 + dirY*ay1 + dirZ*az1
	locDirZ := dirX*ax2 + dirY*ay2 + dirZ*az2

	hitDist, hitFace, okHit := v.IntersectRayFace(locOx, locOy, locOz, locDirX, locDirY, locDirZ, maxDist)
	if !okHit || hitFace == nil {
		return maxDist, geometry.XYZ{}, false
	}
	lnx, lny, lnz := hitFace.GetNormal()
	wnx := lnx*ax0 + lny*ax1 + lnz*ax2
	wny := lnx*ay0 + lny*ay1 + lnz*ay2
	wnz := lnx*az0 + lny*az1 + lnz*az2
	return hitDist, geometry.XYZ{X: wnx, Y: wny, Z: wnz}, true
}

var standingReferenceNames = []string{"stand", "idle", "legs_idle", "hover"}

// findReferenceFrameIndex returns the first frame index matching an idle/standing pose
// so shared collider bounds match the upright stance instead of the final animation frame.
func (v *Element3D) findReferenceFrameIndex() int {
	for _, act := range v.actions {
		if act == nil {
			continue
		}
		for _, refName := range standingReferenceNames {
			if strings.Contains(act.Name, refName) && act.Start >= 0 && act.Start < v.numFrames {
				if v.tpl != nil && v.tpl.frames[act.Start].faceCount > 0 {
					return act.Start
				}
				if v.materializedAll {
					if vol := v.vols.Get(act.Start); vol != nil && vol.GetFaceCount() > 0 {
						return act.Start
					}
				}
			}
		}
	}
	return 0
}

// rebuildSolverFromVolume updates solver collision data from volume face bounds, centroid, and inertia when non-nil.
func rebuildSolverFromVolume(solver *collisions.Solver, vol *volumes.Volume) {
	if vol == nil || solver == nil {
		return
	}
	faces := vol.GetFaces()
	minX, minY, minZ := faces.GetMin()
	w, h, d := faces.GetSize()
	cx, cy, cz := faces.GetCentroid()
	ixx, ixy, ixz, iyy, iyz, izz := faces.GetInertia()
	solver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)
}

// transformVolumeFaces copies and transforms volume faces into dstFaces, optionally updating and returning bounds.
func transformVolumeFaces(srcVol *volumes.Volume, dstFaces *volumes.Faces, rot *articulated_dynamics.TagAxis, isIdent bool, ox, oy, oz float64, minX, minY, minZ, maxX, maxY, maxZ float64, trackBounds bool) (float64, float64, float64, float64, float64, float64) {
	if srcVol == nil || dstFaces == nil {
		return minX, minY, minZ, maxX, maxY, maxZ
	}
	noOffset := ox == 0 && oy == 0 && oz == 0
	count := srcVol.GetFaceCount()
	dstLen := dstFaces.Len()
	dstFaces.EnsureLen(count)
	dstFaces.SetCount(count)
	limit := count
	for j := range limit {
		src := srcVol.GetFace(j)
		dst := dstFaces.At(j)
		if src == nil || dst == nil {
			continue
		}
		dst.SetMaterial(src.GetMaterial())
		dst.SetU(src.GetU())
		dst.SetV(src.GetV())
		if isIdent && noOffset {
			dst.CopyPoints(src)
			dst.CopyRenderAttributes(src)
		} else {
			rot.TransformPointsWithMatrix32(dst.GetTri(), src.GetTri(), ox, oy, oz)
			if isIdent {
				dst.CopyRenderAttributes(src)
			} else {
				nx0, ny0, nz0, nx1, ny1, nz1, nx2, ny2, nz2 := src.GetVertexNormalsFlat()
				rx0, ry0, rz0 := rot.RotatePoint(float64(nx0), float64(ny0), float64(nz0))
				rx1, ry1, rz1 := rot.RotatePoint(float64(nx1), float64(ny1), float64(nz1))
				rx2, ry2, rz2 := rot.RotatePoint(float64(nx2), float64(ny2), float64(nz2))
				dst.SetComputedVertexNormals(rx0, ry0, rz0, rx1, ry1, rz1, rx2, ry2, rz2)
				tx0, ty0, tz0, s0, tx1, ty1, tz1, s1, tx2, ty2, tz2, s2 := src.GetTangentsFlat()
				rtx0, rty0, rtz0 := rot.RotatePoint(float64(tx0), float64(ty0), float64(tz0))
				rtx1, rty1, rtz1 := rot.RotatePoint(float64(tx1), float64(ty1), float64(tz1))
				rtx2, rty2, rtz2 := rot.RotatePoint(float64(tx2), float64(ty2), float64(tz2))
				dst.SetTangents(rtx0, rty0, rtz0, rtx1, rty1, rtz1, rtx2, rty2, rtz2, float64(s0), float64(s1), float64(s2))
			}
		}
		if trackBounds {
			minX, minY, minZ, maxX, maxY, maxZ = dst.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
		}
	}
	for j := limit; j < dstLen; j++ {
		if f := dstFaces.At(j); f != nil {
			f.Zero()
		}
	}
	dstFaces.BumpVersion()
	return minX, minY, minZ, maxX, maxY, maxZ
}

// FrameCenterA returns the center coordinates of frame A.
func (v *Element3D) FrameCenterA() geometry.XYZ {
	return v.vols.GetCenter(v.frameIdxA)
}

// FrameCenterB returns the center of frame B.
func (v *Element3D) FrameCenterB() geometry.XYZ {
	return v.vols.GetCenter(v.frameIdxB)
}

// SyncSolverToCurrentFrame updates the solver from the current frame volume when it has changed.
func (v *Element3D) SyncSolverToCurrentFrame() {
	if v.volA == nil {
		return
	}
	v.ensureActiveTree()
	if v.lastVolA == v.volA && v.lastSolverFrameA == v.frameIdxA {
		return
	}
	v.comp.SetVolume(v.volA)
	rebuildSolverFromVolume(v.comp.solver, v.volA)
	v.lastVolA = v.volA
	v.lastSolverFrameA = v.frameIdxA
}

// CompileAttached updates transformed attached volumes and records their combined bounding extents.
func (v *Element3D) CompileAttached(rotA, rotB *articulated_dynamics.TagAxis, dOffX, dOffY, dOffZ float64) {
	v.SyncSolverToCurrentFrame()
	if v.volA == nil {
		return
	}
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	minX, minY, minZ, maxX, maxY, maxZ = transformVolumeFaces(v.volA, v.facesA, rotA, rotA.IsIdentity(), 0, 0, 0, minX, minY, minZ, maxX, maxY, maxZ, true)

	volB := v.volB
	if volB == nil {
		volB = v.volA
	}
	minX, minY, minZ, maxX, maxY, maxZ = transformVolumeFaces(volB, v.facesB, rotB, rotB.IsIdentity(), dOffX, dOffY, dOffZ, minX, minY, minZ, maxX, maxY, maxZ, true)

	if minX > maxX {
		minX, minY, minZ = 0, 0, 0
		maxX, maxY, maxZ = 0, 0, 0
	}
	v.minX, v.minY, v.minZ = minX, minY, minZ
	v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
}

// CompileDetached updates cached rotated volume bounds and rebuilds the entity AABB at its current center.
func (v *Element3D) CompileDetached() {
	if v.volA == nil || v.ent == nil {
		return
	}
	v.ensureActiveTree()
	ent := v.ent
	q := ent.GetQRotation()
	qx, qy, qz, qw := q.Get()
	if v.lastRotVolA != v.volA || v.lastRotFrameA != v.frameIdxA || qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
		v.lastRotVolA = v.volA
		v.lastRotFrameA = v.frameIdxA
		v.lastQx, v.lastQy, v.lastQz, v.lastQw = qx, qy, qz, qw
		isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0
		if !isIdent {
			v.rot.SetFromQuaternion(q)
		}
		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
		minX, minY, minZ, maxX, maxY, maxZ = transformVolumeFaces(v.volA, v.facesA, v.rot, isIdent, 0, 0, 0, minX, minY, minZ, maxX, maxY, maxZ, true)
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

// Compile advances animation, updates rotated bounds and volume state, rebuilds the entity AABB, and returns true.
func (v *Element3D) Compile(tick uint64) bool {
	if v.singleFrame && v.ent != nil && v.volA != nil && v.volB == v.volA &&
		v.lastVolA == v.volA && v.lastRotVolA == v.volA &&
		v.lastRotFrameA == v.frameIdxA && v.lastSolverFrameA == v.frameIdxA &&
		v.treeVol == v.volA && v.treeFrameIdx == v.frameIdxA &&
		v.ent.GetPoseGen() == v.lastPoseGen {
		if faces := v.volA.GetFaces(); faces != nil && v.treeVolVer == faces.GetVersion() {
			v.lerpT = 0.0
			return true
		}
	}

	v.AdvanceAnimation(tick)

	ent := v.ent
	if ent == nil {
		ent = v.comp.GetSolver().GetEntity()
	}
	q := ent.GetQRotation()
	qx, qy, qz, qw := q.Get()
	isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0

	if v.volA != v.lastRotVolA || v.volB != v.lastRotVolB || v.frameIdxA != v.lastRotFrameA || v.frameIdxB != v.lastRotFrameB || qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
		if !isIdent {
			v.rot.SetFromQuaternion(q)
		}
		v.lastQx, v.lastQy, v.lastQz, v.lastQw = qx, qy, qz, qw
		v.lastRotVolA = v.volA
		v.lastRotVolB = v.volB
		v.lastRotFrameA = v.frameIdxA
		v.lastRotFrameB = v.frameIdxB

		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
		minX, minY, minZ, maxX, maxY, maxZ = transformVolumeFaces(v.volA, v.facesA, v.rot, isIdent, 0, 0, 0, minX, minY, minZ, maxX, maxY, maxZ, true)
		if v.volB != v.volA {
			minX, minY, minZ, maxX, maxY, maxZ = transformVolumeFaces(v.volB, v.facesB, v.rot, isIdent, 0, 0, 0, minX, minY, minZ, maxX, maxY, maxZ, true)
		} else {
			_, _, _, _, _, _ = transformVolumeFaces(v.volA, v.facesB, v.rot, isIdent, 0, 0, 0, minX, minY, minZ, maxX, maxY, maxZ, false)
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

	if v.lastVolA != v.volA || v.lastSolverFrameA != v.frameIdxA {
		v.comp.SetVolume(v.volA)
		v.lastVolA = v.volA
		v.lastSolverFrameA = v.frameIdxA
	}
	v.ensureActiveTree()

	cx, cy, cz := ent.GetCenter()
	ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	v.lastPoseGen = ent.GetPoseGen()

	return true
}

// GetHash returns a deterministic state hash combining active frame/morph geometry and animated materials.
func (v *Element3D) GetHash() uint64 {
	return v.GetHashForMode(true)
}

// GetHashForMode computes this element's geometry hash, allowing rigid instancing only when allowRigid is true.
func (v *Element3D) GetHashForMode(allowRigid bool) uint64 {
	if allowRigid && v.volA != nil && (v.singleFrame || v.volA == v.volB) {
		if faces := v.volA.GetFaces(); faces != nil {
			if fHash := faces.GetRenderHash(); fHash != 0 {
				h := geometry.MixU64(0x7269676964336401, fHash)
				if h == 0 {
					return 1
				}
				return h
			}
		}
		return 0
	}
	fHashA := v.facesA.GetRenderHash()
	if fHashA == 0 {
		return 0
	}
	fHashB := v.facesB.GetRenderHash()
	h := geometry.MixU64(0x6d6f727068336402, fHashA)
	h = geometry.MixU64(h, fHashB)
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(v.lerpT)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(vertices.RenderModeModel3D)))
	if h == 0 {
		return 1
	}
	return h
}

// GetUIFaces returns the face sets, interpolation value, and render mode for UI rendering.
func (v *Element3D) GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64) {
	return v.facesA, v.facesB, v.lerpT, vertices.RenderModeModel3D
}

// GetRigidFaces returns the unrotated local-space static Faces template when the model
// is currently on a single non-morphing frame, allowing GPU instanced rendering without CPU vertex copies.
func (v *Element3D) GetRigidFaces() *volumes.Faces {
	if v.volA != nil && (v.singleFrame || v.volA == v.volB) {
		return v.volA.GetFaces()
	}
	return nil
}

// GetVolumes returns the entry's source and target volumes along with the interpolation factor.
func (v *Element3D) GetVolumes() (*volumes.Volume, *volumes.Volume, float64) {
	return v.volA, v.volB, v.lerpT
}

// SetRagdollMode switches the vertices entry to ragdoll mode.
func (v *Element3D) SetRagdollMode() {
}

// IsActionClamped reports whether the action at idx is clamped or death-related; invalid or nil actions return false.
func (v *Element3D) IsActionClamped(idx int) bool {
	if idx < 0 || idx >= len(v.actions) || v.actions[idx] == nil {
		return false
	}
	return v.actions[idx].Clamp
}

// FreezeCurrentFrame locks the entry to the current frame and resets interpolation.
func (v *Element3D) FreezeCurrentFrame() {
	v.singleFrame = true
	if v.volA != nil {
		v.volB = v.volA
	}
	v.lerpT = 0.0
	v.lastPoseGen = 0
	vertices.BumpStateEpoch()
	if v.ent != nil {
		v.ent.BumpPoseGen()
	}
}

// GetTagIndex returns the index for the named tag, or -1 if the tag is not found.
func (v *Element3D) GetTagIndex(name string) int {
	if idx, ok := v.tagMap[name]; ok {
		return idx
	}
	fmt.Printf("[warning] Vertices3DEntry %s: tag not found\n", name)
	return -1
}

// Deform applies an impulse within the given radius and recompiles cached data when geometry changes.
// Deform returns true if any vertex volume was modified.
func (v *Element3D) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	v.ensureMaterialized()
	deformed := v.vols.Deform(impact, impulse, radius)
	if deformed {
		v.lastRotVolA = nil
		v.lastRotVolB = nil
		v.lastRotFrameA = -1
		v.lastRotFrameB = -1
		v.treeVol = nil
		v.treeFrameIdx = -1
		v.lastPoseGen = 0
		vertices.BumpStateEpoch()
		if v.ent != nil {
			v.ent.BumpPoseGen()
		}
		v.Compile(v.tick)
	}
	return deformed
}

// DeformDent applies a dent at impact along the opposite normal, with the given depth and radius.
func (v *Element3D) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	impulse := geometry.XYZ{X: -normal.X * depth, Y: -normal.Y * depth, Z: -normal.Z * depth}
	return v.Deform(impact, impulse, radius)
}

// IntersectRayFace finds the closest solid face hit by a ray within maxDist.
// It returns the hit distance, the intersected face, and whether a hit occurred.
func (v *Element3D) IntersectRayFace(locOx, locOy, locOz, locDirX, locDirY, locDirZ, maxDist float64) (float64, *volumes.Face, bool) {
	if v.volA == nil {
		return maxDist, nil, false
	}
	tree := v.volA.GetFacesTree()
	if tree == nil {
		return maxDist, nil, false
	}
	var hitFace *volumes.Face
	closest := maxDist
	hit := false
	tree.QueryRay(locOx, locOy, locOz, locDirX, locDirY, locDirZ, maxDist, func(obj aabb.IObject, d float64) (float64, bool) {
		fc, ok := obj.(*volumes.Face)
		if !ok || !fc.IsSolid() {
			return closest, false
		}
		if tHit, okHit := fc.IntersectRay(locOx, locOy, locOz, locDirX, locDirY, locDirZ, closest); okHit {
			closest = tHit
			hitFace = fc
			hit = true
			return tHit, true
		}
		return closest, false
	})
	return closest, hitFace, hit
}
