package model

import (
	"github.com/markel1974/godoom/engine/physics"
)

// EntityDef defines the physical properties of an entity, including its radius, height, and mass.
type EntityDef struct {
	Radius float64
	Height float64
	Mass   float64
}

// EntityManager manages a collection of entities, their definitions, and a spatial tree for efficient querying and updates.
type EntityManager struct {
	Tree     *physics.AABBTree
	Entities map[string]*physics.Entity
	Registry map[int]EntityDef
}

// NewEntityManager initializes and returns a new EntityManager with specified maxEntities for the AABBTree and entity storage.
func NewEntityManager(maxEntities uint) *EntityManager {
	return &EntityManager{
		Tree:     physics.NewAABBTree(maxEntities),
		Entities: make(map[string]*physics.Entity),
		Registry: make(map[int]EntityDef),
	}
}

// RegisterType adds a new entity type definition to the registry using the provided type identifier and definition.
func (em *EntityManager) RegisterType(thingType int, def EntityDef) {
	em.Registry[thingType] = def
}

// Spawn creates a new physics.Entity using a specified ConfigThing and registers it within the EntityManager's structures.
func (em *EntityManager) Spawn(thing *ConfigThing) *physics.Entity {
	def, ok := em.Registry[thing.Type]
	if !ok {
		// Fallback failsafe per Type sconosciuti
		def = EntityDef{Radius: 16.0, Height: 56.0, Mass: 100.0}
	}

	// Conversione centro -> top-left per il rect fisico
	w := def.Radius * 2
	h := def.Radius * 2
	x := thing.Position.X - def.Radius
	y := thing.Position.Y - def.Radius

	ent := physics.NewEntity(x, y, w, h, def.Mass)
	ent.Id = thing.Id

	em.Entities[ent.Id] = ent
	em.Tree.InsertObject(ent)

	return ent
}

// Compute esegue l'integrazione e la risoluzione iterativa per le collisioni a catena
// Compute esegue l'integrazione cinematica e il solver iterativo per la propagazione fisica a catena
func (em *EntityManager) Compute() {
	// Fase 1: Integrazione cinematica (Movimento e frizione)
	for _, ent := range em.Entities {
		if ent.Compute() {
			em.Tree.UpdateObject(ent)
		}
	}

	// Fase 2: Iterative Solver per collisioni multiple e propagazione
	const solverIterations = 4

	for i := 0; i < solverIterations; i++ {
		isStable := true

		for _, ent := range em.Entities {
			overlaps := em.Tree.QueryOverlaps(ent)

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
					em.Tree.UpdateObject(ent)
					em.Tree.UpdateObject(otherEnt)

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

// QueryCollisions checks for overlapping entities with the given entity and returns a list of intersecting bounding boxes.
func (em *EntityManager) QueryCollisions(ent *physics.Entity) []physics.IAABB {
	return em.Tree.QueryOverlaps(ent)
}
