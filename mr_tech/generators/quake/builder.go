package quake

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

//MODEL IMPORTAL
//model is Z up
//model is CCW

// Builder manages the construction and handling of graphical assets, leveraging a Textures manager for texture operations.
type Builder struct {
	texManager *Textures
}

// NewBuilder initializes and returns a pointer to a new Builder instance with a default Textures manager.
func NewBuilder() *Builder {
	return &Builder{
		texManager: NewTextures(),
	}
}

// Setup initializes the game environment by loading and processing BSP data, textures, entities, and lights from a .pak file.
func (p *Builder) Setup(pakPath string, level int) (*config.Root, error) {
	bpsPath := "maps" + lumps.PakSeparator + "e1m" + strconv.Itoa(level) + ".bsp"
	palPath := "gfx" + lumps.PakSeparator + "palette.lmp"
	pk := lumps.NewPak()
	if err := pk.Setup(pakPath); err != nil {
		return nil, err
	}
	palReader, err := pk.Open(palPath)
	if err != nil {
		return nil, err
	}
	palette, err := lumps.NewPalette(palReader)
	if err != nil {
		return nil, err
	}
	rs, err := pk.Open(bpsPath)
	if err != nil {
		return nil, err
	}
	// Header
	infos, err := lumps.NewLumpInfos(rs)
	if err != nil {
		return nil, err
	}
	// Geometry
	vertexes, _ := lumps.NewVertexes(rs, infos[lumps.LumpVertexes])
	edges, _ := lumps.NewEdges(rs, infos[lumps.LumpEdges])
	surfEdges, _ := lumps.NewSurfEdges(rs, infos[lumps.LumpSurfEdges])
	faces, _ := lumps.NewFace(rs, infos[lumps.LumpFaces])
	texInfos, _ := lumps.NewTexInfos(rs, infos[lumps.LumpTexInfo])
	mipTextures, _ := lumps.NewMipTextures(rs, infos[lumps.LumpTextures])
	//leaves, _ := lumps.NewLeaves(rs, infos[lumps.LumpLeaves])
	entities, _ := lumps.NewEntities(rs, infos[lumps.LumpEntities])
	//marks, _ := lumps.NewMarks(rs, infos[lumps.LumpMarkSurfaces])
	bspModels, _ := lumps.NewModels(rs, infos[lumps.LumpModels])

	for _, mt := range mipTextures {
		if mt != nil && mt.Name != "" {
			if err = p.texManager.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], palette, false, 255, false); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}

	var playerAngle float64
	var playerPos geometry.XYZ
	cal := config.NewConfigCalibration(true, 0, 0, 0, 0, 0, 0, true)
	//cal.Auto = false
	//cal.OrthoSize = 32092
	//cal.LightCamY = 8000
	//cal.ZNearRoom = 0.1
	//cal.ZFarRoom = 16000

	cal.ScaleFactor = 1.0

	root := config.NewConfigRoot(cal, nil, nil, nil, 1.0, p.texManager)

	for _, ent := range entities {
		classname := ent.Properties["classname"]
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			pos = p.createXYZ(x, y, z)
		}
		modelProp := ent.Properties["model"]

		var angle float64
		if a, ok := ent.Properties["angle"]; ok {
			angle, _ = strconv.ParseFloat(a, 64)
		}
		mangleStr, _ := ent.Properties["mangle"]
		colorStr, _ := ent.Properties["_color"]

		externalBSPPath := GetExternalBModelFileName(classname)

		switch {
		case classname == "worldspawn":
			// Ignoriamo: è la mappa base, la geometria è già gestita da worldModel
			continue
		case classname == "info_player_start":
			playerPos, playerAngle, err = p.createPlayerProps(angle, pos)
			if err != nil {
				fmt.Printf("Warning: %s\n", err.Error())
				continue
			}
		case classname == "light":
			light := p.createLight(ent, angle, mangleStr, colorStr, pos, lightStyle0, false)
			if light == nil {
				continue
			}
			root.Lights = append(root.Lights, light)

		case strings.HasPrefix(classname, "info_"):
			// Marker invisibili: teletrasporti, spawn point deathmatch, nodi di pattuglia.
			// TODO: Salvarli in una lista di waypoint/spawnpoint gameplay.
			continue

		case strings.HasPrefix(classname, "path_"):
			// Marker invisibili: teletrasporti, spawn point deathmatch, nodi di pattuglia.
			// TODO: Salvarli in una lista di waypoint/spawnpoint gameplay.
			continue

		case strings.HasPrefix(classname, "light"):
			style := lightStyle0
			if sIndex, ok := ent.Properties["style"]; ok {
				if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(lightStyles) {
					style = lightStyles[index]
				}
			}
			// Gestisce light, light_fluoro, light_fluorospark
			light := p.createLight(ent, angle, mangleStr, colorStr, pos, style, true)
			if light == nil {
				continue
			}
			root.Lights = append(root.Lights, light)

		case strings.HasPrefix(classname, "ambient_"):
			// Suoni ambientali (es. ambient_drone). Nessuna mesh.
			continue

		case strings.HasPrefix(modelProp, "*"):
			continue

		case strings.HasPrefix(classname, "func_"):
			continue

		case strings.HasPrefix(classname, "trigger_"):
			continue

		case len(externalBSPPath) > 0:
			cThing, err := p.createExternalBModelThing(externalBSPPath, pos, classname, pk, palette)
			if err != nil {
				fmt.Printf("Warning External BModel: %s (Errore: %v)\n", classname, err)
				continue
			}
			root.Things = append(root.Things, cThing)

		default:
			cThing, err := p.createThing(pos, classname, pk, palette)
			if err != nil {
				fmt.Printf("Warning: %s\n", err.Error())
				continue
			}
			root.Things = append(root.Things, cThing)
		}
	}

	// 4. Conversione Geometria Statica: BSP Faces -> Volume
	// Creiamo un singolo volume globale, senza duplicazioni.
	volume := config.NewConfigVolume("quake_world", "quake_bsp")

	worldModel := bspModels[0]

	// Iteriamo ESCLUSIVAMENTE sulle facce che appartengono al mondo
	for i := int32(0); i < worldModel.NumFaces; i++ {
		faceIdx := worldModel.FirstFace + i
		bspFace := faces[faceIdx]
		texInfo := texInfos[bspFace.TexInfo]
		texName := "default"
		isSky := (texInfo.Flags & 4) != 0
		if texInfo.MipTex < uint32(len(mipTextures)) && mipTextures[texInfo.MipTex] != nil {
			texName = mipTextures[texInfo.MipTex].Name
		}
		var points []geometry.XYZ
		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := surfEdges[bspFace.FirstEdge+int32(j)]
			var v *lumps.Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].Vertex0]
			} else {
				v = vertexes[edges[-surfEdgeIdx].Vertex1]
			}
			pos := p.createXYZ(float64(v.X), float64(v.Y), float64(v.Z))
			points = append(points, pos)
		}
		animKind := config.AnimationKindLoop
		if isSky {
			animKind = config.AnimationKindSky
		}
		material := config.NewConfigAnimation([]string{texName}, animKind, 1.0, 1.0)
		triangles := p.triangulateConvex3d(points)
		for _, tri := range triangles {
			volume.Faces = append(volume.Faces, config.NewConfigFace(tri, material, texName))
		}
	}
	if len(volume.Faces) > 0 {
		root.Volumes = append(root.Volumes, volume)
	}
	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 40, 4, 80)
	root.Player.Speed = 1200
	root.Player.JumpForce = 1000

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Factor = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1
	root.Player.Bobbing.SwayScale = 2.0
	root.Player.Bobbing.SwayOffsetX = 50
	root.Player.Bobbing.SwayOffsetY = -0.9
	root.Player.Bobbing.MaxAmplitudeX = 5.0 // ESCURSIONE MASSIMA: 12 unità (circa il 20% dell'altezza player)
	root.Player.Bobbing.MaxAmplitudeY = 5.5
	root.Player.Bobbing.StrideLength = 0.0015 // FREQUENZA: 1000 * 0.0007 = 0.7 rad/frame.
	root.Player.Bobbing.IdleAmpX = 0.9        // Respiro
	root.Player.Bobbing.IdleAmpY = 0.9
	root.Player.Bobbing.IdleDrift = 0.01
	root.Player.Bobbing.SpeedLerp = 0.30 // Reattività istantanea alla velocità
	root.Player.Bobbing.AmpLerp = 0.20
	root.Player.Bobbing.ImpactMax = 1000.0
	root.Player.Bobbing.ImpactScale = 0.02   // ATTERRAGGIO: 1000 * 0.02 = 20 unità di scuotimento verticale
	root.Player.Bobbing.SpringTension = 0.20 // Molla più rigida (ritorno rapido)
	root.Player.Bobbing.SpringDamping = 0.80
	root.Player.Bobbing.TiltAmp = 0.05
	//fmt.Println("TODO REACTIVATE ROOT THINGS!")
	//root.Things = nil
	return root, nil
}

// createPlayerProps extracts player position and angle from an entity and computes the angle in radians.
func (p *Builder) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// createLight creates a new Light instance based on entity properties and position, returning an error if invalid or missing data.
func (p *Builder) createLight(entity *lumps.Entity, angle float64, mangleStr, colorStr string, pos geometry.XYZ, style []float64, isSpot bool) *config.Light {
	intensity := 0.0
	falloff := 0.0
	var kind config.LightKind

	// 1. INTENSITÀ DI BASE
	if l, ok := entity.Properties["light"]; ok {
		intensity, _ = strconv.ParseFloat(l, 64)
	} else {
		intensity = 300.0 // Default fallback tipico di Quake
	}

	// COLORE (Standard Quake 2 / Modern Quake 1)
	r, g, b := 1.0, 1.0, 1.0 // Default Bianco
	if len(colorStr) > 0 {
		if cr, cg, cb, valid := p.parseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r, g, b = cr/255.0, cg/255.0, cb/255.0
			} else {
				r, g, b = cr, cg, cb
			}
		}
	}

	// DIREZIONE DELLO SPOTLIGHT
	dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Default: guarda in basso
	if isSpot {
		kind = config.LightKindSpot
		intensity = intensity * 0.9
		falloff = intensity * 10
		if len(mangleStr) > 0 {
			if yaw, pitch, _, valid := p.parseVector(mangleStr); valid {
				dirX, dirY, dirZ = p.calcDirection(yaw, pitch)
			}
		} else {
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Guarda in alto
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Guarda in basso
			} else {
				dirX, dirY, dirZ = p.calcDirection(angle, 0)
			}
		}
	} else {
		kind = config.LightKindAmbient
		intensity = intensity * 0.05
		falloff = intensity
	}

	// 4. CREAZIONE CONFIGURAZIONE
	cl := config.NewConfigLight(pos, intensity, kind, falloff)
	cl.R = r
	cl.G = g
	cl.B = b

	// Applica il fix al sistema di coordinate se il tuo motore trasforma Z in Y
	// Se nel tuo builder.go trasformi le posizioni con XYZ{X: x, Y: -y, Z: 0},
	// assicurati che anche la direzione segua la stessa swizzle logic.
	cl.DirX = dirX
	cl.DirY = dirY
	cl.DirZ = dirZ
	cl.Style = style

	return cl
}

// createThing creates a new Thing object based on the specified position, classname, Pak file, and color palette.
func (p *Builder) createThing(pos geometry.XYZ, classname string, pk *lumps.Pak, palette []byte) (*config.Thing, error) {
	thingPath := GetModelFileName(classname)
	if len(thingPath) == 0 {
		return nil, fmt.Errorf("unknown thing %s", classname)
	}
	rsMdl, err := pk.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}
	mdl, err := lumps.NewMDLResource(rsMdl)
	if err != nil {
		return nil, fmt.Errorf("can't load MDL %s: %s\n", classname, err.Error())
	}
	var registeredTexNames []string
	for sIdx, skin := range mdl.Skins {
		texName := fmt.Sprintf("%s_skin_%d", classname, sIdx)
		w := int(mdl.Header.SkinWidth)
		h := int(mdl.Header.SkinHeight)
		if err = p.texManager.RegisterPixels(texName, w, h, skin.Data, palette, false, 255, false); err != nil {
			fmt.Printf("Warning: texture %s error: %s\n", texName, err.Error())
			continue
		}
		registeredTexNames = append(registeredTexNames, texName)
	}

	skinTargetIndex := 0
	kind := config.ThingEnemyDef

	if strings.HasPrefix(classname, "item_") {
		itemName := strings.TrimPrefix(classname, "item_")
		switch itemName {
		case "armor2":
			skinTargetIndex = 1
		case "armorInv":
			skinTargetIndex = 2
		}
		kind = config.ThingItemDef
	}

	var anim *config.Animation
	if len(registeredTexNames) > skinTargetIndex {
		targetSkin := []string{registeredTexNames[skinTargetIndex]}
		anim = config.NewConfigAnimation(targetSkin, config.AnimationKindLoop, 1.0, 1.0)
	}
	thingCfg := config.NewConfigThing(classname, pos, 0.0, kind, 16.0, 16.0, 56, 100.0, anim)

	cModel := &config.Model3d{Frames: make([]config.Frame3d, mdl.Header.NumFrames)}
	for idx, f := range mdl.Frames {
		cFrame := config.Frame3d{Triangles: make([][3]config.Vertex3d, mdl.Header.NumTris)}
		skinW := float32(mdl.Header.SkinWidth)
		skinH := float32(mdl.Header.SkinHeight)
		for tIdx, tri := range mdl.Triangles {
			for v := 0; v < 3; v++ {
				vx := tri.Vertices[v]
				tc := mdl.TexCoords[vx]
				s := float32(tc.S)
				t := float32(tc.T)
				if tri.FacesFront == 0 && tc.OnSeam != 0 {
					s += skinW / 2.0
				}
				nU := s / skinW
				nV := 1.0 - (t / skinH)
				cFrame.Triangles[tIdx][v] = config.Vertex3d{Pos: p.createXYZ(f[vx][0], f[vx][1], f[vx][2]), U: nU, V: nV}
			}
		}
		cModel.Frames[idx] = cFrame
		cModel.Frames[idx] = cFrame
	}

	thingCfg.WakeUpDistance = 400
	thingCfg.SetModel3d(cModel)
	return thingCfg, nil
}

func (p *Builder) createExternalBModelThing(bspPath string, pos geometry.XYZ, classname string, pk *lumps.Pak, palette []byte) (*config.Thing, error) {
	rs, err := pk.Open(bspPath)
	if err != nil {
		return nil, fmt.Errorf("impossibile aprire %s: %s", bspPath, err.Error())
	}
	infos, err := lumps.NewLumpInfos(rs)
	if err != nil {
		return nil, err
	}
	bspModels, _ := lumps.NewModels(rs, infos[lumps.LumpModels])
	if len(bspModels) == 0 {
		return nil, fmt.Errorf("nessun modello trovato in %s", bspPath)
	}
	vertexes, _ := lumps.NewVertexes(rs, infos[lumps.LumpVertexes])
	edges, _ := lumps.NewEdges(rs, infos[lumps.LumpEdges])
	surfEdges, _ := lumps.NewSurfEdges(rs, infos[lumps.LumpSurfEdges])
	faces, _ := lumps.NewFace(rs, infos[lumps.LumpFaces])
	//texInfos, _ := lumps.NewTexInfos(rs, infos[lumps.LumpTexInfo])
	mipTextures, _ := lumps.NewMipTextures(rs, infos[lumps.LumpTextures])

	for _, mt := range mipTextures {
		if mt != nil && mt.Name != "" {
			_ = p.texManager.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], palette, false, 255, false)
		}
	}
	// 3. Traduzione Geometria in Model3d Agnostico
	// Raccogliamo tutti i triangoli in questo singolo frame
	var allTriangles [][3]config.Vertex3d
	model := bspModels[0] // Il modello root dell'oggetto
	for i := int32(0); i < model.NumFaces; i++ {
		faceIdx := model.FirstFace + i
		bspFace := faces[faceIdx]
		//texInfo := texInfos[bspFace.TexInfo]
		var points []geometry.XYZ
		// (Per un rendering perfetto delle texture sui BSP servirebbe il calcolo vettoriale
		// di S e T usando i vettori in TexInfo, ma per la geometria bruta ci basta la posizione)
		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := surfEdges[bspFace.FirstEdge+int32(j)]
			var v *lumps.Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].Vertex0]
			} else {
				v = vertexes[edges[-surfEdgeIdx].Vertex1]
			}
			xyz := p.createXYZ(float64(v.X), float64(v.Y), float64(v.Z))
			points = append(points, xyz)
		}
		// Triangolazione del poligono della faccia
		rawTriangles := p.triangulateConvex3d(points)
		for _, rawTri := range rawTriangles {
			tri := [3]config.Vertex3d{
				{Pos: rawTri[0], U: 0.0, V: 0.0}, // Placeholder UV, andrebbe calcolato
				{Pos: rawTri[1], U: 1.0, V: 0.0},
				{Pos: rawTri[2], U: 0.0, V: 1.0},
			}
			allTriangles = append(allTriangles, tri)
		}
	}
	// I BSP non hanno animazioni vertex-morphing, 1 solo frame
	model3d := &config.Model3d{Frames: make([]config.Frame3d, 1)}
	model3d.Frames[0].Triangles = allTriangles

	// 4. Iniezione nel ConfigThing
	// Nota: usiamo config.ThingItemDef e un raggio/altezza fittizi per le collisioni
	thingCfg := config.NewConfigThing(classname, pos, 0.0, config.ThingItemDef, 16.0, 16.0, 32.0, 0.0, nil)
	thingCfg.SetModel3d(model3d)

	return thingCfg, nil
}

// createXYZ creates and returns a geometry.XYZ struct using the provided x, y, and z coordinates.
func (p *Builder) createXYZ(x, y, z float64) geometry.XYZ {
	// Conversione coordinate: Quake Z-up -> Engine Z-up
	//pos := geometry.XYZ{X: x, Y: z, Z: -y}
	pos := geometry.XYZ{X: x, Y: y, Z: z}
	return pos
}

// triangulateConvex3d generates a triangle fan from a convex 3D polygon defined by a list of vertices.
// It returns a slice of slices, each containing exactly three vertices representing a single triangle.
func (p *Builder) triangulateConvex3d(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil // Poligono degenere
	}
	if pLen == 3 {
		return [][]geometry.XYZ{{pts[0], pts[1], pts[2]}}
	}
	output := make([][]geometry.XYZ, 0, pLen-2)
	// Triangle Fan ancorato a pts[0]
	for i := 1; i < pLen-1; i++ {
		output = append(output, []geometry.XYZ{pts[0], pts[i], pts[i+1]})
	}
	return output
}

// triangulateConvex3dInverted triangulates a convex 3D polygon into triangles in inverted winding order.
func (p *Builder) triangulateConvex3dInverted(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil
	}
	if pLen == 3 {
		// INVERTITO: da (0, 1, 2) a (0, 2, 1)
		return [][]geometry.XYZ{{pts[0], pts[2], pts[1]}}
	}

	output := make([][]geometry.XYZ, 0, pLen-2)
	for i := 1; i < pLen-1; i++ {
		// INVERTITO: pts[i+1] viene PRIMA di pts[i]
		output = append(output, []geometry.XYZ{pts[0], pts[i+1], pts[i]})
	}
	return output
}

// parseVector estrae 3 float da una stringa stile Quake (es. "1.0 0.5 0.0")
func (p *Builder) parseVector(s string) (float64, float64, float64, bool) {
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		v1, _ := strconv.ParseFloat(parts[0], 64)
		v2, _ := strconv.ParseFloat(parts[1], 64)
		v3, _ := strconv.ParseFloat(parts[2], 64)
		return v1, v2, v3, true
	}
	return 0, 0, 0, false
}

// calcDirection converte gli angoli Quake (yaw, pitch) in un vettore direzionale normalizzato
func (p *Builder) calcDirection(yaw, pitch float64) (float64, float64, float64) {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0
	dirX := math.Cos(pitchRad) * math.Cos(yawRad)
	dirY := math.Sin(pitchRad)
	dirZ := math.Cos(pitchRad) * math.Sin(yawRad)
	return dirX, dirY, dirZ
}
