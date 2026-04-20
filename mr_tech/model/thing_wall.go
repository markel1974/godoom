package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// ThingWall represents a UI component or control typically used to select a value from a range by sliding a handle.
type ThingWall struct {
	wall    *physics.Entity
	volumes *Volumes
}

// NewThingWall initializes and returns a new ThingWall instance, associating it with the provided Volumes object.
func NewThingWall(volumes *Volumes, restitution, friction float64) *ThingWall {
	return &ThingWall{
		volumes: volumes,
		wall:    physics.NewEntity(0, 0, 0, 0, 0, 0, -1, restitution, friction),
	}
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the ThingWall instance.
func (s *ThingWall) GetAABB() *physics.AABB {
	return s.wall.GetAABB()
}

// GetEntity retrieves the underlying physics.Entity associated with the ThingWall instance.
func (s *ThingWall) GetEntity() *physics.Entity {
	return s.wall
}

// ClosestFace finds the nearest face in a 3D space based on the given positions, velocity, top, bottom, and radius constraints.
// It updates the axis-aligned bounding box (AABB) and queries the volumes for the closest face intersection.
func (s *ThingWall) ClosestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64, float64) {
	minX := math.Min(viewX, pX) - radius
	maxX := math.Max(viewX, pX) + radius
	minY := math.Min(viewY, pY) - radius
	maxY := math.Max(viewY, pY) + radius
	minZ := math.Min(viewZ, pZ) - radius
	maxZ := math.Max(viewZ, pZ) + radius
	s.GetAABB().Rebuild(minX, minY, math.Min(bottom, minZ), maxX, maxY, math.Max(top, maxZ))
	return s.volumes.QueryClosestFace(s, viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius)
}

/*
// ClosestFace finds the nearest face a moving object collides with, given its trajectory and radius, using 3D collision detection.
func (s *ThingWall) ClosestFaceOld(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64) {
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
	s.world.QueryAABB(s, func(vol *Volume) {
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

*/
/*
func (s *ThingWall) ClosestFace3d(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, radius float64) (*Face, float64, float64, float64) {
	minX := math.Min(viewX, pX) - radius
	maxX := math.Max(viewX, pX) + radius
	minY := math.Min(viewY, pY) - radius
	maxY := math.Max(viewY, pY) + radius
	minZ := math.Min(viewZ, pZ) - radius
	maxZ := math.Max(viewZ, pZ) + radius

	s.GetAABB().Rebuild(minX, minY, math.Min(bottom, minZ), maxX, maxY, math.Max(top, maxZ))

	var closestFace *Face = nil
	minT := 1.0
	var colNx, colNy, colNz float64

	s.world.QueryAABB(s, func(vol *Volume) {
		for _, face := range vol.GetFaces() {
			// Skip sui portali attraversabili (se in modalità 2.5D) o ignoriamo le transizioni perfette
			if neighbor := face.GetNeighbor(); neighbor != nil {
				holeLow := math.Max(vol.GetMinZ(), neighbor.GetMinZ())
				holeHigh := math.Min(vol.GetMaxZ(), neighbor.GetMaxZ())
				if top <= holeHigh && bottom >= holeLow {
					continue
				}
			}

			pts := face.GetPoints()
			if len(pts) < 3 {
				continue // Assicuriamo che la faccia sia stata correttamente triangolata
			}
			p0, p1, p2 := pts[0], pts[1], pts[2]
			n := face.GetNormal()

			hit := false
			var hitT float64
			var cNx, cNy, cNz float64

			// FASE A: Discrete Test 3D (Risoluzione Compenetrazione su Rampe e Muri)
			closestX, closestY, closestZ := ClosestPointOnTriangle(pX, pY, pZ, p0, p1, p2)
			diffX, diffY, diffZ := pX-closestX, pY-closestY, pZ-closestZ
			distSq := diffX*diffX + diffY*diffY + diffZ*diffZ

			if distSq < radius*radius {
				hit = true
				hitT = 0.0 // Contatto istantaneo
				cDist := math.Sqrt(distSq)
				if cDist > 0.0001 {
					// Repulsione sferica radiale (fondamentale per scivolare morbidi sugli spigoli)
					cNx, cNy, cNz = diffX/cDist, diffY/cDist, diffZ/cDist
				} else {
					// Compenetrazione esatta, arretriamo lungo la normale nativa
					cNx, cNy, cNz = n.X, n.Y, n.Z
				}
			} else {
				// FASE B: Sweep Test 3D (Continuous Collision Detection) contro il piano
				distStart := (viewX-p0.X)*n.X + (viewY-p0.Y)*n.Y + (viewZ-p0.Z)*n.Z
				distEnd := (pX-p0.X)*n.X + (pY-p0.Y)*n.Y + (pZ-p0.Z)*n.Z

				if distStart >= -0.01 && distEnd < radius {
					dotVel := distEnd - distStart
					if dotVel < 0 {
						timeHit := (radius - distStart) / dotVel
						if timeHit >= 0.0 && timeHit <= 1.0 {
							// Proiezione del punto d'impatto atteso
							hX := viewX + velX*timeHit
							hY := viewY + velY*timeHit
							hZ := viewZ + velZ*timeHit

							// Sfruttiamo il tool matematico per verificare se il punto d'impatto
							// cade all'interno dei bordi del triangolo
							cpX, cpY, cpZ := ClosestPointOnTriangle(hX, hY, hZ, p0, p1, p2)
							dHX, dHY, dHZ := hX-cpX, hY-cpY, hZ-cpZ

							// Se il punto di impatto (h) e il punto più vicino sul triangolo (cp)
							// coincidono (distanza quasi 0), allora il raggio ha centrato il poligono.
							if (dHX*dHX + dHY*dHY + dHZ*dHZ) < 0.001 {
								hit = true
								hitT = timeHit
								cNx, cNy, cNz = n.X, n.Y, n.Z
							}
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

// ClosestPointOnTriangle trova il punto più vicino su un triangolo 3D rispetto a un punto P.
func ClosestPointOnTriangle(px, py, pz float64, p0, p1, p2 geometry.XYZ) (float64, float64, float64) {
	abX, abY, abZ := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	acX, acY, acZ := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	apX, apY, apZ := px-p0.X, py-p0.Y, pz-p0.Z

	d1 := abX*apX + abY*apY + abZ*apZ
	d2 := acX*apX + acY*apY + acZ*apZ
	if d1 <= 0.0 && d2 <= 0.0 {
		return p0.X, p0.Y, p0.Z // Regione del vertice A
	}

	bpX, bpY, bpZ := px-p1.X, py-p1.Y, pz-p1.Z
	d3 := abX*bpX + abY*bpY + abZ*bpZ
	d4 := acX*bpX + acY*bpY + acZ*bpZ
	if d3 >= 0.0 && d4 <= d3 {
		return p1.X, p1.Y, p1.Z // Regione del vertice B
	}

	vc := d1*d4 - d3*d2
	if vc <= 0.0 && d1 >= 0.0 && d3 <= 0.0 {
		v := d1 / (d1 - d3)
		return p0.X + v*abX, p0.Y + v*abY, p0.Z + v*abZ // Bordo AB
	}

	cpX, cpY, cpZ := px-p2.X, py-p2.Y, pz-p2.Z
	d5 := abX*cpX + abY*cpY + abZ*cpZ
	d6 := acX*cpX + acY*cpY + acZ*cpZ
	if d6 >= 0.0 && d5 <= d6 {
		return p2.X, p2.Y, p2.Z // Regione del vertice C
	}

	vb := d5*d2 - d1*d6
	if vb <= 0.0 && d2 >= 0.0 && d6 <= 0.0 {
		w := d2 / (d2 - d6)
		return p0.X + w*acX, p0.Y + w*acY, p0.Z + w*acZ // Bordo AC
	}

	va := d3*d6 - d5*d4
	if va <= 0.0 && (d4-d3) >= 0.0 && (d5-d6) >= 0.0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return p1.X + w*(p1.X-p1.X), p1.Y + w*(p2.Y-p1.Y), p1.Z + w*(p2.Z-p1.Z) // Bordo BC
	}

	// P cade internamente al triangolo: risoluzione tramite coordinate baricentriche
	denom := 1.0 / (va + vb + vc)
	v := vb * denom
	w := vc * denom
	return p0.X + abX*v + acX*w, p0.Y + abY*v + acY*w, p0.Z + abZ*v + acZ*w
}

*/

/*
// Compute calculates the resulting velocity and collision changes by simulating movement and impact within specified limits.
func (s *ThingWall) Compute2(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, radius float64, ballistic bool) (float64, float64, float64, bool) {
	changed := false
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ

	const acceleration = 0.8 //2.0
	const bounce = 0.8       //0.8

	// 1. Risoluzione Planare/Arbitraria (Muri e Geometria 3D)
	closestFace, colNx, colNy, colNz := s.ClosestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, zTop, zBottom, radius)

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
*/
