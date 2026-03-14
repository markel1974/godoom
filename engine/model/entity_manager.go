package model

import (
	"github.com/markel1974/godoom/engine/physics"
)

// EntityDef mappa il Type del WAD (es. 3004 = Imp) alle costanti fisiche
type EntityDef struct {
	Radius float64
	Height float64
	Mass   float64
}

type EntityManager struct {
	Tree     *physics.AABBTree
	Entities map[string]*physics.Entity
	Registry map[int]EntityDef
}

func NewEntityManager(maxEntities uint) *EntityManager {
	return &EntityManager{
		Tree:     physics.NewAABBTree(maxEntities),
		Entities: make(map[string]*physics.Entity),
		Registry: make(map[int]EntityDef),
	}
}

// RegisterType popola il dizionario fisico in fase di inizializzazione
func (em *EntityManager) RegisterType(thingType int, def EntityDef) {
	em.Registry[thingType] = def
}

// Spawn converte il DTO statico in un body fisico e lo inietta nel broad-phase
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

// Compute esegue lo step fisico e aggiorna l'AABBTree solo per le entità in movimento
func (em *EntityManager) Compute() {
	for _, ent := range em.Entities {
		if ent.Compute() {
			em.Tree.UpdateObject(ent)
		}
	}
}

// QueryCollisions espone il broad-phase per raycasting e overlap testing
func (em *EntityManager) QueryCollisions(ent *physics.Entity) []physics.IAABB {
	return em.Tree.QueryOverlaps(ent)
}
