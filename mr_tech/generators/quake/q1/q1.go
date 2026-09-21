package q1

import (
	"encoding/binary"
	"fmt"
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
		isSky := (texInfo.Flags & 4) != 0
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
			texName = q1.mipTextures[texInfo.MipTex].Name
		}
		var points []geometry.XYZ
		var uvs [][2]float64

		// Prepare texture width/height for normalization
		texW, texH := float64(256), float64(256)
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
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
func (q1 *Q1BSPReader) GetModelFileName(classname string) string {
	return _q1DictModelFilename[classname]
}

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

		if externalBSPPath := q1.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := q1.createThingBSP(externalBSPPath, pos, classname)
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
			mangleStr, _ := ent.Properties["mangle"]
			colorStr, _ := ent.Properties["_color"]
			var light *config.Light = nil
			if len(subClass) == 0 {
				light = q1.createLight(ent, angle, mangleStr, colorStr, pos, _q1LightStyle0, false)
			} else {
				style := _q1LightStyle0
				if sIndex, ok := ent.Properties["style"]; ok {
					if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(_q1LightStyles) {
						style = _q1LightStyles[index]
					}
				}
				// Handles light, light_fluoro, light_fluorospark
				light = q1.createLight(ent, angle, mangleStr, colorStr, pos, style, true)
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
			cThing, err := q1.createThing(pos, classname)
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
		triangles := lumps.TriangulateConvex3d(v.Points)

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
func (q1 *Q1BSPReader) createPlayerProps(angle float64, pos geometry.XYZ) (geometry.XYZ, float64, error) {
	playerAngle := angle * (math.Pi / 180.0)
	return pos, playerAngle, nil
}

// createLight creates a new Light instance based on entity properties and position, returning an error if invalid or missing data.
func (q1 *Q1BSPReader) createLight(entity *lumps.Entity, angle float64, mangleStr, colorStr string, pos geometry.XYZ, style []float64, isSpot bool) *config.Light {
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
		if cr, cg, cb, valid := lumps.ParseVector(colorStr); valid {
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
			if yaw, pitch, _, valid := lumps.ParseVector(mangleStr); valid {
				dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)
			}
		} else {
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down
			} else {
				dirX, dirY, dirZ = lumps.CalcDirection(angle, 0)
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
func (q1 *Q1BSPReader) createThing(pos geometry.XYZ, classname string) (*config.Thing, error) {
	thingPath := q1.GetModelFileName(classname)
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
	arc := q1.GetArchive()
	rsMd1, err := arc.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}
	md1 := lumps.NewMD1Resource()
	if err = md1.Parse(rsMd1); err != nil {
		return nil, fmt.Errorf("can't load MDL %s: %s\n", classname, err.Error())
	}
	if skinTargetIndex >= len(md1.Skins) {
		return nil, fmt.Errorf("no skin found for %s", classname)
	}
	skin := md1.Skins[skinTargetIndex]
	skinName := fmt.Sprintf("%s_skin_%d", classname, skinTargetIndex)
	if err = q1.RegisterPixels(skinName, int(md1.Header.SkinWidth), int(md1.Header.SkinHeight), skin.Data, false, 255, false); err != nil {
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
				cTri.Vertices[v] = config.MD1Vertex{Pos: lumps.CreateXYZ(f[vx][0], f[vx][1], f[vx][2]), U: nU, V: nV}
			}
			triangles[tIdx] = cTri
		}
		cFrame := config.NewMD1Frame(triangles)
		cModel.Frames[idx] = cFrame
	}

	thingCfg := q1.createConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// createThingBSP constructs a Thing instance using external BSP model data, applying positions, textures, and materials.
func (q1 *Q1BSPReader) createThingBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	arc := q1.GetArchive()
	rs, err := arc.Open(bspPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", bspPath, err.Error())
	}
	reader := NewQ1BSPReader(arc, rs)
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
			_ = q1.RegisterPixelsRGBA(texName, tw, th, pixels, false)
		}
		rawTriangles := lumps.TriangulateConvex3d(bspFace.Points)
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
	thingCfg := q1.createConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates a Thing configuration object with properties like position, model, animation, and physics.
func (q1 *Q1BSPReader) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.MD1, angle, mass, radius, height, speed float64) *config.Thing {
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
