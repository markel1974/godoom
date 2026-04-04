package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IThing defines an interface for game objects with properties such as ID, position, animation, and lighting,
// and methods for computation and movement handling.
type IThing interface {
	GetId() string

	SetIdentifier(identifier int)

	GetIdentifier() int

	GetKind() config.ThingType

	GetAABB() *physics.AABB

	GetAnimation() *textures.Animation

	GetPosition() (float64, float64)

	GetLight() *Light

	GetFloorY() float64

	GetCeilY() float64

	GetEntity() *physics.Entity

	Compute(playerX float64, playerY float64)

	GetVolume() *Volume

	PhysicsApply()

	IsActive() bool

	SetActive(active bool)

	OnCollide(other IThing)
}

// WallSlidingEffect adjusts the velocity when sliding along a wall to simulate a wall-sliding effect with slight separation.
// Takes the current view coordinates, position, velocity, head and knee positions, and returns the modified velocity.
func WallSlidingEffect(volume *Volume, viewX, viewY, pX, pY, velX, velY, top, bottom float64, radius float64) (float64, float64) {
	const epsilon = 0.5
	// 1. Trova il segmento più vicino (con padding per gli spigoli)
	face := volume.CheckFacesClearance(viewX, viewY, pX, pY, top, bottom, radius)
	if face == nil {
		return velX, velY
	}
	start := face.GetStart()
	end := face.GetEnd()
	xd := end.X - start.X
	yd := end.Y - start.Y
	if lenSq := xd*xd + yd*yd; lenSq > 0 {
		// 2. Proiezione della velocità sul vettore tangenziale del muro
		dot := (velX*xd + velY*yd) / lenSq
		velX = xd * dot
		velY = yd * dot
		// 3. Calcolo della normale
		invLen := 1.0 / math.Sqrt(lenSq)
		nx := -yd * invLen
		ny := xd * invLen
		// 4. DIREZIONE DELLA NORMALE (Cruciale)
		// Calcoliamo il vettore che va dal muro al giocatore.
		// Se la normale calcolata punta nella direzione opposta, la invertiamo.
		// Usiamo viewX/Y come riferimento sicuro (posizione pre-collisione).
		midX := (start.X + end.X) * 0.5
		midY := (start.Y + end.Y) * 0.5
		toPlayerX := viewX - midX
		toPlayerY := viewY - midY
		if (toPlayerX*nx + toPlayerY*ny) < 0 {
			nx = -nx
			ny = -ny
		}
		// 5. Applica l'epsilon push per staccarsi fisicamente dal piano di collisione
		velX += nx * epsilon
		velY += ny * epsilon
	}
	return velX, velY
}
