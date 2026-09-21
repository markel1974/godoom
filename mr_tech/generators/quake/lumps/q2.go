package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// LumpQ2Entities represents the lump index for entities in Quake 2 BSP files.
// LumpQ2Planes represents the lump index for planes in Quake 2 BSP files.
// LumpQ2Vertexes represents the lump index for vertices in Quake 2 BSP files.
// LumpQ2Visibility represents the lump index for visibility in Quake 2 BSP files.
// LumpQ2Nodes represents the lump index for nodes in Quake 2 BSP files.
// LumpQ2TexInfo represents the lump index for texture information in Quake 2 BSP files.
// LumpQ2Faces represents the lump index for faces in Quake 2 BSP files.
// LumpQ2Lighting represents the lump index for lighting in Quake 2 BSP files.
// LumpQ2Leaves represents the lump index for leaves in Quake 2 BSP files.
// LumpQ2LeafFaces represents the lump index for leaf faces in Quake 2 BSP files.
// LumpQ2LeafBrushes represents the lump index for leaf brushes in Quake 2 BSP files.
// LumpQ2Edges represents the lump index for edges in Quake 2 BSP files.
// LumpQ2SurfEdges represents the lump index for surface edges in Quake 2 BSP files.
// LumpQ2Models represents the lump index for models in Quake 2 BSP files.
// LumpQ2Brushes represents the lump index for brushes in Quake 2 BSP files.
// LumpQ2BrushSides represents the lump index for brush sides in Quake 2 BSP files.
// LumpQ2Pop represents the lump index for pop in Quake 2 BSP files (unused or specific purpose).
// LumpQ2Areas represents the lump index for areas in Quake 2 BSP files.
// LumpQ2AreaPortals represents the lump index for area portals in Quake 2 BSP files.
// NumQ2Lumps represents the total number of lumps in Quake 2 BSP files.
const (
	LumpQ2Entities    = 0
	LumpQ2Planes      = 1
	LumpQ2Vertexes    = 2
	LumpQ2Visibility  = 3
	LumpQ2Nodes       = 4
	LumpQ2TexInfo     = 5
	LumpQ2Faces       = 6
	LumpQ2Lighting    = 7
	LumpQ2Leaves      = 8
	LumpQ2LeafFaces   = 9
	LumpQ2LeafBrushes = 10
	LumpQ2Edges       = 11
	LumpQ2SurfEdges   = 12
	LumpQ2Models      = 13
	LumpQ2Brushes     = 14
	LumpQ2BrushSides  = 15
	LumpQ2Pop         = 16
	LumpQ2Areas       = 17
	LumpQ2AreaPortals = 18
	NumQ2Lumps        = 19
)

// Q2Header represents the header structure of a Quake 2 BSP file containing magic, version, and lump information.
type Q2Header struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ2Lumps]struct {
		Offset int32
		Length int32
	}
}

// q2Model represents a Quake 2 model with bounds, origin, BSP tree head, and face-related indices.
type q2Model struct {
	Mins      [3]float32
	Maxs      [3]float32
	Origin    [3]float32
	HeadNode  int32
	FirstFace int32
	NumFaces  int32
}

// q2Face represents a face in the Quake 2 BSP map format.
// PlaneID is the plane index in the BSP plane array associated with this face.
// Side specifies whether the face is oriented in the same or opposite direction to the plane.
// FirstEdge is the starting index in the surface edge array for this face's edges.
// NumEdges indicates the total number of edges defining this face.
// TexInfo is the index into the texture information array for texture details of the face.
// LightTypes contains light style indices for the face's dynamic lighting data.
// Lightmap is the offset in the lightmap data where this face's lightmap starts.
type q2Face struct {
	PlaneID    uint16
	Side       uint16
	FirstEdge  int32
	NumEdges   uint16
	TexInfo    uint16
	LightTypes [4]uint8
	Lightmap   int32
}

// q2TexInfo represents texture mapping information for Quake 2 BSP files.
// Vecs defines two texture vectors used for UV mapping calculations.
// Flags holds attributes for the surface such as visibility or rendering properties.
// Value specifies additional data for the texture, often used for switches or animations.
// TextureName is the name of the texture, stored as a null-terminated string.
// NextTexInfo holds the index of the next texture in the chain, or -1 if none.
type q2TexInfo struct {
	Vecs        [2][4]float32
	Flags       uint32
	Value       uint32
	TextureName [32]byte
	NextTexInfo int32
}

// q2Edge represents an edge in a Quake 2 BSP file, defined by two vertex indices V1 and V2.
type q2Edge struct {
	V1, V2 uint16
}

// q2Vertex represents a 3D point in space with X, Y, and Z coordinates as float32 values.
type q2Vertex struct {
	X, Y, Z float32
}

// Q2BSPReader reads and processes Quake 2 BSP map files, managing textures, palettes, and player metadata.
type Q2BSPReader struct {
	arc         IArchive
	header      Q2Header
	rs          io.ReadSeeker
	rsPal       io.ReadSeeker
	palette     []byte
	texManager  *Textures
	playerAngle float64
	playerPos   geometry.XYZ
}

// NewQ2BSPReader creates a new Q2BSPReader instance with the provided archive, BSP file reader, and optional palette reader.
func NewQ2BSPReader(arc IArchive, rs io.ReadSeeker, rsPal io.ReadSeeker) *Q2BSPReader {
	return &Q2BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: NewTextures(),
		rsPal:      rsPal,
	}
}

// Setup initializes the Q2BSPReader by reading the header and optionally loading the palette for WAL textures.
func (q2 *Q2BSPReader) Setup() error {
	var err error
	if _, err = q2.rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err = binary.Read(q2.rs, binary.LittleEndian, &q2.header); err != nil {
		return err
	}
	ibsp := string(q2.header.Magic[:])
	if ibsp != "IBSP" {
		return fmt.Errorf("invalid ibsp: %s", ibsp)
	}

	// In Quake 2, WAL textures often use dedicated palettes or true-color,
	// but we still load the palette if provided by the builder.
	if q2.rsPal != nil {
		q2.palette, err = NewPalette(q2.rsPal)
		if err != nil {
			fmt.Printf("Warning: palette.lmp not loaded in Q2 (WAL textures include it): %v\n", err)
		}
	}
	return nil
}

// GetArchive retrieves the IArchive instance associated with the Q2BSPReader.
func (q2 *Q2BSPReader) GetArchive() IArchive {
	return q2.arc
}

// GetPlayerInfo returns the player's current angle (in radians) and position as an XYZ coordinate structure.
func (q2 *Q2BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q2.playerAngle, q2.playerPos
}

// GetEntities extracts and parses entities from the BSP file by reading the entities lump and converting it to structured data.
func (q2 *Q2BSPReader) GetEntities() ([]*Entity, error) {
	lump := q2.header.Lumps[LumpQ2Entities]
	if _, err := q2.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q2.rs.Read(data); err != nil {
		return nil, err
	}
	text := FromNullTerminatingString(data)
	return NewEntitiesFromText(text)
}

// GetModels reads and parses the model lump to retrieve an array of BSP sub-models from the map file.
func (q2 *Q2BSPReader) GetModels() ([]*Model, error) {
	lumpModels := q2.header.Lumps[LumpQ2Models]
	if _, err := q2.rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]q2Model, numModels)
	if err := binary.Read(q2.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*Model, numModels)
	for i, m := range models {
		out[i] = &Model{
			Mins:      m.Mins,
			Maxs:      m.Maxs,
			Origin:    m.Origin,
			HeadNode:  [4]int32{m.HeadNode, -1, -1, -1}, // Q2 has a single tree, not 4 hulls like Q1
			FirstFace: m.FirstFace,
			NumFaces:  m.NumFaces,
		}
	}
	return out, nil
}

// RegisterPixels registers pixel-based texture data for a given texture name with specified dimensions and options.
func (q2 *Q2BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q2.texManager.RegisterPixels(name, width, height, indices, q2.palette, isTransparent, transIndex, invertY)
}

// RegisterPixelsRGBA registers an RGBA texture with the given name, dimensions, pixel data, and optional Y-axis inversion.
// Returns an error if the registration process fails.
func (q2 *Q2BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q2.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

// GetRawFaces extracts raw face geometry and texture mapping data for a specified model index in the BSP file.
func (q2 *Q2BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	// Read models to find face offsets
	lumpModels := q2.header.Lumps[LumpQ2Models]
	if _, err := q2.rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]q2Model, numModels)
	if err := binary.Read(q2.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	if modelIdx < 0 || modelIdx >= numModels {
		return nil, fmt.Errorf("modelIdx %d fuori dai limiti (0-%d)", modelIdx, numModels-1)
	}
	targetModel := models[modelIdx]

	// 2. Bulk read topological lumps
	lumpFaces := q2.header.Lumps[LumpQ2Faces]
	if _, err := q2.rs.Seek(int64(lumpFaces.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	faces := make([]q2Face, int(lumpFaces.Length)/20)
	if err := binary.Read(q2.rs, binary.LittleEndian, &faces); err != nil {
		return nil, err
	}

	lumpTexInfos := q2.header.Lumps[LumpQ2TexInfo]
	if _, err := q2.rs.Seek(int64(lumpTexInfos.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	texInfos := make([]q2TexInfo, int(lumpTexInfos.Length)/76)
	if err := binary.Read(q2.rs, binary.LittleEndian, &texInfos); err != nil {
		return nil, err
	}

	lumpSurfEdges := q2.header.Lumps[LumpQ2SurfEdges]
	if _, err := q2.rs.Seek(int64(lumpSurfEdges.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	surfEdges := make([]int32, int(lumpSurfEdges.Length)/4)
	if err := binary.Read(q2.rs, binary.LittleEndian, &surfEdges); err != nil {
		return nil, err
	}

	lumpEdges := q2.header.Lumps[LumpQ2Edges]
	if _, err := q2.rs.Seek(int64(lumpEdges.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	edges := make([]q2Edge, int(lumpEdges.Length)/4)
	if err := binary.Read(q2.rs, binary.LittleEndian, &edges); err != nil {
		return nil, err
	}

	lumpVerts := q2.header.Lumps[LumpQ2Vertexes]
	if _, err := q2.rs.Seek(int64(lumpVerts.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	vertexes := make([]q2Vertex, int(lumpVerts.Length)/12)
	if err := binary.Read(q2.rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, err
	}

	// Resolve indirection and generate RawFaces
	var rawFaces []*RawFace
	for i := int32(0); i < targetModel.NumFaces; i++ {
		faceIdx := targetModel.FirstFace + i
		face := faces[faceIdx]
		texInfo := texInfos[face.TexInfo]

		// Decode texture name (fixed 32-byte null-terminated C string)
		texNameBytes := make([]byte, 0, 32)
		for _, b := range texInfo.TextureName {
			if b == 0 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.ToLower(string(texNameBytes))

		// In Quake 2 flags are included in TexInfo. SURF_SKY is bitmask 0x4
		if (texInfo.Flags & 0x80) != 0 {
			continue // SURF_NODRAW
		}
		isSky := (texInfo.Flags & 0x4) != 0

		// Resolve SurfEdge -> Edge -> Vertex
		var points []geometry.XYZ
		var uvs [][2]float64
		texW, texH := float64(256), float64(256)
		if texes := q2.texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tw, th := texes[0].Size()
			texW = float64(tw)
			texH = float64(th)
			if texW == 0 {
				texW = 256
			}
			if texH == 0 {
				texH = 256
			}
		}

		for j := uint16(0); j < face.NumEdges; j++ {
			surfEdgeIdx := surfEdges[face.FirstEdge+int32(j)]

			var v q2Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].V1] // Positive direction (counter-clockwise)
			} else {
				v = vertexes[edges[-surfEdgeIdx].V2] // Reversed direction (clockwise)
			}

			// Apply standard z-up axis transformation using CreateXYZ
			points = append(points, CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z)))

			u := (float64(v.X) * float64(texInfo.Vecs[0][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[0][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[0][2])) +
				float64(texInfo.Vecs[0][3])
			vt := (float64(v.X) * float64(texInfo.Vecs[1][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[1][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[1][2])) +
				float64(texInfo.Vecs[1][3])
			uvs = append(uvs, [2]float64{u / texW, vt / texH})
		}
		rf := NewRawFace(points, uvs, texName, isSky)
		rawFaces = append(rawFaces, rf)
	}

	q2.compileTextures(rawFaces)
	return rawFaces, nil
}

// GetTextures returns a reference to the Textures manager associated with the Q2BSPReader.
func (q2 *Q2BSPReader) GetTextures() *Textures {
	return q2.texManager
}

// compileTextures processes and registers unique textures extracted from the given raw faces, excluding "sky" textures.
func (q2 *Q2BSPReader) compileTextures(faces []*RawFace) {
	uniqueTextures := make(map[string]bool)
	for _, f := range faces {
		uniqueTextures[f.TexName] = true
	}
	for texName := range uniqueTextures {
		if texName == "sky" || len(texName) == 0 {
			continue
		}
		walPath := "textures" + PakSeparator + texName + ".wal"
		walFile, walErr := q2.arc.Open(walPath)
		if walErr != nil {
			fmt.Printf("Warning: missing asset %s: %s\n \n", walPath, walErr.Error())
			continue
		}

		walTex, walErr := ParseWal(walFile)
		if walErr != nil {
			fmt.Printf("Warning: can't open asset %s: %s\n", walPath, walErr.Error())
			continue
		}

		err := q2.RegisterPixels(texName, int(walTex.Header.Width), int(walTex.Header.Height), walTex.Pixels, false, 255, false)
		if err != nil {
			fmt.Printf("Warning: can't register asset %s: %s\n", walPath, err.Error())
			continue
		}
	}
}

// GetExternalBModelFileName returns the external BModel file name associated with the given classname.
func (q2 *Q2BSPReader) GetExternalBModelFileName(classname string) string {
	return _q2DictBModel[classname]
}

// GetModelFileName retrieves the file name of the model associated with the given classname from the predefined dictionary.
func (q2 *Q2BSPReader) GetModelFileName(classname string) string {
	return _q2DictModelFilename[classname]
}

// Build processes the Quake 2 BSP data and constructs the corresponding game world structure in the provided root configuration.
func (q2 *Q2BSPReader) Build(root *config.Root) error {
	const chunkSize = float64(1024)
	mIdx := 0
	faces, rfErr := q2.GetRawFaces(mIdx)
	if rfErr != nil {
		return rfErr
	}
	entities, eErr := q2.GetEntities()
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

		if externalBSPPath := q2.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := q2.createThingBSP(externalBSPPath, pos, classname)
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
				q2.playerPos, q2.playerAngle, err = q2.createPlayerProps(angle, pos)
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
				light = q2.createLight(ent, angle, mangleStr, colorStr, pos, _q1LightStyle0, false)
			} else {
				style := _q1LightStyle0
				if sIndex, ok := ent.Properties["style"]; ok {
					if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(_q1LightStyles) {
						style = _q1LightStyles[index]
					}
				}
				// Handles light, light_fluoro, light_fluorospark
				light = q2.createLight(ent, angle, mangleStr, colorStr, pos, style, true)
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
			cThing, err := q2.createThing(pos, classname)
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

// createPlayerProps calculates the player's position and orientation in radians based on the provided angle and position.
func (q2 *Q2BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// createLight initializes a Light object based on entity properties, position, style, color, and light type (spot or ambient).
func (q2 *Q2BSPReader) createLight(entity *Entity, angle float64, mangleStr, colorStr string, pos geometry.XYZ, style []float64, isSpot bool) *config.Light {
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

// createThing creates a new game entity (Thing) based on its position and classname, returning the entity or an error.
func (q2 *Q2BSPReader) createThing(pos geometry.XYZ, classname string) (*config.Thing, error) {
	thingPath := q2.GetModelFileName(classname)
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
	arc := q2.GetArchive()
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
	if err = q2.RegisterPixels(skinName, int(md1.Header.SkinWidth), int(md1.Header.SkinHeight), skin.Data, false, 255, false); err != nil {
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

	thingCfg := q2.createConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// createThingBSP creates a Thing entity from a BSP file at the specified position and with the given classname.
// It extracts and converts BSP models, textures, and faces, constructing a Thing with appropriate geometry data.
func (q2 *Q2BSPReader) createThingBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	reader, err := NewBSPReader(q2.GetArchive(), bspPath)
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
			_ = q2.RegisterPixelsRGBA(texName, tw, th, pixels, false)
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
	thingCfg := q2.createConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing initializes and returns a Thing configuration object with provided properties and logic handlers.
func (q2 *Q2BSPReader) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.MD1, angle, mass, radius, height, speed float64) *config.Thing {
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
