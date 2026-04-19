package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Contact represents a physics collision contact point between two things.
// A and B are the things involved in the collision.
// Nx, Ny, Nz represent the normal vector of the contact.
// Penetration denotes the depth of the intersection between things.
// AccumulatedImpulse tracks the accumulated impulse applied during resolution.
type Contact struct {
	a, b               *physics.Entity
	nx, ny, nz         float64
	penetration        float64
	accumulatedImpulse float64
}

// Update updates the contact with new things, normal vector components, and penetration depth, resetting impulse to zero.
func (c *Contact) Update(a, b *physics.Entity, nx, ny, nz float64, penetration float64) {
	c.a = a
	c.b = b
	c.nx = nx
	c.ny = ny
	c.nz = nz
	c.penetration = penetration
	c.accumulatedImpulse = 0
}

// Resolve handles the collision response between two things by resolving penetration and applying impulses.
func (c *Contact) Resolve() {
	// 1. Velocità relativa
	rvX := c.a.GetVx() - c.b.GetVx()
	rvY := c.a.GetVy() - c.b.GetVy()
	rvZ := c.a.GetVz() - c.b.GetVz()
	// 2. Velocità lungo la normale
	velAlongNormal := (rvX * c.nx) + (rvY * c.ny) + (rvZ * c.nz)
	// Se si stanno già separando, il vincolo è soddisfatto
	if velAlongNormal > 0 {
		return
	}
	// BAUMGARTE STABILIZATION (Positional Bias)
	const slop = 0.05   // Tolleranza: permette agli oggetti di penetrare leggermente senza vibrare
	const percent = 0.2 // Corregge il 20% dell'errore ad ogni frame
	// Calcoliamo la velocità extra necessaria per spingerli fuori
	bias := math.Max(c.penetration-slop, 0.0) * percent
	// Se la velocità relativa (velAlongNormal) è già sufficiente a separarli, ignoriamo il bias
	// altrimenti lo aggiungiamo al calcolo dell'impulso
	e := math.Min(c.a.GetRestitution(), c.b.GetRestitution())
	invMassA := c.a.GetInvMass()
	invMassB := c.b.GetInvMass()
	totalInvMass := invMassA + invMassB
	// PREVIENI LA DIVISIONE PER ZERO (Entrambi gli oggetti statici)
	if totalInvMass <= 0.00001 {
		return
	}
	// Aggiungiamo il termine "bias" all'equazione di J
	j := (-(1.0 + e) * velAlongNormal) + bias
	j /= invMassA + invMassB
	// PASSAGGIO PROIETTIVO (PGS)
	// Conserviamo l'impulso calcolato e lo proiettiamo per non "tirare" i corpi
	oldImpulse := c.accumulatedImpulse
	c.accumulatedImpulse = math.Max(oldImpulse+j, 0.0)
	// L'impulso effettivo da applicare in questa singola iterazione
	jDelta := c.accumulatedImpulse - oldImpulse
	// 4. Applica il delta di velocità
	impulseX := jDelta * c.nx
	impulseY := jDelta * c.ny
	impulseZ := jDelta * c.nz
	c.a.AddV(impulseX*invMassA, impulseY*invMassA, impulseZ*invMassA)
	c.b.SubV(impulseX*invMassB, impulseY*invMassB, impulseZ*invMassB)
}

// Things manages game objects, their spatial partitioning, and contact interactions within a simulation environment.
type Things struct {
	gScale       float64
	config       []*config.Thing
	volumes      *Volumes
	animations   *Animations
	tree         *physics.AABBTree
	entities     map[int]IThing
	identifier   int
	moving       []IThing
	movingLen    int
	contacts     []Contact
	contactsLen  int
	activeIdx    int
	activeThings []IThing
	event        *ThingEvent
}

// NewThings initializes and returns an instance of Things with the specified maximum number of things.
func NewThings(gScale float64, cfg []*config.Thing, volumes *Volumes, animations *Animations) *Things {
	e := &Things{
		gScale:       gScale,
		tree:         physics.NewAABBTree(uint(len(cfg)*2), 4.0),
		entities:     make(map[int]IThing),
		identifier:   0,
		movingLen:    0,
		contacts:     make([]Contact, 1024),
		contactsLen:  0,
		config:       cfg,
		volumes:      volumes,
		animations:   animations,
		activeIdx:    0,
		activeThings: make([]IThing, 1024),
		event:        NewThingEvent(),
	}
	for _, ct := range cfg {
		if _, err := e.createThing(ct); err != nil {
			fmt.Println("Warning: ", err)
		}
	}
	return e
}

// GetGlobalScale retrieves the global scaling factor applied to all objects managed by the Things instance.
func (th *Things) GetGlobalScale() float64 {
	return th.gScale
}

// GetVolumes returns the Volumes instance managed by the Things object.
func (th *Things) GetVolumes() *Volumes {
	return th.volumes
}

// GetTextures fetches the ITextures instance from the associated Animations object.
func (th *Things) GetTextures() textures.ITextures {
	return th.animations.GetTextures()
}

// GetActive returns the slice of active IThing objects and the current index of active things.
func (th *Things) GetActive() ([]IThing, int) {
	return th.activeThings, th.activeIdx
}

// QueryRay performs a raycast query within the spatial tree, invoking the callback for each intersected object.
// Parameters:
// - oX, oY, oZ: Origin coordinates of the ray.
// - dirX, dirY, dirZ: Direction vector of the ray.
// - maxDistance: Maximum distance the ray can travel.
// - callback: Function invoked for each intersected object, receives the object and its distance as arguments.
func (th *Things) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object physics.IAABB, distance float64) (float64, bool)) {
	th.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// CreateThing creates a new IThing instance based on the provided Thing and adds it to the Things collection.
func (th *Things) createThing(ct *config.Thing) (IThing, error) {
	volume := th.volumes.LocateVolume(ct.Position.X, ct.Position.Y, ct.Position.Z)
	if volume == nil {
		return nil, fmt.Errorf("can't find thing location at %f, %f, %f", ct.Position.X, ct.Position.Y, ct.Position.Z)
	}

	const disableEnemies = false
	if disableEnemies {
		if ct.Kind == config.ThingEnemyDef {
			ct.Kind = config.ThingItemDef
		}
	}

	var thing IThing
	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = NewThingEnemy(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	case config.ThingWeaponDef:
		thing = NewThingItem(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	case config.ThingBulletDef:
		thing = NewThingItem(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	case config.ThingKeyDef:
		thing = NewThingItem(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	case config.ThingItemDef:
		thing = NewThingItem(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	default:
		thing = NewThingItem(th, ct, th.animations.GetAnimation(ct.Animation), volume)
	}
	return thing, nil
}

// CreateBullet creates a new bullet in the specified sector at the given position (x, y) with the given angle.
func (th *Things) CreateBullet(volume *Volume, pos geometry.XYZ, angle, pitch, mass, radius, speed float64) {
	//TODO now is an hack
	const ammoIndex = 2
	if len(th.config) <= ammoIndex {
		return
	}
	c := th.config[ammoIndex]
	id := utils.NextUUId()
	cfg := config.NewConfigThing(id, pos, angle, config.ThingBulletDef, c.Mass, c.Radius, c.Radius, speed, c.Animation)
	NewThingBullet(th, cfg, th.animations.GetAnimation(cfg.Animation), volume, pitch)
}

// UpdateThing updates the position of the specified IThing in the entity manager and updates its spatial location in the tree.
func (th *Things) UpdateThing(thing IThing, px float64, py float64, pz float64) {
	ent := thing.GetEntity()
	eRadius := ent.GetWidth() / 2.0
	ent.MoveTo(px-eRadius, py-eRadius, pz)
	th.tree.UpdateObject(thing)
}

// AddThing adds a new IThing to the entity collection, assigns it a unique identifier, and updates related structures.
func (th *Things) AddThing(ent IThing) {
	th.entities[th.identifier] = ent
	ent.SetIdentifier(th.identifier)
	th.identifier++
	if len(th.entities) > cap(th.moving) {
		th.moving = make([]IThing, len(th.entities)*2)
	}
	th.tree.InsertObject(ent)
	ent.StartLoop()
}

// RemoveThing removes the specified IThing entity from the spatial tree and the things map in Things.
func (th *Things) RemoveThing(ent IThing) {
	th.tree.RemoveObject(ent)
	delete(th.entities, ent.GetIdentifier())
}

func (th *Things) Compute(pX float64, pY float64, pZ float64) {
	th.computeActive(pX, pY, pZ)
	th.processCollision()
}

// Compute updates the state of all IThing objects in the collection using the provided position coordinates (pX, pY).
func (th *Things) computeActive(pX float64, pY float64, pZ float64) {
	th.activeIdx = 0

	th.event.SetStage(StageThinking)
	th.event.SetCoords(pX, pY, pZ)

	for _, t2 := range th.entities {
		if !t2.IsActive() {
			th.RemoveThing(t2)
			continue
		}
		th.event.wg.Add(1)
		t2.PostMessage(th.event)

		//t.Compute(pX, pY, pZ)
		if th.activeIdx >= len(th.activeThings) {
			newThings := make([]IThing, len(th.activeThings)*2)
			copy(newThings, th.activeThings)
			th.activeThings = newThings
		}
		th.activeThings[th.activeIdx] = t2
		th.activeIdx++
	}

	th.event.wg.Wait()
}

// Compute updates the state of all entities, processes collisions, resolves contacts, and integrates final positions.
func (th *Things) processCollision() {
	th.movingLen = 0
	th.contactsLen = 0

	//ACQUIRING
	for _, thing := range th.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			th.tree.UpdateObject(thing)
			th.moving[th.movingLen] = thing
			th.movingLen++
		}
	}

	if th.movingLen == 0 {
		return
	}

	th.event.SetStage(StageCollision)
	// DETECTION (Costruzione del Jacobiano)
	for x := 0; x < th.movingLen; x++ {
		thing := th.moving[x]
		ent := thing.GetEntity()

		th.tree.QueryOverlaps(thing, func(object physics.IAABB) bool {
			otherThing, ok := object.(IThing)
			if !ok || otherThing == thing {
				return false
			}
			otherEnt := otherThing.GetEntity()
			// Tie-breaker 3D
			otherIsMoving := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
			if otherIsMoving && thing.GetIdentifier() > otherThing.GetIdentifier() {
				return false
			}
			normX, normY, normZ, minPenetration, hasCollision := ent.ComputeCollision(otherEnt)
			if !hasCollision {
				return false
			}
			if th.contactsLen >= len(th.contacts) {
				newContacts := make([]Contact, len(th.contacts)*2)
				copy(newContacts, th.contacts)
				th.contacts = newContacts
			}
			th.contacts[th.contactsLen].Update(ent, otherEnt, normX, normY, normZ, minPenetration)
			th.contactsLen++
			//TODO MESSAGES!!!!!
			thing.OnCollide(otherThing)
			otherThing.OnCollide(thing)
			return false
		})
	}

	// RESOLUTION (Il Solver PGS)
	const solverIterations = 4
	for i := 0; i < solverIterations; i++ {
		for c := 0; c < th.contactsLen; c++ {
			th.contacts[c].Resolve()
		}
	}

	th.event.SetStage(StagePhysics)
	// PHYSYCS APPLY
	for x := 0; x < th.movingLen; x++ {
		t2 := th.moving[x]
		th.event.wg.Add(1)
		t2.PostMessage(th.event)
		//tx, ty, tz := m.PhysicsApply()
		//th.UpdateThing(m, tx, ty, tz)
	}
	th.event.wg.Wait()

	// COMMIT SPAZIALE E INTEGRAZIONE
	for x := 0; x < th.movingLen; x++ {
		t2 := th.moving[x]
		th.tree.UpdateObject(t2)
		tx, ty, tz := t2.GetPosition()
		th.UpdateThing(t2, tx, ty, tz)
	}
}
