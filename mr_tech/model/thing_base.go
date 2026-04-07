package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	id         string
	position   geometry.XYZ
	mass       float64
	radius     float64
	height     float64
	angle      float64
	maxStep    float64
	kind       config.ThingType
	speed      float64
	volume     *Volume
	animation  *textures.Animation
	volumes    *Volumes
	entities   *Entities
	entity     *physics.Entity
	isActive   bool
	identifier int
	lastTx     float64
	lastTy     float64
	slider     *Slider
}

// NewThingBase creates a new ThingBase instance with specified configuration, animation, sector, volumes, and entities.
func NewThingBase(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, volumes *Volumes, entities *Entities) *ThingBase {
	diameter := cfg.Radius * 2
	w := diameter
	h := diameter
	d := cfg.Height // In 3D, la profondità è l'altezza reale dell'entità
	maxStep := cfg.Height * 0.5

	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius
	z := volume.GetMinZ() // Spawn esatto sul pavimento del settore

	position := cfg.Position
	position.Z = z // Forza la sincronizzazione della Z iniziale

	thing := &ThingBase{
		id:         cfg.Id,
		position:   position,
		angle:      cfg.Angle,
		kind:       cfg.Kind,
		mass:       cfg.Mass,
		radius:     cfg.Radius,
		height:     cfg.Height,
		speed:      cfg.Speed,
		volume:     volume,
		animation:  anim,
		volumes:    volumes,
		entities:   entities,
		maxStep:    maxStep,
		entity:     physics.NewEntity(x, y, z, w, h, d, cfg.Mass),
		isActive:   true,
		identifier: -1,
		slider:     NewSlider(volumes),
	}
	return thing
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

// GetVolume retrieves the current volume associated with the ThingBase instance.
func (t *ThingBase) GetVolume() *Volume {
	return t.volume
}

// GetPosition returns the X, Y, and Z coordinates of the ThingBase instance as a tuple of three float64 values.
func (t *ThingBase) GetPosition() (float64, float64, float64) {
	return t.position.X, t.position.Y, t.position.Z
}

// GetLight retrieves the Light object associated with the ThingBase's current sector.
func (t *ThingBase) GetLight() *Light {
	return t.volume.Light
}

// GetMinZ retrieves the minimum Z-coordinate (floor height) of the volume associated with the ThingBase instance.
func (t *ThingBase) GetMinZ() float64 {
	return t.volume.GetMinZ()
}

// GetMaxZ retrieves the maximum Z-coordinate (height) of the volume associated with the ThingBase instance.
func (t *ThingBase) GetMaxZ() float64 {
	return t.volume.GetMaxZ()
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

// PhysicsApply updates the position of the object based on passive and active physics-driven deltas.
func (t *ThingBase) PhysicsApply() {
	// 1. Recupero dati dal motore impulsivo
	eX, eY, eZ := t.entity.GetCenter()
	currentBaseZ := eZ - (t.entity.GetDepth() / 2.0)
	// 2. Calcolo dei delta
	velX := (eX - t.position.X) + t.entity.GetVx()
	velY := (eY - t.position.Y) + t.entity.GetVy()
	velZ := (currentBaseZ - t.position.Z) + t.entity.GetVz()
	if velX == 0 && velY == 0 && velZ == 0 {
		return
	}
	viewX, viewY, viewZ := t.position.X, t.position.Y, t.position.Z
	zBottom := viewZ
	zTop := viewZ + t.height
	zMinLimit := t.volume.GetMinZ()            // + t.getEyeHeight()
	zMaxLimit := t.volume.GetMaxZ() - t.height //.headMargin
	vx, vy, vz, _ := t.slider.AdjustPassage(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, t.radius)
	//vx, vy, vz := t.adjustPassage(viewX, viewY, viewZ, tx, ty, tz, zTop, zBottom, t.maxStep)
	// 4. Applichiamo il movimento se significativo
	if math.Abs(vx) > minMovement || math.Abs(vy) > minMovement || math.Abs(vz) > minMovement {
		t.position.X += vx
		t.position.Y += vy
		t.position.Z += vz
		baseZ := t.position.Z
		topZ := t.position.Z + t.height
		if newVolume := t.volumes.SearchVolume3d(t.volume, t.position.X, t.position.Y, baseZ, topZ, t.maxStep); newVolume != nil && newVolume != t.volume {
			t.volume = newVolume
		}
		// Sincronizzazione AABB Tree
		t.entities.UpdateThing(t, t.position.X, t.position.Y, t.position.Z)
	}
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

/*
func (t *ThingBase) adjustPassage(viewX, viewY, viewZ, velX, velY, velZ, top, bottom, maxStep float64) (float64, float64, float64) {
	// 1. Parametri fisici correnti
	// Correzione: Il bottom deve essere la quota piedi reale per il wall-sliding.
	// Usiamo il maxStep solo per "filtrare" cosa ignorare durante lo scivolamento orizzontale.
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ

	// 2. Wall Sliding (Collisione orizzontale)
	// Se velX/velY portano contro uno scalino < maxStep, il sistema di sliding
	// deve permettere l'avanzamento invece di azzerare il vettore.
	velX, velY, velZ = t.slider.WallSlidingEffect(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom+maxStep, t.radius)

	// 3. Pre-fetch del settore Target (Sonda avanzata)
	targetVol := t.volume
	// Creiamo una sonda che guardi leggermente avanti rispetto alla posizione attuale
	// per intercettare il settore dello scalino prima di sbatterci.
	probeX := viewX + velX
	probeY := viewY + velY
	if math.Abs(velX) < 0.05 && math.Abs(velY) < 0.05 {
		// Se quasi fermo, proietta una sonda minima nella direzione di sguardo
		probeX += math.Cos(t.angle) * 0.1
		probeY += math.Sin(t.angle) * 0.1
	}
	// Cerchiamo se la sonda finisce in un nuovo volume compatibile con il nostro maxStep
	if nv := t.volumes.SearchVolume3d(t.volume, probeX, probeY, viewZ, viewZ+t.height, maxStep); nv != nil {
		targetVol = nv
	}
	minZ, maxZ := targetVol.GetMinZ(), targetVol.GetMaxZ()
	if maxZ <= minZ {
		maxZ = math.MaxFloat64 // Cielo aperto
	}
	nextZ := viewZ + velZ
	// 4. Risoluzione Vincoli Verticali (Salita, Discesa e Gravità)
	if nextZ < minZ {
		// STEP UP: Il nuovo pavimento è più alto (scalino in salita).
		// Ci "tiriamo su" istantaneamente sulla quota del nuovo settore.
		velZ = minZ - viewZ
		t.entity.SetVz(0)
	} else if nextZ+t.height > maxZ {
		// COLLISIONE SOFFITTO: Abbassiamo la testa sotto il soffitto.
		velZ = (maxZ - t.height) - viewZ
		t.entity.SetVz(0)
	} else if nextZ > minZ {
		// STEP DOWN / GRAVITÀ: Se siamo sopra il suolo, applichiamo forza di caduta.
		// Se l'entità non ha una velocità di salto/caduta attiva, iniettiamo la gravità passiva.
		if math.Abs(velZ) < 0.01 {
			velZ = -0.15 // Valore leggermente superiore a minMovement per garantire il movimento
		}
	}
	return velX, velY, velZ
}
*/
