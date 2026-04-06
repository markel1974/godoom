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

	GetPosition() (float64, float64, float64)

	GetLight() *Light

	GetMinZ() float64

	GetMaxZ() float64

	GetEntity() *physics.Entity

	Compute(playerX float64, playerY float64, playerZ float64)

	GetVolume() *Volume

	PhysicsApply()

	IsActive() bool

	SetActive(active bool)

	OnCollide(other IThing)
}

// Slider represents a control or mechanism that uses an axis-aligned bounding box (AABB) for its spatial definition.
type Slider struct {
	aabb    *physics.AABB
	volumes *Volumes
}

// NewSlider creates and returns a pointer to a new Slider instance with an initialized AABB having default zero bounds.
func NewSlider(volumes *Volumes) *Slider {
	return &Slider{
		volumes: volumes,
		aabb:    physics.NewAABB(0, 0, 0, 0, 0, 0),
	}
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the Slider instance.
func (s *Slider) GetAABB() *physics.AABB {
	return s.aabb
}

// WallSlidingEffect adjusts the 3D velocity when sliding along a face to simulate physical sliding with separation.
// It implements Continuous Collision Detection (Sweep Test) and Discrete Point-to-Segment to prevent tunneling and corner snagging.
func (s *Slider) WallSlidingEffect(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (float64, float64, float64) {
	// 1. Broad-phase: Estendiamo l'AABB per includere il raggio e il movimento
	minX := math.Min(viewX, pX) - radius
	maxX := math.Max(viewX, pX) + radius
	minY := math.Min(viewY, pY) - radius
	maxY := math.Max(viewY, pY) + radius

	s.GetAABB().Rebuild(minX, minY, bottom, maxX, maxY, top)

	var closestFace *Face
	minT := 1.0 // Earliest Time of Impact (1.0 = nessun impatto in questo frame)
	var colNx, colNy, colNz float64

	s.volumes.QueryAABB(s, func(vol *Volume) {
		for _, face := range vol.GetFaces() {
			// Gestione Portali (Passaggio tra settori)
			if neighbor := face.GetNeighbor(); neighbor != nil {
				holeLow := math.Max(vol.GetMinZ(), neighbor.GetMinZ())
				holeHigh := math.Min(vol.GetMaxZ(), neighbor.GetMaxZ())
				// Se il giocatore passa attraverso il "buco" del portale, ignoriamo la collisione
				if top <= holeHigh && bottom >= holeLow {
					continue
				}
			}

			n := face.GetNormal()
			start := face.GetStart()
			end := face.GetEnd()

			edgeX := end.X - start.X
			edgeY := end.Y - start.Y
			edgeLenSq := edgeX*edgeX + edgeY*edgeY

			// Distanze proiettate contro il piano del muro (infinita)
			distStart := (viewX-start.X)*n.X + (viewY-start.Y)*n.Y
			distEnd := (pX-start.X)*n.X + (pY-start.Y)*n.Y

			hit := false
			var hitT float64
			var cNx, cNy float64

			// Fase A: Sweep Test (Continuous Collision Detection) - Previene il tunneling ad alte velocità
			// Verifichiamo se passiamo dal davanti al dietro (o dentro il raggio d'azione)
			if distStart >= -0.01 && distEnd < radius {
				dotVel := distEnd - distStart // Velocità proiettata sulla normale
				if dotVel < 0 {
					// Calcoliamo la frazione di tempo 't' in cui il cilindro tocca il piano
					t := (radius - distStart) / dotVel
					if t < 0 {
						t = 0
					} // Già in compenetrazione

					if t <= 1.0 {
						// Coordinate esatte al momento dell'impatto sul piano
						hitX := viewX + velX*t
						hitY := viewY + velY*t

						// Proiezione sul segmento per assicurarci di aver colpito il muro reale, non il vuoto
						vX := hitX - start.X
						vY := hitY - start.Y
						dotEdge := vX*edgeX + vY*edgeY

						// Tolleranza per non perdere gli spigoli
						if dotEdge >= -0.1 && dotEdge <= edgeLenSq+0.1 {
							hit = true
							hitT = t
							cNx, cNy = n.X, n.Y
						}
					}
				}
			}

			// Fase B: Discrete Point-to-Segment (Collision Detection sui Vertici) - Previene incagliamenti sugli spigoli
			if !hit {
				vX := pX - start.X
				vY := pY - start.Y
				tProj := 0.0
				if edgeLenSq > 0 {
					tProj = (vX*edgeX + vY*edgeY) / edgeLenSq
					tProj = math.Max(0.0, math.Min(1.0, tProj)) // Clamp ai confini del segmento
				}
				closestX := start.X + (tProj * edgeX)
				closestY := start.Y + (tProj * edgeY)

				diffX := pX - closestX
				diffY := pY - closestY
				distSq := diffX*diffX + diffY*diffY

				if distSq < radius*radius {
					hit = true
					hitT = 0.0 // Le collisioni discrete hanno priorità immediata (compenetrazione in atto)
					cDist := math.Sqrt(distSq)

					if tProj > 0.0 && tProj < 1.0 {
						// Impatto sul lato piatto
						cNx, cNy = n.X, n.Y
					} else {
						// Impatto sullo spigolo (Vertice): Creiamo una normale radiale per "scivolare" morbidamente attorno al cardine
						if cDist > 0.0001 {
							cNx, cNy = diffX/cDist, diffY/cDist
						} else {
							cNx, cNy = n.X, n.Y // Fallback di sicurezza
						}
					}
				}
			}

			// Se abbiamo registrato un hit, salviamo quello che è avvenuto PRIMA temporalmente (minT)
			if hit && hitT <= minT {
				minT = hitT
				closestFace = face
				colNx, colNy, colNz = cNx, cNy, n.Z
			}
		}
	})

	// 2. Risoluzione Newtoniana sul muro più vicino (o più precoce)
	if closestFace != nil {
		// Proiezione del vettore velocità: eliminiamo solo la forza perpendicolare alla faccia
		dot := (velX * colNx) + (velY * colNy) + (velZ * colNz)
		if dot < 0 {
			velX = velX - (dot * colNx)
			velY = velY - (dot * colNy)
			velZ = velZ - (dot * colNz)
		}
	}

	return velX, velY, velZ
}
