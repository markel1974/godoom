package q3

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
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// LumpQ3Entities represents the lump index for storing entity data in a Quake 3 map.
// LumpQ3Textures represents the lump index for storing texture data in a Quake 3 map.
// LumpQ3Planes represents the lump index for storing plane data in a Quake 3 map.
// LumpQ3Nodes represents the lump index for storing node data in a Quake 3 map.
// LumpQ3Leafs represents the lump index for storing leaf data in a Quake 3 map.
// LumpQ3LeafFaces represents the lump index for storing leaf face data in a Quake 3 map.
// LumpQ3LeafBrushes represents the lump index for storing leaf brush data in a Quake 3 map.
// LumpQ3Models represents the lump index for storing model data in a Quake 3 map.
// LumpQ3Brushes represents the lump index for storing brush data in a Quake 3 map.
// LumpQ3BrushSides represents the lump index for storing brush side data in a Quake 3 map.
// LumpQ3Vertexes represents the lump index for storing vertex data in a Quake 3 map.
// LumpQ3MeshVerts represents the lump index for storing mesh vertex data in a Quake 3 map.
// LumpQ3Effects represents the lump index for storing special effect data in a Quake 3 map.
// LumpQ3Faces represents the lump index for storing face data in a Quake 3 map.
// LumpQ3Lightmaps represents the lump index for storing lightmap data in a Quake 3 map.
// LumpQ3LightVols represents the lump index for storing light volume data in a Quake 3 map.
// LumpQ3VisData represents the lump index for storing visibility data in a Quake 3 map.
// NumQ3Lumps represents the total number of lumps in a Quake 3 map.
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

// BSPVersionQ3 represents the BSP version number used for Quake 3 map files.
const BSPVersionQ3 int = 46

// HeaderQ3 represents the header structure of a Quake 3 BSP file, containing metadata and lump information.
type HeaderQ3 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ3Lumps]struct {
		Offset int32
		Length int32
	}
}

// q3Texture represents a texture used in a Quake III BSP file.
// It includes the texture name, rendering flags, and content flags.
type q3Texture struct {
	Name     [64]byte
	Flags    uint32
	Contents uint32
}

// q3Model represents a 3D model in a Quake 3 BSP file, including bounds and references to associated geometry data.
type q3Model struct {
	Mins       [3]float32
	Maxs       [3]float32
	FirstFace  int32
	NumFaces   int32
	FirstBrush int32
	NumBrushes int32
}

// q3Face represents a face in a Quake 3 BSP map, including its geometry, texture, and lightmap information.
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

// q3Vertex represents a single vertex in a Quake 3 BSP model structure.
// Position defines the 3D coordinates of the vertex in space.
// TexCoord specifies the 2D texture mapping coordinates.
// LMapCoord provides the lightmap texture coordinates.
// Normal defines the surface normal vector at the vertex.
// Color stores the RGBA color values of the vertex.
type q3Vertex struct {
	Position  [3]float32
	TexCoord  [2]float32
	LMapCoord [2]float32
	Normal    [3]float32
	Color     [4]uint8
}

// Q3BSPReader provides functionality to parse and read Quake 3 BSP (Binary Space Partitioning) map files.
type Q3BSPReader struct {
	arc         interfaces.IArchive
	header      HeaderQ3
	rs          io.ReadSeeker
	texManager  *lumps.Textures
	playerAngle float64
	playerPos   geometry.XYZ
	shaders     *Shaders
	//additiveMats map[string]bool
}

// NewQ3BSPReader creates a new instance of Q3BSPReader with the provided archive and ReadSeeker.
func NewQ3BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *Q3BSPReader {
	q3 := &Q3BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: lumps.NewTextures(),
		shaders:    NewShaders(),
		//additiveMats: nil,
	}

	return q3
}

// Setup initializes the Q3BSPReader by parsing the header, validating file format, and loading shaders for additive materials.
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
	q3.shaders = NewShaders()
	if err := q3.shaders.Parse(q3.arc); err != nil {
		return err
	}
	//q3.additiveMats = shaders.Retrieve()

	return nil
}

// GetArchive returns the IArchive instance associated with the Q3BSPReader.
func (q3 *Q3BSPReader) GetArchive() interfaces.IArchive {
	return q3.arc
}

// GetPlayerInfo retrieves the player's view angle (in radians) and position in 3D space.
func (q3 *Q3BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q3.playerAngle, q3.playerPos
}

// GetEntities retrieves all entities from the BSP file by parsing the entities lump and returns them as a slice.
func (q3 *Q3BSPReader) GetEntities() ([]*lumps.Entity, error) {
	lump := q3.header.Lumps[LumpQ3Entities]
	if _, err := q3.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q3.rs.Read(data); err != nil {
		return nil, err
	}
	return lumps.NewEntitiesFromText(lumps.FromNullTerminatingString(data))
}

// GetModels extracts and returns all BSP sub-models from the lump data, including static and moving brush models.
func (q3 *Q3BSPReader) GetModels() ([]*lumps.Model, error) {
	lModels := q3.header.Lumps[LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}

	numModels := int(lModels.Length) / 40
	models := make([]q3Model, numModels)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*lumps.Model, numModels)
	for i, m := range models {
		out[i] = &lumps.Model{
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

// RegisterPixels registers a texture by name with specified dimensions and pixel indices data, supporting transparency and Y-inversion.
func (q3 *Q3BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return nil // In Q3 le texture sono solitamente .tga o .jpg lette dal VFS nativamente come RGBA, il manager andrà adattato
}

// RegisterPixelsRGBA registers an RGBA texture in the texture manager with the given name, dimensions, and pixel data.
// Pixels should be provided as a byte slice in RGBA format, and setting invertY to true flips the texture vertically.
// Returns an error if the registration fails.
func (q3 *Q3BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q3.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

// GetTextures retrieves the texture manager containing the loaded textures for the current Q3 BSP file.
func (q3 *Q3BSPReader) GetTextures() *lumps.Textures {
	return q3.texManager
}

// GetRawFaces retrieves all raw face data for a specific model index from the BSP file, including geometry and texture info.
// Returns a slice of RawFace objects or an error if the operation fails.
func (q3 *Q3BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace, error) {
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

	var rawFaces []*lumps.RawFace

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
					tri = append(tri, lumps.CreateXYZ(float64(v.Position[0]), float64(v.Position[1]), float64(v.Position[2])))
					uvs = append(uvs, [2]float64{float64(v.TexCoord[0]), float64(v.TexCoord[1])})
				}
				rawFaces = append(rawFaces, &lumps.RawFace{
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
						rawFaces = append(rawFaces, &lumps.RawFace{
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

// compileTextures loads and registers unique textures from a list of faces, supporting JPEG and TGA formats.
func (q3 *Q3BSPReader) compileTextures(faces []*lumps.RawFace) {
	uniqueTextures := make(map[string]bool)
	for _, f := range faces {
		uniqueTextures[f.TexName] = true
	}

	for texName := range uniqueTextures {
		if texName == "noshader" || len(texName) == 0 {
			continue
		}

		baseName := BaseName(texName)
		var img image.Image
		var err error

		// Prova prima con TGA (perché supporta l'alpha channel natively)
		tgaPath := baseName + ".tga"
		fileTga, errTga := q3.arc.Open(tgaPath)
		if errTga == nil {
			img, err = common.DecodeTGA(fileTga)
		} else {
			// Fallback su JPEG
			jpgPath := baseName + ".jpg"
			fileJpg, errJpg := q3.arc.Open(jpgPath)
			if errJpg == nil {
				img, _, err = image.Decode(fileJpg)
			} else {
				// Shader Fallback (se il nome è uno shader noto, puntiamo alla texture base)
				fallbackName, ok := _q3ShaderFallback[baseName]
				if !ok {
					// Also try original texName just in case
					fallbackName, ok = _q3ShaderFallback[texName]
					if !ok {
						fmt.Printf("warning: missing asset %s (%s | %s)\n", texName, tgaPath, jpgPath)
						continue
					}
				}
				if fallbackName == "" {
					continue // Skips textures explicitly mapped to empty string (e.g. fog)
				}
				// Riprova con il fallback (sempre prima TGA poi JPG)
				fbTga := fallbackName + ".tga"
				if fTga, e := q3.arc.Open(fbTga); e == nil {
					img, err = common.DecodeTGA(fTga)
				} else {
					fbJpg := fallbackName + ".jpg"
					if fJpg, e := q3.arc.Open(fbJpg); e == nil {
						img, _, err = image.Decode(fJpg)
					} else {
						fmt.Printf("warning: missing asset %s (.jpg/.tga) (fallito anche il fallback)\n", texName)
						continue
					}
				}
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
		// =================================================================================================
		// Invio del buffer [R,G,B,A, R,G,B,A...] al manager.
		// NOTA: Usa un metodo specifico per i 32-bit (es. RegisterPixelsRGBA)
		// bypassando la logica della palette a 8-bit usata in Q1/Q2.
		err = q3.texManager.RegisterPixelsRGBA(texName, bounds.Dx(), bounds.Dy(), rgba.Pix, true)

		if err != nil {
			fmt.Printf("Warning: registrazione texture fallita %s: %v\n", texName, err)
		}
	}
}

// evalBezier computes the value of a quadratic Bézier curve given control points p0, p1, p2 and a parameter t [0,1].
func (q3 *Q3BSPReader) evalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// tessellatePatch calculates a triangle mesh from a 3x3 patch of control points, using a specified tessellation level.
// The method returns a list of 3D vertices and corresponding UV texture coordinates for the generated mesh.
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
				p[row] = lumps.CreateXYZ(
					float64(q3.evalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(q3.evalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(q3.evalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
				puv[row] = [2]float32{
					q3.evalBezier(cp[idx].TexCoord[0], cp[idx+1].TexCoord[0], cp[idx+2].TexCoord[0], tU),
					q3.evalBezier(cp[idx].TexCoord[1], cp[idx+1].TexCoord[1], cp[idx+2].TexCoord[1], tU),
				}
			}
			grid[i*L+j] = lumps.CreateXYZ(
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

// GetExternalBModelFileName retrieves the external BSP model filename associated with the given classname.
func (q3 *Q3BSPReader) GetExternalBModelFileName(classname string) string {
	return _q3DictBModel[classname]
}

// GetModelFileName retrieves the file path of the model associated with the given classname from the model filename map.
func (q3 *Q3BSPReader) GetModelFileName(classname string) string {
	return _q3DictModelFilename[classname]
}

// Build processes entities and geometry from a Q3BSPReader, organizing them into the root config structure.
func (q3 *Q3BSPReader) Build(root *config.Root) error {
	mIdx := 0
	rawFaces, rfErr := q3.GetRawFaces(mIdx)
	if rfErr != nil {
		return rfErr
	}
	entities, eErr := q3.GetEntities()
	if eErr != nil {
		return eErr
	}
	things := NewThings(q3.arc, q3.shaders, q3.texManager)
	lights := NewLights(entities)
	faces := NewFaces(q3.shaders)

	playerSpawned := false
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		baseClass := classname
		if z := strings.Split(classname, "_"); len(z) > 1 {
			baseClass = z[0]
		}
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			pos = lumps.CreateXYZ(x, y, z)
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
			if cThing, err := things.CreateBSP(externalBSPPath, pos, classname); err != nil {
				fmt.Printf("warning on external bmodel %s: %v)\n", classname, err)
			} else {
				root.Things = append(root.Things, cThing)
			}
			continue
		}

		switch baseClass {
		case "worldspawn":
			// Ignored: it is the base map, geometry is already handled by worldModel
		case "info":
			if classname == "info_player_start" || classname == "info_player_deathmatch" {
				if !playerSpawned {
					var err error
					q3.playerPos, q3.playerAngle, err = q3.createPlayerProps(angle, pos)
					if err != nil {
						fmt.Printf("Warning: %s\n", err.Error())
					}
					playerSpawned = true
				} else {
					// We use remaining spawn points as Bot spawners
					enemyClass := "enemy_bot"
					thingPath := q3.GetModelFileName(enemyClass)
					cThing, err := things.Create(thingPath, pos, enemyClass)
					if err == nil {
						root.Things = append(root.Things, cThing)
					} else {
						fmt.Printf("Warning BotSpawner: %s\n", err.Error())
					}
				}
			} else {
				// Invisible markers: teleports, deathmatch spawn points, patrol nodes.
				// TODO: Save them in a gameplay waypoint/spawnpoint list.
			}
		case "light":
			if cLight, err := lights.Create(ent, angle, pos); err != nil {
				fmt.Printf("Warning can't create light: %s\n", err.Error())
			} else {
				root.Lights = append(root.Lights, cLight)
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
		case "target", "misc", "shooter":
		// TODO: ignore invisible targets and misc models for now
		default:
			thingPath := q3.GetModelFileName(classname)
			if cThing, err := things.Create(thingPath, pos, classname); err != nil {
				fmt.Printf("Warning can't create thing: %s\n", err.Error())
			} else {
				root.Things = append(root.Things, cThing)
			}
		}
	}
	if cVolumes, err := faces.CreateFaces(mIdx, rawFaces); err != nil {
		fmt.Printf("Warning can't create faces: %s\n", err.Error())
	} else {
		root.Volumes = cVolumes
	}
	return nil
}

// createPlayerProps calculates the player's angle in radians and returns updated position, angle, and error if any.
func (q3 *Q3BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}
