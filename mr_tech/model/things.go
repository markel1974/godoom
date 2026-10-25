package model

import (
	"fmt"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

const solverJitter = 1e-6

// Things manages game objects, their spatial partitioning, and contact interactions within a simulation environment.
type Things struct {
	gScale           geometry.XYZ
	config           []*config.Thing
	volumes          *Volumes
	materials        *Materials
	tree             *aabb.Tree
	player           *ThingPlayer
	entities         map[uint64]IThing
	running          []IThing
	inactives        []IThing
	pending          []IThing
	ready            []IThing
	thinkers         []IThing
	statics          []IThing
	runningLen       int
	readyLen         int
	thinkersLen      int
	inactivesLen     int
	pendingLen       atomic.Int32
	hasPending       bool
	solverIterations int
	event            *ThingEvent
	updated          bool
}

// NewThings initializes and returns an instance of Things with the specified maximum number of things.
func NewThings(gScale geometry.XYZ, solverIterations int, cfg []*config.Thing, volumes *Volumes, materials *Materials) *Things {
	const defaultLen = 1024
	e := &Things{
		gScale:           gScale,
		solverIterations: solverIterations,
		tree:             aabb.NewTree(uint(len(cfg)*2), 4.0),
		entities:         make(map[uint64]IThing),
		running:          make([]IThing, defaultLen),
		ready:            make([]IThing, defaultLen),
		thinkers:         make([]IThing, defaultLen),
		pending:          make([]IThing, defaultLen),
		runningLen:       0,
		readyLen:         0,
		thinkersLen:      0,
		inactivesLen:     0,
		hasPending:       false,
		config:           cfg,
		volumes:          volumes,
		materials:        materials,
		event:            NewThingEvent(0, solverJitter),
		statics:          make([]IThing, 0, 16),
		updated:          false,
	}
	e.pendingLen.Store(0)
	const enableThingsCreation = true
	if enableThingsCreation {
		for _, ct := range cfg {
			if ct == nil {
				continue
			}
			volume, _ := e.volumes.QueryPoint(ct.Position.X, ct.Position.Y, ct.Position.Z)
			if volume == nil {
				//volume = e.volumes.LocateVolume(ct.Position.X, ct.Position.Y, ct.Position.Z)
				fmt.Printf("Warning can't find thing location at %f, %f, %f\n", ct.Position.X, ct.Position.Y, ct.Position.Z)
				continue
			}
			t2 := e.createThing(ct, volume)
			e.addThing(t2)
		}
	}
	return e
}

// GetMaterials returns a pointer to the Materials associated with the Things instance.
func (th *Things) GetMaterials() *Materials {
	return th.materials
}

// Len returns the number of elements in the container.
func (th *Things) Len() int {
	return len(th.ready)
}

// QueryCollisionCage checks collisions between the specified CollisionCage and others in the spatial structure (tree).
func (th *Things) QueryCollisionCage(lCage *CollisionCage) {
	lThing := lCage.GetThing()
	th.tree.QueryOverlaps(lCage, func(object aabb.IObject) bool {
		rThing := object.(IThing)
		if lThing == rThing {
			return false
		}
		rCage := rThing.GetCage()
		if lCage.HasSeen(rCage) {
			return false
		}
		lCage.Seen(rCage)
		rCage.Seen(lCage)
		lEntityL, deltaX, deltaY, deltaZ := lCage.TranslateCage(0, rCage)
		rThing.GetVolume().QueryOverlaps(lEntityL, func(rEnt aabb.IObject) bool {
			rFace := rEnt.(*Face)
			z := lCage.Translate(1, rFace, deltaX, deltaY, deltaZ)
			// 3. Query sul BVH del Self
			lThing.GetVolume().QueryOverlaps(z, func(lEnt aabb.IObject) bool {
				lFace := lEnt.(*Face)
				// Integrazione nel Manifold (Assicurati che AddFace accetti/conosca i delta
				// per allineare i vertici allo stesso sistema di riferimento durante il calcolo SAT)
				lCage.AddFace(rFace, lFace)
				return false
			})
			return false
		})
		lCage.Commit(rCage)
		return false
	})
}

// QueryMultiFrustum performs a spatial query against two frustums, invoking the callback for each intersected IAABB object.
func (th *Things) QueryMultiFrustum(rear *aabb.Frustum, front *aabb.Frustum, callback func(object aabb.IObject) bool) {
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
func (th *Things) GetPlayer() *ThingPlayer {
	return th.player
}

// SetPlayer assigns a ThingPlayer to the Things collection and integrates it into the entity management system.
func (th *Things) SetPlayer(p *ThingPlayer) {
	th.player = p
	th.addThing(p)
}

// GetGlobalScale retrieves the global scaling factor applied to all objects managed by the Things instance.
func (th *Things) GetGlobalScale() geometry.XYZ {
	return th.gScale
}

// GetTextures fetches the ITextures instance from the associated Materials object.
func (th *Things) GetTextures() textures.ITextures {
	return th.materials.GetTextures()
}

// GetActive returns the slice of active IThing objects and the current index of active things.
func (th *Things) GetActive() ([]IThing, int) {
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

// CreateThing creates a new IThing instance based on the provided Thing and adds it to the Things collection.
func (th *Things) createThing(ct *config.Thing, volume *Volume) IThing {
	const disableEnemies = false
	if disableEnemies {
		if ct.Kind == config.ThingEnemyDef {
			ct.Kind = config.ThingItemDef
		}
	}
	var thing IThing
	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = NewThingEnemy(th, ct, volume)
	case config.ThingWeaponDef:
		thing = NewThingItem(th, ct, volume)
	case config.ThingThrowableDef:
		thing = NewThingThrowable(th, ct, volume)
	case config.ThingKeyDef:
		thing = NewThingItem(th, ct, volume)
	case config.ThingInterfaceDef:
		thing = NewThingInterface(th, ct, volume)
	case config.ThingItemDef:
		thing = NewThingItem(th, ct, volume)
	default:
		thing = NewThingItem(th, ct, volume)
	}
	entity := thing.GetEntity()
	entity.SetOnGround(false)
	entity.MoveTo(ct.Position.X, ct.Position.Y, ct.Position.Z)
	return thing
}

// CreateThrowable creates a throwable object with specified position, angle, pitch, mass, radius, and speed, adding it to the pending list.
func (th *Things) CreateThrowable(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *Volume, pos geometry.XYZ, angle, pitch, speed float64) {
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
	throwable := th.createThing(dst, volume)
	throwable.GetEntity().SetOnGround(false)
	th.pending[slot] = throwable
	th.hasPending = true
}

// Compute updates the state of all managed entities by computing their active state and processing collisions.
func (th *Things) Compute(pX float64, pY float64, pZ float64) {
	if !th.updated || th.hasPending || th.inactivesLen > 0 {
		th.updated = true
		th.updateCollections()
	}
	th.computeActive(pX, pY, pZ)
	th.processCollision()
}

// updateCollections restructures and updates the internal collections of entities based on their current state and activity.
func (th *Things) updateCollections() {
	//Remove inactive entities
	if th.inactivesLen > 0 {
		for x := 0; x < th.inactivesLen; x++ {
			th.removeThing(th.inactives[x])
		}
		th.inactivesLen = 0
	}

	//Adding Pending entities
	if th.hasPending {
		for x := 0; x < int(th.pendingLen.Load()); x++ {
			th.addThing(th.pending[x])
		}
		th.pendingLen.Store(0)
		th.hasPending = false
	}

	//Rebuild ready and thinkers collections
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
}

// Compute updates the state of all IThing objects in the collection using the provided position coordinates (pX, pY).
func (th *Things) computeActive(pX float64, pY float64, pZ float64) {
	//Apply thinking
	if th.thinkersLen > 0 {
		th.event.SetStage(StageThinking)
		th.event.SetCoords(pX, pY, pZ)
		for x := 0; x < th.thinkersLen; x++ {
			t2 := th.thinkers[x]
			th.event.wg.Add(1)
			t2.PostMessage(th.event)
		}
		th.event.wg.Wait()
	}

	//Apply Physics
	th.runningLen = 0
	for x := 0; x < th.readyLen; x++ {
		t2 := th.ready[x]
		if !t2.IsActive() {
			th.inactives[th.inactivesLen] = t2
			th.inactivesLen++
			continue
		}
		if !t2.StagePrepare() {
			continue
		}
		th.running[th.runningLen] = t2
		th.runningLen++
	}
}

// Compute updates the state of all entities, processes collisions, resolves contacts, and integrates final positions.
func (th *Things) processCollision() {
	if th.runningLen == 0 {
		return
	}
	//th.event.SetStage(StageCompute)
	for x := 0; x < th.runningLen; x++ {
		t2 := th.running[x]
		// Collisioni Statiche (Mondo/Mappa)
		th.volumes.QueryCollisionCage(t2.GetCage())
		// Collisioni Dinamiche (Entità)
		th.QueryCollisionCage(t2.GetCage())
		//t2.StageCompute()
		//th.event.wg.Add(1)
		//t2.PostMessage(th.event)
	}
	//th.event.wg.Wait()
	for si := 0; si < th.solverIterations; si++ {
		//th.event.SetSolver(si, solverJitter)
		//th.event.SetStage(StageResolve)
		for x := 0; x < th.runningLen; x++ {
			t2 := th.running[x]
			//th.event.wg.Add(1)
			//t2.PostMessage(th.event)
			//TODO MULTIPLE ACTION INSIDE [SELF -> OTHER]
			t2.StageResolve(si, solverJitter)
		}
		//th.event.wg.Wait()
	}
	th.event.SetStage(StageApply)
	// PHYSYCS APPLY
	for x := 0; x < th.runningLen; x++ {
		t2 := th.running[x]
		th.event.wg.Add(1)
		t2.PostMessage(th.event)
	}
	th.event.wg.Wait()
	// COMMIT SPAZIALE E INTEGRAZIONE
	for x := 0; x < th.runningLen; x++ {
		t2 := th.running[x]
		th.tree.UpdateObject(t2)
	}
}

// AddStaticThing adds a static object to the scene without physics or spatial partitioning.
func (th *Things) AddStaticThing(ent IThing) {
	th.statics = append(th.statics, ent)
}

// addThing adds a new IThing to the entity collection, assigns it a unique identifier, and updates related structures.
func (th *Things) addThing(ent IThing) {
	th.entities[ent.GetEntity().GetId()] = ent
	if len(th.entities) > cap(th.running) {
		th.running = make([]IThing, len(th.entities)*4)
		th.inactives = make([]IThing, len(th.entities)*4)
		th.pending = make([]IThing, len(th.entities)*4)
	}
	th.tree.InsertObject(ent)
	ent.StartLoop()
}

// removeThing removes an IThing instance from the spatial tree and the entities map.
func (th *Things) removeThing(ent IThing) {
	th.tree.RemoveObject(ent)
	delete(th.entities, ent.GetEntity().GetId())
}
