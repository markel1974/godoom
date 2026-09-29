package q3

import (
	"encoding/binary"
	"fmt"
	_ "image/jpeg"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// BSPVersionQ3 represents the BSP version number used for Quake 3 map files.
const BSPVersionQ3 int = 46

// BSPReader provides functionality to parse and read Quake 3 BSP (Binary Space Partitioning) map files.
type BSPReader struct {
	arc         interfaces.IArchive
	header      lumps.Header3
	rs          io.ReadSeeker
	texManager  *lumps.Textures
	playerAngle float64
	playerPos   geometry.XYZ
	shaders     *Shaders
}

// NewQ3BSPReader creates a new instance of BSPReader with the provided archive and ReadSeeker.
func NewQ3BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *BSPReader {
	q3 := &BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: lumps.NewTextures(),
		shaders:    NewShaders(),
	}

	return q3
}

// Setup initializes the BSPReader by parsing the header, validating file format, and loading shaders for additive materials.
func (q3 *BSPReader) Setup() error {
	_, err := q3.rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	q3.header, err = lumps.NewHeader3(q3.rs)
	if string(q3.header.Magic[:]) != "IBSP" || q3.header.Version != 46 {
		return fmt.Errorf("formato Quake 3 non valido (Magic: %s, Versione: %d)", string(q3.header.Magic[:]), q3.header.Version)
	}
	q3.shaders = NewShaders()
	if err = q3.shaders.Parse(q3.arc); err != nil {
		return err
	}
	return nil
}

// GetArchive returns the IArchive instance associated with the BSPReader.
func (q3 *BSPReader) GetArchive() interfaces.IArchive {
	return q3.arc
}

// GetPlayerInfo retrieves the player's view angle (in radians) and position in 3D space.
func (q3 *BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q3.playerAngle, q3.playerPos
}

// GetEntities retrieves all entities from the BSP file by parsing the entities lump and returns them as a slice.
func (q3 *BSPReader) GetEntities() ([]*lumps.Entity, error) {
	lump := q3.header.Lumps[lumps.LumpQ3Entities]
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
func (q3 *BSPReader) GetModels() ([]*lumps.Model, error) {
	lModels := q3.header.Lumps[lumps.LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}

	numModels := int(lModels.Length) / 40
	models := make([]lumps.Model3, numModels)
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

// GetTextures retrieves the texture manager containing the loaded textures for the current Q3 BSP file.
func (q3 *BSPReader) GetTextures() *lumps.Textures {
	return q3.texManager
}

// GetRawFaces retrieves all raw face data for a specific model index from the BSP file, including geometry and texture info.
// Returns a slice of RawFace objects or an error if the operation fails.
func (q3 *BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace, error) {
	lModels := q3.header.Lumps[lumps.LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to models lump: %w", err)
	}
	models := make([]lumps.Model3, int(lModels.Length)/40)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, fmt.Errorf("failed to read models lump: %w", err)
	}

	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("modelIdx out of range")
	}
	targetModel := models[modelIdx]

	//Geometrical lumps
	lFaces := q3.header.Lumps[lumps.LumpQ3Faces]
	if _, err := q3.rs.Seek(int64(lFaces.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to faces lump: %w", err)
	}
	faces := make([]lumps.Face3, int(lFaces.Length)/104)
	if err := binary.Read(q3.rs, binary.LittleEndian, &faces); err != nil {
		return nil, fmt.Errorf("failed to read faces lump: %w", err)
	}

	lVerts := q3.header.Lumps[lumps.LumpQ3Vertexes]
	if _, err := q3.rs.Seek(int64(lVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to vertexes lump: %w", err)
	}
	vertexes := make([]lumps.Vertex3, int(lVerts.Length)/44)
	if err := binary.Read(q3.rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, fmt.Errorf("failed to read vertexes lump: %w", err)
	}

	lMeshVerts := q3.header.Lumps[lumps.LumpQ3MeshVerts]
	if _, err := q3.rs.Seek(int64(lMeshVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to mesh vertices lump: %w", err)
	}
	meshVerts := make([]int32, int(lMeshVerts.Length)/4)
	if err := binary.Read(q3.rs, binary.LittleEndian, &meshVerts); err != nil {
		return nil, fmt.Errorf("failed to read mesh vertices lump: %w", err)
	}

	lTextures := q3.header.Lumps[lumps.LumpQ3Textures]
	if _, err := q3.rs.Seek(int64(lTextures.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to textures lump: %w", err)
	}
	textures := make([]lumps.Texture3, int(lTextures.Length)/72)
	if err := binary.Read(q3.rs, binary.LittleEndian, &textures); err != nil {
		return nil, fmt.Errorf("failed to read textures lump: %w", err)
	}

	var rawFaces []*lumps.RawFace

	// 3. Risoluzione Topologica
	for i := int32(0); i < targetModel.NumFaces; i++ {
		face := faces[targetModel.FirstFace+i]
		tex := textures[face.TextureID]
		if (tex.Flags & 0x80) != 0 {
			continue // SURF_NODRAW
		}
		texNameBytes := make([]byte, 0, len(tex.Name))
		for _, b := range tex.Name {
			if b == 0 || len(texNameBytes) >= len(tex.Name)-1 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.TrimSpace(strings.ToLower(string(texNameBytes)))
		if q3.shaders.IsNodraw(texName) {
			continue // shader has surfaceparm nodraw
		}
		isSky := (tex.Flags & 0x4) != 0 // SURF_SKY
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
					var cp [9]lumps.Vertex3
					for row := 0; row < 3; row++ {
						for col := 0; col < 3; col++ {
							cpIdx := face.VertexStart + int32((y*2+row)*w+(x*2+col))
							cp[row*3+col] = vertexes[cpIdx]
						}
					}

					// Livello di Tassellatura (LOD). 5 = Risoluzione standard.
					triangles, uvs := lumps.Tessellate(cp, 5)

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
func (q3 *BSPReader) compileTextures(faces []*lumps.RawFace) {
	// Map to track if a physical texture NEEDS alpha test.
	// If it doesn't need alpha test, we can force it to be opaque.
	needsAlphaTest := make(map[string]bool)

	for _, f := range faces {
		texNameLC := f.TexName
		hasAlpha := q3.shaders.HasAlphaTest(texNameLC)

		if animMap := q3.shaders.GetAnimMap(texNameLC); len(animMap) > 0 {
			for _, frameTex := range animMap {
				frameLC := strings.ToLower(frameTex)
				if existing, ok := needsAlphaTest[frameLC]; !ok || (!existing && hasAlpha) {
					needsAlphaTest[frameLC] = hasAlpha
				}
			}
		} else {
			targetTex := texNameLC
			if diffMap := q3.shaders.GetDiffuseMap(texNameLC); diffMap != "" {
				targetTex = strings.ToLower(diffMap)
			}
			if existing, ok := needsAlphaTest[targetTex]; !ok || (!existing && hasAlpha) {
				needsAlphaTest[targetTex] = hasAlpha
			}
		}
	}

	il := NewImageLoader(q3.arc, q3.texManager, q3.shaders)
	for texName, requiresAlpha := range needsAlphaTest {
		forceOpaque := !requiresAlpha
		if err := il.Load(texName, forceOpaque); err != nil {
			fmt.Printf("Warning: %s\n", err.Error())
			continue
		}
	}
}

// GetExternalBModelFileName retrieves the external BSP model filename associated with the given classname.
func (q3 *BSPReader) GetExternalBModelFileName(classname string) string {
	return _q3DictBModel[classname]
}

// GetModelFileName retrieves the file path of the model associated with the given classname from the model filename map.
func (q3 *BSPReader) GetModelFileName(classname string) string {
	return _q3DictModelFilename[classname]
}

// Build processes entities and geometry from a BSPReader, organizing them into the root config structure.
func (q3 *BSPReader) Build(root *config.Root) error {
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
	volumes := NewVolumes(q3.shaders)

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
					q3.playerPos = pos
					q3.playerAngle = angle * (math.Pi / 180.0)
					playerSpawned = true
				} else {
					// We use remaining spawn points as Bot spawners
					enemyClass := "enemy_bot"
					thingPath := q3.GetModelFileName(enemyClass)

					var cThing *config.Thing
					var err error
					if strings.HasSuffix(thingPath, "/") {
						cThing, err = things.CreatePlayer(thingPath, pos, enemyClass)
					} else {
						cThing, err = things.Create(thingPath, pos, enemyClass)
					}

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
			var cThing *config.Thing
			var err error
			if strings.HasSuffix(thingPath, "/") {
				cThing, err = things.CreatePlayer(thingPath, pos, classname)
			} else {
				cThing, err = things.Create(thingPath, pos, classname)
			}

			if err != nil {
				fmt.Printf("Warning can't create thing: %s\n", err.Error())
			} else {
				root.Things = append(root.Things, cThing)
			}
		}
	}
	if cVolumes, err := volumes.Create(mIdx, rawFaces); err != nil {
		fmt.Printf("Warning can't create faces: %s\n", err.Error())
	} else {
		root.Volumes = cVolumes
	}
	return nil
}
