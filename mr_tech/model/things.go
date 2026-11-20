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

const solverJitter = 1e-6

// Things manages game objects, their spatial partitioning, and contact interactions within a simulation environment.
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

// NewThings initializes and returns an instance of Things with the specified maximum number of things.
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

// Setup initializes the necessary configurations or resources required for the Things object to function properly.
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

// GetMaterials returns a pointer to the Materials associated with the Things instance.
func (th *Things) GetMaterials() *materials.Materials {
	return th.materials
}

// GetVolumes returns a IVolumesHandler interface.
func (th *Things) GetVolumes() things.IVolumesHandler {
	return th.volumes
}

// Len returns the number of staged elements in the coordinator.
func (th *Things) Len() int {
	return th.coordinator.StagedLen()
}

// QueryCollisionCage checks collisions between the specified CollisionCage and others in the spatial structure (tree).
func (th *Things) QueryCollisionCage(lCage *collisions.Cage, maxOverlaps int) {
	th.queryCollisionCage(lCage, maxOverlaps, nil)
}

func (th *Things) querySolverContacts(cage *collisions.Cage, wake collisions.WakeRunningFn) {
	if s := cage.GetSolver(); s == nil || !s.IgnoresStatic() {
		th.volumes.QueryCollisionCage(cage, th.maxOverlapsWorld)
	}
	th.queryCollisionCage(cage, th.maxOverlapsThings, wake)
}

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

// QueryMultiFrustum performs a spatial query against two frustums, invoking the callback for each intersected IAABB object.
func (th *Things) QueryMultiFrustum(rear *aabb.Frustum, front *aabb.Frustum, callback func(object aabb.IObject, isFront bool, isBack bool) bool) {
	th.tree.QueryMultiFrustum(rear, front, callback)
}

// QueryFrustum performs a spatial query within the specified frustum, invoking the callback for each intersected object.
func (th *Things) QueryFrustum(front *aabb.Frustum, callback func(object aabb.IObject) bool) {
	th.tree.QueryFrustum(front, callback)
}

// QueryStatic iterates over all static objects, applying the provided callback function to each object.
func (th *Things) QueryStatic(callback func(object aabb.IObject) bool) {
	for _, st := range th.statics {
		callback(st)
	}
}

// GetPlayer returns the current player instance.
func (th *Things) GetPlayer() *things.Player {
	return th.player
}

// SetPlayer assigns a ThingPlayer to the Things collection and integrates it into the entity management system.
func (th *Things) SetPlayer(p *things.Player) {
	th.player = p
	th.addThing(p)
}

func (th *Things) GetThing(id string) ithing.IThing {
	for _, thing := range th.entities {
		if thing.GetId() == id {
			return thing
		}
	}
	return nil
}

// GetTextures fetches the ITextures instance from the associated Materials object.
func (th *Things) GetTextures() textures.ITextures {
	return th.materials.GetTextures()
}

// QueryRay performs a raycast query within the spatial tree, invoking the callback for each intersected object.
// Parameters:
// - oX, oY, oZ: Origin coordinates of the ray.
// - dirX, dirY, dirZ: Direction vector of the ray.
// - maxDistance: Maximum distance the ray can travel.
// - callback: Function invoked for each intersected object, receives the object and its distance as arguments.
func (th *Things) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool)) {
	th.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// QueryVolume returns the volume containing (x, y, z), or nil if none is found.
func (th *Things) QueryVolume(x, y, z float64) *volumes.Volume {
	vol, _ := th.volumes.QueryPoint(x, y, z)
	return vol
}

// resolveTeleportTargets resolves TeleportSpec.Target references (matching another Thing's TargetName or Id)
// into concrete Destination coordinates and TargetAngle when Destination is unset.
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

// Create creates a new IThing instance based on the provided Thing and adds it to the Things collection.
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
	case config.ThingThrowableDef:
		thing = things.NewThingThrowable(th, ct, volume)
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

// CreateThrowable creates a throwable object with specified position, angle, pitch, mass, radius, and speed, adding it to the pending list.
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
	if self != nil && throwable.GetSolver() != nil && throwable.GetSolver().GetCage() != nil {
		sx, sy, _ := self.GetEntity().GetCenter()
		throwable.GetSolver().GetCage().SetSpawnOrigin(sx, sy, pos.Z)
	}
	th.pending[slot] = throwable
	th.hasPending = true
}

// AddJoint registers an active bilateral PGS joint constraint with the coordinator.
func (th *Things) AddJoint(joint *collisions.JointConstraint) {
	th.coordinator.AddJoint(joint)
}

// StagePart queues a newly released 6-DoF segment for immediate staging in the current physics tick.
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

// StageRemove queues a deactivated Thing for safe removal from Things.tree and Things.entities.
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

// StagePrepare resets staging state before collecting ready things and thinkers.
func (th *Things) StagePrepare() {
	th.coordinator.Reset()
	th.thinkersLen = 0
	th.stageEpoch++
}

// StageAdd queues t2 as ready (or routes attached sub-parts to their owning Thing) and avoids duplicate per-tick staging.
func (th *Things) StageAdd(t2 ithing.IThing) {
	if sub, ok := t2.(ithing.ISubPart); ok && !sub.IsPhysicsFree() {
		if owner := sub.GetOwner(); owner != nil {
			th.stageOne(owner)
		}
		return
	}
	th.stageOne(t2)
}

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

// applySafeStepUp verifies vertical headroom for a step-up before keeping it so beveled walls under
// ceilings or overhangs do not launch the entity into ceiling clamps that kick it backward.
func (th *Things) applySafeStepUp(cage *collisions.Cage, stepZ float64) {
	cage.ApplySafeStepUp(stepZ, th.volumes.ClampStaticCage)
}

// Compute updates the state of all managed entities by computing their active state and processing collisions.
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

// AddStaticThing adds a static object to the scene without physics or spatial partitioning.
func (th *Things) AddStaticThing(ent ithing.IThing) {
	th.statics = append(th.statics, ent)
}

// addThing registers ent in entities and inserts either all articulated segment sub-things or ent itself into tree.
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

// removeThing removes an IThing instance from the spatial tree (when tree-resident) and the entities map.
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
