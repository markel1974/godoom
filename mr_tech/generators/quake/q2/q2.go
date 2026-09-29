package q2

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const BSPVersionQ2 int = 38

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

// q2Header represents the header structure of a Quake 2 BSP file containing magic, version, and lump information.
type q2Header struct {
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

// BSPReader reads and processes Quake 2 BSP map files, managing textures, palettes, and player metadata.
type BSPReader struct {
	arc         interfaces.IArchive
	header      q2Header
	rs          io.ReadSeeker
	palette     [256]color.RGBA
	texManager  *lumps.Textures
	playerAngle float64
	playerPos   geometry.XYZ
}

// NewQ2BSPReader creates a new BSPReader instance with the provided reader.go, BSP file reader, and optional palette reader.
func NewQ2BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *BSPReader {
	return &BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: lumps.NewTextures(),
	}
}

// Setup initializes the BSPReader by reading the header and optionally loading the palette for WAL textures.
func (q2 *BSPReader) Setup() error {
	if _, err := q2.rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(q2.rs, binary.LittleEndian, &q2.header); err != nil {
		return err
	}
	ibsp := string(q2.header.Magic[:])
	if ibsp != "IBSP" {
		return fmt.Errorf("invalid ibsp: %s", ibsp)
	}
	const palettePath = "pics" + lumps.PakSeparator + "colormap.pcx"
	rsPal, err := q2.arc.Open(palettePath)
	if err != nil {
		return err
	}
	palette := lumps.NewPalette(0.8)
	q2.palette, err = palette.ParseFromPCX(rsPal)
	if err != nil {
		fmt.Printf("Warning: %s not loaded: %v\n", palettePath, err)
	}
	return nil
}

// GetArchive retrieves the IArchive instance associated with the BSPReader.
func (q2 *BSPReader) GetArchive() interfaces.IArchive {
	return q2.arc
}

// GetTextures returns a reference to the Textures manager associated with the BSPReader.
func (q2 *BSPReader) GetTextures() *lumps.Textures {
	return q2.texManager
}

// GetPlayerInfo returns the player's current angle (in radians) and position as an XYZ coordinate structure.
func (q2 *BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q2.playerAngle, q2.playerPos
}

// GetEntities extracts and parses entities from the BSP file by reading the entities lump and converting it to structured data.
func (q2 *BSPReader) GetEntities() ([]*lumps.Entity, error) {
	lump := q2.header.Lumps[LumpQ2Entities]
	if _, err := q2.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q2.rs.Read(data); err != nil {
		return nil, err
	}
	text := lumps.FromNullTerminatingString(data)
	return lumps.NewEntitiesFromText(text)
}

// GetModels reads and parses the model lump to retrieve an array of BSP sub-models from the map file.
func (q2 *BSPReader) GetModels() ([]*lumps.Model, error) {
	lumpModels := q2.header.Lumps[LumpQ2Models]
	if _, err := q2.rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]q2Model, numModels)
	if err := binary.Read(q2.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*lumps.Model, numModels)
	for i, m := range models {
		out[i] = &lumps.Model{
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

// GetExternalBModelFileName returns the external BModel file name associated with the given classname.
func (q2 *BSPReader) GetExternalBModelFileName(classname string) string {
	return _q2DictBModel[classname]
}

// GetModelFileName retrieves the file name of the model associated with the given classname from the predefined dictionary.
func (q2 *BSPReader) GetModelFileName(classname string) string {
	return _q2DictModelFilename[classname]
}

// Build processes the Quake 2 BSP data and constructs the corresponding game world structure in the provided root configuration.
func (q2 *BSPReader) Build(root *config.Root) error {
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

	lights := NewLights(entities)
	volumes := NewVolumes(mIdx, chunkSize)
	things := NewThings(q2.arc, q2.texManager, q2.palette)

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

		if externalBSPPath := q2.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := things.CreateThingBSP(externalBSPPath, pos, classname)
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
			if classname == "info_player_start" || classname == "info_player_deathmatch" {
				q2.playerPos = pos
				q2.playerAngle = angle * (math.Pi / 180.0)
			} else {
				// Invisible markers: teleports, deathmatch spawn points, patrol nodes.
				// TODO: Save them in a gameplay waypoint/spawnpoint list.
			}
		case "light":
			if light := lights.CreateLight(ent, pos, subClass); light != nil {
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
			thingPath := q2.GetModelFileName(classname)
			cThing, err := things.CreateThing(thingPath, pos, classname)
			if err != nil {
				fmt.Printf("Warning: %s\n", err.Error())
				continue
			}
			root.Things = append(root.Things, cThing)
		}
	}
	root.Volumes = volumes.Create(faces)

	return nil
}

// GetRawFaces extracts raw face geometry and texture mapping data for a specified model index in the BSP file.
func (q2 *BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace, error) {
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
	var rawFaces []*lumps.RawFace
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

		texW, texH, err := q2.registerTexture(texName)
		if err != nil {
			fmt.Printf("Warning: can't register asset %s: %s\n", texName, err.Error())
			continue
		}
		if texW == 0 {
			texW = 256
		}
		if texH == 0 {
			texH = 256
		}

		// Resolve SurfEdge -> Edge -> Vertex
		var points []geometry.XYZ
		var uvs [][2]float64

		for j := uint16(0); j < face.NumEdges; j++ {
			surfEdgeIdx := surfEdges[face.FirstEdge+int32(j)]

			var v q2Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].V1] // Positive direction (counter-clockwise)
			} else {
				v = vertexes[edges[-surfEdgeIdx].V2] // Reversed direction (clockwise)
			}

			// Apply standard z-up axis transformation using CreateXYZ
			points = append(points, lumps.CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z)))

			u := (float64(v.X) * float64(texInfo.Vecs[0][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[0][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[0][2])) +
				float64(texInfo.Vecs[0][3])
			vt := (float64(v.X) * float64(texInfo.Vecs[1][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[1][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[1][2])) +
				float64(texInfo.Vecs[1][3])
			uvs = append(uvs, [2]float64{u / float64(texW), vt / float64(texH)})
		}
		rf := lumps.NewRawFace(points, uvs, texName, isSky)
		rawFaces = append(rawFaces, rf)
	}

	return rawFaces, nil
}

// registerTexture loads a .wal texture, parses its data, and registers it with the texture manager using a palette.
// Returns the width, height, and potential error of the operation.
func (q2 *BSPReader) registerTexture(texName string) (int, int, error) {
	walPath := "textures" + lumps.PakSeparator + texName + ".wal"
	walFile, err := q2.arc.Open(walPath)
	if err != nil {
		return 0, 0, err
	}
	walTex, err := lumps.ParseWal(walFile)
	if err != nil {
		return 0, 0, err
	}
	err = q2.texManager.RegisterPixelsPalette(texName, int(walTex.Header.Width), int(walTex.Header.Height), walTex.Pixels, q2.palette, false, 255, false)
	if err != nil {
		return 0, 0, err
	}
	return int(walTex.Header.Width), int(walTex.Header.Height), nil
}
