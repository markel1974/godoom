package model

import (
	"fmt"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// solverJitter defines a small epsilon used to stabilize solver event calculations.
const solverJitter = 1e-6

// Things manages game entities, their spatial state, collision coordination, and staged lifecycle updates.
type Things struct {
	config            []*config.Thing
	volumes           *Volumes
	materials         *materials.Materials
	tree              *aabb.Tree
	player            *things.Player
	entities          map[uint64]ithing.IThing
	pending           []ithing.IThing
	thinkers          []ithing.IThing
	statics           []ithing.IThing
	thinkersLen       int
	pendingLen        atomic.Int32
	hasPending        bool
	pendingStage      []ithing.IThing
	pendingStageLen   int
	pendingRemove     []ithing.IThing
	pendingRemoveLen  int
	stageEpoch        uint64
	event             *ithing.ThingEvent
	cfg               []*config.Thing
	maxOverlapsWorld  int
	maxOverlapsThings int
	coordinator       *collisions.Coordinator
	throwable         []*config.Thing
}

// NewThings creates and initializes a Things manager from thing configs, volumes, materials, and collision settings.
func NewThings(maxOverlaps int, solverIterations int, cfg []*config.Thing, vols *Volumes, materials *materials.Materials) *Things {
	const defaultLen = 1024
	if maxOverlaps <= 0 {
		maxOverlaps = 100
	}
	e := &Things{
		maxOverlapsWorld:  maxOverlaps,
		maxOverlapsThings: maxOverlaps / 2,
		cfg:               cfg,
		tree:              aabb.NewTree(uint(len(cfg)*2), 4.0),
		entities:          make(map[uint64]ithing.IThing),
		thinkers:          make([]ithing.IThing, defaultLen),
		pending:           make([]ithing.IThing, defaultLen),
		pendingStage:      make([]ithing.IThing, 128),
		pendingRemove:     make([]ithing.IThing, 128),
		thinkersLen:       0,
		hasPending:        false,
		config:            cfg,
		volumes:           vols,
		materials:         materials,
		event:             ithing.NewThingEvent(0, solverJitter),
		statics:           make([]ithing.IThing, 0, 16),
	}
	e.coordinator = collisions.NewCoordinator(solverIterations, defaultLen, e.querySolverContacts, vols.ClampStaticCage)
	e.pendingLen.Store(0)
	return e
}

// Setup initializes things from configuration, resolves volumes and teleport targets, and stages created entities.
func (th *Things) Setup() error {
	th.resolveTeleportTargets()
	for _, ct := range th.cfg {
		if ct == nil {
			continue
		}
		volume, _ := th.volumes.QueryPoint(ct.Position.X, ct.Position.Y, ct.Position.Z)
		if volume == nil && ct.Kind == config.ThingDoorDef {
			if vols := th.volumes.GetVolumes(); vols != nil && vols.Len() > 0 {
				volume = vols.Get(0)
			}
		}
		if volume == nil {
			fmt.Printf("Warning can't find thing location at %f, %f, %f\n", ct.Position.X, ct.Position.Y, ct.Position.Z)
		}
		t2 := th.create(ct, volume)
		if t2.GetKind() == config.ThingItemDef {
			th.throwable = append(th.throwable, ct)
		}
		th.addThing(t2)
	}

	th.StagePrepare()
	for _, t2 := range th.entities {
		th.StageAdd(t2)
	}
	return nil
}

// GetMaterials returns the materials associated with this Things instance.
func (th *Things) GetMaterials() *materials.Materials {
	return th.materials
}

// GetVolumes returns the volumes handler.
func (th *Things) GetVolumes() things.IVolumesHandler {
	return th.volumes
}

// Len returns the number of staged things.
func (th *Things) Len() int {
	return th.coordinator.StagedLen()
}

// QueryCollisionCage checks for collisions within lCage, limiting overlap processing to maxOverlaps.
func (th *Things) QueryCollisionCage(lCage *collisions.Cage, maxOverlaps int) {
	th.queryCollisionCage(lCage, maxOverlaps, nil)
}

// querySolverContacts populates collision contacts for a cage against world volumes and other things.
func (th *Things) querySolverContacts(cage *collisions.Cage, wake collisions.WakeRunningFn) {
	if s := cage.GetSolver(); s == nil || !s.IgnoresStatic() {
		th.volumes.QueryCollisionCage(cage, th.maxOverlapsWorld)
	}
	th.queryCollisionCage(cage, th.maxOverlapsThings, wake)
}

// queryCollisionCage queries nearby cages, records face overlaps up to maxOverlaps, and wakes sleeping solvers when needed.
func (th *Things) queryCollisionCage(lCage *collisions.Cage, maxOverlaps int, wake collisions.WakeRunningFn) {
	lThing := lCage.GetObject().(ithing.IThing)
	th.tree.QueryOverlaps(lCage, func(object aabb.IObject) bool {
		rThing := object.(ithing.IThing)
		if lThing == rThing {
			return false
		}
		rCage := rThing.GetSolver().GetCage()
		if lGroup := lCage.GetCollisionGroup(); lGroup != 0 && lGroup == rCage.GetCollisionGroup() {
			return false
		}
		if lCage.HasSeen(rCage) {
			return false
		}
		wasSleeping := rThing.IsSleeping()
		lCage.Seen(rCage)
		if !wasSleeping {
			rCage.Seen(lCage)
		}
		pairCounter := 0
		lEntityL, deltaX, deltaY, deltaZ := lCage.TranslateCage(0, rCage)
		rThing.QueryOverlaps(lEntityL, func(rEnt aabb.IObject) bool {
			rFace := rEnt.(*volumes.Face)
			z := lCage.Translate(1, rFace, deltaX, deltaY, deltaZ)
			// Query sul BVH del Self: basta il primo lFace sovrapposto per validare il contatto con rFace
			lThing.QueryOverlaps(z, func(lEnt aabb.IObject) bool {
				lFace := lEnt.(*volumes.Face)
				lCage.AddFace(rFace, lFace)
				pairCounter++
				return true
			})
			return pairCounter > maxOverlaps
		})
		if wasSleeping {
			if pairCounter == 0 {
				return false
			}
			// Real face intersection with a sleeping body: wake its connected island before rebuilding/committing rCage.
			if wake != nil {
				if rSolver := rCage.GetSolver(); rSolver != nil {
					wake(rSolver)
				}
			}
			rCage.Seen(lCage)
		}
		lCage.Commit(rCage)
		return false
	})
}

// QueryMultiFrustum queries objects against rear and front frustums, invoking callback with membership flags.
func (th *Things) QueryMultiFrustum(rear *aabb.Frustum, front *aabb.Frustum, callback func(object aabb.IObject, isFront bool, isBack bool) bool) {
	th.tree.QueryMultiFrustum(rear, front, callback)
}

// QueryFrustum calls callback for each object intersecting the frustum, stopping if callback returns false.
func (th *Things) QueryFrustum(front *aabb.Frustum, callback func(object aabb.IObject) bool) {
	th.tree.QueryFrustum(front, callback)
}

// QueryStatic calls callback for each static object.
func (th *Things) QueryStatic(callback func(object aabb.IObject) bool) {
	for _, st := range th.statics {
		callback(st)
	}
}

// GetPlayer returns the current player.
func (th *Things) GetPlayer() *things.Player {
	return th.player
}

// SetPlayer sets the active player and adds it to the collection of things.
func (th *Things) SetPlayer(p *things.Player) {
	th.player = p
	th.addThing(p)
}

// GetThing returns the thing with the specified ID, or nil if no matching thing is found.
func (th *Things) GetThing(id string) ithing.IThing {
	for _, thing := range th.entities {
		if thing.GetId() == id {
			return thing
		}
	}
	return nil
}

// GetTextures returns the texture collection associated with the Things materials.
func (th *Things) GetTextures() textures.ITextures {
	return th.materials.GetTextures()
}

// QueryRay finds objects intersected by a ray up to maxDistance and invokes callback with each hit distance.
// QueryRay uses the callback return values to update the search distance and decide whether to continue.
func (th *Things) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool)) {
	th.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// QueryVolume returns the volume containing the point at the given coordinates, or nil if none is found.
func (th *Things) QueryVolume(x, y, z float64) *volumes.Volume {
	vol, _ := th.volumes.QueryPoint(x, y, z)
	return vol
}

// resolveTeleportTargets resolves teleport destinations and angles from configured target thing IDs or target names.
func (th *Things) resolveTeleportTargets() {
	targets := make(map[string]*config.Thing)
	for _, ct := range th.cfg {
		if ct == nil {
			continue
		}
		if ct.Id != "" {
			targets[ct.Id] = ct
		}
		if ct.Teleport != nil && ct.Teleport.TargetName != "" {
			targets[ct.Teleport.TargetName] = ct
		}
	}
	for _, ct := range th.cfg {
		if ct == nil || ct.Kind != config.ThingTeleportDef || ct.Teleport == nil || ct.Teleport.Target == "" {
			continue
		}
		if dest, ok := targets[ct.Teleport.Target]; ok {
			if ct.Teleport.Destination == (geometry.XYZ{}) {
				ct.Teleport.Destination = dest.Position
				if !ct.Teleport.PreserveAngle && ct.Teleport.TargetAngle == 0 {
					ct.Teleport.TargetAngle = dest.Angle
				}
			}
		}
	}
}

// create initializes callbacks, constructs a thing for the config kind, positions it, and returns it.
func (th *Things) create(ct *config.Thing, volume *volumes.Volume) ithing.IThing {
	if ct.OnCollision == nil {
		fmt.Println("onCollision is nil for thing:" + ct.Id)
		ct.OnCollision = func(other config.IThingConfig, energy float64) {}
	}
	if ct.OnImpact == nil {
		fmt.Println("OnImpact is nil for thing:" + ct.Id)
		ct.OnImpact = func(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {}
	}
	if ct.OnBind == nil {
		fmt.Println("OnBind is nil for thing:" + ct.Id)
		ct.OnBind = func(self config.IThingConfig) {}
	}

	const disableEnemies = false
	if disableEnemies {
		if ct.Kind == config.ThingEnemyDef {
			ct.Kind = config.ThingItemDef
		}
	}
	var thing ithing.IThing
	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = things.NewEnemy(th, ct, volume)
	case config.ThingWeaponDef:
		thing = things.NewWeapon(th, ct, volume)
		th.throwable = append(th.throwable, ct)
	case config.ThingThrowableDef:
		thing = things.NewThingThrowable(th, ct, volume)
		th.throwable = append(th.throwable, ct)
	case config.ThingKeyDef:
		thing = things.NewKey(th, ct, volume)
	case config.ThingDoorDef:
		thing = things.NewDoor(th, ct, volume)
	case config.ThingTeleportDef:
		thing = things.NewTeleport(th, ct, volume)
	case config.ThingInterfaceDef:
		thing = things.NewUI(th, ct)
	case config.ThingItemDef:
		thing = things.NewItem(th, ct, volume)
		th.throwable = append(th.throwable, ct)
	default:
		thing = things.NewItem(th, ct, volume)
	}
	if ct.Kind != config.ThingDoorDef {
		entity := thing.GetEntity()
		if ct.Kind == config.ThingThrowableDef {
			entity.MoveCenterTo(ct.Position.X, ct.Position.Y, ct.Position.Z)
		} else {
			entity.MoveTo(ct.Position.X, ct.Position.Y, ct.Position.Z)
		}
	}
	return thing
}

// CreateThrowable clones a throwable template, initializes its state, and queues it for creation in the given volume.
func (th *Things) CreateThrowable(self ithing.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64) {
	if len(th.throwable) <= throwableIndex {
		return
	}
	src := th.throwable[throwableIndex]
	dst := src.Clone()
	dst.Id = utils.NextUUId()
	dst.Kind = config.ThingThrowableDef
	dst.Position = pos
	dst.Angle = angle
	dst.Pitch = pitch
	dst.Speed = speed
	dst.OnCollision = onCollision
	dst.OnImpact = onImpact
	slot := th.pendingLen.Add(1) - 1
	if slot >= int32(len(th.pending)) {
		fmt.Println("max slot reached!")
		return
	}
	throwable := th.create(dst, volume)
	ent := throwable.GetEntity()
	sx, sy, _ := ent.GetCenter()
	solver := throwable.GetSolver()
	solver.GetCage().SetSpawnOrigin(sx, sy, pos.Z)
	th.pending[slot] = throwable
	th.hasPending = true
}

// AddJoint adds a joint constraint to the coordinator.
func (th *Things) AddJoint(joint *collisions.JointConstraint) {
	th.coordinator.AddJoint(joint)
}

// StagePart queues a non-nil thing for staging, expanding the pending stage buffer as needed.
func (th *Things) StagePart(part ithing.IThing) {
	if part == nil {
		return
	}
	if th.pendingStageLen >= len(th.pendingStage) {
		n := make([]ithing.IThing, len(th.pendingStage)*2)
		copy(n, th.pendingStage)
		th.pendingStage = n
	}
	th.pendingStage[th.pendingStageLen] = part
	th.pendingStageLen++
}

// StageRemove queues thing for removal, ignoring nil values.
func (th *Things) StageRemove(thing ithing.IThing) {
	if thing == nil {
		return
	}
	if th.pendingRemoveLen >= len(th.pendingRemove) {
		n := make([]ithing.IThing, len(th.pendingRemove)*2)
		copy(n, th.pendingRemove)
		th.pendingRemove = n
	}
	th.pendingRemove[th.pendingRemoveLen] = thing
	th.pendingRemoveLen++
}

// flushPendingRemovals removes all queued things and coordinator removables, then clears the pending removal lists.
func (th *Things) flushPendingRemovals() {
	if th.pendingRemoveLen > 0 {
		for x := 0; x < th.pendingRemoveLen; x++ {
			th.removeThing(th.pendingRemove[x])
		}
		th.pendingRemoveLen = 0
	}
	if removablesLen := th.coordinator.RemovablesLen(); removablesLen > 0 {
		for x := 0; x < removablesLen; x++ {
			s := th.coordinator.RemovablesAt(x)
			th.removeThing(s.GetCage().GetObject().(ithing.IThing))
		}
		th.coordinator.ClearRemovables()
	}
}

// StagePrepare resets staging state and advances the stage epoch.
func (th *Things) StagePrepare() {
	th.coordinator.Reset()
	th.thinkersLen = 0
	th.stageEpoch++
}

// StageAdd stages the given thing, or its owner when it is a non-physics-free subpart.
func (th *Things) StageAdd(t2 ithing.IThing) {
	if sub, ok := t2.(ithing.ISubPart); ok && !sub.IsPhysicsFree() {
		if owner := sub.GetOwner(); owner != nil {
			th.stageOne(owner)
		}
		return
	}
	th.stageOne(t2)
}

// stageOne stages an active thing's solver once per epoch and tracks it for thinking when applicable.
func (th *Things) stageOne(t2 ithing.IThing) {
	if t2 == nil || !t2.IsActive() {
		return
	}
	if !t2.GetSolver().TryMarkStaged(th.stageEpoch) {
		return
	}
	th.coordinator.Stage(t2.GetSolver())
	if t2.HasThinking() {
		th.thinkers[th.thinkersLen] = t2
		th.thinkersLen++
	}
}

// StageCommit stages pending changes for commit.
func (th *Things) StageCommit() {

}

// applySafeStepUp applies a safe upward step to the cage, clamped against static volumes.
func (th *Things) applySafeStepUp(cage *collisions.Cage, stepZ float64) {
	cage.ApplySafeStepUp(stepZ, th.volumes.ClampStaticCage)
}

// Compute processes pending changes, dispatches thinking events at the given coordinates, steps physics, and updates objects.
func (th *Things) Compute(pX float64, pY float64, pZ float64) {
	th.flushPendingRemovals()
	if th.hasPending {
		for x := 0; x < int(th.pendingLen.Load()); x++ {
			item := th.pending[x]
			th.addThing(item)
			th.StageAdd(item)
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	// Apply Thinking
	if th.thinkersLen > 0 {
		th.event.SetStage(ithing.StageThinking)
		th.event.SetCoords(pX, pY, pZ)
		for x := 0; x < th.thinkersLen; x++ {
			t2 := th.thinkers[x]
			th.event.Add()
			t2.PostMessage(th.event)
		}
		th.event.Wait()
	}

	// Flush and stage any throwable or newly released ragdoll/detached parts from Thinking on tick 0.
	if th.hasPending {
		pLen := int(th.pendingLen.Load())
		for x := 0; x < pLen; x++ {
			item := th.pending[x]
			th.addThing(item)
			th.StageAdd(item)
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	if th.pendingStageLen > 0 {
		for x := 0; x < th.pendingStageLen; x++ {
			th.StageAdd(th.pendingStage[x])
		}
		th.pendingStageLen = 0
	}

	th.coordinator.Step()
	th.flushPendingRemovals()

	if runningLen := th.coordinator.RunningLen(); runningLen > 0 {
		for rIdx := 0; rIdx < runningLen; rIdx++ {
			s := th.coordinator.RunningAt(rIdx)
			obj := s.GetCage().GetObject()
			if owner, ok := obj.(ithing.IPartOwner); ok {
				parts := owner.GetPartLen()
				if parts > 0 {
					owner.SyncAttachedParts()
					for x := 0; x < parts; x++ {
						p := owner.GetPartAt(x)
						if sub, sOk := p.(ithing.ISubPart); sOk && !sub.IsPhysicsFree() {
							th.tree.UpdateObject(p)
						}
					}
					continue
				}
			}
			th.tree.UpdateObject(obj)
		}
	}
}

// AddStaticThing adds a static thing to the collection.
func (th *Things) AddStaticThing(ent ithing.IThing) {
	th.statics = append(th.statics, ent)
}

// addThing registers a thing, indexes it or its parts, grows internal buffers as needed, and starts its loop.
func (th *Things) addThing(ent ithing.IThing) {
	th.entities[ent.GetEntity().GetId()] = ent
	var parts int
	owner, ok := ent.(ithing.IPartOwner)
	if ok {
		parts = owner.GetPartLen()
	}
	needed := len(th.entities) + parts
	if needed > cap(th.pending) {
		newCap := needed * 4
		th.thinkers = make([]ithing.IThing, newCap)
		th.pending = make([]ithing.IThing, newCap)
		th.coordinator.EnsureCapacity(newCap)
	}
	if parts > 0 {
		ent.Compile(0)
		for x := 0; x < parts; x++ {
			p := owner.GetPartAt(x)
			th.entities[p.GetEntity().GetId()] = p
			th.tree.InsertObject(p)
		}
	} else {
		th.tree.InsertObject(ent)
	}
	ent.StartLoop()
}

// removeThing removes the given thing from the registry and, when it owns no parts, from the tree.
func (th *Things) removeThing(ent ithing.IThing) {
	if ent == nil {
		return
	}
	id := ent.GetEntity().GetId()
	if _, ok := th.entities[id]; !ok {
		return
	}
	if owner, ok := ent.(ithing.IPartOwner); !ok || owner.GetPartLen() == 0 {
		th.tree.RemoveObject(ent)
	}
	delete(th.entities, id)
}
