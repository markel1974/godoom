package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	id           string
	pos          geometry.XYZ
	mass         float64
	radius       float64
	height       float64
	angle        float64
	maxStep      float64
	kind         config.ThingType
	speed        float64
	acceleration float64
	location     *Volume
	animation    *textures.Animation
	world        *Volumes
	things       *Things
	entity       *physics.Entity
	isActive     bool
	identifier   int
	wall         *ThingWall
	volume       *Volume
}

// NewThingBaseSprite creates a new ThingBase instance with specified configuration, animation, sector, world, and things.
func NewThingBaseSprite(things *Things, cfg *config.Thing, pos geometry.XYZ, anim *textures.Animation, volume *Volume) *ThingBase {
	volumes := things.GetVolumes()
	entX := pos.X - cfg.Radius
	entY := pos.Y - cfg.Radius
	entZ := pos.Z
	entW := cfg.Radius * 2
	entH := cfg.Radius * 2
	entD := cfg.Height
	t := &ThingBase{
		id:           cfg.Id,
		angle:        cfg.Angle,
		kind:         cfg.Kind,
		mass:         cfg.Mass,
		radius:       cfg.Radius,
		height:       cfg.Height,
		speed:        cfg.Speed,
		acceleration: cfg.Acceleration,
		pos:          pos,
		location:     volume,
		animation:    anim,
		world:        volumes,
		things:       things,
		maxStep:      cfg.Height * 0.5,
		entity:       physics.NewEntity(entX, entY, entZ, entW, entH, entD, cfg.Mass, cfg.Restitution, 0.2),
		isActive:     true,
		identifier:   -1,
		wall:         NewThingWall(volumes, 0, 0),
		volume:       NewVolume3d(0, cfg.Id, "thing"),
	}
	t.entity.SetOnGround(false)

	height := 0.0
	halfW := 0.0
	if t.animation != nil {
		tex := t.animation.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			scaleW, scaleH := t.animation.ScaleFactor()
			width := float64(texW) * scaleW
			height = float64(texH) * scaleH
			halfW = width / 2.0
		}
	}

	t1 := [3]geometry.XYZ{{X: -halfW, Y: height, Z: 0.0}, {X: -halfW, Y: 0.0, Z: 0.0}, {X: halfW, Y: 0.0, Z: 0.0}}
	face0 := NewFace(nil, t1, "", t.animation)
	face0.SetUV(0.0, 0.0, 0.0, 1.0, 1.0, 1.0)
	face0.LockUV(true)
	t.volume.AddFace(face0)

	t2 := [3]geometry.XYZ{{X: -halfW, Y: height, Z: 0.0}, {X: halfW, Y: 0.0, Z: 0.0}, {X: halfW, Y: height, Z: 0.0}}
	face1 := NewFace(nil, t2, "", t.animation)
	face1.SetUV(0.0, 0.0, 1.0, 1.0, 1.0, 0.0)
	face1.LockUV(true)
	t.volume.AddFace(face1)

	t.volume.SetBillboard(1.0)

	t.volume.Rebuild()

	return t
}

// GetVertices retrieves the vertices of the ThingBase's associated triangular entity after updating their origin positions.
func (t *ThingBase) GetVertices() *Volume {
	return t.volume
}

// GetId returns the identifier string of the ThingBase instance.
func (t *ThingBase) GetId() string {
	return t.id
}

// GetKind returns the type of the ThingBase as a value of the ThingType enumeration.
func (t *ThingBase) GetKind() config.ThingType {
	return t.kind
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the associated physics entity.
func (t *ThingBase) GetAABB() *physics.AABB {
	return t.entity.GetAABB()
}

// GetEntity returns the physics.Entity associated with the current ThingBase instance.
func (t *ThingBase) GetEntity() *physics.Entity {
	return t.entity
}

// GetAnimation returns the animation associated with the ThingBase instance.
func (t *ThingBase) GetAnimation() *textures.Animation {
	return t.animation
}

// GetLocation retrieves the current location associated with the ThingBase instance.
func (t *ThingBase) GetLocation() *Volume {
	return t.location
}

// GetRadius retrieves the radius of the ThingBase instance as a float64 value.
func (t *ThingBase) GetRadius() float64 {
	return t.radius
}

// GetPosition returns the X, Y, and Z coordinates of the ThingBase instance as a tuple of three float64 values.
func (t *ThingBase) GetPosition() (float64, float64, float64) {
	return t.pos.X, t.pos.Y, t.pos.Z
}

// GetLight retrieves the Light object associated with the ThingBase's current sector.
func (t *ThingBase) GetLight() *Light {
	return t.location.Light
}

// GetMinZ retrieves the minimum Z-coordinate (floor height) of the location associated with the ThingBase instance.
func (t *ThingBase) GetMinZ() float64 {
	return t.location.GetMinZ()
}

// GetMaxZ retrieves the maximum Z-coordinate (height) of the location associated with the ThingBase instance.
func (t *ThingBase) GetMaxZ() float64 {
	return t.location.GetMaxZ()
}

// Compute performs computations or updates related to the ThingBase object based on the player's coordinates.
func (t *ThingBase) Compute(playerX float64, playerY float64, playerZ float64) {
	//nothing to do
}

// SetIdentifier sets the unique identifier for the ThingBase instance.
func (t *ThingBase) SetIdentifier(identifier int) {
	t.identifier = identifier
}

// GetIdentifier returns the unique identifier of the ThingBase instance.
func (t *ThingBase) GetIdentifier() int {
	return t.identifier
}

// OnCollide handles interactions when the current object collides with another object implementing the IThing interface.
func (t *ThingBase) OnCollide(other IThing) {
	//fmt.Println("COLLISION -> ", other.GetId())
}

// IsActive checks if the ThingBase instance is currently active.
func (t *ThingBase) IsActive() bool {
	return t.isActive
}

// SetActive updates the activation state of the ThingBase instance and returns the updated state as a boolean.
func (t *ThingBase) SetActive(active bool) {
	t.isActive = active
}

// PhysicsApply applies physics calculations to the ThingBase instance using its height attribute.
func (t *ThingBase) PhysicsApply() {
	t.doPhysics(t.height)
}

// doPhysics handles the physics computations for movement, collision detection, and location transitions for the entity.
func (t *ThingBase) doPhysics(tHeight float64) {
	// extract displacement delta
	dx, dy, dz := t.entity.GetDisplacement()
	if dx == 0.0 && dy == 0.0 && dz == 0.0 {
		return
	}
	isGrounded := false
	pX, pY, pZ := t.pos.X+dx, t.pos.Y+dy, t.pos.Z+dz
	hPos := t.pos.Z + tHeight
	elevBaseZ := t.pos.Z + t.maxStep
	// continuous collision detection (ccd) & sliding
	face, nX, nY, nZ := t.wall.ClosestFace(t.pos.X, t.pos.Y, t.pos.Z, pX, pY, pZ, dx, dy, dz, hPos, elevBaseZ, t.radius)
	if face != nil {
		// apply physical response to the entity
		t.entity.ResolveImpact(t.wall.GetEntity(), nX, nY, nZ)
		vx, vy, vz := t.entity.GetVelocity()
		// handle landing on walkable planes (slope)
		if nZ >= 0.7 && vz < 0 {
			vz = 0
			isGrounded = true
		}
		// project residual velocity onto tangent plane (sliding kcc)
		newVx, newVy, newVz := t.entity.ClipVelocity(vx, vy, vz, nX, nY, nZ)
		t.entity.SetVx(newVx)
		t.entity.SetVy(newVy)
		t.entity.SetVz(newVz)
		// recalculate effective displacement for current frame
		dx, dy, dz = t.entity.GetDisplacement()
		pX, pY, pZ = t.pos.X+dx, t.pos.Y+dy, t.pos.Z+dz
	}
	// location transition (3d portals)
	topZ := pZ + tHeight
	newVolume := t.location.Neighbor(pX, pY, pZ)
	if newVolume == nil {
		newVolume = t.world.QueryPoint3d(pX, pY, pZ)
	}
	newVolume = isValidZ(newVolume, pZ, topZ, t.maxStep)
	if newVolume != nil && newVolume != t.location {
		if t.entity.GetVz() <= 0 {
			actualStep := newVolume.GetMinZ() - t.location.GetMinZ()
			// automatic handling of height difference (step-up for stairs)
			if actualStep > 0 || (actualStep < 0 && math.Abs(actualStep) < t.maxStep) {
				pZ = newVolume.GetMinZ()
				t.entity.SetVz(0.0)
				isGrounded = true
			}
		}
		t.location = newVolume
	}
	// vertical topological limits
	minZ, maxZ := t.location.GetMinZ(), t.location.GetMaxZ()
	if pZ <= minZ {
		pZ = minZ
		isGrounded = true
		t.entity.SetVz(0.0)
	} else if (pZ + tHeight) > maxZ {
		t.entity.ResolveImpact(t.wall.GetEntity(), 0, 0, -1)
		pZ = maxZ - tHeight
	}
	// physical state synchronization
	t.entity.SetOnGround(isGrounded)
	// final application
	t.pos.X, t.pos.Y, t.pos.Z = pX, pY, pZ
	t.things.UpdateThing(t, t.pos.X, t.pos.Y, t.pos.Z)
}

// MoveTowards adjusts the entity's velocity towards a target speed in a specified direction using acceleration forces.
func (t *ThingBase) MoveTowards(dirX, dirY, targetSpeed, accelForce float64) {
	vx, vy, _ := t.entity.GetVelocity()
	desiredVx := dirX * targetSpeed
	desiredVy := dirY * targetSpeed
	deltaVx := desiredVx - vx
	deltaVy := desiredVy - vy
	t.entity.AddForce(deltaVx*accelForce, deltaVy*accelForce, 0.0)
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingBase) LaunchObject(pos geometry.XYZ, angle, pitch float64) {
	t.things.CreateBullet(t.location, pos, angle, pitch, 1.0, 1.0, 10)
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingBase) FireHitscan(pos geometry.XYZ, dirX, dirY, dirZ float64) {
	const maxDistance = 4096.0
	var closestDist = maxDistance
	var closestObj physics.IAABB

	// Usiamo l'origine (pos) e il vettore direzione (dir) calcolato esternamente.
	// QueryRay richiede invDir (1.0/dir) che viene calcolato internamente.
	t.things.QueryRay(pos.X, pos.Y, pos.Z, dirX, dirY, dirZ, maxDistance, func(object physics.IAABB, distance float64) (float64, bool) {
		// Self-hit culling: l'entità che spara non deve colpire se stessa
		if t.GetAABB() == object.GetAABB() {
			return maxDistance, false
		}
		//if object == sender {
		//	return maxDistance, false
		//}
		closestObj = object
		closestDist = distance
		// Ray Shrinking: restringiamo il raggio d'azione dell'albero alla distanza dell'impatto trovato.
		return distance, true
	})

	if closestObj != nil {
		// 2. Calcolo del punto d'impatto reale (Origine + Direzione * Distanza)
		impactX := pos.X + (dirX * closestDist)
		impactY := pos.Y + (dirY * closestDist)
		impactZ := pos.Z + (dirZ * closestDist)
		// 3. Risoluzione dell'impatto

		const force = 5000.0
		if enemy, ok := closestObj.(*ThingEnemy); ok {
			enemy.entity.AddForce(dirX*force, dirY*force, dirZ*force)
			// t.spawnBloodEffect(impactX, impactY, impactZ)
			return
		}
		if thing, ok := closestObj.(*ThingItem); ok {
			thing.entity.AddForce(dirX*force, dirY*force, dirZ*force)
			return
		}
		if thing, ok := closestObj.(*ThingBullet); ok {
			thing.entity.AddForce(dirX*force, dirY*force, dirZ*force)
			return
		}
		t.spawnBulletHole(impactX, impactY, impactZ, closestObj)
	}
}

// IsValidZ checks if the entity's base and top Z positions are within valid bounds of the location, considering maxStep.
func isValidZ(volume *Volume, baseZ, topZ, maxStep float64) *Volume {
	if volume == nil {
		return nil
	}
	minZ := volume.GetMinZ()
	maxZ := volume.GetMaxZ()
	// 1. Gestione soffitti a cielo aperto
	if maxZ <= minZ {
		maxZ = math.MaxFloat64
	}
	// 2. Controllo Pavimento (L'entità può scavalcare questo dislivello?)
	// Se baseZ è maggiore di floor (es. stiamo cadendo o saltando), la condizione è ampiamente soddisfatta.
	if baseZ+maxStep < minZ {
		return nil
	}
	// 3. Controllo Soffitto (C'è spazio sufficiente per l'altezza totale?)
	// Calcoliamo la quota base attesa (il massimo tra la nostra Z e il pavimento del nuovo settore)
	expectedBase := math.Max(baseZ, minZ)
	entityHeight := topZ - baseZ
	if expectedBase+entityHeight > maxZ {
		return nil
	}
	return volume
}

// spawnBulletHole creates a temporary visual entity at the specified coordinates to simulate a bullet hole effect.
// It offsets slightly from the surface to avoid Z-fighting and applies a visual decal for a limited duration.
func (t *ThingBase) spawnBulletHole(x, y, z float64, target physics.IAABB) {
	// Creiamo un'entità visiva temporanea tramite il gestore Things
	// Deve essere posizionata leggermente "staccata" dalla superficie (offset 0.1)
	// per evitare lo Z-fighting durante il rendering.

	// Se il target è un muro, possiamo estrarre la normale per ruotare la decalcomania
	// ma per ora posizioniamola semplicemente nel punto XYZ.
	//p.things.CreateDecal("BULLET_HOLE", x, y, z, 5.0) // 5.0 secondi di durata
}
