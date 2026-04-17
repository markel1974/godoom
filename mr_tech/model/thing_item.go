package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(full3d bool, things *Things, cfg *config.Thing, anim *textures.Animation, volume *Volume) *ThingItem {
	pos := cfg.Position
	if !full3d {
		pos.Z = volume.GetMinZ()
	}
	thing := &ThingItem{
		ThingBase: NewThingBase(things, cfg, pos, anim, volume),
	}
	thing.things.AddThing(thing)
	return thing
}
