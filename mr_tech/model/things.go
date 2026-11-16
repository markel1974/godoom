package model

import (
	"fmt"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
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
	entities          map[uint64]volumes.IThing
	pending           []volumes.IThing
	thinkers          []volumes.IThing
	statics           []volumes.IThing
	thinkersLen       int
	pendingLen        atomic.Int32
	hasPending        bool
	event             *volumes.ThingEvent
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
		entities:          make(map[uint64]volumes.IThing),
		thinkers:          make([]volumes.IThing, defaultLen),
		pending:           make([]volumes.IThing, defaultLen),
		thinkersLen:       0,
		hasPending:        false,
		config:            cfg,
		volumes:           vols,
		materials:         materials,
		event:             volumes.NewThingEvent(0, solverJitter),
		statics:           make([]volumes.IThing, 0, 16),
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
	th.volumes.QueryCollisionCage(cage, th.maxOverlapsWorld)
	th.queryCollisionCage(cage, th.maxOverlapsThings, wake)
}

func (th *Things) queryCollisionCage(lCage *collisions.Cage, maxOverlaps int, wake func(*collisions.Solver)) {
	lThing := lCage.GetObject().(volumes.IThing)
	th.tree.QueryOverlaps(lCage, func(object aabb.IObject) bool {
		rThing := object.(volumes.IThing)
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
func (th *Things) Create(ct *config.Thing, volume *volumes.Volume) volumes.IThing {
	const disableEnemies = false
	if disableEnemies {
		if ct.Kind == config.ThingEnemyDef {
			ct.Kind = config.ThingItemDef
		}
	}
	var thing volumes.IThing
	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = things.NewEnemy(th, ct, volume)
	case config.ThingWeaponDef:
		thing = things.NewItem(th, ct, volume)
	case config.ThingThrowableDef:
		thing = things.NewThingThrowable(th, ct, volume)
	case config.ThingKeyDef:
		thing = things.NewItem(th, ct, volume)
	case config.ThingInterfaceDef:
		thing = things.NewUI(th, ct)
	case config.ThingItemDef:
		thing = things.NewItem(th, ct, volume)
	default:
		thing = things.NewItem(th, ct, volume)
	}
	entity := thing.GetEntity()
	entity.MoveTo(ct.Position.X, ct.Position.Y, ct.Position.Z)
	return thing
}

// CreateThrowable creates a throwable object with specified position, angle, pitch, mass, radius, and speed, adding it to the pending list.
func (th *Things) CreateThrowable(self volumes.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64) {
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
	th.pending[slot] = throwable
	th.hasPending = true
}

// CreateDetached wraps vol as an autonomous 6-DoF rigid-body Thing, queues it into the scene, and returns it.
func (th *Things) CreateDetached(vol *volumes.Volume, location *volumes.Volume, pos geometry.XYZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64, collisionGroup uint64) volumes.IThing {
	slot := th.pendingLen.Add(1) - 1
	if slot >= int32(len(th.pending)) {
		fmt.Println("max slot reached!")
		return nil
	}
	detached := things.NewDetached(th, vol, location, utils.NextUUId(), pos, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz, collisionGroup)
	th.pending[slot] = detached
	th.hasPending = true
	return detached
}

// AddJoint registers an active bilateral PGS joint constraint with the coordinator.
func (th *Things) AddJoint(joint *collisions.JointConstraint) {
	th.coordinator.AddJoint(joint)
}

// StagePrepare resets staging state before collecting ready things and thinkers.
func (th *Things) StagePrepare() {
	th.coordinator.Reset()
	th.thinkersLen = 0
}

// StageAdd queues t2 as ready and tracks it as a thinker when it supports thinking.
func (th *Things) StageAdd(t2 volumes.IThing) {
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
	if removables := th.coordinator.Removables(); len(removables) > 0 {
		for _, s := range removables {
			th.removeThing(s.GetCage().GetObject().(volumes.IThing))
		}
		th.coordinator.ClearRemovables()
	}
	if th.hasPending {
		for x := 0; x < int(th.pendingLen.Load()); x++ {
			th.addThing(th.pending[x])
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	// Apply Thinking
	if th.thinkersLen > 0 {
		th.event.SetStage(volumes.StageThinking)
		th.event.SetCoords(pX, pY, pZ)
		for x := 0; x < th.thinkersLen; x++ {
			t2 := th.thinkers[x]
			th.event.Add()
			t2.PostMessage(th.event)
		}
		th.event.Wait()
	}

	// Flush and stage any ragdoll/detached/throwable bodies spawned during Thinking so joints start synchronized on tick 0.
	if th.hasPending {
		for x := 0; x < int(th.pendingLen.Load()); x++ {
			item := th.pending[x]
			th.addThing(item)
			th.StageAdd(item)
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	th.coordinator.Step(th.queryContactsFn, th.clampStaticFn)

	for _, s := range th.coordinator.Running() {
		th.tree.UpdateObject(s.GetCage().GetObject().(aabb.IObject))
	}
}

// AddStaticThing adds a static object to the scene without physics or spatial partitioning.
func (th *Things) AddStaticThing(ent volumes.IThing) {
	th.statics = append(th.statics, ent)
}

// addThing adds a new IThing to the entity collection, assigns it a unique identifier, and updates related structures.
func (th *Things) addThing(ent volumes.IThing) {
	th.entities[ent.GetEntity().GetId()] = ent
	if len(th.entities) > cap(th.pending) {
		newCap := len(th.entities) * 4
		th.thinkers = make([]volumes.IThing, newCap)
		th.pending = make([]volumes.IThing, newCap)
		th.coordinator.EnsureCapacity(newCap)
	}
	th.tree.InsertObject(ent)
	ent.StartLoop()
}

// removeThing removes an IThing instance from the spatial tree and the entities map.
func (th *Things) removeThing(ent volumes.IThing) {
	th.tree.RemoveObject(ent)
	delete(th.entities, ent.GetEntity().GetId())
}
