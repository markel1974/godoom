package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesWAX represents a collection of vertices organized by actions, angles, and associated faces for 3D operations.
type VerticesWAX struct {
	volume      *Volume
	baseTexName string

	currentAction int
	currentAngle  int // Da 0 a 31
	numAngles     int

	faces [][][]*Face
}

// NewVerticesWAX constructs a VerticesWAX instance using the provided configuration, position, and optional material.
// It initializes the geometry and UV mapping for rendering, and sets physical dimensions and properties.
func NewVerticesWAX(cfg *config.Thing, pos geometry.XYZ, anim *textures.Material) *VerticesWAX {
	x := pos.X - cfg.Radius
	y := pos.Y - cfg.Radius
	z := pos.Z
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	d := cfg.Height

	height := h
	width := w
	halfW := width / 2.0
	if anim != nil {
		tex := anim.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			scaleW, scaleH := tex.GetScaleFactor()
			width = float64(texW) * scaleW
			height = float64(texH) * scaleH
			halfW = width / 2.0
		}
	}

	volume := NewVolumeDetails3d(0, "material", "thing", x, y, z, width, height, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesWAX{volume: volume}
	//f.volume.SetBillboard(1.0)

	// Triangolo 0: Top-Left, Bottom-Left, Bottom-Right
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(nil, t0, "", anim)
	// Passiamo V=0 per il top e V=-1 per il bottom (diventerà 1 nel renderer)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	f.volume.AddFace(f0)

	// Triangolo 1: Top-Left, Bottom-Right, Top-Right
	t1 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
		{X: halfW, Y: 0.0, Z: height},  // TR
	}
	f1 := NewFace(nil, t1, "", anim)
	// TL: (0,0), BR: (1,-1), TR: (1,0)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	f.volume.AddFace(f1)

	f.volume.Rebuild()

	return f
}

// GetVolume returns the Volume instance associated with the VerticesWAX object.
func (v *VerticesWAX) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves front and back face data along with a dummy interpolation value based on the current tick value.
func (v *VerticesWAX) GetVertices(tick uint64) ([]*Face, []*Face, float64) {
	actionFaces := v.faces[v.currentAction]

	// Fallback se l'angolazione corrente è vuota (alcune WAX non hanno tutte le 32 view)
	viewFaces := actionFaces[v.currentAngle]
	if len(viewFaces) == 0 {
		viewFaces = actionFaces[0] // Fallback alla vista frontale
	}

	if len(viewFaces) == 0 {
		return nil, nil, 0.0
	}

	// Stessa logica di sequencer che hai usato in VertexMD2
	animLength := len(viewFaces)
	const groupSize = 6.0
	frameFloat := textures.TickGrouped(tick, int(groupSize))

	idxA := int(frameFloat) % animLength
	// Per gli sprite 2D solitamente non si fa lerp geometrico,
	// si restituisce semplicemente il frame corrente
	return []*Face{viewFaces[idxA]}, []*Face{viewFaces[idxA]}, 0.0
}

// SetViewAngle calculates and sets the current view angle index based on camera position, entity position, and entity yaw.
func (v *VerticesWAX) SetViewAngle(cameraPos, entityPos geometry.XYZ, entityYaw float64) {
	// 1. Calcolo del vettore direzione dalla telecamera all'entità
	dx := cameraPos.X - entityPos.X
	dy := cameraPos.Y - entityPos.Y // Considerando Y come profondità (Z nel tuo CreateCoords)

	// 2. Angolo assoluto tra telecamera ed entità
	angleToCam := math.Atan2(dy, dx)

	// 3. Delta angolare (Angolo Telecamera - Yaw Entità)
	// Normalizziamo l'angolo per assicurarci che sia tra 0 e 2*Pi
	relativeAngle := math.Mod(angleToCam-entityYaw+(math.Pi*2), math.Pi*2)

	// 4. Mappatura su 32 indici (0-31)
	// Aggiungiamo metà step (Pi / 32) per arrotondare correttamente al settore più vicino
	sectorSize := (math.Pi * 2) / 32.0
	index := int(math.Floor((relativeAngle + (sectorSize / 2.0)) / sectorSize))

	v.currentAngle = index % 32
}

// SetAction updates the current action index to the specified value if it is within the valid range of available actions.
func (v *VerticesWAX) SetAction(idx int) {
	if idx >= 0 && idx < len(v.faces) {
		v.currentAction = idx
	}
}

// GetDisplacement returns the displacement coordinates (x, y, z) of the entity's bottom-left corner from the volume.
func (v *VerticesWAX) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetBottomLeft()
}

// GetBillboard returns a constant float64 value representing the billboard configuration for the object.
func (v *VerticesWAX) GetBillboard() float64 {
	return 1.0
}

// SetThing assigns the provided IThing instance to the associated Volume within the VerticesWAX object.
func (v *VerticesWAX) SetThing(t IThing) {
	v.volume.SetThing(t)
}
