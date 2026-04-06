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
	// Kinematic integration (Movement and friction)
	em.counter = 0
	for _, thing := range em.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			// FIX: L'albero mappa istanze IThing, non *physics.Entity
			em.tree.UpdateObject(thing)
			em.moving[em.counter] = thing
			em.counter++
		}
	}
	if em.counter == 0 {
		return nil
	}
	// Iterative solver for multiple collisions and propagation
	const solverIterations = 4

	for i := 0; i < solverIterations; i++ {
		isStable := true

		for x := 0; x < em.counter; x++ {
			thing := em.moving[x]

			// Se l'entità è stata disattivata nel loop precedente (es. proiettile esploso), ignoriamola
			if !thing.IsActive() {
				continue
			}

			ent := thing.GetEntity()
			em.tree.QueryOverlaps(thing, func(object physics.IAABB) bool {
				otherThing, ok := object.(IThing)
				if !ok || otherThing == thing {
					return false
				}

				// Early exit per entità morte
				if !otherThing.IsActive() || !thing.IsActive() {
					return false
				}

				otherEnt := otherThing.GetEntity()

				// Apply the tie-breaker ONLY if otherEnt is also in motion.
				// If otherEnt is stationary (sleeping), it's up to the active body (ent) to resolve the collision for both.
				otherIsActive := otherEnt.GetVx() != 0 || otherEnt.GetVy() != 0
				if otherIsActive && thing.GetIdentifier() > otherThing.GetIdentifier() {
					return false
				}

				distance := ent.Distance(otherEnt)
				sumRadii := (ent.GetWidth() / 2.0) + (otherEnt.GetWidth() / 2.0)

				if distance < sumRadii {
					// 1. Risolviamo prima gli Eventi di Gioco (Callbacks, Danni, Pickup)
					thing.OnCollide(otherThing)
					otherThing.OnCollide(thing)

					// 2. Risoluzione della compenetrazione FISICA.
					// Viene eseguita SOLO SE entrambe le entità sono sopravvissute all'impatto (es. due mostri solidi)
					if thing.IsActive() && otherThing.IsActive() {
						ent.SetupCollision(otherEnt)
					}

					// FIX: Passare l'istanza corretta IThing per l'aggiornamento dell'albero
					em.tree.UpdateObject(thing)
					em.tree.UpdateObject(otherThing)
					isStable = false
				}
				return false
			})
		}

		// Early Exit: if the scene no longer presents overlaps, the solver stops saving CPU
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
