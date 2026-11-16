package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type IThingHandler interface {
	GetMaterials() *materials.Materials

	QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool))

	CreateThrowable(t volumes.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64)
}

// Base represents the fundamental attributes and behaviors of an object in the system.
type Base struct {
	volumes.IVertices
	solver         *collisions.Solver
	ent            *rigid_body.Entity
	id             string
	kind           config.ThingType
	angle          float64
	speed          float64
	impulse        float64
	wakeupDistance float64
	acceleration   float64
	jumpForce      float64
	things         IThingHandler
	inbox          chan *volumes.ThingEvent
	onCollision    config.CollisionFunc
	onImpact       config.ImpactFunc
	maxStepFactor  float64
	done           chan struct{}
}

// NewBase creates a new Base instance with specified configuration, material, sector, world, and things.
func NewBase(things IThingHandler, cfg *config.Thing) *Base {
	if cfg.OnCollision == nil {
		panic("onCollision is nil for thing:" + cfg.Id)
	}
	if cfg.OnImpact == nil {
		panic("OnImpact is nil for thing:" + cfg.Id)
	}
	if cfg.Mass == 0 {
		panic("mass is zero for thing:" + cfg.Id)
	}
	t := &Base{
		IVertices:      volumes.CreateVertices(cfg, things.GetMaterials()),
		id:             cfg.Id,
		angle:          cfg.Angle,
		kind:           cfg.Kind,
		speed:          cfg.Speed,
		impulse:        cfg.Impulse,
		wakeupDistance: cfg.WakeupDistance,
		jumpForce:      cfg.JumpForce,
		acceleration:   cfg.Acceleration,
		maxStepFactor:  cfg.MaxStepFactor,
		things:         things,
		inbox:          make(chan *volumes.ThingEvent, 16),
		done:           make(chan struct{}),
		onImpact:       cfg.OnImpact,
		onCollision:    cfg.OnCollision,
	}
	return t
}

// Prepare initializes the volume thing, configures its solver, applies location and movement limits, and resets state.
func (t *Base) Prepare(thing volumes.IThing, location *volumes.Volume) error {
	if err := t.IVertices.Setup(thing); err != nil {
		return err
	}
	t.solver = t.IVertices.GetSolver()
	t.ent = t.solver.GetEntity()
	maxStep := 0.0
	if t.maxStepFactor > 0 {
		maxStep = t.ent.GetDepth() * t.maxStepFactor
	}
	t.solver.SetElasticCollision(t.onElasticCollision)
	t.solver.SetInelasticCollision(t.onInelasticCollision)
	t.solver.SetPrepareHooks(t.beforePrepare, nil)
	t.solver.SetLocation(location)
	t.solver.SetCanSleep(t.kind != config.ThingPlayerDef)
	t.solver.SetMaxStep(maxStep)

	t.ent.SetOnGround(false)
	t.SetAngle(t.angle)
	return nil
}

// GetId returns the identifier string of the Base instance.
func (t *Base) GetId() string {
	return t.id
}

// GetKind returns the type of the Base as a value of the ThingType enumeration.
func (t *Base) GetKind() config.ThingType {
	return t.kind
}

// GetLocation retrieves the current location associated with the Base instance.
func (t *Base) GetLocation() *volumes.Volume {
	loc, _ := t.solver.GetLocation().(*volumes.Volume)
	return loc
}

// GetAngle returns the current rotation angle of the Base instance as a float64 value.
func (t *Base) GetAngle() float64 {
	return t.angle
}

// SetAngle updates the rotation angle of the Base instance and synchronizes the entity's orientation quaternion.
func (t *Base) SetAngle(angle float64) {
	t.angle = angle
	t.ent.SetRotationAngleZ(angle)
}

// GetMaxStep returns the maximum step value associated with the Base instance.
func (t *Base) GetMaxStep() float64 {
	return t.solver.GetMaxStep()
}

// GetAcceleration returns the current acceleration value of the Base.
func (t *Base) GetAcceleration() float64 {
	return t.acceleration
}

// GetSpeed returns the current speed of the Base instance as a float64.
func (t *Base) GetSpeed() float64 {
	return t.speed
}

// GetDistSq returns the squared distance from the light's position to the given 3D point.
func (t *Base) GetDistSq(pX, pY, pZ float64) float64 {
	tx, ty, tz := t.ent.GetCenter()
	dx := tx - pX
	dy := ty - pY
	dz := tz - pZ
	distSq := dx*dx + dy*dy + dz*dz
	return distSq
}

// GetEntity returns the rigid body entity associated with the base.
func (t *Base) GetEntity() *rigid_body.Entity {
	return t.ent
}

// GetImpulse returns the
func (t *Base) GetImpulse() float64 {
	return t.impulse
}

// GetWakeUpDistance retrieves the
func (t *Base) GetWakeUpDistance() float64 {
	return t.wakeupDistance
}

// IsActive checks if the Base instance is currently active.
func (t *Base) IsActive() bool {
	return t.solver.IsActive()
}

// SetActive updates the activation state of the Base instance and returns the updated state as a boolean.
func (t *Base) SetActive(active bool) {
	t.solver.SetActive(active)
}

// SetMaxStep sets the maximum step value for the Base instance, which determines the allowable step height.
func (t *Base) SetMaxStep(maxStep float64) {
	t.solver.SetMaxStep(maxStep)
}

// IsVisible determines whether the Base instance is currently visible, returning true if it is visible.
func (t *Base) IsVisible() bool {
	return true
}

// CanSleep reports whether this entity type is allowed to enter the sleeping state.
func (t *Base) CanSleep() bool {
	return t.solver.CanSleep()
}

// IsSleeping reports whether the entity's rigid body is currently asleep.
func (t *Base) IsSleeping() bool {
	return t.solver.IsSleeping()
}

// WakeUp wakes the entity from sleep so it participates in physics steps.
func (t *Base) WakeUp() {
	t.solver.WakeUp()
}

// EnterSleep puts the entity to sleep and clears any pending frame displacement.
func (t *Base) EnterSleep() {
	t.solver.EnterSleep()
}

// UpdateSleepState evaluates post-solver motion and returns whether this entity qualifies to sleep.
func (t *Base) UpdateSleepState() bool {
	return t.solver.UpdateSleepState()
}

// SetSolverIndex records this entity's active index in the current solver tick (-1 when inactive).
func (t *Base) SetSolverIndex(idx int) {
	t.solver.SetSolverIndex(idx)
}

// GetSolverIndex returns this entity's active index in the current solver tick (-1 when inactive).
func (t *Base) GetSolverIndex() int {
	return t.solver.GetSolverIndex()
}

// GetSolver returns the collisions.Solver managing this thing's physics simulation.
func (t *Base) GetSolver() *collisions.Solver {
	return t.solver
}

// beforePrepare updates the base angle from its entity yaw and compiles textures for the current global tick.
func (t *Base) beforePrepare() {
	t.angle = t.ent.GetYaw()
	t.Compile(textures.GlobalTick())
}

// StagePrepare prepares the entity for staging by updating it and rebuilding the cage if the entity is moving.
func (t *Base) StagePrepare(session uint64) {
	t.solver.StagePrepare(session)
}

// StageWarmStart pre-applies persisted impulses on active contacts before PGS iterations begin.
func (t *Base) StageWarmStart() {
	t.solver.StageWarmStart()
}

// StageResolve resolves collisions and interactions for the entity by applying impulse and friction adjustments.
func (t *Base) StageResolve() {
	t.solver.StageResolve()
}

// StageApply advances entity position by its frame displacement, applies step-ups, and dispatches collision callbacks.
func (t *Base) StageApply() {
	t.solver.StageApply()
}

// StageResolvePosition performs one Gauss-Seidel positional push-out sweep across this entity's active contacts.
func (t *Base) StageResolvePosition() {
	t.solver.StageResolvePosition()
}

// MoveTowards adjusts the entity's velocity towards a target speed in a specified direction using acceleration forces.
func (t *Base) MoveTowards(dirX, dirY, speed, accelForce float64) {
	entity := t.ent
	vx, vy, _ := entity.GetVelocity()
	desiredVx := dirX * speed
	desiredVy := dirY * speed
	deltaVx := desiredVx - vx
	deltaVy := desiredVy - vy
	entity.AddForce(deltaVx*accelForce, deltaVy*accelForce, 0.0)
}

// Jump applies an upward force to the entity based on its mass, jump force, and a given factor if it is on the ground.
func (t *Base) Jump(self volumes.IThing, leapX float64, leapY float64, zFactor float64) bool {
	entity := t.ent
	if !entity.IsOnGround() {
		return false
	}
	fz := (entity.GetMass() * t.jumpForce) * zFactor
	entity.AddForce(leapX, leapY, fz)
	entity.SetOnGround(false)
	return true
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *Base) FireHitscan(self volumes.IThing, id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	const maxDistance = 4096.0
	var closestDist = maxDistance
	var closestThing volumes.IThing

	// Usiamo l'origine (pos) e il vettore direzione (dir) calcolato esternamente.
	// QueryRay richiede invDir (1.0/dir) che viene calcolato internamente.
	t.things.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, maxDistance, func(object aabb.IObject, distance float64) (float64, bool) {
		// Self-hit culling: l'entità che spara non deve colpire se stessa
		other, ok := object.(volumes.IThing)
		if !ok {
			return maxDistance, false
		}
		if self.IsEqual(other) {
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
		// Calcolo del punto d'impatto reale (Origine + Direzione * Distanza)
		impactX := pos.X + (dirX * closestDist)
		impactY := pos.Y + (dirY * closestDist)
		impactZ := pos.Z + (dirZ * closestDist)

		closestThing.GetEntity().AddForceAtPoint(dirX*force, dirY*force, dirZ*force, impactX, impactY, impactZ)
		closestThing.ApplyImpact(dirX*force, dirY*force, dirZ*force, impactX, impactY, impactZ)
		closestThing.Impact(self, id, force, closestDist, impactX, impactY, impactZ)
		t.spawnBulletHole(impactX, impactY, impactZ, closestThing)
	}
}

// Impact handles the interaction logic when this object collides with another object.
// other refers to the configuration of the colliding object.
// id is a unique identifier for the impact event.
// force denotes the magnitude of the impact force.
// closestDist represents the closest penetration between the objects upon collision.
// dirX, dirY, and dirZ specify the directional vector of the impact in 3D space.
func (t *Base) Impact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	t.onImpact(other, id, force, closestDist, dirX, dirY, dirZ)
}

// spawnBulletHole creates a temporary visual entity at the specified coordinates to simulate a bullet hole effect.
// It offsets slightly from the surface to avoid Z-fighting and applies a visual decal for a limited duration.
func (t *Base) spawnBulletHole(x, y, z float64, target volumes.IThing) {
	// Creiamo un'entità visiva temporanea tramite il gestore Things
	// Deve essere posizionata leggermente "staccata" dalla superficie (offset 0.1)
	// per evitare lo Z-fighting durante il rendering.

	// Se il target è un muro, possiamo estrarre la normale per ruotare la decalcomania
	// ma per ora posizioniamola semplicemente nel punto XYZ.
	//p.things.CreateDecal("BULLET_HOLE", x, y, z, 5.0) // 5.0 secondi di durata
}

func (t *Base) onElasticCollision(other collisions.IObject, energy float64) {
	z, ok := other.(volumes.IThing)
	if ok {
		t.onCollision(z, energy)
	}
}

func (t *Base) onInelasticCollision(other collisions.IObject, energy float64) {
	//TODO
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
