package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	IVertices
	id           string
	kind         config.ThingType
	angle        float64
	maxStep      float64
	speed        float64
	acceleration float64
	jumpForce    float64
	location     *Volume
	things       *Things
	isActive     bool
	cage         *CollisionCage

	inbox       chan *ThingEvent
	onCollision config.CollisionFunc
	onImpact    config.ImpactFunc
	done        chan struct{}
}

// NewThingBase creates a new ThingBase instance with specified configuration, material, sector, world, and things.
func NewThingBase(thing IThing, things *Things, cfg *config.Thing, location *Volume) *ThingBase {
	if cfg.OnCollision == nil {
		panic("onCollision is nil for thing:" + cfg.Id)
	}
	if cfg.OnImpact == nil {
		panic("OnImpact is nil for thing:" + cfg.Id)
	}
	if cfg.Mass == 0 {
		panic("mass is zero for thing:" + cfg.Id)
	}

	const cageMargin = 0.001
	t := &ThingBase{
		IVertices:    VerticesFactory(thing, cfg, things.GetMaterials()),
		id:           cfg.Id,
		angle:        cfg.Angle, // * (math.Pi / 180.0),
		kind:         cfg.Kind,
		speed:        cfg.Speed,
		acceleration: cfg.Acceleration,
		jumpForce:    cfg.JumpForce,
		location:     location,
		things:       things,
		maxStep:      0,
		isActive:     true,
		inbox:        make(chan *ThingEvent, 16),
		done:         make(chan struct{}),
		cage:         NewCollisionCage(thing, cageMargin),
		onImpact:     cfg.OnImpact,
		onCollision:  cfg.OnCollision,
	}

	entity := t.GetEntity()
	entity.SetOnGround(false)
	//TODO FROM CONFIG
	t.maxStep = entity.GetDepth() * 0.5 //cfg.Height * 0.5,
	return t
}

// GetId returns the identifier string of the ThingBase instance.
func (t *ThingBase) GetId() string {
	return t.id
}

// GetKind returns the type of the ThingBase as a value of the ThingType enumeration.
func (t *ThingBase) GetKind() config.ThingType {
	return t.kind
}

// GetLocation retrieves the current location associated with the ThingBase instance.
func (t *ThingBase) GetLocation() *Volume {
	return t.location
}

// GetCage retrieves the CollisionCage instance associated with the ThingBase, which defines its physical boundaries.
func (t *ThingBase) GetCage() *CollisionCage {
	return t.cage
}

// GetAngle returns the current rotation angle of the ThingBase instance as a float64 value.
func (t *ThingBase) GetAngle() float64 {
	return t.angle
}

// SetAngle updates the rotation angle of the ThingBase instance to the specified float64 value.
func (t *ThingBase) SetAngle(angle float64) {
	t.angle = angle
}

// GetMaxStep returns the maximum step value associated with the ThingBase instance.
func (t *ThingBase) GetMaxStep() float64 {
	return t.maxStep
}

// GetAcceleration returns the current acceleration value of the ThingBase.
func (t *ThingBase) GetAcceleration() float64 {
	return t.acceleration
}

// GetSpeed returns the current speed of the ThingBase instance as a float64.
func (t *ThingBase) GetSpeed() float64 {
	return t.speed
}

// IsActive checks if the ThingBase instance is currently active.
func (t *ThingBase) IsActive() bool {
	return t.isActive
}

// SetActive updates the activation state of the ThingBase instance and returns the updated state as a boolean.
func (t *ThingBase) SetActive(active bool) {
	t.isActive = active
}

// StagePrepare prepares the entity for staging by updating it and rebuilding the cage if the entity is moving.
func (t *ThingBase) StagePrepare() bool {
	entity := t.GetEntity()
	entity.Update()
	if !entity.IsMoving() {
		return false
	}
	t.cage.Rebuild(t.maxStep)
	return true
}

// StageResolve esegue la Fase Cinetica (Velocity Solver) dell'architettura split-solver.
// Calcola gli impulsi per risolvere gli urti elastici e l'attrito, demandando la correzione
// posizionale allo StageApply. solverJitter assorbe le fluttuazioni numeriche in virgola mobile.
func (t *ThingBase) StageResolve(solverIndex int, solverJitter float64) {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}

	entity := t.GetEntity()

	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)

		iMode := slot.GetImpactMode()
		switch iMode {
		case ImpactNone, ImpactStep:
			continue
		}

		penetration := slot.GetPenetration() + solverJitter
		nX, nY, nZ := slot.GetNormal()

		rFace := slot.GetRemoteFace()
		rParent := rFace.GetParent()
		rParentEnt := rParent.GetEntity()

		// Risoluzione Impulsi e Attrito (Fase Cinetica).
		// Forziamo penetration = 0.0 per inibire la stabilizzazione Baumgarte interna
		// al metodo ResolveImpact, evitando di iniettare energia cinetica fittizia.
		penetration = 0.0
		entity.ResolveImpact(rParentEnt, nX, nY, nZ, penetration)

		if thing := rParent.GetThing(); thing != nil {
			t.onCollision(t, thing)
		}
	}
}

// StageApply applies physical and geometrical adjustments to the entity based on collision resolution and position corrections.
func (t *ThingBase) StageApply(solverJitter float64) {
	if location := t.cage.GetVolume(); location != nil {
		t.location = location
	}
	entity := t.GetEntity()

	onGround := t.cage.BucketCount(BucketFloor) > 0
	entity.SetOnGround(onGround)

	// INTEGRAZIONE ORIGINALE (Sostituisce GetDisplacement)
	// Spostiamo l'oggetto usando il vettore pre-urto. Questo porta l'AABB
	// esattamente nel punto (tX, tY, tZ) dove la gabbia ha misurato la compenetrazione.
	dx, dy, dz := t.cage.GetDisplacement()
	entity.AddTo(dx, dy, dz)

	const slop = 0.01
	const positionalPercent = 1.0

	// RISOLUZIONE GEOMETRICA (Push-out)
	// Ora che siamo nel punto di impatto, le correzioni spingeranno
	// l'entità esattamente sulla superficie dell'ostacolo.
	for i := 0; i < t.cage.GetSlotsLen(); i++ {
		slot := t.cage.GetSlot(i)

		iMode := slot.GetImpactMode()
		switch iMode {
		case ImpactStep:
			// Il MoveToZ sovrascrive dz calcolato prima,
			// posizionandolo sopra il gradino
			entity.MoveToZ(slot.GetMaxZ())
		case ImpactInelastic:
			penetration := slot.GetPenetration()
			if penetration <= slop {
				continue
			}
			correction := ((penetration - slop) * positionalPercent) + solverJitter
			if correction <= 0.0 {
				continue
			}
			nX, nY, nZ := slot.GetNormal()
			entity.AddTo(nX*correction, nY*correction, nZ*correction)
		case ImpactElastic:
			/*
				penetration := slot.GetPenetration()
				if penetration <= slop {
					continue
				}
				correction := ((penetration - slop) * positionalPercent) + solverJitter
				if correction <= 0.0 {
					continue
				}
				otherEnt := slot.GetRemoteFace().GetParent().GetEntity()
				invMass1 := entity.GetInvMass()
				invMass2 := otherEnt.GetInvMass()
				invMassSum := invMass1 + invMass2

				ratio1 := invMass1 / invMassSum
				ratio2 := invMass2 / invMassSum
				p1 := correction * ratio1
				p2 := correction * ratio2

				nX, nY, nZ := slot.GetNormal()
				entity.AddTo(nX*p1, nY*p1, nZ*p1)
				otherEnt.AddTo(-nX*p2, -nY*p2, -nZ*p2)
			*/
		}
	}
}

// MoveTowards adjusts the entity's velocity towards a target speed in a specified direction using acceleration forces.
func (t *ThingBase) MoveTowards(dirX, dirY, targetSpeed, accelForce float64) {
	entity := t.GetEntity()
	vx, vy, _ := entity.GetVelocity()
	desiredVx := dirX * targetSpeed
	desiredVy := dirY * targetSpeed
	deltaVx := desiredVx - vx
	deltaVy := desiredVy - vy
	entity.AddForce(deltaVx*accelForce, deltaVy*accelForce, 0.0)
}

// GetBase returns the current instance of ThingBase. Useful for method chaining or accessing the base object.
func (t *ThingBase) GetBase() *ThingBase {
	return t
}

// Jump applies an upward force to the entity based on its mass, jump force, and a given factor if it is on the ground.
func (t *ThingBase) Jump(leapX float64, leapY float64, zFactor float64) bool {
	entity := t.GetEntity()
	if !entity.IsOnGround() {
		return false
	}
	fz := (entity.GetMass() * t.jumpForce) * zFactor
	entity.AddForce(leapX, leapY, fz)
	entity.SetOnGround(false)
	return true
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingBase) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
	t.things.CreateThrowable(throwableIndex, onCollision, onImpact, t.location, pos, angle, pitch, speed)
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingBase) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	const maxDistance = 4096.0
	var closestDist = maxDistance
	var closestThing IThing

	// Usiamo l'origine (pos) e il vettore direzione (dir) calcolato esternamente.
	// QueryRay richiede invDir (1.0/dir) che viene calcolato internamente.
	t.things.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, maxDistance, func(object physics.IAABB, distance float64) (float64, bool) {
		// Self-hit culling: l'entità che spara non deve colpire se stessa
		other, ok := object.(IThing)
		if !ok {
			return maxDistance, false
		}
		if t == other.GetBase() {
			return maxDistance, false
		}
		//if object == sender {
		//	return maxDistance, false
		//}
		closestThing = other
		closestDist = distance
		// Ray Shrinking: restringiamo il raggio d'azione dell'albero alla distanza dell'impatto trovato.
		return distance, true
	})

	if closestThing != nil {
		// 2. Calcolo del punto d'impatto reale (Origine + Direzione * Distanza)
		impactX := pos.X + (dirX * closestDist)
		impactY := pos.Y + (dirY * closestDist)
		impactZ := pos.Z + (dirZ * closestDist)

		fmt.Println("IMPACT: ", force, closestThing.GetId(), impactX, impactY, impactZ)
		// 3. Risoluzione dell'impatto
		force *= 100

		closestThing.GetEntity().AddForce(dirX*force, dirY*force, dirZ*force)
		closestThing.Impact(closestThing, id, force, closestDist, dirX, dirY, dirZ)
		t.spawnBulletHole(impactX, impactY, impactZ, closestThing)
	}
}

// Impact handles the interaction logic when this object collides with another object.
// other refers to the configuration of the colliding object.
// id is a unique identifier for the impact event.
// force denotes the magnitude of the impact force.
// closestDist represents the closest penetration between the objects upon collision.
// dirX, dirY, and dirZ specify the directional vector of the impact in 3D space.
func (t *ThingBase) Impact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	t.onImpact(t, other, id, force, closestDist, dirX, dirY, dirZ)
}

// spawnBulletHole creates a temporary visual entity at the specified coordinates to simulate a bullet hole effect.
// It offsets slightly from the surface to avoid Z-fighting and applies a visual decal for a limited duration.
func (t *ThingBase) spawnBulletHole(x, y, z float64, target IThing) {
	// Creiamo un'entità visiva temporanea tramite il gestore Things
	// Deve essere posizionata leggermente "staccata" dalla superficie (offset 0.1)
	// per evitare lo Z-fighting durante il rendering.

	// Se il target è un muro, possiamo estrarre la normale per ruotare la decalcomania
	// ma per ora posizioniamola semplicemente nel punto XYZ.
	//p.things.CreateDecal("BULLET_HOLE", x, y, z, 5.0) // 5.0 secondi di durata
}

/*
// deadZone checks if the provided velocity components are within a threshold, sets velocity to zero, and returns true if so.
func (t *ThingBase) deadZone(dx, dy, dz float64) bool {
	const sleepEpsilon = 0.005
	if math.Abs(dx) < sleepEpsilon && math.Abs(dy) < sleepEpsilon && math.Abs(dz) < sleepEpsilon {
		entity := t.vertices.GetEntity()
		entity.SetVx(0.0)
		entity.SetVy(0.0)
		if entity.IsOnGround() {
			entity.SetVz(0.0)
		}
		return true
	}
	return false
}
*/
