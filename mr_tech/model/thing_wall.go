package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// ThingWall represents a UI component or control typically used to select a value from a range by sliding a handle.
type ThingWall struct {
	aabb    *physics.AABB
	volumes *Volumes
}

// NewThingWall initializes and returns a new ThingWall instance, associating it with the provided Volumes object.
func NewThingWall(volumes *Volumes) *ThingWall {
	return &ThingWall{
		volumes: volumes,
		aabb:    physics.NewAABB(0, 0, 0, 0, 0, 0),
	}
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the ThingWall instance.
func (s *ThingWall) GetAABB() *physics.AABB {
	return s.aabb
}

// Compute calculates the resulting velocity and collision changes by simulating movement and impact within specified limits.
func (s *ThingWall) Compute(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, radius float64, ballistic bool) (float64, float64, float64, bool) {
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
			// Se l'impatto è superiore alla soglia, rimbalza
			if math.Abs(velZ) > 1.0 { // Soglia: di solito 2x o 3x la forza di gravità
				velZ = math.Abs(velZ) * bounce
			} else {
				// Resting contact: l'energia è troppo bassa, azzera il vettore
				velZ = zMinLimit - viewZ
			}
		}
		if nextZ > zMaxLimit {
			changed = true
			if math.Abs(velZ) > 1.0 {
				velZ = -math.Abs(velZ) * bounce
			} else {
				velZ = zMaxLimit - viewZ
			}
		}
	}
	return velX, velY, velZ, changed
}

// closestFace finds the nearest face a moving object collides with, given its trajectory and radius, using 3D collision detection.
func (s *ThingWall) closestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64) {
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
