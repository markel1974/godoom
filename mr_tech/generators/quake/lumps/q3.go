package lumps

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	LumpQ3Entities    = 0
	LumpQ3Textures    = 1
	LumpQ3Planes      = 2
	LumpQ3Nodes       = 3
	LumpQ3Leafs       = 4
	LumpQ3LeafFaces   = 5
	LumpQ3LeafBrushes = 6
	LumpQ3Models      = 7
	LumpQ3Brushes     = 8
	LumpQ3BrushSides  = 9
	LumpQ3Vertexes    = 10
	LumpQ3MeshVerts   = 11
	LumpQ3Effects     = 12
	LumpQ3Faces       = 13
	LumpQ3Lightmaps   = 14
	LumpQ3LightVols   = 15
	LumpQ3VisData     = 16
	NumQ3Lumps        = 17
)

type HeaderQ3 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ3Lumps]struct {
		Offset int32
		Length int32
	}
}

type q3Texture struct {
	Name     [64]byte
	Flags    uint32
	Contents uint32
}

type q3Model struct {
	Mins       [3]float32
	Maxs       [3]float32
	FirstFace  int32
	NumFaces   int32
	FirstBrush int32
	NumBrushes int32
}

type q3Face struct {
	TextureID   int32
	Effect      int32
	Type        int32 // 1=Polygon, 2=Patch, 3=Mesh, 4=Billboard
	VertexStart int32
	NumVertexes int32
	MeshStart   int32
	NumMesh     int32
	LightmapID  int32
	LMapCorner  [2]int32
	LMapSize    [2]int32
	LMapOrigin  [3]float32
	LMapVecs    [2][3]float32
	Normal      [3]float32
	PatchSize   [2]int32
}

type q3Vertex struct {
	Position  [3]float32
	TexCoord  [2]float32
	LMapCoord [2]float32
	Normal    [3]float32
	Color     [4]uint8
}

// Q3BSPReader analizza le mappe in formato idTech 3 (Quake 3 / Return to Castle Wolfenstein)
type Q3BSPReader struct {
	arc         IArchive
	header      HeaderQ3
	rs          io.ReadSeeker
	texManager  *Textures
	playerAngle float64
	playerPos   geometry.XYZ
}

func NewQ3BSPReader(arc IArchive, rs io.ReadSeeker) *Q3BSPReader {
	return &Q3BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: NewTextures(),
	}
}

func (q3 *Q3BSPReader) Setup() error {
	if _, err := q3.rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(q3.rs, binary.LittleEndian, &q3.header); err != nil {
		return err
	}
	if string(q3.header.Magic[:]) != "IBSP" || q3.header.Version != 46 {
		return fmt.Errorf("formato Quake 3 non valido (Magic: %s, Versione: %d)", string(q3.header.Magic[:]), q3.header.Version)
	}
	return nil
}

// GetArchive returns the IArchive instance associated with the Q1BSPReader, used for file access and data retrieval.
func (q3 *Q3BSPReader) GetArchive() IArchive {
	return q3.arc
}

// GetPlayerInfo retrieves the player's viewing angle in radians and position in 3D space as a geometry.XYZ struct.
func (q3 *Q3BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q3.playerAngle, q3.playerPos
}

func (q3 *Q3BSPReader) GetEntities() ([]*Entity, error) {
	lump := q3.header.Lumps[LumpQ3Entities]
	if _, err := q3.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q3.rs.Read(data); err != nil {
		return nil, err
	}
	return NewEntitiesFromText(FromNullTerminatingString(data))
}

func (q3 *Q3BSPReader) GetModels() ([]*Model, error) {
	lModels := q3.header.Lumps[LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}

	numModels := int(lModels.Length) / 40
	models := make([]q3Model, numModels)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*Model, numModels)
	for i, m := range models {
		out[i] = &Model{
			Mins:      m.Mins,
			Maxs:      m.Maxs,
			FirstFace: m.FirstFace,
			NumFaces:  m.NumFaces,
			// Q3 non usa Origin/HeadNode/VisLeafs nel lump Models, le collisioni
			// sono basate sui Brush associati (FirstBrush, NumBrushes).
		}
	}
	return out, nil
}

func (q3 *Q3BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return nil // In Q3 le texture sono solitamente .tga o .jpg lette dal VFS nativamente come RGBA, il manager andrà adattato
}

func (q3 *Q3BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q3.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

func (q3 *Q3BSPReader) GetTextures() *Textures {
	return q3.texManager
}

// GetRawFaces risolve nativamente Index Buffer (MeshVerts) e Patch di Bezier
func (q3 *Q3BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	lModels := q3.header.Lumps[LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to models lump: %w", err)
	}
	models := make([]q3Model, int(lModels.Length)/40)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, fmt.Errorf("failed to read models lump: %w", err)
	}

	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("modelIdx out of range")
	}
	targetModel := models[modelIdx]

	//Geometrical lumps
	lFaces := q3.header.Lumps[LumpQ3Faces]
	if _, err := q3.rs.Seek(int64(lFaces.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to faces lump: %w", err)
	}
	faces := make([]q3Face, int(lFaces.Length)/104)
	if err := binary.Read(q3.rs, binary.LittleEndian, &faces); err != nil {
		return nil, fmt.Errorf("failed to read faces lump: %w", err)
	}

	lVerts := q3.header.Lumps[LumpQ3Vertexes]
	if _, err := q3.rs.Seek(int64(lVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to vertexes lump: %w", err)
	}
	vertexes := make([]q3Vertex, int(lVerts.Length)/44)
	if err := binary.Read(q3.rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, fmt.Errorf("failed to read vertexes lump: %w", err)
	}

	lMeshVerts := q3.header.Lumps[LumpQ3MeshVerts]
	if _, err := q3.rs.Seek(int64(lMeshVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to mesh verts lump: %w", err)
	}
	meshVerts := make([]int32, int(lMeshVerts.Length)/4)
	if err := binary.Read(q3.rs, binary.LittleEndian, &meshVerts); err != nil {
		return nil, fmt.Errorf("failed to read mesh verts lump: %w", err)
	}

	lTextures := q3.header.Lumps[LumpQ3Textures]
	if _, err := q3.rs.Seek(int64(lTextures.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to textures lump: %w", err)
	}
	textures := make([]q3Texture, int(lTextures.Length)/72)
	if err := binary.Read(q3.rs, binary.LittleEndian, &textures); err != nil {
		return nil, fmt.Errorf("failed to read textures lump: %w", err)
	}

	var rawFaces []*RawFace

	// 3. Risoluzione Topologica
	for i := int32(0); i < targetModel.NumFaces; i++ {
		face := faces[targetModel.FirstFace+i]
		tex := textures[face.TextureID]

		texNameBytes := make([]byte, 0, 64)
		for _, b := range tex.Name {
			if b == 0 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.ToLower(string(texNameBytes))
		isSky := (tex.Flags & 0x4) != 0 // SURF_SKY

		if (tex.Flags & 0x80) != 0 {
			continue // SURF_NODRAW
		}

		switch face.Type {
		case 1, 3: // Poligono Convesso (1) o Mesh Complessa (3)
			// Q3 usa l'indicizzazione per formare direttamente triangoli
			for j := int32(0); j < face.NumMesh; j += 3 {
				var tri []geometry.XYZ
				var uvs [][2]float64
				for k := int32(0); k < 3; k++ {
					vIdx := face.VertexStart + meshVerts[face.MeshStart+j+k]
					v := vertexes[vIdx]
					tri = append(tri, CreateXYZ(float64(v.Position[0]), float64(v.Position[1]), float64(v.Position[2])))
					uvs = append(uvs, [2]float64{float64(v.TexCoord[0]), float64(v.TexCoord[1])})
				}
				rawFaces = append(rawFaces, &RawFace{
					Points:  tri, // Il Builder non dovrà fare il Fan se riceve già 3 punti
					UVs:     uvs,
					TexName: texName,
					IsSky:   isSky,
				})
			}

		case 2: // PATCH DI BEZIER (Biquadratica)
			w := int(face.PatchSize[0])
			h := int(face.PatchSize[1])

			// Le patch in Q3 sono griglie 3x3 unite. Troviamo quante sub-patch ci sono.
			numPatchesX := (w - 1) / 2
			numPatchesY := (h - 1) / 2

			for y := 0; y < numPatchesY; y++ {
				for x := 0; x < numPatchesX; x++ {
					var cp [9]q3Vertex
					for row := 0; row < 3; row++ {
						for col := 0; col < 3; col++ {
							cpIdx := face.VertexStart + int32((y*2+row)*w+(x*2+col))
							cp[row*3+col] = vertexes[cpIdx]
						}
					}

					// Livello di Tassellatura (LOD). 5 = Risoluzione standard.
					triangles, uvs := q3.tessellatePatch(cp, 5)

					for t := 0; t < len(triangles); t += 3 {
						rawFaces = append(rawFaces, &RawFace{
							Points:  []geometry.XYZ{triangles[t], triangles[t+1], triangles[t+2]},
							UVs:     [][2]float64{uvs[t], uvs[t+1], uvs[t+2]},
							TexName: texName,
							IsSky:   isSky,
						})
					}
				}
			}
		}
	}

	q3.compileTextures(rawFaces)

	return rawFaces, nil
}

// compileTextures cerca e decodifica i file JPEG associati alle facce estratte
func (q3 *Q3BSPReader) compileTextures(faces []*RawFace) {
	uniqueTextures := make(map[string]bool)
	for _, f := range faces {
		uniqueTextures[f.TexName] = true
	}

	for texName := range uniqueTextures {
		if texName == "noshader" || len(texName) == 0 {
			continue
		}

		var img image.Image
		var err error

		// Prova prima con JPEG
		jpgPath := texName + ".jpg"
		file, errJpg := q3.arc.Open(jpgPath)
		if errJpg == nil {
			img, _, err = image.Decode(file)
		} else {
			// Fallback su TGA
			tgaPath := texName + ".tga"
			fileTga, errTga := q3.arc.Open(tgaPath)
			if errTga == nil {
				img, err = common.DecodeTGA(fileTga)
			} else {
				fmt.Printf("Warning: asset mancante %s (.jpg/.tga)\n", texName)
				continue
			}
		}

		if err != nil {
			fmt.Printf("Warning: decodifica fallita per %s: %v\n", texName, err)
			continue
		}

		// Normalizzazione in memoria spaziale lineare a 32-bit (RGBA)
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

		// Invio del buffer [R,G,B,A, R,G,B,A...] al manager.
		// NOTA: Usa un metodo specifico per i 32-bit (es. RegisterPixelsRGBA)
		// bypassando la logica della palette a 8-bit usata in Q1/Q2.
		err = q3.texManager.RegisterPixelsRGBA(texName, bounds.Dx(), bounds.Dy(), rgba.Pix, false)

		if err != nil {
			fmt.Printf("Warning: registrazione texture fallita %s: %v\n", texName, err)
		}
	}
}

// evalBezier calcola la coordinata lungo la curva di grado 2 per il fattore t [0.0 - 1.0]
func (q3 *Q3BSPReader) evalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// tessellatePatch espande i 9 punti di controllo in un array flat di triangoli
func (q3 *Q3BSPReader) tessellatePatch(cp [9]q3Vertex, level int) ([]geometry.XYZ, [][2]float64) {
	var points []geometry.XYZ
	var uvs [][2]float64
	step := 1.0 / float32(level)
	L := level + 1
	grid := make([]geometry.XYZ, L*L)
	gridUV := make([][2]float64, L*L)

	// Calcolo interpolazione griglia
	for i := 0; i <= level; i++ {
		tV := float32(i) * step
		for j := 0; j <= level; j++ {
			tU := float32(j) * step
			var p [3]geometry.XYZ
			var puv [3][2]float32
			for row := 0; row < 3; row++ {
				idx := row * 3
				p[row] = CreateXYZ(
					float64(q3.evalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(q3.evalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(q3.evalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
				puv[row] = [2]float32{
					q3.evalBezier(cp[idx].TexCoord[0], cp[idx+1].TexCoord[0], cp[idx+2].TexCoord[0], tU),
					q3.evalBezier(cp[idx].TexCoord[1], cp[idx+1].TexCoord[1], cp[idx+2].TexCoord[1], tU),
				}
			}
			grid[i*L+j] = CreateXYZ(
				float64(q3.evalBezier(float32(p[0].X), float32(p[1].X), float32(p[2].X), tV)),
				float64(q3.evalBezier(float32(p[0].Y), float32(p[1].Y), float32(p[2].Y), tV)),
				float64(q3.evalBezier(float32(p[0].Z), float32(p[1].Z), float32(p[2].Z), tV)),
			)
			gridUV[i*L+j] = [2]float64{
				float64(q3.evalBezier(puv[0][0], puv[1][0], puv[2][0], tV)),
				float64(q3.evalBezier(puv[0][1], puv[1][1], puv[2][1], tV)),
			}
		}
	}

	// Chiusura dei quadrati in triangoli (Winding Order CCW)
	for i := 0; i < level; i++ {
		for j := 0; j < level; j++ {
			idx0 := (i * L) + j
			idx1 := (i * L) + j + 1
			idx2 := ((i + 1) * L) + j
			idx3 := ((i + 1) * L) + j + 1

			points = append(points, grid[idx0], grid[idx2], grid[idx1])
			uvs = append(uvs, gridUV[idx0], gridUV[idx2], gridUV[idx1])

			points = append(points, grid[idx1], grid[idx2], grid[idx3])
			uvs = append(uvs, gridUV[idx1], gridUV[idx2], gridUV[idx3])
		}
	}
	return points, uvs
}

// GetExternalBModelFileName retrieves the file name of an external BModel based on the provided classname.
func (q3 *Q3BSPReader) GetExternalBModelFileName(classname string) string {
	return _q3DictBModel[classname]
}

// GetModelFileName returns the file name of the model associated with the specified classname.
func (q3 *Q3BSPReader) GetModelFileName(classname string) string {
	return _q3DictModelFilename[classname]
}

func (q3 *Q3BSPReader) Build(root *config.Root) error {
	const chunkSize = float64(1024)
	mIdx := 0
	faces, rfErr := q3.GetRawFaces(mIdx)
	if rfErr != nil {
		return rfErr
	}
	entities, eErr := q3.GetEntities()
	if eErr != nil {
		return eErr
	}
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		baseClass := classname
		subClass := ""
		if z := strings.Split(classname, "_"); len(z) > 1 {
			baseClass = z[0]
			subClass = z[1]
		}
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			pos = CreateXYZ(x, y, z)
		}

		var angle float64
		if a, ok := ent.Properties["angle"]; ok {
			angle, _ = strconv.ParseFloat(a, 64)
		}

		// TODO: Currently we are ignoring sub-models (*1, *2, etc.) like func_door or func_plat.
		// Before focusing on "accessories", let's ensure that worldspawn (the base map)
		// is rendered correctly. When ready, we will remove this continue
		// and instantiate bmodels using GetModels() from IBSPReader.
		if modelProp := ent.Properties["model"]; strings.HasPrefix(modelProp, "*") {
			continue
		}

		if externalBSPPath := q3.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := q3.createThingBSP(externalBSPPath, pos, classname)
			if err != nil {
				fmt.Printf("warning on external bmodel %s: %v)\n", classname, err)
				continue
			}
			root.Things = append(root.Things, cThing)
			continue
		}

		switch baseClass {
		case "worldspawn":
			// Ignored: it is the base map, geometry is already handled by worldModel
		case "info":
			if classname == "info_player_start" {
				var err error
				q3.playerPos, q3.playerAngle, err = q3.createPlayerProps(angle, pos)
				if err != nil {
					fmt.Printf("Warning: %s\n", err.Error())
				}
			} else {
				// Invisible markers: teleports, deathmatch spawn points, patrol nodes.
				// TODO: Save them in a gameplay waypoint/spawnpoint list.
			}
		case "light":
			mangleStr, _ := ent.Properties["mangle"]
			colorStr, _ := ent.Properties["_color"]
			var light *config.Light = nil
			if len(subClass) == 0 {
				light = q3.createLight(ent, angle, mangleStr, colorStr, pos, _q1LightStyle0, false)
			} else {
				style := _q1LightStyle0
				if sIndex, ok := ent.Properties["style"]; ok {
					if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(_q1LightStyles) {
						style = _q1LightStyles[index]
					}
				}
				// Handles light, light_fluoro, light_fluorospark
				light = q3.createLight(ent, angle, mangleStr, colorStr, pos, style, true)
			}
			if light != nil {
				root.Lights = append(root.Lights, light)
			}
		case "path":
			// Invisible markers: teleports, deathmatch spawn points, patrol nodes.
			// TODO: Save them in a gameplay waypoint/spawnpoint list.
		case "ambient":
			// TODO:
		case "func":
			// TODO:
		case "trigger":
		// TODO:
		//case "trap":
		//TODO
		default:
			cThing, err := q3.createThing(pos, classname)
			if err != nil {
				fmt.Printf("Warning: %s\n", err.Error())
				continue
			}
			root.Things = append(root.Things, cThing)
		}
	}
	vIdx := strconv.Itoa(mIdx)

	chunks := make(map[string]*config.Volume)
	for _, v := range faces {
		animKind := config.MaterialKindLoop
		if v.IsSky {
			animKind = config.MaterialKindSky
		}
		material := config.NewConfigMaterial([]string{v.TexName}, animKind, 1.0, 1.0, 0, 0)
		triangles := TriangulateConvex3d(v.Points)

		for _, tri := range triangles {
			var triUvs [][2]float64
			if len(v.UVs) > 0 {
				triUvs = make([][2]float64, 3)
				for k := 0; k < 3; k++ {
					pos := tri[k]
					for idx, pt := range v.Points {
						if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
							if len(v.UVs) > idx {
								triUvs[k] = v.UVs[idx]
							}
							break
						}
					}
				}
			}

			// 2. Find the triangle centroid
			cx := (tri[0].X + tri[1].X + tri[2].X) / 3.0
			cy := (tri[0].Y + tri[1].Y + tri[2].Y) / 3.0
			cz := (tri[0].Z + tri[1].Z + tri[2].Z) / 3.0

			// 3. Calculate the spatial hashing key (grid coordinates)
			gridX := int(math.Floor(cx / chunkSize))
			gridY := int(math.Floor(cy / chunkSize))
			gridZ := int(math.Floor(cz / chunkSize))

			chunkKey := fmt.Sprintf("%d_%d_%d", gridX, gridY, gridZ)
			volume, exists := chunks[chunkKey]
			if !exists {
				chunkId := fmt.Sprintf("quake_world_%s_chunk_%s", vIdx, chunkKey)
				volume = config.NewConfigVolume(chunkId, "quake_bsp_chunk")
				chunks[chunkKey] = volume
				root.Volumes = append(root.Volumes, volume)
			}
			volume.Faces = append(volume.Faces, config.NewConfigFace(tri, triUvs, material, v.TexName))
		}
	}
	return nil
}

// createPlayerProps extracts player position and angle from an entity and computes the angle in radians.
func (q3 *Q3BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// createLight creates a new Light instance based on entity properties and position, returning an error if invalid or missing data.
func (q3 *Q3BSPReader) createLight(entity *Entity, angle float64, mangleStr, colorStr string, pos geometry.XYZ, style []float64, isSpot bool) *config.Light {
	intensity := 0.0
	falloff := 0.0
	var kind config.LightKind

	// BASE INTENSITY
	if l, ok := entity.Properties["light"]; ok {
		intensity, _ = strconv.ParseFloat(l, 64)
		//intensity *= 0.3
	} else {
		intensity = 300 // Typical Quake default fallback
	}

	// COLOR (Standard Quake 2 / Modern Quake 1)
	r, g, b := 1.0, 1.0, 1.0 // Default White
	if len(colorStr) > 0 {
		if cr, cg, cb, valid := ParseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r, g, b = cr/255.0, cg/255.0, cb/255.0
			} else {
				r, g, b = cr, cg, cb
			}
		}
	}

	// SPOTLIGHT DIRECTION
	dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Default: look down
	if isSpot {
		kind = config.LightKindSpot
		intensity = intensity * 0.9
		falloff = intensity * 10
		if len(mangleStr) > 0 {
			if yaw, pitch, _, valid := ParseVector(mangleStr); valid {
				dirX, dirY, dirZ = CalcDirection(yaw, pitch)
			}
		} else {
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down
			} else {
				dirX, dirY, dirZ = CalcDirection(angle, 0)
			}
		}
	} else {
		kind = config.LightKindAmbient
		intensity = intensity * 0.05
		falloff = intensity
	}

	// CONFIGURATION CREATION
	cl := config.NewConfigLight(pos, intensity, kind, falloff)
	cl.R = r
	cl.G = g
	cl.B = b

	cl.DirX = dirX
	cl.DirY = dirY
	cl.DirZ = dirZ
	cl.Style = style

	return cl
}

// createThing creates a new Thing object based on the specified position, classname, Pak file, and color palette.
func (q3 *Q3BSPReader) createThing(pos geometry.XYZ, classname string) (*config.Thing, error) {
	thingPath := q3.GetModelFileName(classname)
	if len(thingPath) == 0 {
		return nil, fmt.Errorf("unknown thing %s", classname)
	}

	skinTargetIndex := 0
	kind := config.ThingEnemyDef
	var category string
	var definition string
	if c := strings.Split(classname, "_"); len(c) > 1 {
		category = c[0]
		definition = c[1]
	}
	items := map[string]int{"armor1": 0, "armor2": 1, "armorInv": 2}
	switch category {
	case "item":
		kind = config.ThingItemDef
		if skinTIndex, ok := items[definition]; ok {
			skinTargetIndex = skinTIndex
		}
	case "weapon":
		kind = config.ThingItemDef
	case "enemy":
		kind = config.ThingEnemyDef
	case "monster":
		kind = config.ThingEnemyDef
	default:
		return nil, fmt.Errorf("unknown thing %s", classname)
	}
	arc := q3.GetArchive()
	rsMd1, err := arc.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}
	md1 := NewMD1Resource()
	if err = md1.Parse(rsMd1); err != nil {
		return nil, fmt.Errorf("can't load MDL %s: %s\n", classname, err.Error())
	}
	if skinTargetIndex >= len(md1.Skins) {
		return nil, fmt.Errorf("no skin found for %s", classname)
	}
	skin := md1.Skins[skinTargetIndex]
	skinName := fmt.Sprintf("%s_skin_%d", classname, skinTargetIndex)
	if err = q3.RegisterPixels(skinName, int(md1.Header.SkinWidth), int(md1.Header.SkinHeight), skin.Data, false, 255, false); err != nil {
		return nil, fmt.Errorf("Warning: texture %s error: %s\n", skinName, err.Error())
	}
	anim := config.NewConfigMaterial([]string{skinName}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)

	cModel := config.NewMD1(int(md1.Header.NumFrames), md1.FrameNames)
	for idx, f := range md1.Frames {
		triangles := make([]config.MD1Triangle, int(md1.Header.NumTris))
		skinW := float32(md1.Header.SkinWidth)
		skinH := float32(md1.Header.SkinHeight)
		for tIdx, tri := range md1.Triangles {
			cTri := config.NewMD1Triangle(anim)
			for v := 0; v < 3; v++ {
				vx := tri.Vertices[v]
				tc := md1.TexCoords[vx]
				s := float32(tc.S)
				t := float32(tc.T)
				if tri.FacesFront == 0 && tc.OnSeam != 0 {
					s += skinW / 2.0
				}
				nU := s / skinW
				nV := 1.0 - (t / skinH)
				cTri.Vertices[v] = config.MD1Vertex{Pos: CreateXYZ(f[vx][0], f[vx][1], f[vx][2]), U: nU, V: nV}
			}
			triangles[tIdx] = cTri
		}
		cFrame := config.NewMD1Frame(triangles)
		cModel.Frames[idx] = cFrame
	}

	thingCfg := q3.createConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// createThingBSP constructs a Thing instance using external BSP model data, applying positions, textures, and materials.
func (q3 *Q3BSPReader) createThingBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	reader, err := NewBSPReader(q3.GetArchive(), bspPath)
	if err != nil {
		return nil, err
	}
	if err = reader.Setup(); err != nil {
		return nil, err
	}
	bspModels, err := reader.GetModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get models from %s: %v", bspPath, err)
	}
	if len(bspModels) == 0 {
		return nil, fmt.Errorf("no model found in %s", bspPath)
	}
	rawFaces, err := reader.GetRawFaces(0)
	if err != nil {
		return nil, err
	}
	texManager := reader.GetTextures()
	// Geometry translation into agnostic MD1, collect all triangles in this single frame
	var allTriangles []config.MD1Triangle
	for _, bspFace := range rawFaces {
		// RETRIEVAL OF SPECIFIC TEXTURE
		texName := bspFace.TexName
		animKind := config.MaterialKindLoop
		if bspFace.IsSky {
			animKind = config.MaterialKindSky
		}
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, 0, 0)
		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tw, th, pixels := texes[0].RGBA()
			_ = q3.RegisterPixelsRGBA(texName, tw, th, pixels, false)
		}
		rawTriangles := TriangulateConvex3d(bspFace.Points)
		// Assignment of pre-calculated UVs from IBSPReader
		for _, rawTri := range rawTriangles {
			tri := config.NewMD1Triangle(specificMaterial)
			for k := 0; k < 3; k++ {
				pos := rawTri[k]
				u, v := float32(0.0), float32(0.0)
				// Find corresponding UV index for vertex
				for idx, pt := range bspFace.Points {
					if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
						if len(bspFace.UVs) > idx {
							u = float32(bspFace.UVs[idx][0])
							v = float32(bspFace.UVs[idx][1])
						}
						break
					}
				}
				tri.Vertices[k] = config.MD1Vertex{Pos: pos, U: u, V: v}
			}
			allTriangles = append(allTriangles, tri)
		}
	}
	// BSPs do not have vertex-morphing animations, 1 single frame
	model3d := config.NewMD1(1, []string{"default"})
	model3d.Frames[0] = config.NewMD1Frame(allTriangles)
	thingCfg := q3.createConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates a Thing configuration object with properties like position, model, animation, and physics.
func (q3 *Q3BSPReader) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.MD1, angle, mass, radius, height, speed float64) *config.Thing {
	const gForce = 9.8 * 14
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.GForce = gForce
	thingCfg.MD1 = cModel
	if thingCfg.Kind == config.ThingEnemyDef {
		var actions []string
		if thingCfg.MD1 != nil {
			actions = thingCfg.MD1.ActionDefinitions
		}
		enemyLogic := common.NewEnemy(actions, 300)
		thingCfg.OnThinking = enemyLogic.OnThinking
		thingCfg.OnCollision = enemyLogic.OnCollision
		thingCfg.OnImpact = enemyLogic.OnImpact
		thingCfg.WakeUpDistance = 400
	} else {
		itemLogic := common.NewItem()
		thingCfg.OnCollision = itemLogic.OnCollision
		thingCfg.OnImpact = itemLogic.OnImpact
	}
	return thingCfg
}
