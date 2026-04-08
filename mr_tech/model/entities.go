package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// minMovement defines the minimum threshold for movement to be considered significant in physics calculations.
const minMovement = 0.001

// Contact represents a physics collision contact point between two entities.
// A and B are the entities involved in the collision.
// Nx, Ny, Nz represent the normal vector of the contact.
// Penetration denotes the depth of the intersection between entities.
// AccumulatedImpulse tracks the accumulated impulse applied during resolution.
type Contact struct {
	a, b               *physics.Entity
	nx, ny, nz         float64
	penetration        float64
	accumulatedImpulse float64
}

// Update updates the contact with new entities, normal vector components, and penetration depth, resetting impulse to zero.
func (c *Contact) Update(a, b *physics.Entity, nx, ny, nz float64, penetration float64) {
	c.a = a
	c.b = b
	c.nx = nx
	c.ny = ny
	c.nz = nz
	c.penetration = penetration
	c.accumulatedImpulse = 0
}

// Resolve handles the collision response between two entities by resolving penetration and applying impulses.
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

// Entities manages game objects, their spatial partitioning, and contact interactions within a simulation environment.
type Entities struct {
	tree        *physics.AABBTree
	entities    map[int]IThing
	identifier  int
	moving      []IThing
	movingLen   int
	contacts    []Contact
	contactsLen int
}

// NewEntities initializes and returns an instance of Entities with the specified maximum number of entities.
func NewEntities(maxEntities uint) *Entities {
	return &Entities{
		tree:        physics.NewAABBTree(maxEntities),
		entities:    make(map[int]IThing),
		identifier:  0,
		movingLen:   0,
		contacts:    make([]Contact, 1024),
		contactsLen: 0,
	}
}

// UpdateThing updates the position of the specified IThing in the entity manager and updates its spatial location in the tree.
func (em *Entities) UpdateThing(thing IThing, px float64, py float64, pz float64) {
	ent := thing.GetEntity()
	eRadius := ent.GetWidth() / 2.0
	// NOTA: Se hai aggiornato physics.Entity per tenere traccia della Z internamente,
	// qui dovrai chiamare la versione 3D, es: ent.MoveTo3d(px-eRadius, py-eRadius, pz)
	ent.MoveTo(px-eRadius, py-eRadius, pz)

	em.tree.UpdateObject(thing)
}

// AddThing adds a new IThing to the entity collection, assigns it a unique identifier, and updates related structures.
func (em *Entities) AddThing(ent IThing) {
	em.entities[em.identifier] = ent
	ent.SetIdentifier(em.identifier)
	em.identifier++
	if len(em.entities) > cap(em.moving) {
		em.moving = make([]IThing, len(em.entities)*2)
	}
	em.tree.InsertObject(ent)
}

// RemoveThing removes the specified IThing entity from the spatial tree and the entities map in Entities.
func (em *Entities) RemoveThing(ent IThing) {
	em.tree.RemoveObject(ent)
	delete(em.entities, ent.GetIdentifier())
}

// Compute updates the state of all entities, processes collisions, resolves contacts, and integrates final positions.
func (em *Entities) Compute() {
	em.movingLen = 0
	em.contactsLen = 0

	for _, thing := range em.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			em.tree.UpdateObject(thing)
			em.moving[em.movingLen] = thing
			em.movingLen++
		}
	}

	if em.movingLen == 0 {
		return
	}

	// DETECTION (Costruzione del Jacobiano)
	for x := 0; x < em.movingLen; x++ {
		thing := em.moving[x]
		ent := thing.GetEntity()

		em.tree.QueryOverlaps(thing, func(object physics.IAABB) bool {
			// 1. CAST SAFE e check di auto-collisione
			otherThing, ok := object.(IThing)
			if !ok || otherThing == thing {
				return false
			}
			otherEnt := otherThing.GetEntity()

			// 2. Tie-breaker 3D
			otherIsMoving := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
			if otherIsMoving && thing.GetIdentifier() > otherThing.GetIdentifier() {
				return false
			}

			x1Min, x1Max := ent.GetXRange()
			x2Min, x2Max := otherEnt.GetXRange()
			y1Min, y1Max := ent.GetYRange()
			y2Min, y2Max := otherEnt.GetYRange()

			// SAT: Collisione AABB Planare Veloce
			if x1Max > x2Min && x1Min < x2Max && y1Max > y2Min && y1Min < y2Max {
				z1Min, z1Max := ent.GetZRange()
				z2Min, z2Max := otherEnt.GetZRange()

				// Supporto Swept Z per il Continuous Collision Detection verticale
				if math.Abs(ent.GetVz()) >= ent.GetGForce() {
					z1Min, z1Max = ent.GetSweptZRange()
				}
				if math.Abs(otherEnt.GetVz()) >= otherEnt.GetGForce() {
					z2Min, z2Max = otherEnt.GetSweptZRange()
				}

				if z1Max > z2Min && z1Min < z2Max {
					pX1 := x1Max - x2Min
					pX2 := x2Max - x1Min
					pY1 := y1Max - y2Min
					pY2 := y2Max - y1Min
					pZ1 := z1Max - z2Min
					pZ2 := z2Max - z1Min

					minPenetration := pX1
					var normX, normY, normZ float64 = -1, 0, 0

					// Troviamo l'asse di minima compenetrazione
					if pX2 < minPenetration {
						minPenetration = pX2
						normX, normY, normZ = 1, 0, 0
					}
					if pY1 < minPenetration {
						minPenetration = pY1
						normX, normY, normZ = 0, -1, 0
					}
					if pY2 < minPenetration {
						minPenetration = pY2
						normX, normY, normZ = 0, 1, 0
					}
					if pZ1 < minPenetration {
						minPenetration = pZ1
						normX, normY, normZ = 0, 0, -1
					}
					if pZ2 < minPenetration {
						minPenetration = pZ2
						normX, normY, normZ = 0, 0, 1
					}

					if minPenetration > 0.001 {
						// Dynamic Growth della memoria pre-allocata
						if em.contactsLen >= len(em.contacts) {
							newContacts := make([]Contact, len(em.contacts)*2)
							copy(newContacts, em.contacts)
							em.contacts = newContacts
						}
						// Riciclo memoria
						em.contacts[em.contactsLen].Update(ent, otherEnt, normX, normY, normZ, minPenetration)
						em.contactsLen++
						thing.OnCollide(otherThing)
						otherThing.OnCollide(thing)
					}
				}
			}
			return false
		})
	}

	// FASE 2: RESOLUTION (Il Solver PGS)
	const solverIterations = 4
	for i := 0; i < solverIterations; i++ {
		for c := 0; c < em.contactsLen; c++ {
			em.contacts[c].Resolve()
		}
	}

	// FASE 3: COMMIT SPAZIALE E INTEGRAZIONE
	for x := 0; x < em.movingLen; x++ {
		m := em.moving[x]
		em.tree.UpdateObject(m)
		m.PhysicsApply()
	}
}
