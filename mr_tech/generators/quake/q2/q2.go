package q2

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const BSPVersionQ2 int = 38

// BSPReader reads and processes Quake 2 BSP map files, managing textures, palettes, and player metadata.
type BSPReader struct {
	arc         interfaces.IArchive
	header      lumps.Headers2
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

// GetHeaders retrieves the BSP file headers, including magic identifier, version, and lump metadata.
func (q2 *BSPReader) GetHeaders() lumps.Headers2 {
	return q2.header
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

	entities, eErr := lumps.NewEntities2(q2.rs, q2.header)
	if eErr != nil {
		return eErr
	}

	lights := NewLights(entities)
	volumes := NewVolumes(mIdx, chunkSize)
	things := NewThings(q2.arc, q2.texManager, q2.palette)

	for _, ent := range entities {
		classname, _ := ent.GetProperty("classname")
		parts := strings.Split(classname, "_")
		nameSpace := parts[0]
		//class := ""
		//subType := ""
		//if len(parts) > 1 {
		//	class = parts[1]
		//}
		//if len(parts) > 2 {
		//	subType = parts[2]
		//}

		var pos geometry.XYZ
		if origin, ok := ent.GetProperty("origin"); ok {
			x, y, z, _ := lumps.ParseVector(origin)
			pos = lumps.CreateXYZ(x, y, z)
		}

		// TODO: Currently we are ignoring sub-models (*1, *2, etc.) like func_door or func_plat.
		// Before focusing on "accessories", let's ensure that worldspawn (the base map)
		// is rendered correctly. When ready, we will remove this continue
		// and instantiate bmodels using GetModels() from IBSPReader.
		if modelProp, _ := ent.GetProperty("model"); strings.HasPrefix(modelProp, "*") {
			continue
		}

		if externalBSPPath := q2.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := things.CreateThingBSP(externalBSPPath, pos, classname)
			if err != nil {
				fmt.Printf("warning on external bmodel %s: %v\n", classname, err)
				continue
			}
			root.Things = append(root.Things, cThing)
			continue
		}

		switch nameSpace {
		case "worldspawn":
			// Ignored: it is the base map, geometry is already handled by worldModel
		case "info":
			if classname == "info_player_start" || classname == "info_player_deathmatch" {
				q2.playerPos = pos
				q2.playerAngle = 0.0
				if a, ok := ent.GetProperty("angle"); ok {
					angle, _ := lumps.ParseFloat(a)
					q2.playerAngle = angle * (math.Pi / 180.0)
				}
			} else {
				// Invisible markers: teleports, deathmatch spawn points, patrol nodes.
				// TODO: Save them in a gameplay waypoint/spawnpoint list.
			}
		case "light":
			if light := lights.CreateLight(ent, pos); light != nil {
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
			//TODO
		case "target":
			//TODO
		case "point":
			//TODO
		case "misc":
			//TODO
		case "turret":
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
func (q2 *BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace2, error) {
	// Read models to find face offsets
	lumpModels := q2.header.Lumps[lumps.LumpModels2]
	if _, err := q2.rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]lumps.Model2, numModels)
	if err := binary.Read(q2.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	if modelIdx < 0 || modelIdx >= numModels {
		return nil, fmt.Errorf("modelIdx %d fuori dai limiti (0-%d)", modelIdx, numModels-1)
	}
	targetModel := models[modelIdx]

	// 2. Bulk read topological lumps
	lumpFaces := q2.header.Lumps[lumps.LumpFaces2]
	if _, err := q2.rs.Seek(int64(lumpFaces.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	faces := make([]lumps.Face2, int(lumpFaces.Length)/20)
	if err := binary.Read(q2.rs, binary.LittleEndian, &faces); err != nil {
		return nil, err
	}

	lumpTexInfos := q2.header.Lumps[lumps.LumpTexInfo2]
	if _, err := q2.rs.Seek(int64(lumpTexInfos.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	texInfos := make([]lumps.TexInfo2, int(lumpTexInfos.Length)/76)
	if err := binary.Read(q2.rs, binary.LittleEndian, &texInfos); err != nil {
		return nil, err
	}

	lumpSurfEdges := q2.header.Lumps[lumps.LumpSurfEdges2]
	if _, err := q2.rs.Seek(int64(lumpSurfEdges.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	surfEdges := make([]int32, int(lumpSurfEdges.Length)/4)
	if err := binary.Read(q2.rs, binary.LittleEndian, &surfEdges); err != nil {
		return nil, err
	}

	lumpEdges := q2.header.Lumps[lumps.LumpEdges2]
	if _, err := q2.rs.Seek(int64(lumpEdges.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	edges := make([]lumps.Edge2, int(lumpEdges.Length)/4)
	if err := binary.Read(q2.rs, binary.LittleEndian, &edges); err != nil {
		return nil, err
	}

	lumpVerts := q2.header.Lumps[lumps.LumpVertexes2]
	if _, err := q2.rs.Seek(int64(lumpVerts.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	vertexes := make([]lumps.Vertex2, int(lumpVerts.Length)/12)
	if err := binary.Read(q2.rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, err
	}

	// Resolve indirection and generate RawFaces
	var rawFaces []*lumps.RawFace2
	for i := int32(0); i < targetModel.NumFaces; i++ {
		faceIdx := targetModel.FirstFace + i
		face := faces[faceIdx]
		texInfo := texInfos[face.TexInfo]
		texName := strings.ToLower(lumps.FromNullTerminatingString(texInfo.TextureName[:]))
		if texInfo.NoDraw() {
			continue
		}
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
			var v lumps.Vertex2
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
		rf := lumps.NewRawFace2(points, uvs, texName, texInfo)
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
