package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// TeleportSpec defines destination coordinates, orientation, and linking metadata for a ThingTeleportDef.
type TeleportSpec struct {
	Destination   geometry.XYZ `json:"destination"`
	TargetAngle   float64      `json:"targetAngle,omitempty"`
	PreserveAngle bool         `json:"preserveAngle,omitempty"`
	ResetVelocity bool         `json:"resetVelocity,omitempty"`
	Cooldown      float64      `json:"cooldown,omitempty"`
	TargetName    string       `json:"targetName,omitempty"`
	Target        string       `json:"target,omitempty"`
	PlayerOnly    bool         `json:"playerOnly,omitempty"`
}
