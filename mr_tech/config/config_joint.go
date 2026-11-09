package config

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// JointConfig defines physics and constraint settings for a skeletal joint in animated and ragdoll states.
type JointConfig struct {
	DofScale        geometry.XYZ
	MaxAngleAlive   float64
	MaxAngleRagdoll float64
	Stiffness       float64
	Damping         float64
	DetachThreshold float64
	CanDetach       bool
	Tags            []string
}

// JointsConfig groups joint behavior presets and a fallback configuration for matching model links by tag.
type JointsConfig struct {
	Joints  []*JointConfig
	Default *JointConfig
}

// NewJointsConfig creates the default joint configuration set for model links.
// NewJointsConfig includes presets for torso, head, limbs, weapons, and a fallback joint.
func NewJointsConfig() *JointsConfig {
	// Torso / Spine: 3 DoF (Pitch, Roll, Yaw), core trunk stays connected
	torso := &JointConfig{}
	torso.DofScale = geometry.XYZ{X: 1.0, Y: 1.0, Z: 0.85}
	torso.MaxAngleAlive = 0.48
	torso.MaxAngleRagdoll = 0.85
	torso.Stiffness = 58.0
	torso.Damping = 7.5
	torso.CanDetach = false
	torso.Tags = []string{"torso", "chest", "waist", "pelvis", "hips", "belly", "spine", "shoulders", "upper"}

	// Head / Neck: 2-3 DoF with decapitation threshold
	head := &JointConfig{}
	head.DofScale = geometry.XYZ{X: 1.0, Y: 1.0, Z: 0.75}
	head.MaxAngleAlive = 0.60
	head.MaxAngleRagdoll = 1.15
	head.Stiffness = 48.0
	head.Damping = 6.5
	//j.canDetach = true
	head.CanDetach = false
	head.DetachThreshold = 1.25
	head.Tags = []string{"head", "neck", "jaw"}

	// Distal limbs (forearms, hands, lower legs): 1-2 DoF hinge, detachable
	limbs := &JointConfig{}
	limbs.DofScale = geometry.XYZ{X: 1.0, Y: 0.90, Z: 0.40}
	limbs.MaxAngleAlive = 0.72
	limbs.MaxAngleRagdoll = 1.35
	limbs.Stiffness = 40.0
	limbs.Damping = 5.5
	//j.canDetach = true
	limbs.CanDetach = false
	limbs.DetachThreshold = 1.05
	limbs.Tags = []string{"lloarm", "rloarm", "lhand", "rhand", "lloleg", "rloleg", "lfoot", "rfoot"}

	// Held weapon: disarms/flies off on hit
	weapon := &JointConfig{}
	weapon.DofScale = geometry.XYZ{X: 1.0, Y: 1.0, Z: 0.65}
	weapon.MaxAngleAlive = 0.68
	weapon.MaxAngleRagdoll = 1.35
	weapon.Stiffness = 42.0
	weapon.Damping = 5.5
	//j.canDetach = true
	weapon.CanDetach = false
	weapon.DetachThreshold = 0.95
	weapon.Tags = []string{"weapon"}

	// Upper arms and upper legs (luparm, ruparm, lupleg, rupleg, tail, etc.): 2-3 DoF ball joint, detachable on heavy/ragdoll hits
	j := &JointConfig{}
	j.DofScale = geometry.XYZ{X: 1.0, Y: 1.0, Z: 0.65}
	j.MaxAngleAlive = 0.62
	j.MaxAngleRagdoll = 1.20
	j.Stiffness = 46.0
	j.Damping = 6.2
	//j.canDetach = true
	j.CanDetach = false
	j.DetachThreshold = 1.30

	js := &JointsConfig{
		Joints:  []*JointConfig{torso, head, limbs, weapon},
		Default: j,
	}

	return js
}

// Get returns the first matching joint configuration for tag, or the default configuration.
func (jc *JointsConfig) Get(tag string) *JointConfig {
	lowerTag := strings.ToLower(tag)
	for _, j := range jc.Joints {
		for _, t := range j.Tags {
			if strings.Contains(lowerTag, t) {
				return j
			}
		}
	}
	return jc.Default
}
