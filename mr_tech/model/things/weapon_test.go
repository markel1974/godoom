package things

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
)

func newTestWeapon(cfg *config.Weapon) *Weapon {
	ct := &config.Thing{
		Weapon:      cfg,
		OnCollision: func(other config.IThingConfig, energy float64) {},
		OnImpact:    func(other config.IThingConfig, energy float64, pos, normal geometry.XYZ) {},
		OnBind:      func(self config.IThingConfig) {},
	}
	return NewWeapon(nil, ct, nil)
}

func TestWeaponFiringAndCooldown(t *testing.T) {
	cfg := &config.Weapon{
		Force:        60000,
		FireInterval: 0.5,
		RechargeTime: 2.0,
		MagazineSize: 4,
	}
	w := newTestWeapon(cfg)

	if w.GetAmmo() != 4 {
		t.Fatalf("expected 4 rounds, got %d", w.GetAmmo())
	}
	if w.GetForce() != 60000 {
		t.Errorf("expected force 60000, got %f", w.GetForce())
	}
	if !w.CanFire() {
		t.Fatalf("expected weapon to be ready to fire")
	}

	// First shot: should succeed
	if !w.Fire() {
		t.Fatalf("first shot should succeed")
	}
	if w.GetAmmo() != 3 {
		t.Errorf("expected 3 rounds remaining, got %d", w.GetAmmo())
	}

	// Immediate next shot: must fail due to cooldown
	if w.CanFire() {
		t.Errorf("weapon should not be ready immediately after firing")
	}
	if w.Fire() {
		t.Errorf("shot should fail while on cooldown")
	}

	// Advance time by 0.3s (less than 0.5s fire interval)
	w.Update(0.3)
	if w.CanFire() {
		t.Errorf("weapon should still be on cooldown after 0.3s")
	}

	// Advance time by remaining 0.25s (total 0.55s > 0.5s)
	w.Update(0.25)
	if !w.CanFire() {
		t.Errorf("weapon should be ready to fire after cooldown elapsed")
	}
	if !w.Fire() {
		t.Errorf("second shot should succeed after cooldown")
	}
	if w.GetAmmo() != 2 {
		t.Errorf("expected 2 rounds remaining, got %d", w.GetAmmo())
	}
}

func TestWeaponReloadMechanics(t *testing.T) {
	cfg := &config.Weapon{
		Force:        50000,
		FireInterval: 0.1,
		RechargeTime: 1.0,
		MagazineSize: 2,
	}
	w := newTestWeapon(cfg)

	// Fire shot 1
	w.Fire()
	w.Update(0.15)
	// Fire shot 2 (empties magazine)
	w.Fire()
	if w.GetAmmo() != 0 {
		t.Fatalf("expected 0 ammo, got %d", w.GetAmmo())
	}

	// Depletion should automatically trigger reload
	if !w.IsReloading() {
		t.Fatalf("weapon should automatically enter reloading state when empty")
	}
	if w.CanFire() {
		t.Fatalf("weapon cannot fire while reloading")
	}

	// Progress during reload
	w.Update(0.5)
	progress := w.GetReloadProgress()
	if progress < 0.45 || progress > 0.55 {
		t.Errorf("expected reload progress ~0.5, got %f", progress)
	}

	// Finish reload
	w.Update(0.55)
	if w.IsReloading() {
		t.Errorf("weapon should have finished reloading")
	}
	if w.GetAmmo() != 2 {
		t.Errorf("ammo should be restored to magazine size 2, got %d", w.GetAmmo())
	}
	if !w.CanFire() {
		t.Errorf("weapon should be ready to fire after reload")
	}
}

func TestWeaponManualReload(t *testing.T) {
	cfg := &config.Weapon{
		Force:        50000,
		FireInterval: 0.1,
		RechargeTime: 1.5,
		MagazineSize: 6,
	}
	w := newTestWeapon(cfg)

	// Manual reload on full magazine should return false
	if w.Reload() {
		t.Errorf("should not reload full magazine")
	}

	w.Fire()
	if w.GetAmmo() != 5 {
		t.Errorf("expected 5 ammo, got %d", w.GetAmmo())
	}

	// Manual reload with 5/6 ammo
	if !w.Reload() {
		t.Errorf("manual reload should succeed with partial magazine")
	}
	if !w.IsReloading() {
		t.Errorf("weapon should be in reloading state")
	}

	// Complete reload
	w.Update(1.6)
	if w.GetAmmo() != 6 {
		t.Errorf("expected full magazine 6, got %d", w.GetAmmo())
	}
}
