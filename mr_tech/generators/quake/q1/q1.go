package q1

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	BSPVersionQ1 int = 29
)

// Q1BSPReader reads and processes Quake 1 BSP files by managing lumps, textures, and geometry data.
type Q1BSPReader struct {
	arc         interfaces.IArchive
	rs          io.ReadSeeker
	rsPal       io.ReadSeeker
	infos       []*lumps.LumpInfo
	palette     []byte
	mipTextures []*lumps.MipTexture
	faces       []*lumps.Face
	surfEdges   []int32
	edges       []*lumps.Edge
	vertexes    []*lumps.Vertex
	texInfos    []*lumps.TexInfo
	texManager  *lumps.Textures
	playerAngle float64
	playerPos   geometry.XYZ
}

// NewQ1BSPReader initializes and returns a pointer to a new Q1BSPReader instance using the provided io.ReadSeeker streams.
func NewQ1BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *Q1BSPReader {
	return &Q1BSPReader{
		arc:        arc,
		rs:         rs,
		rsPal:      nil,
		texManager: lumps.NewTextures(),
	}
}

// Setup initializes the Q1BSPReader by loading BSP data, textures, palettes, and related metadata from the provided reader.
func (q1 *Q1BSPReader) Setup() error {
	q1.rsPal, _ = q1.arc.Open("gfx" + lumps.PakSeparator + "palette.lmp")

	if _, err := q1.rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rewind stream: %w", err)
	}
	var version int32
	if err := binary.Read(q1.rs, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != int32(BSPVersionQ1) {
		return fmt.Errorf("unsupported Q1 BSP version: %d", version)
	}
	if _, err := q1.rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rewind stream: %w", err)
	}
	var err error
	if q1.infos, err = lumps.NewLumpInfos(q1.rs); err != nil {
		return err
	}
	if q1.palette, err = lumps.NewPalette(q1.rsPal); err != nil {
		return err
	}
	if q1.faces, err = q1.getFaces(); err != nil {
		return err
	}
	if q1.surfEdges, err = q1.getSurfEdges(); err != nil {
		return err
	}
	if q1.edges, err = q1.getEdges(); err != nil {
		return err
	}
	if q1.vertexes, err = q1.getVertexes(); err != nil {
		return err
	}
	if q1.texInfos, err = q1.getTexInfos(); err != nil {
		return err
	}
	if q1.mipTextures, err = q1.getMipTextures(); err != nil {
		return err
	}
	for _, mt := range q1.mipTextures {
		if mt != nil && mt.Name != "" {
			if err = q1.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], false, 255, false); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}
	return nil
}

// GetArchive returns the IArchive instance associated with the Q1BSPReader, used for file access and data retrieval.
func (q1 *Q1BSPReader) GetArchive() interfaces.IArchive {
	return q1.arc
}

// GetPlayerInfo retrieves the player's angle in radians and position in 3D space as geometry.XYZ coordinates.
func (q1 *Q1BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q1.playerAngle, q1.playerPos
}

// GetEntities retrieves all entities from the BSP file and returns them as a slice of Entity pointers or an error.
func (q1 *Q1BSPReader) GetEntities() ([]*lumps.Entity, error) {
	return lumps.NewEntities(q1.rs, q1.infos[lumps.LumpEntities])
}

// GetModels retrieves the BSP models from the lump data and returns a slice of models or an error if reading fails.
func (q1 *Q1BSPReader) GetModels() ([]*lumps.Model, error) {
	return lumps.NewModels(q1.rs, q1.infos[lumps.LumpModels])
}

// GetModelFileName retrieves the file name of a BSP model corresponding to the given classname.
func (q1 *Q1BSPReader) GetModelFileName(classname string) string {
	return GetModelFileName(classname)
}

// GetTextures returns the texture manager instance containing textures defined in the BSP file.
func (q1 *Q1BSPReader) GetTextures() *lumps.Textures {
	return q1.texManager
}

// RegisterPixels registers a texture by name with specified dimensions, pixel data, palette, transparency, and alignment.
func (q1 *Q1BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q1.texManager.RegisterPixels(name, width, height, indices, q1.palette, isTransparent, transIndex, invertY)
}

// RegisterPixelsRGBA registers a texture using raw RGBA pixel data with optional Y-axis inversion.
func (q1 *Q1BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q1.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

// GetRawFaces extracts raw face data for a specified model index, including geometry, texture names, and UV coordinates.
func (q1 *Q1BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace, error) {
	models, _ := q1.GetModels()
	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("invalid model index")
	}
	model := models[modelIdx]

	var rawFaces []*lumps.RawFace

	// Itera solo sulle facce di questo modello (0 = World, 1+ = BModels)
	for i := int32(0); i < model.NumFaces; i++ {
		faceIdx := model.FirstFace + i
		bspFace := q1.faces[faceIdx]
		texInfo := q1.texInfos[bspFace.TexInfo]
		texName := "default"
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
			texName = q1.mipTextures[texInfo.MipTex].Name
		}
		isSky := strings.HasPrefix(strings.ToLower(texName), "sky")
		var points []geometry.XYZ
		var uvs [][2]float64

		// Prepare texture width/height for normalization
		texW, texH := float64(256), float64(256)
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
			texName = q1.mipTextures[texInfo.MipTex].Name

			texW = float64(q1.mipTextures[texInfo.MipTex].Width)
			texH = float64(q1.mipTextures[texInfo.MipTex].Height)
			if texW == 0 {
				texW = 256
			}
			if texH == 0 {
				texH = 256
			}
		}

		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := q1.surfEdges[bspFace.FirstEdge+int32(j)]
			var v *lumps.Vertex
			if surfEdgeIdx >= 0 {
				v = q1.vertexes[q1.edges[surfEdgeIdx].Vertex0]
			} else {
				v = q1.vertexes[q1.edges[-surfEdgeIdx].Vertex1]
			}
			pos := lumps.CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z))
			points = append(points, pos)

			// Compute UVs using Quake 1 vector projection
			// Note: Q1 raw vertex coords are used for projection
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
		rf := lumps.NewRawFace(points, uvs, texName, isSky)
		rawFaces = append(rawFaces, rf)
	}

	return rawFaces, nil
}

// GetExternalBModelFileName returns the file name of the external BSP model associated with the given classname.
func (q1 *Q1BSPReader) GetExternalBModelFileName(classname string) string {
	return _q1DictBModel[classname]
}

// GetModelFileName returns the file name of a model corresponding to the given classname from the predefined model dictionary.
//func (q1 *Q1BSPReader) GetModelFileName(classname string) string {
//	return _q1DictModelFilename[classname]
//}

func (q1 *Q1BSPReader) Build(root *config.Root) error {
	const chunkSize = float64(1024)
	mIdx := 0
	faces, rfErr := q1.GetRawFaces(mIdx)
	if rfErr != nil {
		return rfErr
	}
	entities, eErr := q1.GetEntities()
	if eErr != nil {
		return eErr
	}

	things := NewThings(q1.arc, q1.texManager, q1.palette)
	lights := NewLights(entities)

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

		if modelProp := ent.Properties["model"]; strings.HasPrefix(modelProp, "*") {
			//TODO IMPLEMENT
			continue
			modelIdx, _ := strconv.Atoi(modelProp[1:])
			rawFaces, err := q1.GetRawFaces(modelIdx)
			if err != nil {
				fmt.Printf("warning on internal bmodel %s (index %d): %v", classname, modelIdx, err)
				continue
			}
			cThing, err := things.CreateInternalBModel(rawFaces, pos, classname)
			if err != nil {
				fmt.Printf("warning on internal bmodel %s (index %d): %v", classname, modelIdx, err)
				continue
			}
			root.Things = append(root.Things, cThing)
			continue
		}

		if externalBSPPath := q1.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
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
			if classname == "info_player_start" {
				var err error
				q1.playerPos, q1.playerAngle, err = q1.createPlayerProps(angle, pos)
				if err != nil {
					fmt.Printf("Warning: %s\n", err.Error())
				}
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
			cThing, err := things.CreateThing(pos, classname)
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
		velX, velY := 0.0, 0.0
		if v.IsSky {
			animKind = config.MaterialKindSky
			velX, velY = 0.05, 0.05
		} else if len(v.TexName) > 0 && v.TexName[0] == '*' {
			animKind = config.MaterialKindLiquid
		}
		material := config.NewConfigMaterial([]string{v.TexName}, animKind, 1.0, 1.0, velX, velY)
		triangles := lumps.TriangulateConvex3d(v.Points)

		//isLiquid := len(v.TexName) > 0 && v.TexName[0] == '*'
		for _, rawTri := range triangles {
			var rawTriUvs [][2]float64
			if len(v.UVs) > 0 {
				rawTriUvs = make([][2]float64, 3)
				for k := 0; k < 3; k++ {
					pos := rawTri[k]
					for idx, pt := range v.Points {
						if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
							if len(v.UVs) > idx {
								rawTriUvs[k] = v.UVs[idx]
							}
							break
						}
					}
				}

				// Find the triangle centroid
				cx := (rawTri[0].X + rawTri[1].X + rawTri[2].X) / 3.0
				cy := (rawTri[0].Y + rawTri[1].Y + rawTri[2].Y) / 3.0
				cz := (rawTri[0].Z + rawTri[1].Z + rawTri[2].Z) / 3.0

				// Calculate the spatial hashing key (grid coordinates)
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

				volume.Faces = append(volume.Faces, config.NewConfigFace([]geometry.XYZ{rawTri[0], rawTri[1], rawTri[2]}, rawTriUvs, material, v.TexName))
			}
		}
	}
	return nil
}

// createPlayerProps extracts player position and angle from an entity and computes the angle in radians.
func (q1 *Q1BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// getVertexes retrieves the vertex data from the BSP file using the lump information and returns a slice of vertex pointers.
func (q1 *Q1BSPReader) getVertexes() ([]*lumps.Vertex, error) {
	return lumps.NewVertexes(q1.rs, q1.infos[lumps.LumpVertexes])
}

// getEdges loads and returns the list of edges from the edges lump data or an error if the operation fails.
func (q1 *Q1BSPReader) getEdges() ([]*lumps.Edge, error) {
	return lumps.NewEdges(q1.rs, q1.infos[lumps.LumpEdges])
}

// getSurfEdges retrieves an array of surface edges, representing directed edge indices used for face definitions.
// Returns a slice of int32 values and an error if reading fails.
func (q1 *Q1BSPReader) getSurfEdges() ([]int32, error) {
	return lumps.NewSurfEdges(q1.rs, q1.infos[lumps.LumpSurfEdges])
}

// getFaces retrieves all Face structures from the BSP file using lump metadata and returns them or an error if encountered.
func (q1 *Q1BSPReader) getFaces() ([]*lumps.Face, error) {
	return lumps.NewFace(q1.rs, q1.infos[lumps.LumpFaces])
}

// getTexInfos retrieves texture mapping information from the level data and returns a slice of TexInfo and an error.
func (q1 *Q1BSPReader) getTexInfos() ([]*lumps.TexInfo, error) {
	return lumps.NewTexInfos(q1.rs, q1.infos[lumps.LumpTexInfos])
}

// getMipTextures reads and decodes all mipmap textures from the lump data in the BSP file. Returns an error on failure.
func (q1 *Q1BSPReader) getMipTextures() ([]*lumps.MipTexture, error) {
	return lumps.NewMipTextures(q1.rs, q1.infos[lumps.LumpTextures])
}
