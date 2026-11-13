package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/material"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type IThingHandler interface {
	GetMaterials() *materials.Materials

	QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool))

	CreateThrowable(self volumes.IThing, throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, volume *volumes.Volume, pos geometry.XYZ, angle, pitch, speed float64)
}

type IVolumeHandler interface {
	QueryPoint(px, py, pz float64) (*volumes.Volume, *volumes.Face)
}

// Base represents the fundamental attributes and behaviors of an object in the system.
type Base struct {
	volumes.IVertices
	id             string
	kind           config.ThingType
	angle          float64
	maxStep        float64
	speed          float64
	impulse        float64
	wakeupDistance float64
	acceleration   float64
	jumpForce      float64
	location       *volumes.Volume
	things         IThingHandler
	isActive       bool
	cage           *collisions.Cage
	inbox          chan *volumes.ThingEvent
	onCollision    config.CollisionFunc
	onImpact       config.ImpactFunc
	done           chan struct{}
}

// NewBase creates a new Base instance with specified configuration, material, sector, world, and things.
func NewBase(thing volumes.IThing, things IThingHandler, cfg *config.Thing, location *volumes.Volume) *Base {
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
		IVertices:      volumes.VerticesFactory(thing, cfg, things.GetMaterials()),
		id:             cfg.Id,
		angle:          cfg.Angle,
		kind:           cfg.Kind,
		speed:          cfg.Speed,
		impulse:        cfg.Impulse,
		wakeupDistance: cfg.WakeupDistance,
		jumpForce:      cfg.JumpForce,
		acceleration:   cfg.Acceleration,
		location:       location,
		things:         things,
		maxStep:        0,
		isActive:       true,
		inbox:          make(chan *volumes.ThingEvent, 16),
		done:           make(chan struct{}),
		onImpact:       cfg.OnImpact,
		onCollision:    cfg.OnCollision,
	}

	entity := t.VerticesGetVolume().GetEntity()
	entity.SetOnGround(false)
	if cfg.RestitutionCombine != 0 || cfg.FrictionCombine != 0 {
		mat := entity.GetPhysicsMaterial()
		if cfg.RestitutionCombine != 0 {
			mat.SetRestitutionCombine(material.CombineMode(cfg.RestitutionCombine))
		}
		if cfg.FrictionCombine != 0 {
			mat.SetFrictionCombine(material.CombineMode(cfg.FrictionCombine))
		}
		entity.SetPhysicsMaterial(mat)
	}
	t.SetAngle(cfg.Angle)
	if cfg.MaxStepFactor > 0 {
		t.maxStep = entity.GetDepth() * cfg.MaxStepFactor
	}
	t.cage = collisions.NewCage(thing)

	return t
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
	return t.location
}

// GetCage retrieves the CollisionCage instance associated with the Base, which defines its physical boundaries.
func (t *Base) GetCage() volumes.ICollisionCage {
	return t.cage
}

// GetAngle returns the current rotation angle of the Base instance as a float64 value.
func (t *Base) GetAngle() float64 {
	return t.angle
}

// SetAngle updates the rotation angle of the Base instance and synchronizes the entity's orientation quaternion.
func (t *Base) SetAngle(angle float64) {
	t.angle = angle
	t.VerticesGetVolume().GetEntity().SetRotationAngleZ(angle)
}

// GetMaxStep returns the maximum step value associated with the Base instance.
func (t *Base) GetMaxStep() float64 {
	return t.maxStep
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
	entity := t.VerticesGetVolume().GetEntity()
	tx, ty, tz := entity.GetCenter()
	dx := tx - pX
	dy := ty - pY
	dz := tz - pZ
	distSq := dx*dx + dy*dy + dz*dz
	return distSq
}

func (t *Base) GetEntity() *rigid_body.Entity {
	return t.VerticesGetVolume().GetEntity()
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
	return t.isActive
}

// SetActive updates the activation state of the Base instance and returns the updated state as a boolean.
func (t *Base) SetActive(active bool) {
	t.isActive = active
}

// SetMaxStep sets the maximum step value for the Base instance, which determines the allowable step height.
func (t *Base) SetMaxStep(maxStep float64) {
	t.maxStep = maxStep
}

// IsVisible determines whether the Base instance is currently visible, returning true if it is visible.
func (t *Base) IsVisible() bool {
	return true
}

// StagePrepare prepares the entity for staging by updating it and rebuilding the cage if the entity is moving.
func (t *Base) StagePrepare() {
	entity := t.VerticesGetVolume().GetEntity()
	entity.Update()
	t.angle = entity.GetYaw()
	t.cage.Rebuild(t.maxStep)
}

// StageResolve resolves collisions and interactions for the entity by applying impulse and friction adjustments.
func (t *Base) StageResolve() {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}

	entity := t.VerticesGetVolume().GetEntity()

	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)

		iMode := slot.GetImpactMode()
		switch iMode {
		case collisions.ImpactStep:
			continue
		}
		var rParentEnt *rigid_body.Entity

		if rCage := slot.GetRCage(); rCage != nil {
			rParentEnt = rCage.GetEntity()
		} else {
			rFace := slot.GetRemoteFace()
			rParent := rFace.GetParent().(*volumes.Volume)
			rParentEnt = rParent.GetEntity()
		}
		// Risoluzione Impulsi e Attrito (Fase Cinetica) con Proprietà Esplicite del Materiale di Contatto.
		nX, nY, nZ := slot.GetNormal()
		//matOther := slot.GetPhysicsMaterial()
		//rParentEnt.Cinematic
		entity.ResolveImpact(rParentEnt, nX, nY, nZ)
	}
}

// StageApply applies physical and geometrical adjustments to the entity based on collision resolution and position corrections.
func (t *Base) StageApply() {
	if location := t.cage.GetVolume(); location != nil {
		t.location = location.(*volumes.Volume)
	}
	entity := t.VerticesGetVolume().GetEntity()
	// Spostiamo l'oggetto usando il vettore pre-urto. Questo porta AABB esattamente nel punto (tX, tY, tZ) dove la gabbia ha misurato la compenetrazione.
	dx, dy, dz := t.cage.GetDisplacement()
	entity.AddTo(dx, dy, dz)
	t.cage.SetApplied()
	// RISOLUZIONE GEOMETRICA (Push-out)
	// Ora che siamo nel punto di impatto, le correzioni spingeranno l'entità esattamente sulla superficie dell'ostacolo.
	for i := 0; i < t.cage.GetSlotsLen(); i++ {
		slot := t.cage.GetSlot(i)
		iMode := slot.GetImpactMode()
		switch iMode {
		case collisions.ImpactStep:
			entity.MoveToZ(slot.GetMaxZ())
			continue
		case collisions.ImpactElastic:
			nX, nY, nZ := slot.GetNormal()
			rCage := slot.GetRCage()
			sPushX, sPushY, sPushZ := t.cage.GetPushOffset()
			rPushX, rPushY, rPushZ := rCage.GetPushOffset()
			alreadyResolved := (sPushX-rPushX)*nX + (sPushY-rPushY)*nY + (sPushZ-rPushZ)*nZ
			if pen := slot.GetPenetration() - alreadyResolved; pen > 0 {
				if !slot.Penetrable() {
					entity.ResolveInelasticPenetration(nX, nY, nZ, pen)
				} else {
					entity.ResolveElasticPenetration(rCage.GetEntity(), rCage.HasFloor(), nX, nY, nZ, pen)
				}
			}
			t.onCollision(rCage.GetObject().(volumes.IThing))
		case collisions.ImpactInelastic:
			nX, nY, nZ := slot.GetNormal()
			sPushX, sPushY, sPushZ := t.cage.GetPushOffset()
			alreadyResolved := sPushX*nX + sPushY*nY + sPushZ*nZ
			if alreadyResolved < 0.0 {
				alreadyResolved = 0.0
			}
			pen := slot.GetPenetration() - alreadyResolved
			if pen > 0 {
				entity.ResolveInelasticPenetration(nX, nY, nZ, pen)
			}
			rParent := slot.GetRemoteFace().GetParent().(*volumes.Volume)
			if thing := rParent.GetThing(); thing != nil {
				t.onCollision(thing)
			}
		}
	}
}

// MoveTowards adjusts the entity's velocity towards a target speed in a specified direction using acceleration forces.
func (t *Base) MoveTowards(dirX, dirY, speed, accelForce float64) {
	entity := t.VerticesGetVolume().GetEntity()
	vx, vy, _ := entity.GetVelocity()
	desiredVx := dirX * speed
	desiredVy := dirY * speed
	deltaVx := desiredVx - vx
	deltaVy := desiredVy - vy
	entity.AddForce(deltaVx*accelForce, deltaVy*accelForce, 0.0)
}

// GetBase returns the current instance of Base. Useful for method chaining or accessing the base object.
func (t *Base) GetBase() *Base {
	return t
}

// Jump applies an upward force to the entity based on its mass, jump force, and a given factor if it is on the ground.
func (t *Base) Jump(self volumes.IThing, leapX float64, leapY float64, zFactor float64) bool {
	entity := t.VerticesGetVolume().GetEntity()
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
		closestThing.VerticesApplyImpact(dirX*force, dirY*force, dirZ*force, impactX, impactY, impactZ)
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
