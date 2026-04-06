package model

import (
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

				if !otherThing.IsActive() || !thing.IsActive() {
					return false
				}

				otherEnt := otherThing.GetEntity()

				// Tie-breaker 3D: considera anche la velocità Z
				otherIsActive := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0 || otherEnt.GetVz() != 0
				if otherIsActive && thing.GetIdentifier() > otherThing.GetIdentifier() {
					return false
				}

				// --- COLLISION CHECK 3D ---
				// 1. Check Orizzontale usando DistanceSq (evita math.Sqrt)
				distSq := ent.DistanceSq(otherEnt)
				sumRadii := (ent.GetWidth() / 2.0) + (otherEnt.GetWidth() / 2.0)

				if distSq < sumRadii*sumRadii {
					// 2. Check Verticale (Z-Overlap) usando GetZRange
					z1Min, z1Max := ent.GetZRange()
					z2Min, z2Max := otherEnt.GetZRange()

					// Se le proiezioni Z si sovrappongono
					if z1Max > z2Min && z1Min < z2Max {
						// COLLISIONE CONFERMATA
						thing.OnCollide(otherThing)
						otherThing.OnCollide(thing)

						if thing.IsActive() && otherThing.IsActive() {
							// SetupCollision risolverà la compenetrazione
							ent.SetupCollision(otherEnt)
						}

						// Aggiornamento immediato dell'albero per la prossima query
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
