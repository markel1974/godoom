package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

const minMovement = 0.001

// Entities is responsible for managing entities and their spatial relationships within an AABBTree structure.
type Entities struct {
	tree       *physics.AABBTree
	entities   map[string]IThing
	moving     []IThing
	counter    int
	identifier int
}

// NewEntities initializes and returns a new Entities with a specified maximum number of entities.
func NewEntities(maxEntities uint) *Entities {
	return &Entities{
		tree:       physics.NewAABBTree(maxEntities),
		entities:   make(map[string]IThing),
		identifier: 0,
		counter:    0,
	}
}

// Compute performs movement integration, updates the spatial tree, resolves collisions iteratively, and stabilizes the system.
func (em *Entities) Compute() []IThing {
	// Fase 1: Integrazione cinematica (Movimento e frizione)
	em.counter = 0
	for _, thing := range em.entities {
		ent := thing.GetEntity()
		if ent.Update() {
			em.tree.UpdateObject(ent)
			em.moving[em.counter] = thing
			em.counter++
		}
	}
	if em.counter == 0 {
		return nil
	}
	// Fase 2: Iterative Solver per collisioni multiple e propagazione
	const solverIterations = 4

	for i := 0; i < solverIterations; i++ {
		isStable := true

		for x := 0; x < em.counter; x++ {
			thing := em.moving[x]
			ent := thing.GetEntity()
			overlaps := em.tree.QueryOverlaps(thing)

			// Dentro Fase 2: Iterative Solver
			for _, overlapObj := range overlaps {
				otherThing, ok := overlapObj.(IThing)
				if !ok || otherThing == thing {
					continue
				}
				otherEnt := otherThing.GetEntity()

				// FIX REPULSIONE: Applica il tie-breaker SOLO se anche otherEnt è in movimento.
				// Se otherEnt è fermo (sleeping), spetta al body attivo (ent) risolvere l'urto per entrambi.
				otherIsActive := otherEnt.Vx != 0 || otherEnt.Vy != 0
				if otherIsActive && ent.GetId() > otherEnt.GetId() {
					continue
				}

				distance := ent.Distance(otherEnt)
				sumRadii := (ent.GetWidth() / 2.0) + (otherEnt.GetWidth() / 2.0)

				if distance < sumRadii {
					ent.SetupCollision(otherEnt)
					em.tree.UpdateObject(ent)
					em.tree.UpdateObject(otherEnt)
					isStable = false
				}
			}
		}

		// Early Exit: se la scena non presenta più compenetrazioni, il solver si ferma risparmiando CPU
		if isStable {
			break
		}
	}
	return em.moving[:em.counter]
}

// QueryCollisions checks for overlapping entities in the spatial tree and returns a list of bounding boxes for collisions.
func (em *Entities) QueryCollisions(ent IThing) []physics.IAABB {
	return em.tree.QueryOverlaps(ent)
}

func (em *Entities) UpdateThing(thing IThing, px float64, py float64) {
	ent := thing.GetEntity()
	eRadius := ent.GetWidth() / 2.0
	ent.MoveTo(px-eRadius, py-eRadius)

	em.tree.UpdateObject(thing)
}

// AddThing adds the given entity to the manager, adjusts the moving entity slice, and inserts it into the spatial tree.
func (em *Entities) AddThing(ent IThing) IThing {
	em.entities[ent.GetId()] = ent
	if len(em.entities) > len(em.moving) {
		em.moving = make([]IThing, len(em.entities)*2)
	}
	em.tree.InsertObject(ent)
	return ent
}
