package doom3

import (
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// TeleportDest describes a named teleport destination with its target position and facing angle.
type TeleportDest struct {
	Pos   geometry.XYZ
	Angle float64
}

// NewTeleportDest creates a teleport destination with the given position and angle.
func NewTeleportDest(pos geometry.XYZ, angle float64) *TeleportDest {
	return &TeleportDest{Pos: pos, Angle: angle}
}

// Teleport stores teleport destinations by target name for lookup after collecting map entities.
type Teleport struct {
	destinations map[string]*TeleportDest
}

// NewTeleport creates an empty teleport destination registry.
func NewTeleport() *Teleport {
	return &Teleport{}
}

// Get returns the teleport destination for target, or nil when target is empty or not found.
func (t *Teleport) Get(target string) *TeleportDest {
	if len(target) == 0 {
		return nil
	}
	return t.destinations[target]
}

// Collect rebuilds the teleport destination lookup from supported destination entities.
// It stores destinations by name with valid origins and yaw angles.
func (t *Teleport) Collect(entities []*proc.MapEntity) {
	t.destinations = make(map[string]*TeleportDest)
	for _, ent := range entities {
		classname, _ := ent.GetProperty("classname")
		if classname != "info_player_teleport" &&
			classname != "info_teleport_destination" &&
			classname != "target_teleporter" &&
			classname != "misc_teleporter_dest" {
			continue
		}
		name, ok := ent.GetProperty("name")
		if !ok || name == "" {
			continue
		}
		origin, err := ent.GetOrigin()
		if err != nil {
			continue
		}
		t.destinations[name] = NewTeleportDest(origin, ent.GetAngle())
	}
}
