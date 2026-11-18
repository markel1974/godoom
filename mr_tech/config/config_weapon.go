package config

type Weapon struct {
	Force        float64
	FireInterval float64
	RechargeTime float64
	MagazineSize int
}

func NewWeapon() *Weapon {
	return &Weapon{
		Force:        50000,
		FireInterval: 1,
		RechargeTime: 3,
		MagazineSize: 6,
	}
}
