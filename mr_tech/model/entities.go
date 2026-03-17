package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Entities is responsible for managing entities and their spatial relationships within an AABBTree structure.
type Entities struct {
	tree     *physics.AABBTree
	entities map[string]*physics.Entity
	moving   []*physics.Entity
	counter  int
}

// NewEntities initializes and returns a new Entities with a specified maximum number of entities.
func NewEntities(maxEntities uint) *Entities {
	return &Entities{
		tree:     physics.NewAABBTree(maxEntities),
		entities: make(map[string]*physics.Entity),
	}
}

// GetEntity retrieves an entity by its ID from the entity manager's collection and returns a pointer to the entity.
func (em *Entities) GetEntity(id string) *physics.Entity {
	return em.entities[id]
}

// Spawn creates a new entity with the specified ID, position, radius, and mass, and inserts it into the spatial tree.
//func (em *Entities) Spawn(id string, pX, pY, radius, mass float64) *physics.Entity {
//	w := radius * 2
//	h := radius * 2
//	x := pX - radius
//	y := pY - radius
//	ent := physics.NewEntity(x, y, w, h, mass)
//	em.addEntity(id, ent)
//	return ent
//}

// Compute performs movement integration, updates the spatial tree, resolves collisions iteratively, and stabilizes the system.
func (em *Entities) Compute() []*physics.Entity {
	// Fase 1: Integrazione cinematica (Movimento e frizione)
	em.counter = 0
	for _, ent := range em.entities {
		if ent.Compute() {
			em.tree.UpdateObject(ent)
			em.moving[em.counter] = ent
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
			ent := em.moving[x]
			overlaps := em.tree.QueryOverlaps(ent)

			// Dentro Fase 2: Iterative Solver
			for _, overlapObj := range overlaps {
				otherEnt, ok := overlapObj.(*physics.Entity)
				if !ok || otherEnt == ent {
					continue
				}

				// FIX REPULSIONE: Applica il tie-breaker SOLO se anche otherEnt è in movimento.
				// Se otherEnt è fermo (sleeping), spetta al body attivo (ent) risolvere l'urto per entrambi.
				otherIsActive := otherEnt.Vx != 0 || otherEnt.Vy != 0
				if otherIsActive && ent.Id > otherEnt.Id {
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
func (em *Entities) QueryCollisions(ent *physics.Entity) []physics.IAABB {
	return em.tree.QueryOverlaps(ent)
}

func (em *Entities) UpdateObject(ent *physics.Entity) {
	em.tree.UpdateObject(ent)
}

// AddEntity adds the given entity to the manager, adjusts the moving entity slice, and inserts it into the spatial tree.
func (em *Entities) AddEntity(id string, ent *physics.Entity) *physics.Entity {
	ent.Id = id
	em.entities[ent.Id] = ent
	if len(em.entities) > len(em.moving) {
		em.moving = make([]*physics.Entity, len(em.entities)*2)
	}
	em.tree.InsertObject(ent)
	return ent
}
