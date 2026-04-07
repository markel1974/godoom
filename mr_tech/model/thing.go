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

// AdjustPassage adjusts the player's movement vector while accounting for collisions, wall sliding, and vertical clipping constraints.
func (s *Slider) AdjustPassage(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, radius, height float64) (float64, float64, float64, bool) {
	// 1. Broad-phase vertical bounds (ingombro fisico del giocatore)
	// Coordinate target per il narrow-phase
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ
	velX, velY, velZ, _ = s.effectWallSliding(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, zTop, zBottom, radius)
	// 3. Vertical Clipping (Floor/Ceiling)
	// Limiti rigidi basati sul volume (settore) corrente.
	nextZ := viewZ + velZ
	if nextZ < zMinLimit {
		velZ = zMinLimit - viewZ
	}
	if viewZ+velZ > zMaxLimit {
		velZ = zMaxLimit - viewZ
	}
	return velX, velY, velZ, true
}

func (s *Slider) AdjustBounce(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, radius, height float64) (float64, float64, float64, bool) {
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ
	velX, velY, velZ, _ = s.effectWallBounce(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, zTop, zBottom, radius)
	nextZ := viewZ + velZ
	if nextZ < zMinLimit {
		// Rimbalzo sul pavimento
		velZ = math.Abs(velZ) * 0.8 // Perde un 20% di energia
		//t.entity.SetVz(velZ)
	} else if nextZ+height > zMaxLimit {
		// Rimbalzo sul soffitto
		velZ = -math.Abs(velZ) * 0.8
		//t.entity.SetVz(velZ)
	}

	return velX, velY, velZ, true
}

// effectWallSliding adjusts the 3D velocity when sliding along a face to simulate physical sliding with separation.
// It implements Continuous Collision Detection (Sweep Test) and Discrete Point-to-Segment to prevent tunneling and corner snagging.
func (s *Slider) effectWallSliding(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (float64, float64, float64, bool) {
	// 1. Broad-phase: Estendiamo l'AABB per includere il raggio e il movimento
	closestFace, colNx, colNy, colNz := s.closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius)
	// 2. Risoluzione Newtoniana sul muro più vicino (o più precoce)
	if closestFace != nil {
		// Proiezione del vettore velocità: eliminiamo solo la forza perpendicolare alla faccia
		dot := (velX * colNx) + (velY * colNy) + (velZ * colNz)
		if dot < 0 {
			velX = velX - (dot * colNx)
			velY = velY - (dot * colNy)
			velZ = velZ - (dot * colNz)
			return velX, velY, velZ, true
		}
	}
	return velX, velY, velZ, false
}

// effectWallBounce calculates the resulting direction of a projectile after collision applying continuous 3D bounce physics.
func (s *Slider) effectWallBounce(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (float64, float64, float64, bool) {
	closestFace, colNx, colNy, colNz := s.closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius)
	// 2. Risoluzione Newtoniana (Riflessione Vettoriale 3D)
	if closestFace != nil {
		restitution := 1.0 // 1.0 = Elastico (rimbalza senza perdere forza)
		// Prodotto scalare 3D completo
		dot := (velX * colNx) + (velY * colNy) + (velZ * colNz)
		if dot < 0 {
			velX -= (1.0 + restitution) * dot * colNx
			velY -= (1.0 + restitution) * dot * colNy
			velZ -= (1.0 + restitution) * dot * colNz
			return velX, velY, velZ, true
		}
	}
	return velX, velY, velZ, false
}

func (s *Slider) closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64) {
	minX := math.Min(viewX, pX) - radius
	maxX := math.Max(viewX, pX) + radius
	minY := math.Min(viewY, pY) - radius
	maxY := math.Max(viewY, pY) + radius
	minZ := math.Min(viewZ, pZ) - radius
	maxZ := math.Max(viewZ, pZ) + radius
	// Broad-phase: estendiamo l'AABB unendo i limiti logici (top/bottom) con la traiettoria sferica
	s.GetAABB().Rebuild(minX, minY, math.Min(bottom, minZ), maxX, maxY, math.Max(top, maxZ))
	var closestFace *Face = nil
	minT := 1.0                     // Earliest Time of Impact
	var colNx, colNy, colNz float64 // Aggiunto l'asse Z per la normale di impatto
	s.volumes.QueryAABB(s, func(vol *Volume) {
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
			// Vettori 3D del segmento
			edgeX := end.X - start.X
			edgeY := end.Y - start.Y
			edgeZ := end.Z - start.Z // Calcolo Z
			edgeLenSq := (edgeX * edgeX) + (edgeY * edgeY) + (edgeZ * edgeZ)
			// Distanze proiettate contro il piano 3D (Include l'asse Z)
			distStart := (viewX-start.X)*n.X + (viewY-start.Y)*n.Y + (viewZ-start.Z)*n.Z
			distEnd := (pX-start.X)*n.X + (pY-start.Y)*n.Y + (pZ-start.Z)*n.Z
			hit := false
			var hitT float64
			var cNx, cNy, cNz float64 // Variabili temporanee per la normale
			// Fase A: Sweep Test (Continuous Collision Detection 3D)
			if distStart >= -0.01 && distEnd < radius {
				dotVel := distEnd - distStart
				if dotVel < 0 {
					timeHit := (radius - distStart) / dotVel
					if timeHit < 0 {
						timeHit = 0
					}
					if timeHit <= 1.0 {
						hX := viewX + velX*timeHit
						hY := viewY + velY*timeHit
						hZ := viewZ + velZ*timeHit // Z d'impatto
						vX := hX - start.X
						vY := hY - start.Y
						vZ := hZ - start.Z // Offset Z sul piano
						dotEdge := (vX * edgeX) + (vY * edgeY) + (vZ * edgeZ)
						if dotEdge >= -0.1 && dotEdge <= edgeLenSq+0.1 {
							hit = true
							hitT = timeHit
							cNx, cNy, cNz = n.X, n.Y, n.Z
						}
					}
				}
			}
			// Fase B: Discrete Point-to-Segment 3D (Collision Detection sui Vertici)
			if !hit {
				vX := pX - start.X
				vY := pY - start.Y
				vZ := pZ - start.Z // Offset Z
				tProj := 0.0
				if edgeLenSq > 0 {
					tProj = ((vX * edgeX) + (vY * edgeY) + (vZ * edgeZ)) / edgeLenSq
					tProj = math.Max(0.0, math.Min(1.0, tProj))
				}
				closestX := start.X + (tProj * edgeX)
				closestY := start.Y + (tProj * edgeY)
				closestZ := start.Z + (tProj * edgeZ) // Quota Z più vicina
				diffX := pX - closestX
				diffY := pY - closestY
				diffZ := pZ - closestZ // Delta Z
				// Distanza sferica 3D dallo spigolo
				distSq := (diffX * diffX) + (diffY * diffY) + (diffZ * diffZ)
				if distSq < radius*radius {
					hit = true
					hitT = 0.0
					cDist := math.Sqrt(distSq)
					if tProj > 0.0 && tProj < 1.0 {
						cNx, cNy, cNz = n.X, n.Y, n.Z
					} else {
						if cDist > 0.0001 {
							// Normale radiale sferica 3D (Rimbalzo sullo spigolo vivo)
							cNx, cNy, cNz = diffX/cDist, diffY/cDist, diffZ/cDist
						} else {
							cNx, cNy, cNz = n.X, n.Y, n.Z
						}
					}
				}
			}
			if hit && hitT <= minT {
				minT = hitT
				closestFace = face
				colNx, colNy, colNz = cNx, cNy, cNz
			}
		}
	})
	return closestFace, colNx, colNy, colNz
}
