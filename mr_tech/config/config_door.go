package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// DoorActivation defines how a door can be triggered by an actor.
type DoorActivation int

const (
	// DoorActivateTouch opens the door automatically on actor collision as well as explicit Use.
	DoorActivateTouch DoorActivation = iota
	// DoorActivateUse opens the door only when explicitly activated via Use or trigger target.
	DoorActivateUse
)

// DoorSpec defines kinematic translation, timing, and key lock parameters for a ThingDoorDef.
type DoorSpec struct {
	RequiredKey string         `json:"requiredKey,omitempty"`
	OpenOffset  geometry.XYZ   `json:"openOffset"`
	Speed       float64        `json:"speed"`
	WaitSeconds float64        `json:"waitSeconds"`
	StartOpen   bool           `json:"startOpen,omitempty"`
	TargetName  string         `json:"targetName,omitempty"`
	Target      string         `json:"target,omitempty"`
	Activation  DoorActivation `json:"activation"`
}
