package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// minMovement defines the minimum threshold for movement to be considered significant in physics calculations.
const minMovement = 0.001

// Entities represent a collection of game objects managed within a spatial partitioning structure for efficient queries.
type Entities struct {
	tree       *physics.AABBTree
	entities   map[int]IThing
	moving     []IThing
	counter    int
	identifier int
}

// NewEntities initializes and returns a new Entities structure with a defined maximum capacity for the AABB tree.
func NewEntities(maxEntities uint) *Entities {
	return &Entities{
		tree:       physics.NewAABBTree(maxEntities),
		entities:   make(map[int]IThing),
		identifier: 0,
		counter:    0,
	}
}

// Compute processes the physics and collision solving for the entities, returning a list of actively moving entities.
func (em *Entities) ComputeOld() []IThing {
	em.counter = 0
	for _, thing := range em.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			em.tree.UpdateObject(thing)
			em.moving[em.counter] = thing
			em.counter++
		}
	}

	if em.counter == 0 {
		return nil
	}

	const solverIterations = 4
	for i := 0; i < solverIterations; i++ {
		isStable := true

		for x := 0; x < em.counter; x++ {
			thing := em.moving[x]
			if !thing.IsActive() {
				continue
			}

			ent := thing.GetEntity()

			em.tree.QueryOverlaps(thing, func(object physics.IAABB) bool {
				otherThing, ok := object.(IThing)
				if !ok || otherThing == thing {
					return false
				}

				//if !otherThing.IsActive() || !thing.IsActive() {
				//	return false
				//}

				otherEnt := otherThing.GetEntity()

				// Tie-breaker 3D: considera anche la velocità Z
				otherIsActive := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
				if otherIsActive && thing.GetIdentifier() > otherThing.GetIdentifier() {
					return false
				}

				// 1. Check Orizzontale Statico (Veloce)
				x1Min, x1Max := ent.GetXRange()
				x2Min, x2Max := otherEnt.GetXRange()
				y1Min, y1Max := ent.GetYRange()
				y2Min, y2Max := otherEnt.GetYRange()
				if x1Max > x2Min && x1Min < x2Max && y1Max > y2Min && y1Min < y2Max {
					// 2. Check Verticale: Qui decidiamo se usare lo Swept
					z1Min, z1Max := ent.GetZRange()
					z2Min, z2Max := otherEnt.GetZRange()
					// Se l'entità ha una velocità Z significativa, espandi il check
					if math.Abs(ent.GetVz()) >= ent.GetGForce() {
						z1Min, z1Max = ent.GetSweptZRange()
					}
					if math.Abs(otherEnt.GetVz()) >= otherEnt.GetGForce() {
						z2Min, z2Max = otherEnt.GetSweptZRange()
					}
					if z1Max > z2Min && z1Min < z2Max {
						// COLLISIONE CONFERMATA
						thing.OnCollide(otherThing)
						otherThing.OnCollide(thing)
						//if thing.IsActive() && otherThing.IsActive() {
						// SetupCollision risolverà la compenetrazione usando la logica degli impulsi
						ent.SetupCollision(otherEnt)
						//}
						em.tree.UpdateObject(thing)
						em.tree.UpdateObject(otherThing)
						isStable = false
					}
				}
				return false
			})
		}
		if isStable {
			break
		}
	}
	return em.moving[:em.counter]
}

// UpdateThing updates the position of the given IThing and adjusts its spatial data in the AABBTree.
func (em *Entities) UpdateThing(thing IThing, px float64, py float64, pz float64) {
	ent := thing.GetEntity()
	eRadius := ent.GetWidth() / 2.0
	// NOTA: Se hai aggiornato physics.Entity per tenere traccia della Z internamente,
	// qui dovrai chiamare la versione 3D, es: ent.MoveTo3d(px-eRadius, py-eRadius, pz)
	ent.MoveTo(px-eRadius, py-eRadius, pz)

	em.tree.UpdateObject(thing)
}

// AddThing adds an IThing instance to the Entities collection, sets its identifier, and inserts it into the AABB tree.
func (em *Entities) AddThing(ent IThing) {
	em.entities[em.identifier] = ent
	ent.SetIdentifier(em.identifier)
	em.identifier++
	if len(em.entities) > cap(em.moving) {
		em.moving = make([]IThing, len(em.entities)*2)
	}
	em.tree.InsertObject(ent)
}

func (em *Entities) RemoveThing(ent IThing) {
	em.tree.RemoveObject(ent)
	delete(em.entities, ent.GetIdentifier())
}

type Contact struct {
	A, B               *physics.Entity
	Nx, Ny, Nz         float64
	Penetration        float64
	AccumulatedImpulse float64
}

func (em *Entities) Compute() []IThing {
	em.counter = 0
	for _, thing := range em.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			em.tree.UpdateObject(thing)
			em.moving[em.counter] = thing
			em.counter++
		}
	}

	if em.counter == 0 {
		return nil
	}

	var contacts []Contact

	// FASE 1: DETECTION (Costruzione del Jacobiano)
	for x := 0; x < em.counter; x++ {
		thing := em.moving[x]
		ent := thing.GetEntity()

		em.tree.QueryOverlaps(thing, func(object physics.IAABB) bool {
			otherThing := object.(IThing)
			otherEnt := otherThing.GetEntity()
			otherThing, ok := object.(IThing)
			if !ok || otherThing == thing {
				return false
			}

			// Tie-breaker 3D: considera anche la velocità Z
			//otherIsActive := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
			otherIsMoving := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
			if otherIsMoving && thing.GetIdentifier() > otherThing.GetIdentifier() {
				return false
			}

			x1Min, x1Max := ent.GetXRange()
			x2Min, x2Max := otherEnt.GetXRange()
			y1Min, y1Max := ent.GetYRange()
			y2Min, y2Max := otherEnt.GetYRange()
			if x1Max > x2Min && x1Min < x2Max && y1Max > y2Min && y1Min < y2Max {
				// 2. Check Verticale: Qui decidiamo se usare lo Swept
				z1Min, z1Max := ent.GetZRange()
				z2Min, z2Max := otherEnt.GetZRange()
				// Se l'entità ha una velocità Z significativa, espandi il check
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
					// 3. INVERTI I SEGNI DELLE NORMALI
					minPenetration := pX1
					var normX, normY, normZ float64 = -1, 0, 0 // A è a sinistra, spingi a sinistra
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
						contacts = append(contacts, Contact{
							A: ent, B: otherEnt,
							Nx: normX, Ny: normY, Nz: normZ,
							Penetration: minPenetration,
						})
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
		for c := 0; c < len(contacts); c++ {
			contacts[c].Resolve()
		}
	}
	for x := 0; x < em.counter; x++ {
		em.tree.UpdateObject(em.moving[x])
	}
	return em.moving[:em.counter]
}

func (c *Contact) Resolve() {
	// 1. Velocità relativa
	rvX := c.A.GetVx() - c.B.GetVx()
	rvY := c.A.GetVy() - c.B.GetVy()
	rvZ := c.A.GetVz() - c.B.GetVz()
	// 2. Velocità lungo la normale
	velAlongNormal := (rvX * c.Nx) + (rvY * c.Ny) + (rvZ * c.Nz)
	// Se si stanno già separando, il vincolo è soddisfatto
	if velAlongNormal > 0 {
		return
	}
	// --- BAUMGARTE STABILIZATION (Positional Bias) ---
	const slop = 0.05   // Tolleranza: permette agli oggetti di penetrare leggermente senza vibrare
	const percent = 0.2 // Corregge il 20% dell'errore ad ogni frame
	// Calcoliamo la velocità extra necessaria per spingerli fuori
	bias := math.Max(c.Penetration-slop, 0.0) * percent
	// Se la velocità relativa (velAlongNormal) è già sufficiente a separarli, ignoriamo il bias
	// altrimenti lo aggiungiamo al calcolo dell'impulso
	e := math.Min(c.A.GetRestitution(), c.B.GetRestitution())
	invMassA := c.A.GetInvMass()
	invMassB := c.B.GetInvMass()

	totalInvMass := invMassA + invMassB
	// PREVIENI LA DIVISIONE PER ZERO (Entrambi gli oggetti statici)
	if totalInvMass <= 0.00001 {
		return
	}
	// Aggiungiamo il termine "bias" all'equazione di J
	j := (-(1.0 + e) * velAlongNormal) + bias
	j /= invMassA + invMassB
	// --- IL PASSAGGIO PROIETTIVO (PGS) ---
	// Conserviamo l'impulso calcolato e lo proiettiamo per non "tirare" i corpi
	oldImpulse := c.AccumulatedImpulse
	c.AccumulatedImpulse = math.Max(oldImpulse+j, 0.0)
	// L'impulso effettivo da applicare in questa singola iterazione
	jDelta := c.AccumulatedImpulse - oldImpulse
	// 4. Applica il delta di velocità
	impulseX := jDelta * c.Nx
	impulseY := jDelta * c.Ny
	impulseZ := jDelta * c.Nz

	// Applica a entità A
	c.A.SetVx(c.A.GetVx() + (impulseX * invMassA))
	c.A.SetVy(c.A.GetVy() + (impulseY * invMassA))
	c.A.SetVz(c.A.GetVz() + (impulseZ * invMassA))

	// Sottrai da entità B
	c.B.SetVx(c.B.GetVx() - (impulseX * invMassB))
	c.B.SetVy(c.B.GetVy() - (impulseY * invMassB))
	c.B.SetVz(c.B.GetVz() - (impulseZ * invMassB))
}
