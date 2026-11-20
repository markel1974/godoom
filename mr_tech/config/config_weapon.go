package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// weaponTest identifies the default test weapon used in player and root configuration.
const weaponTest = "weapon_test"

// WeaponSpec defines firing and reload characteristics for a weapon attached to a game entity.
type WeaponSpec struct {
	Force        float64
	FireInterval float64
	RechargeTime float64
	MagazineSize int
}

// NewWeaponDefault creates and returns a Weapon initialized with default gameplay settings.
func NewWeaponDefault() *WeaponSpec {
	return &WeaponSpec{
		Force:        500000.0,
		FireInterval: 0.3,
		RechargeTime: 1,
		MagazineSize: 12,
	}
}

// NewConfigWeapon creates a weapon Thing configuration with the specified ID, position, angle, and mass.
// It initializes the Thing with default weapon settings.
func NewConfigWeapon(id string, pos geometry.XYZ, angle float64, mass float64) *Thing {
	c := NewConfigThing(id, pos, angle, ThingWeaponDef, mass, 0, 0, 0)
	c.Weapon = NewWeaponDefault()
	return c
}
