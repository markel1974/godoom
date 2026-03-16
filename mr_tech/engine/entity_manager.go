package engine

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// EntityManager is responsible for managing entities and their spatial relationships within an AABBTree structure.
type EntityManager struct {
	tree     *physics.AABBTree
	entities map[string]*physics.Entity
	moving   []*physics.Entity
}

// NewEntityManager initializes and returns a new EntityManager with a specified maximum number of entities.
func NewEntityManager(maxEntities uint) *EntityManager {
	return &EntityManager{
		tree:     physics.NewAABBTree(maxEntities),
		entities: make(map[string]*physics.Entity),
	}
}

// Spawn creates a new entity with the specified ID, position, radius, and mass, and inserts it into the spatial tree.
func (em *EntityManager) Spawn(id string, pX, pY, radius, mass float64) *physics.Entity {
	w := radius / 10
	h := radius / 10
	x := pX - radius
	y := pY - radius
	ent := em.addEntity(id, x, y, w, h, mass)
	return ent
}

// Compute performs movement integration, updates the spatial tree, resolves collisions iteratively, and stabilizes the system.
func (em *EntityManager) Compute() {
	// Fase 1: Integrazione cinematica (Movimento e frizione)
	counter := 0
	for _, ent := range em.entities {
		if ent.Compute() {
			em.tree.UpdateObject(ent)
			em.moving[counter] = ent
			counter++
		}
	}

	if counter == 0 {
		return
	}
	// Fase 2: Iterative Solver per collisioni multiple e propagazione
	const solverIterations = 4

	for i := 0; i < solverIterations; i++ {
		isStable := true

		for x := 0; x < counter; x++ {
			ent := em.moving[x]
			overlaps := em.tree.QueryOverlaps(ent)

			for _, overlapObj := range overlaps {
				otherEnt, ok := overlapObj.(*physics.Entity)
				if !ok || otherEnt == ent {
					continue
				}

				// Ottimizzazione: previene la risoluzione bidirezionale (A->B calcolato, B->A skippato)
				if ent.Id > otherEnt.Id {
					continue
				}

				distance := ent.Distance(otherEnt)
				sumRadii := (ent.GetWidth() / 2.0) + (otherEnt.GetWidth() / 2.0)

				// Narrow-Phase radiale
				if distance < sumRadii {
					// SetupCollision applica l'impulso elastico e separa i centri
					ent.SetupCollision(otherEnt)

					// UpdateObject propaga il nuovo AABB, essenziale per l'urto successivo B->C
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
}

// QueryCollisions checks for overlapping entities in the spatial tree and returns a list of bounding boxes for collisions.
func (em *EntityManager) QueryCollisions(ent *physics.Entity) []physics.IAABB {
	return em.tree.QueryOverlaps(ent)
}

// addEntity adds the given entity to the manager, adjusts the moving entity slice, and inserts it into the spatial tree.
func (em *EntityManager) addEntity(id string, x float64, y float64, w float64, h float64, mass float64) *physics.Entity {
	ent := physics.NewEntity(x, y, w, h, mass)
	ent.Id = id
	em.entities[ent.Id] = ent
	if len(em.entities) > len(em.moving) {
		em.moving = make([]*physics.Entity, len(em.entities)+128)
	}
	em.tree.InsertObject(ent)
	return ent
}
