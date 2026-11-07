package model

import (
	"fmt"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/collisions"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

const solverJitter = 1e-6

// Things manages game objects, their spatial partitioning, and contact interactions within a simulation environment.
type Things struct {
	config           []*config.Thing
	volumes          *Volumes
	materials        *materials.Materials
	tree             *aabb.Tree
	player           *things.Player
	entities         map[uint64]volumes.IThing
	running          []volumes.IThing
	removables       []volumes.IThing
	pending          []volumes.IThing
	ready            []volumes.IThing
	thinkers         []volumes.IThing
	statics          []volumes.IThing
	readyLen         int
	thinkersLen      int
	removablesLen    int
	pendingLen       atomic.Int32
	hasPending       bool
	solverIterations int
	event            *volumes.ThingEvent
	cfg              []*config.Thing
	maxOverlaps      int
}

// NewThings initializes and returns an instance of Things with the specified maximum number of things.
func NewThings(maxOverlaps int, solverIterations int, cfg []*config.Thing, vols *Volumes, materials *materials.Materials) *Things {
	const defaultLen = 1024
	if maxOverlaps <= 0 {
		maxOverlaps = 100
	}
	if solverIterations <= 0 {
		solverIterations = 10
	}
	e := &Things{
		maxOverlaps:      maxOverlaps,
		solverIterations: solverIterations,
		cfg:              cfg,
		tree:             aabb.NewTree(uint(len(cfg)*2), 4.0),
		entities:         make(map[uint64]volumes.IThing),
		running:          make([]volumes.IThing, defaultLen),
		ready:            make([]volumes.IThing, defaultLen),
		thinkers:         make([]volumes.IThing, defaultLen),
		pending:          make([]volumes.IThing, defaultLen),
		readyLen:         0,
		thinkersLen:      0,
		removablesLen:    0,
		hasPending:       false,
		config:           cfg,
		volumes:          vols,
		materials:        materials,
		event:            volumes.NewThingEvent(0, solverJitter),
		statics:          make([]volumes.IThing, 0, 16),
	}
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

	th.readyLen = 0
	th.thinkersLen = 0
	for _, t2 := range th.entities {
		th.ready[th.readyLen] = t2
		th.readyLen++
		if t2.HasThinking() {
			th.thinkers[th.thinkersLen] = t2
			th.thinkersLen++
		}
	}
	return nil
}

// GetMaterials returns a pointer to the Materials associated with the Things instance.
func (th *Things) GetMaterials() *materials.Materials {
	return th.materials
}

// Len returns the number of elements in the container.
func (th *Things) Len() int {
	return len(th.ready)
}

// QueryCollisionCage checks collisions between the specified CollisionCage and others in the spatial structure (tree).
func (th *Things) QueryCollisionCage(lCage *collisions.Cage, maxOverlaps int) {
	lThing := lCage.GetThing()
	counter := 0
	th.tree.QueryOverlaps(lCage, func(object aabb.IObject) bool {
		rThing := object.(volumes.IThing)
		if lThing == rThing {
			return false
		}
		rCage := rThing.GetCage().(*collisions.Cage)
		if lCage.HasSeen(rCage) {
			return false
		}
		lCage.Seen(rCage)
		rCage.Seen(lCage)
		lEntityL, deltaX, deltaY, deltaZ := lCage.TranslateCage(0, rCage)
		rThing.VerticesGetVolume().QueryOverlaps(lEntityL, func(rEnt aabb.IObject) bool {
			rFace := rEnt.(*volumes.Face)
			z := lCage.Translate(1, rFace, deltaX, deltaY, deltaZ)
			// Query sul BVH del Self
			lThing.VerticesGetVolume().QueryOverlaps(z, func(lEnt aabb.IObject) bool {
				lFace := lEnt.(*volumes.Face)
				// Integrazione nel Manifold (verificare che AddFace accetti/conosca i delta
				// per allineare i vertici allo stesso sistema di riferimento durante il calcolo SAT)
				lCage.AddFace(rFace, lFace)
				counter++
				if counter > maxOverlaps {
					return true
				}
				return false
			})
			if counter > maxOverlaps {
				return true
			}
			return false
		})
		lCage.Commit(rCage)
		if counter > maxOverlaps {
			return true
		}
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

// GetActive returns the slice of active IThing objects and the current index of active things.
func (th *Things) GetActive() ([]volumes.IThing, int) {
	return th.ready, th.readyLen
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
	entity := thing.VerticesGetEntity()
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

// StagePrepare resets staging state before collecting ready things and thinkers.
func (th *Things) StagePrepare() {
	th.readyLen = 0
	th.thinkersLen = 0
}

// StageAdd queues t2 as ready and tracks it as a thinker when it supports thinking.
func (th *Things) StageAdd(t2 volumes.IThing) {
	th.ready[th.readyLen] = t2
	th.readyLen++
	if t2.HasThinking() {
		th.thinkers[th.thinkersLen] = t2
		th.thinkersLen++
	}
}

// StageCommit stages pending changes for commit.
func (th *Things) StageCommit() {

}

// Compute updates the state of all managed entities by computing their active state and processing collisions.
func (th *Things) Compute(pX float64, pY float64, pZ float64) {
	if th.removablesLen > 0 {
		for x := 0; x < th.removablesLen; x++ {
			th.removeThing(th.removables[x])
		}
		th.removablesLen = 0
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

	//Apply Physics
	runningLen := 0
	for x := 0; x < th.readyLen; x++ {
		t2 := th.ready[x]
		if !t2.IsActive() {
			th.removables[th.removablesLen] = t2
			th.removablesLen++
			continue
		}
		t2.StagePrepare()
		th.running[runningLen] = t2
		runningLen++
	}

	if runningLen == 0 {
		return
	}

	for x := 0; x < runningLen; x++ {
		t2 := th.running[x]
		cage := t2.GetCage().(*collisions.Cage)
		th.volumes.QueryCollisionCage(cage, th.maxOverlaps) //world collision
		th.QueryCollisionCage(cage, th.maxOverlaps)         //things collision
	}

	for x := 0; x < runningLen; x++ {
		t2 := th.running[x]
		cage := t2.GetCage().(*collisions.Cage)
		t2.VerticesGetEntity().SetOnGround(cage.HasFloor())
	}

	//fmt.Println("Ready:", th.readyLen, "Thinkers:", th.thinkersLen, "Running:", runningLen)
	//th.event.wg.Wait()
	for si := 0; si < th.solverIterations; si++ {
		for x := 0; x < runningLen; x++ {
			t2 := th.running[x]
			//th.event.wg.Add(1)
			//t2.PostMessage(th.event)
			//TODO MULTIPLE ACTION INSIDE [SELF -> OTHER]
			t2.StageResolve()
		}
		//th.event.wg.Wait()
	}

	// PHYSYCS APPLY
	for x := 0; x < runningLen; x++ {
		t2 := th.running[x]
		t2.StageApply()
	}

	// COMMIT SPAZIALE E INTEGRAZIONE
	for x := 0; x < runningLen; x++ {
		t2 := th.running[x]
		th.tree.UpdateObject(t2)
	}
}

// AddStaticThing adds a static object to the scene without physics or spatial partitioning.
func (th *Things) AddStaticThing(ent volumes.IThing) {
	th.statics = append(th.statics, ent)
}

// addThing adds a new IThing to the entity collection, assigns it a unique identifier, and updates related structures.
func (th *Things) addThing(ent volumes.IThing) {
	th.entities[ent.VerticesGetEntity().GetId()] = ent
	if len(th.entities) > cap(th.running) {
		th.running = make([]volumes.IThing, len(th.entities)*4)
		th.removables = make([]volumes.IThing, len(th.entities)*4)
		th.pending = make([]volumes.IThing, len(th.entities)*4)
	}
	th.tree.InsertObject(ent)
	ent.StartLoop()
}

// removeThing removes an IThing instance from the spatial tree and the entities map.
func (th *Things) removeThing(ent volumes.IThing) {
	th.tree.RemoveObject(ent)
	delete(th.entities, ent.VerticesGetEntity().GetId())
}
