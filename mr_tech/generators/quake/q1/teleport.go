package q1

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// TeleportDest describes a teleport target with its destination position and facing angle.
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
// It stores destinations by targetname with parsed origin and angle in radians.
func (t *Teleport) Collect(entities []*lumps.Entity) {
	t.destinations = make(map[string]*TeleportDest)
	for _, ent := range entities {
		classname, _ := ent.GetProperty("classname")
		if classname != "info_teleport_destination" && classname != "misc_teleporter_dest" && classname != "target_position" {
			continue
		}
		targetName, ok := ent.GetProperty("targetname")
		if !ok || targetName == "" {
			continue
		}
		var pos geometry.XYZ
		if origin, okOrig := ent.GetProperty("origin"); okOrig {
			x, y, z, _ := lumps.ParseVector(origin)
			pos = lumps.CreateXYZ(x, y, z)
		}
		angle := 0.0
		if rawAngle, okAng := ent.GetProperty("angle"); okAng {
			if parsed, okParse := lumps.ParseFloat(rawAngle); okParse {
				angle = parsed * (math.Pi / 180.0)
			}
		}
		t.destinations[targetName] = NewTeleportDest(pos, angle)
	}
}
