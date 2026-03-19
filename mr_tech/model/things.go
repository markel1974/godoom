package model

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Things represents a collection of game entities, animations, textures, and configuration for managing game objects.
type Things struct {
	animations map[string]*textures.Animation
	tex        textures.ITextures
	things     []IThing
	config     *ConfigRoot
	sectors    *Sectors
	entities   *Entities
}

// NewThings initializes a Things instance with textures, configuration, sectors, and entities, and prepares animations.
func NewThings(cfg *ConfigRoot) *Things {
	return &Things{
		config:     cfg,
		sectors:    nil,
		entities:   nil,
		tex:        cfg.Textures,
		animations: make(map[string]*textures.Animation),
		things:     nil,
	}
}

// GetTextures returns the texture interface associated with the current Things instance.
func (r *Things) GetTextures() textures.ITextures {
	return r.tex
}

// GetThings returns a slice of IThing objects managed by the Things instance.
func (r *Things) GetThings() []IThing {
	return r.things
}

// Setup initializes and creates all `IThing` instances defined in the configuration. Returns an error if creation fails.
func (r *Things) Setup(sectors *Sectors, entities *Entities) error {
	r.sectors = sectors
	r.entities = entities
	for _, ct := range r.config.Things {
		if err := r.CreateThing(ct); err != nil {
			return err
		}
	}
	return nil
}

// CreateThing initializes and adds a new `IThing` instance based on the provided configuration within the `Things` manager.
// It determines the type, sector, and properties of the thing and appends it to the internal `things` slice.
func (r *Things) CreateThing(ct *ConfigThing) error {
	sector := r.sectors.GetSector(ct.Sector)
	if sector == nil {
		return fmt.Errorf("can't find thing sector at %s", ct.Sector)
	}
	var thing IThing

	switch ct.Kind {
	case ThingEnemyDef:
		thing = NewThingEnemy(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case ThingWeaponDef:
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case ThingBulletDef:
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case ThingKeyDef:
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	case ThingItemDef:
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	default:
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	}

	if ct.Speed > 0 {
		thing = NewThingEnemy(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	} else {
		thing = NewThingItem(ct, r.GetAnimation(ct.Animation), sector, r.sectors, r.entities)
	}
	r.things = append(r.things, thing)
	return nil
}

func (r *Things) CreateBullet(sector *Sector, x float64, y float64, angle float64) {
	//TODO now is an hack
	//test zero index
	c := r.config.Things[2]
	id := utils.NextUUId()
	pos := XY{X: x, Y: y}
	cfg := NewConfigThing(id, pos, angle, ThingBulletDef, sector.Id, 500.0, 1.0, 5.0, 1.0, c.Animation)
	thing := NewThingBullet(cfg, r.GetAnimation(cfg.Animation), sector, r.sectors, r.entities)
	r.things = append(r.things, thing)
}

// GetAnimation retrieves or creates an animation instance based on the provided configuration.
func (r *Things) GetAnimation(ca *ConfigAnimation) *textures.Animation {
	if ca == nil {
		return textures.NewAnimation(nil, int(AnimationKindNone), 1, 1)
	}
	key := strings.Join(ca.Frames, ";")
	animation, ok := r.animations[key]
	if ok {
		return animation
	}
	tex := r.tex.Get(ca.Frames)
	animation = textures.NewAnimation(tex, int(ca.Kind), ca.ScaleW, ca.ScaleH)
	r.animations[key] = animation
	return animation
}

func (r *Things) Compute(pX float64, pY float64) {
	for _, t := range r.things {
		t.Compute(pX, pY)
	}
}
