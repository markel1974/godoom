package q1

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

const (
	BSPVersionQ1 int = 29
)

// BSPReader reads and processes Quake 1 BSP files by managing lumps, textures, and geometry data.
type BSPReader struct {
	arc         interfaces.IArchive
	rs          io.ReadSeeker
	rsPal       io.ReadSeeker
	infos       []*lumps.LumpInfo
	palette     [256]color.RGBA
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

// NewQ1BSPReader initializes and returns a pointer to a new BSPReader instance using the provided io.ReadSeeker streams.
func NewQ1BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *BSPReader {
	return &BSPReader{
		arc:        arc,
		rs:         rs,
		rsPal:      nil,
		texManager: lumps.NewTextures(),
	}
}

// Setup initializes the BSPReader by loading BSP data, textures, palettes, and related metadata from the provided reader.
func (q1 *BSPReader) Setup() error {
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
	palette := lumps.NewPalette(0.8)
	if q1.palette, err = palette.Parse(q1.rsPal); err != nil {
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
		if mt != nil && len(mt.Name) > 0 {
			if err = q1.texManager.RegisterPixelsPalette(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], q1.palette, false, 255, false); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}
	return nil
}

// GetArchive returns the IArchive instance associated with the BSPReader, used for file access and data retrieval.
func (q1 *BSPReader) GetArchive() interfaces.IArchive {
	return q1.arc
}

// GetPlayerInfo retrieves the player's angle in radians and position in 3D space as geometry.XYZ coordinates.
func (q1 *BSPReader) GetPlayerInfo() (float64, geometry.XYZ) {
	return q1.playerAngle, q1.playerPos
}

// GetEntities retrieves all entities from the BSP file and returns them as a slice of Entity pointers or an error.
func (q1 *BSPReader) GetEntities() ([]*lumps.Entity, error) {
	return lumps.NewEntities(q1.rs, q1.infos[lumps.LumpEntities])
}

// GetModels retrieves the BSP models from the lump data and returns a slice of models or an error if reading fails.
func (q1 *BSPReader) GetModels() ([]*lumps.Model, error) {
	return lumps.NewModels(q1.rs, q1.infos[lumps.LumpModels])
}

// GetModelFileName retrieves the file name of a BSP model corresponding to the given classname.
func (q1 *BSPReader) GetModelFileName(classname string) string {
	return GetModelFileName(classname)
}

// GetTextures returns the texture manager instance containing textures defined in the BSP file.
func (q1 *BSPReader) GetTextures() *lumps.Textures {
	return q1.texManager
}

// GetExternalBModelFileName returns the file name of the external BSP model associated with the given classname.
func (q1 *BSPReader) GetExternalBModelFileName(classname string) string {
	return GetExternalBModelFileName(classname)
}

// Build processes the BSP file's data, including faces, entities, lights, and models, and populates the provided Root structure.
func (q1 *BSPReader) Build(root *config.Root) error {
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
	volumes := NewVolumes(mIdx, chunkSize)

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
			modelIdx, _ := strconv.Atoi(modelProp[1:])
			rawFaces, err := q1.GetRawFaces(modelIdx)
			if err != nil {
				fmt.Printf("warning on internal bmodel %s (index %d): %v\n", classname, modelIdx, err)
				continue
			}
			cThing, err := things.CreateInternalBModel(rawFaces, pos, classname)
			if err != nil {
				fmt.Printf("warning on internal bmodel %s (index %d): %v\n", classname, modelIdx, err)
				continue
			}
			root.Things = append(root.Things, cThing)
			continue
		}

		if externalBSPPath := GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := things.CreateThingBSP(externalBSPPath, pos, classname)
			if err != nil {
				fmt.Printf("warning on external bmodel %s: %v\n", classname, err)
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

	root.Volumes = volumes.Generate(faces)

	return nil
}

// createPlayerProps extracts player position and angle from an entity and computes the angle in radians.
func (q1 *BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// getVertexes retrieves the vertex data from the BSP file using the lump information and returns a slice of vertex pointers.
func (q1 *BSPReader) getVertexes() ([]*lumps.Vertex, error) {
	return lumps.NewVertexes(q1.rs, q1.infos[lumps.LumpVertexes])
}

// getEdges loads and returns the list of edges from the edges lump data or an error if the operation fails.
func (q1 *BSPReader) getEdges() ([]*lumps.Edge, error) {
	return lumps.NewEdges(q1.rs, q1.infos[lumps.LumpEdges])
}

// getSurfEdges retrieves an array of surface edges, representing directed edge indices used for face definitions.
// Returns a slice of int32 values and an error if reading fails.
func (q1 *BSPReader) getSurfEdges() ([]int32, error) {
	return lumps.NewSurfEdges(q1.rs, q1.infos[lumps.LumpSurfEdges])
}

// getFaces retrieves all Face structures from the BSP file using lump metadata and returns them or an error if encountered.
func (q1 *BSPReader) getFaces() ([]*lumps.Face, error) {
	return lumps.NewFace(q1.rs, q1.infos[lumps.LumpFaces])
}

// getTexInfos retrieves texture mapping information from the level data and returns a slice of TexInfo and an error.
func (q1 *BSPReader) getTexInfos() ([]*lumps.TexInfo, error) {
	return lumps.NewTexInfos(q1.rs, q1.infos[lumps.LumpTexInfos])
}

// getMipTextures reads and decodes all mipmap textures from the lump data in the BSP file. Returns an error on failure.
func (q1 *BSPReader) getMipTextures() ([]*lumps.MipTexture, error) {
	return lumps.NewMipTextures(q1.rs, q1.infos[lumps.LumpTextures])
}

// GetRawFaces extracts raw face data for a specified model index, including geometry, texture names, and UV coordinates.
func (q1 *BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace, error) {
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
		info := q1.texInfos[bspFace.TexInfo]
		texName := "default"
		if info.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[info.MipTex] != nil {
			texName = q1.mipTextures[info.MipTex].Name
		}
		isSky := strings.HasPrefix(strings.ToLower(texName), "sky")
		var points []geometry.XYZ
		var uvs [][2]float64
		// Prepare texture width/height for normalization
		texW, texH := float64(256), float64(256)
		if info.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[info.MipTex] != nil {
			texName = q1.mipTextures[info.MipTex].Name
			texW = float64(q1.mipTextures[info.MipTex].Width)
			texH = float64(q1.mipTextures[info.MipTex].Height)
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
			u := (float64(v.X) * float64(info.Vecs[0][0])) + (float64(v.Y) * float64(info.Vecs[0][1])) + (float64(v.Z) * float64(info.Vecs[0][2])) + float64(info.Vecs[0][3])
			vt := (float64(v.X) * float64(info.Vecs[1][0])) + (float64(v.Y) * float64(info.Vecs[1][1])) + (float64(v.Z) * float64(info.Vecs[1][2])) + float64(info.Vecs[1][3])
			uvs = append(uvs, [2]float64{u / texW, vt / texH})
		}
		rf := lumps.NewRawFace(points, uvs, texName, isSky)
		rawFaces = append(rawFaces, rf)
	}

	return rawFaces, nil
}
