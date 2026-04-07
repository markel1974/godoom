package model

import (
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities) *ThingItem {
	pos := cfg.Position
	pos.Z = volume.GetMinZ()
	thing := &ThingItem{
		ThingBase: NewThingBase(cfg, pos, anim, volume, sectors, entities),
	}
	thing.entities.AddThing(thing)
	return thing
}
