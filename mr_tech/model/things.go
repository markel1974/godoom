package model

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
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
	entities          map[uint64]vertices.IThing
	pending           []vertices.IThing
	thinkers          []vertices.IThing
	statics           []vertices.IThing
	thinkersLen       int
	pendingLen        atomic.Int32
	hasPending        bool
	pendingStage      []vertices.IThing
	pendingStageMu    sync.Mutex
	pendingRemove     []vertices.IThing
	pendingRemoveMu   sync.Mutex
	stageEpoch        uint64
	event             *vertices.ThingEvent
	cfg               []*config.Thing
	maxOverlapsWorld  int
	maxOverlapsThings int
	coordinator       *collisions.Coordinator
	queryContactsFn   func(cage *collisions.Cage, wake func(*collisions.Solver))
	clampStaticFn     func(cage *collisions.Cage)
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
		entities:          make(map[uint64]vertices.IThing),
		thinkers:          make([]vertices.IThing, defaultLen),
		pending:           make([]vertices.IThing, defaultLen),
		pendingStage:      make([]vertices.IThing, 0, 64),
		thinkersLen:       0,
		hasPending:        false,
		config:            cfg,
		volumes:           vols,
		materials:         materials,
		event:             vertices.NewThingEvent(0, solverJitter),
		statics:           make([]vertices.IThing, 0, 16),
		coordinator:       collisions.NewCoordinator(solverIterations, defaultLen),
	}
	e.queryContactsFn = e.querySolverContacts
	e.clampStaticFn = vols.ClampStaticCage
	e.pendingLen.Store(0)

	return e
}

// Setup initializes the necessary configurations or resources required for the Things object to function properly.
func (th *Things) Setup() error {
	for _, ct := range th.cfg {
		if ct == nil {
			continue
		}
		volume, _ := th.volumes.QueryPoint(ct.Position.X, ct.Position.Y, ct.Position.Z)
		if volume == nil && ct.Kind == config.ThingDoorDef {
			if vols := th.volumes.GetVolumes(); len(vols) > 0 {
				volume = vols[0]
			}
		}
		if volume == nil {
			fmt.Printf("Warning can't find thing location at %f, %f, %f\n", ct.Position.X, ct.Position.Y, ct.Position.Z)
			continue
		}
		t2 := th.Create(ct, volume)
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

// Len returns the number of staged elements in the coordinator.
func (th *Things) Len() int {
	return th.coordinator.StagedLen()
}

// QueryCollisionCage checks collisions between the specified CollisionCage and others in the spatial structure (tree).
func (th *Things) QueryCollisionCage(lCage *collisions.Cage, maxOverlaps int) {
	th.queryCollisionCage(lCage, maxOverlaps, nil)
}

func (th *Things) querySolverContacts(cage *collisions.Cage, wake func(*collisions.Solver)) {
	if s := cage.GetSolver(); s == nil || !s.IgnoresStatic() {
		th.volumes.QueryCollisionCage(cage, th.maxOverlapsWorld)
	}
	th.queryCollisionCage(cage, th.maxOverlapsThings, wake)
}

func (th *Things) queryCollisionCage(lCage *collisions.Cage, maxOverlaps int, wake func(*collisions.Solver)) {
	lThing := lCage.GetObject().(vertices.IThing)
	th.tree.QueryOverlaps(lCage, func(object aabb.IObject) bool {
		rThing := object.(vertices.IThing)
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

// Create creates a new IThing instance based on the provided Thing and adds it to the Things collection.
func (th *Things) Create(ct *config.Thing, volume *volumes.Volume) vertices.IThing {
	const disableEnemies = false
	if disableEnemies {
		if ct.Kind == config.ThingEnemyDef {
			ct.Kind = config.ThingItemDef
		}
	}
	var thing vertices.IThing
	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = things.NewEnemy(th, ct, volume)
	case config.ThingWeaponDef:
		thing = things.NewItem(th, ct, volume)
	case config.ThingThrowableDef:
		thing = things.NewThingThrowable(th, ct, volume)
	case config.ThingKeyDef:
		thing = things.NewKey(th, ct, volume)
	case config.ThingDoorDef:
		thing = things.NewDoor(th, ct, volume)
	case config.ThingInterfaceDef:
		thing = things.NewUI(th, ct)
	case config.ThingItemDef:
		thing = things.NewItem(th, ct, volume)
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
func (th *Things) CreateThrowable(self vertices.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64) {
	if len(th.config) <= throwableIndex {
		return
	}
	src := th.config[throwableIndex]
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
	throwable := th.Create(dst, volume)
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
func (th *Things) StagePart(part vertices.IThing) {
	if part == nil {
		return
	}
	th.pendingStageMu.Lock()
	th.pendingStage = append(th.pendingStage, part)
	th.pendingStageMu.Unlock()
}

// StageRemove queues a deactivated Thing for safe removal from Things.tree and Things.entities.
func (th *Things) StageRemove(thing vertices.IThing) {
	if thing == nil {
		return
	}
	th.pendingRemoveMu.Lock()
	th.pendingRemove = append(th.pendingRemove, thing)
	th.pendingRemoveMu.Unlock()
}

func (th *Things) flushPendingRemovals() {
	th.pendingRemoveMu.Lock()
	if len(th.pendingRemove) > 0 {
		for _, item := range th.pendingRemove {
			th.removeThing(item)
		}
		th.pendingRemove = th.pendingRemove[:0]
	}
	th.pendingRemoveMu.Unlock()

	if removablesLen := th.coordinator.RemovablesLen(); removablesLen > 0 {
		for x := 0; x < removablesLen; x++ {
			s := th.coordinator.RemovablesAt(x)
			th.removeThing(s.GetCage().GetObject().(vertices.IThing))
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
func (th *Things) StageAdd(t2 vertices.IThing) {
	if sub, ok := t2.(vertices.ISubPart); ok && !sub.IsPhysicsFree() {
		if owner := sub.GetOwner(); owner != nil {
			th.stageOne(owner)
		}
		return
	}
	th.stageOne(t2)
}

func (th *Things) stageOne(t2 vertices.IThing) {
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
	collisions.ApplySafeStepUp(cage, stepZ, th.volumes.ClampStaticCage)
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
		th.event.SetStage(vertices.StageThinking)
		th.event.SetCoords(pX, pY, pZ)
		for x := 0; x < th.thinkersLen; x++ {
			t2 := th.thinkers[x]
			th.event.Add()
			t2.PostMessage(th.event)
		}
		th.event.Wait()
	}

	// Flush and stage any throwables or newly released ragdoll/detached parts from Thinking on tick 0.
	if th.hasPending {
		for x := 0; x < int(th.pendingLen.Load()); x++ {
			item := th.pending[x]
			th.addThing(item)
			th.StageAdd(item)
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	if len(th.pendingStage) > 0 {
		for _, part := range th.pendingStage {
			th.StageAdd(part)
		}
		th.pendingStage = th.pendingStage[:0]
	}

	th.coordinator.Step(th.queryContactsFn, th.clampStaticFn)
	th.flushPendingRemovals()

	if runningLen := th.coordinator.RunningLen(); runningLen > 0 {
		for rIdx := 0; rIdx < runningLen; rIdx++ {
			s := th.coordinator.RunningAt(rIdx)
			obj := s.GetCage().GetObject()
			if owner, ok := obj.(vertices.IPartOwner); ok {
				parts := owner.GetPartLen()
				if parts > 0 {
					owner.SyncAttachedParts()
					for x := 0; x < parts; x++ {
						p := owner.GetPartAt(x)
						if sub, sOk := p.(vertices.ISubPart); sOk && !sub.IsPhysicsFree() {
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
func (th *Things) AddStaticThing(ent vertices.IThing) {
	th.statics = append(th.statics, ent)
}

// addThing registers ent in entities and inserts either all articulated segment sub-things or ent itself into tree.
func (th *Things) addThing(ent vertices.IThing) {
	th.entities[ent.GetEntity().GetId()] = ent
	var parts int
	owner, ok := ent.(vertices.IPartOwner)
	if ok {
		parts = owner.GetPartLen()
	}
	needed := len(th.entities) + parts
	if needed > cap(th.pending) {
		newCap := needed * 4
		th.thinkers = make([]vertices.IThing, newCap)
		th.pending = make([]vertices.IThing, newCap)
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
func (th *Things) removeThing(ent vertices.IThing) {
	if ent == nil {
		return
	}
	id := ent.GetEntity().GetId()
	if _, ok := th.entities[id]; !ok {
		return
	}
	if owner, ok := ent.(vertices.IPartOwner); !ok || owner.GetPartLen() == 0 {
		th.tree.RemoveObject(ent)
	}
	delete(th.entities, id)
}
