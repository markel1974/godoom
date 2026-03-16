package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// EntityManager manages a collection of entities and their spatial organization using an AABBTree.
type EntityManager struct {
	tree     *physics.AABBTree
	entities map[string]*physics.Entity
}

// NewEntityManager initializes and returns a new instance of EntityManager with a specified maximum number of entities.
func NewEntityManager(maxEntities uint) *EntityManager {
	return &EntityManager{
		tree:     physics.NewAABBTree(maxEntities),
		entities: make(map[string]*physics.Entity),
	}
}

// Spawn creates a new physics entity from the given Thing, adds it to the entity manager, and inserts it into the spatial tree.
func (em *EntityManager) Spawn(thing *model.Thing) *physics.Entity {
	// Conversione centro -> top-left per il rect fisico
	w := thing.Radius * 2
	h := thing.Radius * 2
	x := thing.Position.X - thing.Radius
	y := thing.Position.Y - thing.Radius

	ent := physics.NewEntity(x, y, w, h, thing.Mass)
	ent.Id = thing.Id

	em.entities[ent.Id] = ent
	em.tree.InsertObject(ent)

	return ent
}

// Compute updates the state of all entities, handling movement, friction, collision detection, and resolution in multiple phases.
func (em *EntityManager) Compute() {
	// Fase 1: Integrazione cinematica (Movimento e frizione)
	counter := 0
	for _, ent := range em.entities {
		if ent.Compute() {
			em.tree.UpdateObject(ent)
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

		for _, ent := range em.entities {
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

// QueryCollisions returns a slice of IAABB containing all entities colliding with the given entity.
func (em *EntityManager) QueryCollisions(ent *physics.Entity) []physics.IAABB {
	return em.tree.QueryOverlaps(ent)
}
