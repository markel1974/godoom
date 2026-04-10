package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingEnemy represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingEnemy struct {
	*ThingBase
	active       bool
	fireCooldown float64 // Timer per controllare il rateo di fuoco
}

// NewThingEnemy creates and initializes a new ThingEnemy instance.
func NewThingEnemy(things *Things, cfg *config.ConfigThing, anim *textures.Animation, volume *Volume) *ThingEnemy {
	pos := cfg.Position
	pos.Z = volume.GetMinZ()
	e := &ThingEnemy{
		ThingBase:    NewThingBase(things, cfg, pos, anim, volume),
		active:       false,
		fireCooldown: 0.0,
	}
	e.things.AddThing(e)
	return e
}

// Compute updates the Thing's direction, position, and attack logic based on the player's coordinates.
// Compute updates the Thing's direction, position, and attack logic based on the player's coordinates.
func (t *ThingEnemy) Compute(thingX float64, thingY float64, thingZ float64) {
	dx := thingX - t.pos.X
	dy := thingY - t.pos.Y

	// Il target Z deve essere circa a metà altezza del giocatore (es. petto) per mirare bene
	targetZ := thingZ + (t.height / 2)
	dz := targetZ - t.pos.Z

	// 1. Attivazione (Aggro): Utilizza la distanza sferica 3D
	dist3d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !t.active {
		if dist3d < 25.0 { // Raggio di risveglio
			t.active = true
			t.fireCooldown = 1.0 // Pausa prima del primo colpo dopo essersi svegliato
		}
		return
	}

	// 2. Aggiornamento timer armi (assumendo dt fisso a 1/60)
	if t.fireCooldown > 0 {
		t.fireCooldown -= 1.0 / 60.0
	}

	// 3. Inseguimento Terrestre
	dist2d := math.Sqrt(dx*dx + dy*dy)
	if dist2d <= 0.001 {
		return
	}

	// Aggiorniamo l'angolo del nemico affinché lo sprite o il modello si giri verso il bersaglio
	t.angle = math.Atan2(dy, dx)
	invDist := 1.0 / dist2d
	nx := dx * invDist
	ny := dy * invDist
	const forceScale = 100.0

	// Applica forza per muoversi verso il giocatore
	fx := nx * forceScale * t.speed
	fy := ny * forceScale * t.speed
	t.entity.AddForce(fx, fy, 0.0)

	t.doJump(thingZ, dist2d, fx, fy)

	t.doFire(dist3d, dist2d, dz)
}

func (t *ThingEnemy) doJump(thingZ, dist2d, fx, fy float64) {
	if t.entity.IsOnGround() {
		vx, vy, _ := t.entity.GetVelocity()
		speedSq := (vx * vx) + (vy * vy)
		// Euristica A: L'entità sta spingendo ma la sua velocità planare è quasi zero (ostacolo insuperabile col maxStep)
		isBlocked := speedSq < 0.1
		// Euristica B: Il pavimento del bersaglio (thingZ) è più in alto del pavimento del nemico (t.pos.Z)
		floorDz := thingZ - t.pos.Z
		playerIsHigher := floorDz > t.maxStep && dist2d < 20.0
		if isBlocked || playerIsHigher {
			// Impulso Z (Regola il 400.0 in base alla gravità e al peso del demone)
			t.entity.AddForce(0.0, 0.0, t.mass*1000)
			// FONDAMENTALE: Svincola dall'attrito radente nel frame di stacco!
			t.entity.SetOnGround(false)
			// Bonus "Balzo": diamo un'extra spinta in avanti per aiutare a scavalcare i gap
			t.entity.AddForce(fx*1.5, fy*1.5, 0.0)
		}
	}
}

func (t *ThingEnemy) doFire(dist3d, dist2d, dz float64) {
	// 4. Logica di Fuoco (Spara se è ricaricato e a portata)
	if t.fireCooldown <= 0 && dist3d < 20.0 {
		weaponForward := t.radius * 2.0
		spawnX := t.pos.X + (math.Cos(t.angle) * weaponForward)
		spawnY := t.pos.Y + (math.Sin(t.angle) * weaponForward)
		spawnZ := t.pos.Z + (t.height * 0.5)
		bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
		aimPitch := math.Atan2(dz, dist2d)
		t.things.CreateBullet(t.volume, bulletPos, t.angle, aimPitch, 1.0, 1.0, 10)
		// Resetta il timer
		t.fireCooldown = 1.5
	}
}
