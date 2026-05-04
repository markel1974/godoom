package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

/*
	// 1. Calcolo dimensioni e offset specifici del frame
	texW := float64(waxFrame.SizeX) // Dimensioni native del frame WAX
	texH := float64(waxFrame.SizeY)

	// Applicazione degli scale factor del motore
	scaleW, scaleH := 1.0, 1.0 // Sostituisci con i fattori di scala globali se necessari
	width := texW * scaleW
	height := texH * scaleH
	halfW := width / 2.0
	halfH := height / 2.0

	// L'insertY in Dark Forces trasla il centro del rendering
	offY := float64(waxFrame.InsertY) * scaleH
	offX := float64(waxFrame.InsertX) * scaleW

	// 2. Costruzione dei vertici del Quad centrato (con offset)
	// Ricorda: asse Z locale va da +halfH a -halfH
	topZ := halfH - offY
	botZ := -halfH - offY
	leftX := -halfW + offX
	rightX := halfW + offX

	t0 := [3]geometry.XYZ{
		{X: leftX, Y: 0.0, Z: topZ},  // TL
		{X: leftX, Y: 0.0, Z: botZ},  // BL
		{X: rightX, Y: 0.0, Z: botZ}, // BR
	}
	t1 := [3]geometry.XYZ{
		{X: leftX, Y: 0.0, Z: topZ},  // TL
		{X: rightX, Y: 0.0, Z: botZ}, // BR
		{X: rightX, Y: 0.0, Z: topZ}, // TR
	}

	// 3. Estrazione della texture specifica per questo frame
	// Usa il MaterialManager o un lookup per ID (es. texName generato nel Builder)
	tag := fmt.Sprintf("%s_A%d_V%d_F%d", cfg.Id, actIdx, viewIdx, frameIdx)

	// Creiamo il materiale per il singolo frame
	// (Se hai già registrato le texture come singole nel manager)
	//frameAnim := textures.NewMaterial([]string{waxFrame.TextureID}, textures.MaterialKindStatic, 1, 1, 0, 0)
	//TODO
	frameAnim := anim
	f0 := NewFace(nil, t0, tag, frameAnim)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	if waxFrame.Flip {
		// Ribaltamento orizzontale nativo per specchiamento
		f0.SetUV(1.0, 0.0, 1.0, -1.0, 0.0, -1.0)
	}
	f0.LockUV(true)

	f1 := NewFace(nil, t1, tag, frameAnim)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	if waxFrame.Flip {
		f1.SetUV(1.0, 0.0, 0.0, -1.0, 0.0, 0.0)
	}
	f1.LockUV(true)

	// Uniamo i due triangoli per formare il quad finale (opzionale se il tuo renderer gestisce slice di Facce)
	// Nel DOM attuale di GetVertices ti aspetterai 2 face (un quad) per frame, quindi l'array finale
	// potrebbe dover contenere i due triangoli concatenati.
	// Per semplicità qui assumo che Face supporti i quad (o che tu concateni le slice nel GetVertices).

*/

// WAXFaces represents a pair of connected Faces within a 3D volume, providing bidirectional linking between structures.
type WAXFaces struct {
	face0 *Face
	face1 *Face
}

// VerticesWAX represents a 3D entity composed of WAXFaces and organized within a Volume.
type VerticesWAX struct {
	volume        *Volume
	baseTexName   string
	currentAction int
	currentAngle  int
	faces         []*WAXFaces
	viewFaces     *WAXFaces
}

// NewVerticesWAX creates a new VerticesWAX instance with geometry, physics, and animation information, based on input config.
func NewVerticesWAX(cfg *config.Thing, pos geometry.XYZ, anim *textures.Material) *VerticesWAX {
	x := pos.X - cfg.Radius
	y := pos.Y - cfg.Radius
	z := pos.Z
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	d := cfg.Height

	volume := NewVolumeDetails3d(0, "wax", "thing", x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesWAX{
		volume: volume,
	}
	f.faces = make([]*WAXFaces, len(cfg.WAX.Views))
	for viewIdx, view := range cfg.WAX.Views {
		if view == nil || len(view.Frames) == 0 {
			continue
		}
		//var animation []string
		//for frameIdx, waxFrame := range view.Frames {
		//	animation = append(animation, waxFrame.TextureID)
		//}
		f0, f1 := f.createFaces(w, h, anim)
		f.faces[viewIdx] = &WAXFaces{face0: f0, face1: f1}
	}

	f.compute()
	return f
}

// GetVolume returns the pointer to the Volume instance associated with the VerticesWAX object.
func (v *VerticesWAX) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves the faces and associated data for the current frame and returns them with a default displacement value.
func (v *VerticesWAX) GetVertices(tick uint64) ([]*Face, []*Face, float64) {
	f := v.volume.GetFaces()
	return f, f, 0.0
}

// SetViewAngle calculates the relative angle between the camera and the entity and updates the current view angle index.
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
	v.compute()
}

// SetAction updates the current action index if the provided index is within bounds and triggers a recomputation of vertices.
func (v *VerticesWAX) SetAction(idx int) {
	if idx < 0 || idx >= len(v.faces) {
		return
	}
	v.currentAction = idx
	v.compute()
}

// GetDisplacement retrieves the displacement coordinates (X, Y, Z) of the volume's bottom-left position.
func (v *VerticesWAX) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetBottomLeft()
}

// GetBillboard returns the billboard orientation value for the VerticesWAX instance.
func (v *VerticesWAX) GetBillboard() float64 {
	return 1.0
}

// SetThing assigns an IThing instance to the internal volume of the VerticesWAX object.
func (v *VerticesWAX) SetThing(t IThing) {
	v.volume.SetThing(t)
}

// compute updates the current view faces and rebuilds the volume geometry based on the active view angle.
func (v *VerticesWAX) compute() {
	viewFaces := v.faces[v.currentAngle]
	if viewFaces == v.viewFaces {
		return
	}
	if v.viewFaces = viewFaces; v.viewFaces == nil {
		if v.viewFaces = v.faces[0]; v.viewFaces == nil {
			return
		}
	}
	v.volume.ClearFace()
	v.volume.AddFace(v.viewFaces.face0)
	v.volume.AddFace(v.viewFaces.face1)
	v.volume.Rebuild()
}

// createFaces generates two triangular faces based on the given width, height, and material animation.
func (v *VerticesWAX) createFaces(width float64, height float64, anim *textures.Material) (*Face, *Face) {
	// Triangolo 0: Top-Left, Bottom-Left, Bottom-Right
	halfW := width * 0.5
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(nil, t0, "", anim)
	// Passiamo V=0 per il top e V=-1 per il bottom (diventerà 1 nel renderer)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
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
	return f0, f1
}
