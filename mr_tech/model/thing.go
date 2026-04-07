package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IThing defines an interface for a game entity with methods for retrieving identifiers, position, and physics properties.
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

// WallPhysics represents a UI component or control typically used to select a value from a range by sliding a handle.
type WallPhysics struct {
	aabb    *physics.AABB
	volumes *Volumes
}

// NewWallPhysics initializes and returns a new WallPhysics instance, associating it with the provided Volumes object.
func NewWallPhysics(volumes *Volumes) *WallPhysics {
	return &WallPhysics{
		volumes: volumes,
		aabb:    physics.NewAABB(0, 0, 0, 0, 0, 0),
	}
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the WallPhysics instance.
func (s *WallPhysics) GetAABB() *physics.AABB {
	return s.aabb
}

// AdjustVelocity elabora dinamicamente lo scivolamento o il rimbalzo analizzando il modulo della velocità.
func (s *WallPhysics) AdjustVelocity(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, radius float64, ballistic bool) (float64, float64, float64, bool) {
	changed := false
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ

	const acceleration = 0.8 //2.0
	const bounce = 0.8       //0.8

	// 1. Risoluzione Planare/Arbitraria (Muri e Geometria 3D)
	closestFace, colNx, colNy, colNz := s.closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, zTop, zBottom, radius)

	if closestFace != nil {
		dot := (velX * colNx) + (velY * colNy) + (velZ * colNz)
		if dot < 0 {
			changed = true
			if ballistic {
				// Comportamento Bouncing (Riflessione elastica pura per proiettili)
				velX -= acceleration * dot * colNx
				velY -= acceleration * dot * colNy
				velZ -= acceleration * dot * colNz
			} else {
				// Comportamento Sliding (Assorbimento dell'impatto per Player/Mostri)
				velX -= dot * colNx
				velY -= dot * colNy
				velZ -= dot * colNz
			}
		}
	}
	nextZ := viewZ + velZ
	if ballistic {
		if nextZ < zMinLimit {
			changed = true
			velZ = math.Abs(velZ) * bounce // Proiettile che rimbalza sul pavimento
		}
		if nextZ > zMaxLimit {
			changed = true
			velZ = -math.Abs(velZ) * bounce // Proiettile che rimbalza sul soffitto
		}
	} else {
		if nextZ < zMinLimit {
			changed = true
			velZ = zMinLimit - viewZ
		}
		if nextZ > zMaxLimit {
			changed = true
			velZ = zMaxLimit - viewZ
		}
	}
	return velX, velY, velZ, changed
}

// closestFace finds the nearest face a moving object collides with, given its trajectory and radius, using 3D collision detection.
func (s *WallPhysics) closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64) {
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
