package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
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
	identifier   int
	cage         *CollisionCage
	vertices     IVertices
	inbox        chan *ThingEvent
	onCollision  config.CollisionFunc
	onImpact     config.ImpactFunc
	done         chan struct{}
}

// NewThingBase creates a new ThingBase instance with specified configuration, material, sector, world, and things.
func NewThingBase(thing IThing, things *Things, cfg *config.Thing, location *Volume) *ThingBase {
	vertices := VerticesFactory(thing, cfg, things.GetMaterials())

	if cfg.OnCollision == nil {
		panic("onCollision is nil for thing:" + cfg.Id)
	}
	if cfg.OnImpact == nil {
		panic("OnImpact is nil for thing:" + cfg.Id)
	}

	const cageMargin = 0.001
	t := &ThingBase{
		vertices:     vertices,
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
		identifier:   -1,
		inbox:        make(chan *ThingEvent, 16),
		done:         make(chan struct{}),
		cage:         NewCollisionCage(thing, cageMargin),
		onImpact:     cfg.OnImpact,
		onCollision:  cfg.OnCollision,
	}

	entity := t.vertices.GetEntity()
	entity.SetOnGround(false)
	t.maxStep = entity.GetDepth() * 0.5 //cfg.Height * 0.5,
	return t
}

// GetVertices retrieves the vertices of the ThingBase's associated triangular entity after updating their origin positions.
func (t *ThingBase) GetVertices() (*[]*Face, int, *[]*Face, int, float64, float64) {
	vCurr, vCurrCount, vNext, vNextCount, lerp := t.vertices.GetVertices(textures.GlobalTick())
	return vCurr, vCurrCount, vNext, vNextCount, lerp, t.vertices.GetBillboard()
}

// GetAngle returns the current rotation angle of the ThingBase instance as a float64 value.
func (t *ThingBase) GetAngle() float64 {
	return t.angle
}

// SetAngle updates the rotation angle of the ThingBase instance to the specified float64 value.
func (t *ThingBase) SetAngle(angle float64) {
	t.angle = angle
}

// SetAction sets the action for a vertex at the specified index in the ThingBase object.
func (t *ThingBase) SetAction(idx int) {
	t.vertices.SetAction(idx)
}

// GetDisplacement returns the x, y, and z coordinates of the position as three float64 values.
func (t *ThingBase) GetDisplacement() (float64, float64, float64) {
	return t.vertices.GetDisplacement()
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

// GetMaxStep returns the maximum step value associated with the ThingBase instance.
func (t *ThingBase) GetMaxStep() float64 {
	return t.maxStep
}

func (t *ThingBase) GetCage() *CollisionCage {
	return t.cage
}

// GetEntity returns the physics.Entity associated with the current ThingBase instance.
func (t *ThingBase) GetEntity() *physics.Entity {
	return t.vertices.GetEntity()
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the associated physics entity.
func (t *ThingBase) GetAABB() *physics.AABB {
	return t.vertices.GetEntity().GetAABB()
}

// GetRadius retrieves the radius of the ThingBase instance as a float64 value.
func (t *ThingBase) GetRadius() float64 {
	return t.vertices.GetEntity().GetWidth() * 0.5
}

// GetSize returns the dimensions (width, height, depth) of the entity as a tuple of float64 values.
func (t *ThingBase) GetSize() (float64, float64, float64) {
	return t.vertices.GetEntity().GetSize()
}

// GetDepth retrieves the depth value of the entity associated with the ThingBase instance.
func (t *ThingBase) GetDepth() float64 {
	return t.vertices.GetEntity().GetDepth()
}

// GetWidth retrieves the width of the underlying entity associated with the ThingBase.
func (t *ThingBase) GetWidth() float64 {
	return t.vertices.GetEntity().GetWidth()
}

// GetMass retrieves the mass value of the underlying entity associated with the ThingBase instance.
func (t *ThingBase) GetMass() float64 {
	return t.vertices.GetEntity().GetMass()
}

// GetVelocity retrieves the current velocity of the entity as a tuple of X, Y, and Z components.
func (t *ThingBase) GetVelocity() (float64, float64, float64) {
	return t.vertices.GetEntity().GetVelocity()
}

// IsOnGround checks if the entity associated with ThingBase is currently on the ground and returns true if it is.
func (t *ThingBase) IsOnGround() bool {
	return t.vertices.GetEntity().IsOnGround()
}

// SetOnGround sets the on-ground state of the entity to the specified boolean value.
func (t *ThingBase) SetOnGround(g bool) {
	t.vertices.GetEntity().SetOnGround(g)
}

// AddForce applies a force vector (fx, fy, fz) to the entity associated with the ThingBase.
func (t *ThingBase) AddForce(fx, fy, fz float64) {
	t.vertices.GetEntity().AddForce(fx, fy, fz)
}

// GetBottomLeft returns the bottom-left coordinates (x, y) and an optional z-value of the entity associated with the ThingBase.
func (t *ThingBase) GetBottomLeft() (float64, float64, float64) {
	return t.vertices.GetEntity().GetBottomLeft()
}

// GetBottomCenter returns the center-bottom coordinates (x, y, z) of the ThingBase entity.
func (t *ThingBase) GetBottomCenter() (float64, float64, float64) {
	return t.vertices.GetEntity().GetBottomCenter()
}

// GetCenter calculates and returns the center coordinates (x, y, z) of the entity within ThingBase.
func (t *ThingBase) GetCenter() (float64, float64, float64) {
	return t.vertices.GetEntity().GetCenter()
}

// GetVolume retrieves the volume associated with the ThingBase instance.
func (t *ThingBase) GetVolume() *Volume {
	return t.vertices.GetVolume()
}

// GetAcceleration returns the current acceleration value of the ThingBase.
func (t *ThingBase) GetAcceleration() float64 {
	return t.acceleration
}

// GetSpeed returns the current speed of the ThingBase instance as a float64.
func (t *ThingBase) GetSpeed() float64 {
	return t.speed
}

// SetIdentifier sets the unique identifier for the ThingBase instance.
func (t *ThingBase) SetIdentifier(identifier int) {
	t.identifier = identifier
}

// GetIdentifier returns the unique identifier of the ThingBase instance.
func (t *ThingBase) GetIdentifier() int {
	return t.identifier
}

// IsActive checks if the ThingBase instance is currently active.
func (t *ThingBase) IsActive() bool {
	return t.isActive
}

// SetActive updates the activation state of the ThingBase instance and returns the updated state as a boolean.
func (t *ThingBase) SetActive(active bool) {
	t.isActive = active
}

func (t *ThingBase) StagePrepare() bool {
	entity := t.vertices.GetEntity()
	entity.Update()
	if !entity.IsMoving() {
		return false
	}
	t.cage.Rebuild()
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

	entity := t.vertices.GetEntity()
	baseZ := t.cage.GetBaseZ()
	stepZ := baseZ + t.maxStep

	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)
		// Filtro topologico per ostacoli scavalcabili (Stair-Stepping).
		// Se l'ostacolo è statico e rientra nel maxStep, inibiamo la normale di collisione
		// orizzontale e programmiamo lo step verticale per lo StageApply.

		if !slot.IsDynamic() {
			//if solverIndex == 0 {
			switch slot.GetBucket() {
			case BucketWallEast, BucketWallWest, BucketWallNorth, BucketWallSouth:
				maxZ := slot.GetMaxZ()
				if maxZ <= baseZ {
					continue // down-hill (in discesa)
				}
				if maxZ <= stepZ {
					slot.SetStep(1, maxZ)
					continue // up-hill (gradino superabile)
				}
				slot.SetStep(0, 0) // Ostacolo insuperabile (muro)
			case BucketFloor, BucketCeiling:
				slot.SetStep(0, 0)
			}
		}

		otherFace := slot.GetRemoteFace()
		penetration := slot.GetPenetration() + solverJitter
		nX, nY, nZ := slot.GetNormal()

		otherParent := otherFace.GetParent()
		otherParentEnt := otherParent.GetEntity()

		// Risoluzione Impulsi e Attrito (Fase Cinetica).
		// Forziamo penetration = 0.0 per inibire la stabilizzazione Baumgarte interna
		// al metodo ResolveImpact, evitando di iniettare energia cinetica fittizia.
		penetration = 0.0
		entity.ResolveImpact(otherParentEnt, nX, nY, nZ, penetration)

		if thing := otherParent.GetThing(); thing != nil {
			t.onCollision(t, thing)
		}
	}
}

// StageApply applies physical and geometrical adjustments to the entity based on collision resolution and position corrections.
func (t *ThingBase) StageApply(solverJitter float64) {
	if location := t.cage.GetVolume(); location != nil {
		t.location = location
	}
	entity := t.vertices.GetEntity()

	isGrounded := t.cage.BucketCount(BucketFloor) > 0
	entity.SetOnGround(isGrounded)

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

		stepMode, stepSize := slot.GetStep()
		switch stepMode {
		case 1:
			// Il MoveToZ sovrascrive dz calcolato prima,
			// posizionando il player esattamente sopra il gradino
			entity.MoveToZ(stepSize)
			continue
		case -1:
			continue
		case 0:
			penetration := slot.GetPenetration()
			if penetration <= slop {
				continue
			}
			correction := ((penetration - slop) * positionalPercent) + solverJitter
			if correction <= 0.0 {
				continue
			}
			if slot.IsDynamic() {
				otherEnt := slot.GetRemoteFace().GetParent().GetEntity()
				invMass1 := entity.GetInvMass()
				invMass2 := otherEnt.GetInvMass()
				invMassSum := invMass1 + invMass2
				if invMassSum > 0.0 {
					nX, nY, nZ := slot.GetNormal()
					ratio1 := invMass1 / invMassSum
					p1 := correction * ratio1
					entity.AddTo(nX*p1, nY*p1, nZ*p1)

					ratio2 := invMass2 / invMassSum
					p2 := correction * ratio2
					otherEnt.AddTo(-nX*p2, -nY*p2, -nZ*p2)
				}
			} else {
				nX, nY, nZ := slot.GetNormal()
				entity.AddTo(nX*correction, nY*correction, nZ*correction)
			}
		}
	}
}

// MoveTowards adjusts the entity's velocity towards a target speed in a specified direction using acceleration forces.
func (t *ThingBase) MoveTowards(dirX, dirY, targetSpeed, accelForce float64) {
	entity := t.vertices.GetEntity()
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
		closestThing.AddForce(dirX*force, dirY*force, dirZ*force)
		closestThing.Impact(closestThing, id, force, closestDist, dirX, dirY, dirZ)
		t.spawnBulletHole(impactX, impactY, impactZ, closestThing)
	}
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
