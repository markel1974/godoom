package things

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_multi_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type IVolumesHandler interface {
	QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool))
}

type IThingHandler interface {
	GetMaterials() *materials.Materials

	GetVolumes() IVolumesHandler

	QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool))

	CreateThrowable(t ithing.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64)

	StagePart(part ithing.IThing)

	StageRemove(thing ithing.IThing)

	AddJoint(joint *collisions.JointConstraint)

	QueryVolume(x, y, z float64) *volumes.Volume
}

// Base represents the fundamental attributes and behaviors of an object in the system.
type Base struct {
	ithing.IVertices
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
	inbox          chan *ithing.ThingEvent
	onCollision    config.CollisionFunc
	onImpact       config.ImpactFunc
	maxStepFactor  float64
	deformable     bool
	done           chan struct{}
}

// NewBase creates a new Base instance with specified configuration, material, sector, world, and things.
func NewBase(things IThingHandler, cfg *config.Thing) *Base {
	return NewBaseWithVertices(things, cfg, CreateVertices(cfg, things.GetMaterials()))
}

// NewBaseWithVertices creates a Base instance using a caller-supplied IVertices implementation.
func NewBaseWithVertices(things IThingHandler, cfg *config.Thing, verts ithing.IVertices) *Base {
	if cfg.Mass == 0 {
		panic("mass is zero for thing:" + cfg.Id)
	}
	t := &Base{
		IVertices:      verts,
		id:             cfg.Id,
		angle:          cfg.Angle,
		kind:           cfg.Kind,
		speed:          cfg.Speed,
		impulse:        cfg.Impulse,
		wakeupDistance: cfg.WakeupDistance,
		jumpForce:      cfg.JumpForce,
		acceleration:   cfg.Acceleration,
		maxStepFactor:  cfg.MaxStepFactor,
		deformable:     cfg.Deformable,
		things:         things,
		inbox:          make(chan *ithing.ThingEvent, 16),
		done:           make(chan struct{}),
		onImpact:       cfg.OnImpact,
		onCollision:    cfg.OnCollision,
	}
	return t
}

// Prepare initializes the volume thing, configures its solver, applies location and movement limits, and resets state.
func (t *Base) Prepare(thing ithing.IThing, location *volumes.Volume) error {
	if err := t.IVertices.Setup(thing); err != nil {
		return err
	}
	if pv, ok := t.IVertices.(vertices_multi_element_3d.IPartedVertices); ok {
		pv.SetupParts(
			func(partVerts ithing.IVertices, collisionGroup uint64) ithing.IThing {
				pv3d := partVerts.(*vertices_multi_element_3d.Part)
				return NewAttachedPart(t.things, thing, location, t.id+"_part", pv3d)
			},
			func(part ithing.IThing) {
				t.things.StagePart(part)
			},
			func(joint *collisions.JointConstraint, childThing ithing.IThing, canDetach bool, breakThreshold float64) {
				t.things.AddJoint(joint)
				if canDetach {
					if binder, ok := childThing.(*Detached); ok {
						binder.BindJoint(joint, breakThreshold)
					}
				}
			},
		)
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

// SetActive updates the activation state of the Base instance and schedules removal when deactivated.
func (t *Base) SetActive(active bool) {
	t.solver.SetActive(active)
	if !active && t.things != nil {
		if selfThing, ok := t.solver.GetObject().(ithing.IThing); ok {
			t.things.StageRemove(selfThing)
		}
	}
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

// GetPartLen returns the number of parts from the vertex owner, or 0 if parts are unsupported.
func (t *Base) GetPartLen() int {
	if owner, ok := t.IVertices.(ithing.IPartOwner); ok {
		return owner.GetPartLen()
	}
	return 0
}

// GetPartAt returns the part at the given index, or nil if no part owner is available.
func (t *Base) GetPartAt(idx int) ithing.IThing {
	if owner, ok := t.IVertices.(ithing.IPartOwner); ok {
		return owner.GetPartAt(idx)
	}
	return nil
}

// SyncAttachedParts synchronizes all still-attached segment poses and tree bounds after owner locomotion.
func (t *Base) SyncAttachedParts() {
	if owner, ok := t.IVertices.(ithing.IPartOwner); ok {
		owner.SyncAttachedParts()
	}
}

// GetSolver returns the collisions.Solver managing this thing's physics simulation.
func (t *Base) GetSolver() *collisions.Solver {
	return t.solver
}

// Deformable reports whether the base can be deformed.
func (t *Base) IsDeformable() bool {
	return t.deformable
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
func (t *Base) Jump(self ithing.IThing, leapX float64, leapY float64, zFactor float64) bool {
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
func (t *Base) FireHitscan(self ithing.IThing, id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	const maxDistance = 4096.0

	var closestDistThing = maxDistance
	var closestThing ithing.IThing
	var closestThingNormal geometry.XYZ
	var hasThingNormal bool

	// Usiamo l'origine (pos) e il vettore direzione (dir) calcolato esternamente.
	// QueryRay richiede invDir (1.0/dir) che viene calcolato internamente.
	t.things.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, maxDistance, func(object aabb.IObject, distance float64) (float64, bool) {
		// Self-hit culling: l'entità che spara non deve colpire se stessa
		other, ok := object.(ithing.IThing)
		if !ok {
			return maxDistance, false
		}
		if self.IsEqual(other) {
			return maxDistance, false
		}
		if sub, sOk := other.(ithing.ISubPart); sOk && !sub.IsPhysicsFree() {
			if owner := sub.GetOwner(); owner != nil && self.IsEqual(owner) {
				return maxDistance, false
			}
		}

		meshDist := distance
		meshNorm := geometry.XYZ{}
		hasNorm := false

		if dHit, norm, okHit := other.IntersectRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, distance+128.0); okHit {
			meshDist = dHit
			meshNorm = norm
			hasNorm = true
		}

		closestThing = other
		closestDistThing = meshDist
		closestThingNormal = meshNorm
		hasThingNormal = hasNorm
		// Ray Shrinking: restringiamo il raggio d'azione dell'albero alla distanza dell'impatto trovato.
		return distance, true
	})

	var closestDistFace = maxDistance
	if closestThing != nil {
		closestDistFace = closestDistThing
	}
	var closestFace *volumes.Face
	var closestVol *volumes.Volume

	// Se il colpo interseca un muro/volume prima o invece di una Thing, il muro intercetta il colpo
	if vols := t.things.GetVolumes(); vols != nil {
		vols.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, closestDistFace, func(object aabb.IObject, distance float64) (float64, bool) {
			vol, ok := object.(*volumes.Volume)
			if !ok {
				return closestDistFace, false
			}
			if !vol.IsDeformable() {
				return closestDistFace, false
			}
			hit := false
			if facesTree := vol.GetFacesTree(); facesTree != nil {
				facesTree.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, closestDistFace, func(fObj aabb.IObject, fDist float64) (float64, bool) {
					face, fOk := fObj.(*volumes.Face)
					if !fOk || !face.IsSolid() {
						return closestDistFace, false
					}
					if tHit, okHit := face.IntersectRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, closestDistFace); okHit {
						closestDistFace = tHit
						closestFace = face
						closestVol = vol
						hit = true
						return tHit, true
					}
					return closestDistFace, false
				})
			}
			if hit {
				return closestDistFace, true
			}
			return closestDistFace, false
		})
	}

	if closestThing != nil && (closestFace == nil || closestDistThing < closestDistFace) {
		// Calcolo del punto d'impatto reale (Origine + Direzione * Distanza)
		impactX := pos.X + (dirX * closestDistThing)
		impactY := pos.Y + (dirY * closestDistThing)
		impactZ := pos.Z + (dirZ * closestDistThing)

		closestThing.WakeUp()
		closestThing.GetEntity().AddForceAtPoint(dirX*force, dirY*force, dirZ*force, impactX, impactY, impactZ)
		closestThing.Impact(self, id, force, closestDistThing, dirX, dirY, dirZ)

		if closestThing.IsDeformable() {
			// Deformiamo il Thing (Porta, Modello 3D, barile, etc.) se supporta la deformazione
			impactPoint := geometry.XYZ{X: impactX, Y: impactY, Z: impactZ}
			dentDepth, dentRadius := t.dentCompute(force)
			var impulse geometry.XYZ
			if hasThingNormal {
				normX, normY, normZ := closestThingNormal.X, closestThingNormal.Y, closestThingNormal.Z
				if dirX*normX+dirY*normY+dirZ*normZ > 0 {
					normX, normY, normZ = -normX, -normY, -normZ
				}
				impulse = geometry.XYZ{X: -normX * dentDepth, Y: -normY * dentDepth, Z: -normZ * dentDepth}
			} else {
				impulse = geometry.XYZ{X: dirX * dentDepth, Y: dirY * dentDepth, Z: dirZ * dentDepth}
			}
			closestThing.Deform(impactPoint, impulse, dentRadius)
		}

		//t.spawnBulletHole(impactX, impactY, impactZ, closestThing)
		return
	}

	if closestFace != nil {
		impactX := pos.X + (dirX * closestDistFace)
		impactY := pos.Y + (dirY * closestDistFace)
		impactZ := pos.Z + (dirZ * closestDistFace)
		impactPoint := geometry.XYZ{X: impactX, Y: impactY, Z: impactZ}
		dentDepth, dentRadius := t.dentCompute(force)
		normX, normY, normZ := closestFace.GetNormal()
		if dirX*normX+dirY*normY+dirZ*normZ > 0 {
			normX, normY, normZ = -normX, -normY, -normZ
		}
		impulse := geometry.XYZ{X: -normX * dentDepth, Y: -normY * dentDepth, Z: -normZ * dentDepth}
		if closestVol != nil {
			closestVol.Deform(impactPoint, impulse, dentRadius)
		} else {
			closestFace.Deform(impactPoint, impulse, dentRadius)
		}
		//t.spawnBulletHole(impactX, impactY, impactZ, nil)
		return
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

// onElasticCollision handles elastic collision energy with another object when it is a thing.
func (t *Base) onElasticCollision(other aabb.IObject, energy float64) {
	z, ok := other.(ithing.IThing)
	if ok {
		t.onCollision(z, energy)
	}
}

// onInelasticCollision handles an inelastic collision with another object using the given collision energy.
func (t *Base) onInelasticCollision(other aabb.IObject, energy float64) {
	//TODO
}

// DeformDent produces an indentation on the thing's vertices around a world-space impact point.
func (t *Base) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	if !t.IsDeformable() {
		return false
	}
	impulse := geometry.XYZ{X: -normal.X * depth, Y: -normal.Y * depth, Z: -normal.Z * depth}
	return t.Deform(impact, impulse, radius)
}

// Deform applies a local mesh deformation from an impact point, impulse, and radius.
// It returns false when the base is not deformable.
func (t *Base) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	if !t.IsDeformable() {
		return false
	}
	cx, cy, cz := t.ent.GetCenter()
	dx := impact.X - cx
	dy := impact.Y - cy
	dz := impact.Z - cz
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := t.ent.GetAxes()
	localImpact := geometry.XYZ{X: dx*ax0 + dy*ay0 + dz*az0, Y: dx*ax1 + dy*ay1 + dz*az1, Z: dx*ax2 + dy*ay2 + dz*az2}
	localImpulse := geometry.XYZ{X: impulse.X*ax0 + impulse.Y*ay0 + impulse.Z*az0, Y: impulse.X*ax1 + impulse.Y*ay1 + impulse.Z*az1, Z: impulse.X*ax2 + impulse.Y*ay2 + impulse.Z*az2}
	return t.IVertices.Deform(localImpact, localImpulse, radius)
}

// GetRigidFaces delegates to IVertices when the underlying renderable exposes a static rigid mesh template.
func (t *Base) GetRigidFaces() *volumes.Faces {
	if rs, ok := t.IVertices.(ithing.IRigidRenderSource); ok {
		return rs.GetRigidFaces()
	}
	return nil
}

// IsStaticRigidPose reports whether this thing uses a fixed single-frame rigid pose independent of tick.
func (t *Base) IsStaticRigidPose() bool {
	if sr, ok := t.IVertices.(interface{ IsStaticRigidPose() bool }); ok {
		return sr.IsStaticRigidPose()
	}
	return false
}

// dentCompute returns the dent depth and radius derived from force, using defaults for non-positive force.
func (t *Base) dentCompute(force float64) (float64, float64) {
	dentDepth := 2.5
	dentRadius := 8.0
	if force > 0 {
		dentDepth = math.Min(math.Max(force*0.00001, 1.5), 6.0)
		dentRadius = math.Min(math.Max(force*0.00003, 6.0), 16.0)
	}
	return dentDepth, dentRadius
}

/*
// spawnBulletHole creates a temporary visual entity at the specified coordinates to simulate a bullet hole effect.
// It offsets slightly from the surface to avoid Z-fighting and applies a visual decal for a limited duration.
func (t *Base) spawnBulletHole(x, y, z float64, target vertices.IThing) {
	// Creiamo un'entità visiva temporanea tramite il gestore Things
	// Deve essere posizionata leggermente "staccata" dalla superficie (offset 0.1)
	// per evitare lo Z-fighting durante il rendering.

	// Se il target è un muro, possiamo estrarre la normale per ruotare la decalcomania
	// ma per ora posizioniamola semplicemente nel punto XYZ.
	//p.things.CreateDecal("BULLET_HOLE", x, y, z, 5.0) // 5.0 secondi di durata
}
*/
