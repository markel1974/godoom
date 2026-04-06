package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Things manages a collection of game objects, their configurations, volumes, entities, and animations.
type Things struct {
	things     []IThing
	config     []*config.ConfigThing
	sectors    *Volumes
	entities   *Entities
	animations *Animations
}

// NewThings initializes a Things instance with the provided configurations, volumes, entities, and animations.
// Returns the created Things instance or an error if initialization of any Thing fails.
func NewThings(cfg []*config.ConfigThing, sectors *Volumes, entities *Entities, animations *Animations) (*Things, error) {
	r := &Things{
		config:     cfg,
		sectors:    sectors,
		entities:   entities,
		animations: animations,
		things:     nil,
	}
	for _, ct := range cfg {
		if err := r.CreateThing(ct); err != nil {
			fmt.Println("Warning: ", err)
			//return nil, err
		}
	}
	return r, nil
}

// GetTextures fetches the ITextures instance from the associated Animations object.
func (r *Things) GetTextures() textures.ITextures {
	return r.animations.GetTextures()
}

// Get retrieves the list of all IThing instances managed by the Things object.
func (r *Things) Get() []IThing {
	return r.things
}

// CreateThing creates a new IThing instance based on the provided ConfigThing and adds it to the Things collection.
func (r *Things) CreateThing(ct *config.ConfigThing) error {
	sector := r.sectors.QueryPoint2d(ct.Position.X, ct.Position.Y)
	if sector == nil {
		return fmt.Errorf("can't find thing sector at %f, %f", ct.Position.X, ct.Position.Y)
	}
	var thing IThing

	switch ct.Kind {
	case config.ThingEnemyDef:
		thing = NewThingEnemy(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case config.ThingWeaponDef:
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case config.ThingBulletDef:
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case config.ThingKeyDef:
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case config.ThingItemDef:
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	default:
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	}

	if ct.Speed > 0 {
		thing = NewThingEnemy(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	} else {
		thing = NewThingItem(ct, r.animations.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	}
	r.things = append(r.things, thing)
	return nil
}

// CreateBullet creates a new bullet in the specified sector at the given position (x, y) with the given angle.
func (r *Things) CreateBullet(volume *Volume, x float64, y float64, z float64, angle float64) {
	//TODO now is an hack
	//test zero index
	c := r.config[2]
	id := utils.NextUUId()
	pos := geometry.XYZ{X: x, Y: y, Z: z}
	cfg := config.NewConfigThing(id, pos, angle, config.ThingBulletDef, 500.0, 1.0, 5.0, 5.0, c.Animation)
	thing := NewThingBullet(cfg, r.animations.GetAnimation(cfg.Animation), volume, r.sectors, r.entities)
	r.things = append(r.things, thing)
}

// Compute updates the state of all IThing objects in the collection using the provided position coordinates (pX, pY).
func (r *Things) Compute(pX float64, pY float64, pZ float64) {
	activeThings := r.things[:0] // Reslice reusing memory

	for _, t := range r.things {
		if !t.IsActive() {
			r.entities.RemoveThing(t)
			continue
		}
		t.Compute(pX, pY, pZ)
		activeThings = append(activeThings, t)
	}

	// Clear dangling pointers per il GC
	for i := len(activeThings); i < len(r.things); i++ {
		r.things[i] = nil
	}
	r.things = activeThings
}
