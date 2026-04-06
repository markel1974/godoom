package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
	floorStartY float64
}

// NewThingBullet creates and initializes a new ThingBullet instance.
func NewThingBullet(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities) *ThingBullet {
	p := &ThingBullet{
		ThingBase:   NewThingBase(cfg, anim, volume, sectors, entities),
		floorStartY: volume.GetMinZ(),
	}
	// Annulla il decadimento inerziale per mantenere una velocità lineare costante
	p.entity.SetFriction(1.0) // 1.0 = nessuna perdita di velocità su X/Y
	p.entity.SetGForce(1.0)
	p.entities.AddThing(p)

	// Calculate the directional vector based on the original firing angle
	dirX := math.Cos(p.angle) * p.speed
	dirY := math.Sin(p.angle) * p.speed

	const acceleration = 0.15
	p.entity.SetVx(p.entity.GetVx()*(1-acceleration) + (dirX * acceleration))
	p.entity.SetVy(p.entity.GetVy()*(1-acceleration) + (dirY * acceleration))
	return p
}

func (t *ThingBullet) GetFloorY() float64 {
	velSq := (t.entity.GetVx() * t.entity.GetVx()) + (t.entity.GetVy() * t.entity.GetVy())
	if velSq <= 0.01 || t.speed <= 0 {
		return t.volume.GetMinZ()
	}
	ratio := math.Sqrt(velSq) / t.speed
	if ratio <= 0 {
		return t.volume.GetMinZ()
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	return t.floorStartY * ratio
}

func (t *ThingBullet) Compute(playerX float64, playerY float64, playerZ float64) {
	// Logica eventuale di homing-missile o timeout qui
}

// PhysicsApply updates the bullet's position based on physics deltas (X, Y, Z)
// and synchronizes its state with the 3D spatial partitioning.
func (t *ThingBullet) PhysicsApply() {
	// 1. Recupero dal motore fisico (Baricentro Reale 3D)
	eX, eY, eZ := t.entity.GetCenter()

	// Calcolo quota base del proiettile
	baseZ := eZ - (t.entity.GetDepth() / 2.0)

	// 2. Calcolo dei delta completi
	tx := (eX - t.position.X) + t.entity.GetVx()
	ty := (eY - t.position.Y) + t.entity.GetVy()
	tz := (baseZ - t.position.Z) + t.entity.GetVz()

	if math.Abs(tx) > minMovement || math.Abs(ty) > minMovement || math.Abs(tz) > minMovement {
		// 3. Risoluzione dei vincoli ambientali 3D (Bounces e Portali)
		vx, vy, vz := t.adjustPassage(tx, ty, tz)

		// 4. Aggiornamento posizione logica
		t.position.X += vx
		t.position.Y += vy
		t.position.Z += vz

		// 5. Aggiornamento AABB Tree (basato sul baricentro per prevenire cambi errati)
		bulletBaseZ := t.position.Z
		bulletTopZ := t.position.Z + t.height
		const bulletStep = 0.0
		if newVolume := t.volumes.SearchVolume3d(t.volume, t.position.X, t.position.Y, bulletBaseZ, bulletTopZ, bulletStep); newVolume != nil && newVolume != t.volume {
			t.volume = newVolume
		}

		t.entities.UpdateThing(t, t.position.X, t.position.Y, t.position.Z)
	}
}

// adjustPassage resolves the 3D trajectory of the bullet, handling bounces via the spatial tree.
func (t *ThingBullet) adjustPassage(velX, velY, velZ float64) (float64, float64, float64) {
	viewZ := t.position.Z
	bottom := viewZ
	top := viewZ + t.height
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ

	// Rimbalzo sui muri (Broad & Narrow phase)
	velX, velY, velZ = t.EffectBounce(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom)

	// Clipping e Rimbalzo Pavimento/Soffitto
	nextZ := viewZ + velZ
	minZ := t.volume.GetMinZ()
	maxZ := t.volume.GetMaxZ()

	if nextZ < minZ {
		// Rimbalzo sul pavimento
		velZ = math.Abs(velZ) * 0.8 // Perde un 20% di energia
		t.entity.SetVz(velZ)
	} else if nextZ+t.height > maxZ {
		// Rimbalzo sul soffitto
		velZ = -math.Abs(velZ) * 0.8
		t.entity.SetVz(velZ)
	}

	return velX, velY, velZ
}

// EffectBounce calculates the resulting direction of a projectile after collision applying continuous 3D bounce physics.
func (t *ThingBullet) EffectBounce(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom float64) (float64, float64, float64) {
	minX := math.Min(viewX, pX) - t.radius
	maxX := math.Max(viewX, pX) + t.radius
	minY := math.Min(viewY, pY) - t.radius
	maxY := math.Max(viewY, pY) + t.radius

	t.slider.GetAABB().Rebuild(minX, minY, bottom, maxX, maxY, top)

	var closestFace *Face = nil
	minT := 1.0 // Tempo di impatto (Continuous Collision)
	var colNx, colNy float64

	t.volumes.QueryAABB(t.slider, func(vol *Volume) {
		for _, face := range vol.GetFaces() {
			if neighbor := face.GetNeighbor(); neighbor != nil {
				holeLow := math.Max(vol.GetMinZ(), neighbor.GetMinZ())
				holeHigh := math.Min(vol.GetMaxZ(), neighbor.GetMaxZ())
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

			distStart := (viewX-start.X)*n.X + (viewY-start.Y)*n.Y
			distEnd := (pX-start.X)*n.X + (pY-start.Y)*n.Y

			hit := false
			var hitT float64
			var cNx, cNy float64

			// Fase A: Sweep Test (Previene il passaggio attraverso i muri)
			if distStart >= -0.01 && distEnd < t.radius {
				dotVel := distEnd - distStart
				if dotVel < 0 {
					timeHit := (t.radius - distStart) / dotVel
					if timeHit < 0 {
						timeHit = 0
					}
					if timeHit <= 1.0 {
						hX := viewX + velX*timeHit
						hY := viewY + velY*timeHit
						vX := hX - start.X
						vY := hY - start.Y
						dotEdge := vX*edgeX + vY*edgeY

						// Assicuriamoci che l'impatto avvenga tra l'inizio e la fine del segmento
						if dotEdge >= -0.1 && dotEdge <= edgeLenSq+0.1 {
							hit = true
							hitT = timeHit
							cNx, cNy = n.X, n.Y
						}
					}
				}
			}

			// Fase B: Point-to-segment (Gestione spigoli vivi)
			if !hit {
				vX := pX - start.X
				vY := pY - start.Y
				tProj := 0.0
				if edgeLenSq > 0 {
					tProj = (vX*edgeX + vY*edgeY) / edgeLenSq
					tProj = math.Max(0.0, math.Min(1.0, tProj))
				}
				closestX := start.X + (tProj * edgeX)
				closestY := start.Y + (tProj * edgeY)

				diffX := pX - closestX
				diffY := pY - closestY
				distSq := diffX*diffX + diffY*diffY

				if distSq < t.radius*t.radius {
					hit = true
					hitT = 0.0
					cDist := math.Sqrt(distSq)
					if tProj > 0.0 && tProj < 1.0 {
						cNx, cNy = n.X, n.Y
					} else {
						if cDist > 0.0001 {
							cNx, cNy = diffX/cDist, diffY/cDist
						} else {
							cNx, cNy = n.X, n.Y
						}
					}
				}
			}

			if hit && hitT <= minT {
				minT = hitT
				closestFace = face
				colNx, colNy = cNx, cNy
			}
		}
	})

	if closestFace != nil {
		// Riflessione Vettoriale Perfetta: V' = V - (1+e)(V·N)N
		restitution := 1.0 // 1.0 = Rimbalzo perfettamente elastico (non perde energia sul muro)
		dot := (velX * colNx) + (velY * colNy)

		if dot < 0 {
			velX = velX - (1.0+restitution)*dot*colNx
			velY = velY - (1.0+restitution)*dot*colNy

			// Trasmettiamo il nuovo vettore velocità al motore fisico per i calcoli successivi
			t.entity.SetVx(velX)
			t.entity.SetVy(velY)

			// Se il proiettile esplode all'impatto invece di rimbalzare, decommentalo qui:
			// t.OnCollide(closestFace)
		}
	}

	return velX, velY, velZ
}

// OnCollide handles the interaction when the bullet collides with another object.
func (t *ThingBullet) OnCollide(other IThing) {
	if enemy, ok := other.(*ThingEnemy); ok {
		_ = enemy
		// enemy.TakeDamage(...)
		// t.SetActive(false)
	}
}

// IsActive checks if the ThingBullet is currently active and operational.
func (t *ThingBullet) IsActive() bool {
	return t.isActive
}
