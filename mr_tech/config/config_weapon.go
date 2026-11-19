package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const weaponTest = "weapon_test"

type Weapon struct {
	Force        float64
	FireInterval float64
	RechargeTime float64
	MagazineSize int
}

func NewWeaponDefault() *Weapon {
	return &Weapon{
		Force:        500000.0,
		FireInterval: 0.3,
		RechargeTime: 1,
		MagazineSize: 12,
	}
}

func NewConfigWeapon(id string, pos geometry.XYZ, angle float64, mass float64) *Thing {
	c := NewConfigThing(id, pos, angle, ThingWeaponDef, mass, 0, 0, 0)
	c.Weapon = NewWeaponDefault()
	//weapon := common.NewWeapon(c)
	//c.OnCollision = weapon.OnCollision
	//c.OnImpact = weapon.OnImpact
	//c.OnBind = weapon.OnBind
	return c
}
