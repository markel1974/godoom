package model

import (
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingItem {
	thing := &ThingItem{
		ThingBase: NewThingBase(cfg, anim, sector, sectors, entities),
	}
	return thing
}
