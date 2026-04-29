package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IVertices represents an interface for managing and retrieving geometric vertices and their associated volume data.
type IVertices interface {
	GetVertices(uint64) ([]*Face, []*Face, float64)

	GetVolume() *Volume
}

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	id            string
	pos           geometry.XYZ
	kind          config.ThingType
	angle         float64
	maxStep       float64
	speed         float64
	acceleration  float64
	jumpForce     float64
	location      *Volume
	world         *Volumes
	material      *textures.Material
	things        *Things
	isActive      bool
	identifier    int
	cage          *CollisionCage
	volume        *Volume
	entity        *physics.Entity
	vertices      IVertices
	collisions    []IThing
	collisionsIdx int
	inbox         chan *ThingEvent
	full3d        bool
	done          chan struct{}
}

// NewThingBase creates a new ThingBase instance with specified configuration, material, sector, world, and things.
func NewThingBase(things *Things, cfg *config.Thing, pos geometry.XYZ, material *textures.Material, location *Volume) *ThingBase {
	volumes := things.GetVolumes()
	radAngle := cfg.Angle // * (math.Pi / 180.0)
	entX := pos.X - cfg.Radius
	entY := pos.Y - cfg.Radius
	entZ := pos.Z
	entW := cfg.Radius * 2
	entH := cfg.Radius * 2
	entD := cfg.Height

	var vertices IVertices
	if cfg.Md2 != nil {
		vertices = NewVertexMD2(cfg.Md2, material, entX, entY, entZ, entW, entH, entD, cfg.Mass, cfg.Restitution, cfg.Friction)
	} else {
		vertices = NewVertexSprite(material, entX, entY, entZ, entW, entH, entD, cfg.Mass, cfg.Restitution, cfg.Friction)
	}
	const cageMargin = 0.001
	volume := vertices.GetVolume()
	t := &ThingBase{
		vertices:      vertices,
		volume:        volume,
		entity:        volume.GetEntity(),
		id:            cfg.Id,
		angle:         radAngle,
		kind:          cfg.Kind,
		speed:         cfg.Speed,
		acceleration:  cfg.Acceleration,
		jumpForce:     cfg.JumpForce,
		pos:           pos,
		location:      location,
		material:      material,
		world:         volumes,
		things:        things,
		maxStep:       cfg.Height * 0.5,
		isActive:      true,
		identifier:    -1,
		cage:          NewCollisionCage(cfg.Id, volume, cageMargin, 0, 0),
		inbox:         make(chan *ThingEvent, 16),
		done:          make(chan struct{}),
		collisions:    make([]IThing, 128),
		collisionsIdx: 0,
		full3d:        things.full3d,
	}
	t.entity.SetOnGround(false)

	return t
}

// GetVertices retrieves the vertices of the ThingBase's associated triangular entity after updating their origin positions.
func (t *ThingBase) GetVertices() ([]*Face, []*Face, float64, float64) {
	vCurr, vNext, lerp := t.vertices.GetVertices(textures.CurrentTick())
	return vCurr, vNext, lerp, t.volume.GetBillboard()
}

// GetAngle returns the current rotation angle of the ThingBase instance as a float64 value.
func (t *ThingBase) GetAngle() float64 {
	return t.angle
}

// SetAngle updates the rotation angle of the ThingBase instance to the specified float64 value.
func (t *ThingBase) SetAngle(angle float64) {
	t.angle = angle
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

func (t *ThingBase) GetCage() *CollisionCage {
	return t.cage
}

// GetEntity returns the physics.Entity associated with the current ThingBase instance.
func (t *ThingBase) GetEntity() *physics.Entity {
	return t.entity
}

// GetMaterial returns the material associated with the ThingBase instance.
func (t *ThingBase) GetMaterial() *textures.Material {
	return t.material
}

// GetLocation retrieves the current location associated with the ThingBase instance.
func (t *ThingBase) GetLocation() *Volume {
	return t.location
}

// GetRadius retrieves the radius of the ThingBase instance as a float64 value.
func (t *ThingBase) GetRadius() float64 {
	return t.entity.GetWidth() / 2.0
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

// GetVolume retrieves the volume associated with the ThingBase instance.
func (t *ThingBase) GetVolume() *Volume {
	return t.volume
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
	if t.collisionsIdx >= len(t.collisions) {
		return
	}
	t.collisions[t.collisionsIdx] = other
	t.collisionsIdx++
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

// StageCompute calculates the displacement and updates the entity's collision cage for collision detection and resolution.
func (t *ThingBase) StageCompute() {
	dx, dy, dz := t.entity.GetDisplacement()
	// 1. DEADZONE
	//const sleepEpsilon = 0.005
	//if math.Abs(dx) < sleepEpsilon && math.Abs(dy) < sleepEpsilon && math.Abs(dz) < sleepEpsilon {
	//	return
	//}
	pX, pY, pZ := t.pos.X, t.pos.Y, t.pos.Z
	eRadX := t.entity.GetWidth() * 0.5
	eRadY := t.entity.GetHeight() * 0.5
	eRadZ := t.entity.GetDepth() * 0.5
	cX, cY, cZ := pX, pY, pZ+eRadZ
	//tX, tY, tZ := cX+dx, cY+dy, cZ+dz
	t.cage.Rebuild(cX, cY, cZ, dx, dy, dz, eRadX, eRadY, eRadZ)
	t.world.QueryCollisionCage(t.cage, t.maxStep)
	t.things.QueryCollisionCage(t.cage, t.maxStep)
}

// StageResolve processes interactions between the current object and others in proximity to resolve collisions or overlaps.
// solverJitter adds a small adjustment to penetration calculations to account for numerical instability.
func (t *ThingBase) StageResolve(solverJitter float64) {
	tX, tY, tZ := t.cage.GetT()
	for bucket := BucketType(0); bucket < BucketSize; bucket++ {
		count := t.cage.counts[bucket]
		if count == 0 {
			continue
		}
		for j := 0; j < count; j++ {
			entry := t.cage.faces[bucket][j]
			otherFace := entry.GetFace()
			nX, nY, nZ := entry.GetNormal()
			rEff := entry.GetREff()

			// Lettura delle coordinate in WORLD SPACE tradotte dalla Cage
			p0X := entry.p0X
			p0Y := entry.p0Y
			p0Z := entry.p0Z

			distTarget := (tX-p0X)*nX + (tY-p0Y)*nY + (tZ-p0Z)*nZ
			if distTarget >= rEff {
				continue
			}

			penetration := (rEff - distTarget) + solverJitter
			otherParentEnt := otherFace.GetParent().GetEntity()

			// Delega totale e assoluta al solutore interno di physics
			t.GetEntity().ResolveImpact(otherParentEnt, nX, nY, nZ, penetration)
		}
	}
}

// StageApply updates the entity's state by processing movement, grounding, and positional integration based on displacement.
func (t *ThingBase) StageApply() {
	dx, dy, dz := t.entity.GetDisplacement()

	// DEADZONE
	const sleepEpsilon = 0.005
	if math.Abs(dx) < sleepEpsilon && math.Abs(dy) < sleepEpsilon && math.Abs(dz) < sleepEpsilon {
		t.entity.SetVx(0.0)
		t.entity.SetVy(0.0)
		if t.entity.IsOnGround() {
			t.entity.SetVz(0.0)
		}
		return
	}

	// TRACKING PAVIMENTO (Grounding & Location)
	isGrounded := false
	if count := t.cage.counts[BucketFloor]; count > 0 {
		isGrounded = true
		if parent := t.cage.faces[BucketFloor][0].GetFace().GetParent(); parent != nil {
			t.location = parent
		}
	}

	// APPLICAZIONE STATO
	t.entity.SetOnGround(isGrounded)

	// 4. INTEGRAZIONE POSIZIONALE PURA
	// La posizione è il risultato diretto del displacement fisico.
	t.pos.X += dx
	t.pos.Y += dy
	t.pos.Z += dz
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
	t.things.CreateThrowable(t.location, pos, angle, pitch, 1.0, 1.0, 10)
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
		if thing, ok := closestObj.(*ThingThrowable); ok {
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
