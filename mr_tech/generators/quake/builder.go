package quake

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const gForce = 9.8 * 14

//MODEL IMPORTAL
//model is Z up
//model is CCW

// Builder manages the construction and handling of graphical assets, leveraging a Textures manager for texture operations.
type Builder struct {
}

// NewBuilder initializes and returns a pointer to a new Builder instance with a default Textures manager.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the game environment by loading and processing BSP data, textures, entities, and lights from a .pak file.
func (p *Builder) Setup(pakPath string, lev int) (*config.Root, error) {
	const chunkSize = float64(1024)
	if lev < 1 {
		lev = 1
	}
	levelIndex := lev - 1
	//bpsPath := "maps" + lumps.PakSeparator + "e1m" + strconv.Itoa(level) + ".bsp"

	arc, aErr := lumps.NewArchive(pakPath)
	if aErr != nil {
		return nil, aErr
	}

	if err := arc.Setup(pakPath); err != nil {
		return nil, err
	}
	maps, _ := arc.ReadDirFilter("maps", "^e.+\\.bsp")
	if len(maps) == 0 {
		maps, _ = arc.ReadDirFilter("maps", "\\.bsp$") // Fallback for Q2/Q3
	}
	if levelIndex >= len(maps) {
		return nil, fmt.Errorf("level %d out of range for available maps", levelIndex)
	}
	bpsPath := "maps" + lumps.PakSeparator + maps[levelIndex]

	reader, bErr := lumps.NewBSPReader(arc, bpsPath)
	if bErr != nil {
		return nil, bErr
	}
	if err := reader.Setup(arc); err != nil {
		return nil, err
	}
	mIdx := 0
	faces, rfErr := reader.GetRawFaces(mIdx)
	if rfErr != nil {
		return nil, rfErr
	}
	entities, eErr := reader.GetEntities()
	if eErr != nil {
		return nil, eErr
	}
	texManager := reader.GetTextures()

	var playerAngle float64
	var playerPos geometry.XYZ
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	//cal.Auto = false
	//cal.OrthoSize = 32092
	//cal.LightCamY = 8000
	//cal.ZNearRoom = 0.1
	//cal.ZFarRoom = 16000

	cal.AspectRatio = 1.0

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, texManager)

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

		if externalBSPPath := reader.GetExternalBModelFileName(classname); len(externalBSPPath) > 0 {
			cThing, err := p.createThingBSP(externalBSPPath, pos, classname, arc, reader)
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
				playerPos, playerAngle, err = p.createPlayerProps(angle, pos)
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
				light = p.createLight(ent, angle, mangleStr, colorStr, pos, lightStyle0, false)
			} else {
				style := lightStyle0
				if sIndex, ok := ent.Properties["style"]; ok {
					if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(lightStyles) {
						style = lightStyles[index]
					}
				}
				// Handles light, light_fluoro, light_fluorospark
				light = p.createLight(ent, angle, mangleStr, colorStr, pos, style, true)
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
			cThing, err := p.createThing(pos, classname, arc, reader)
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
		triangles := p.triangulateConvex3d(v.Points)

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

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 15, 40)
	playerLogic := common.NewPlayer()
	root.Player.OnCollision = playerLogic.OnCollision
	root.Player.OnImpact = playerLogic.OnImpact
	root.Player.GForce = gForce
	root.Player.JumpForce = 1000

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Factor = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1
	root.Player.Bobbing.SwayScale = 2.0
	root.Player.Bobbing.SwayOffsetX = 50
	root.Player.Bobbing.SwayOffsetY = -0.9
	root.Player.Bobbing.MaxAmplitudeX = 5.0 // MAXIMUM EXCURSION: 12 units (approx. 20% of player height)
	root.Player.Bobbing.MaxAmplitudeY = 5.5
	root.Player.Bobbing.StrideLength = 0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	root.Player.Bobbing.IdleAmpX = 0.9        // Breathing
	root.Player.Bobbing.IdleAmpY = 0.9
	root.Player.Bobbing.IdleDrift = 0.01
	root.Player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	root.Player.Bobbing.AmpLerp = 0.20
	root.Player.Bobbing.ImpactMax = 1000.0
	root.Player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	root.Player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
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
		if cr, cg, cb, valid := p.parseVector(colorStr); valid {
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
			if yaw, pitch, _, valid := p.parseVector(mangleStr); valid {
				dirX, dirY, dirZ = p.calcDirection(yaw, pitch)
			}
		} else {
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down
			} else {
				dirX, dirY, dirZ = p.calcDirection(angle, 0)
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
func (p *Builder) createThing(pos geometry.XYZ, classname string, arc lumps.IArchive, reader lumps.IBSPReader) (*config.Thing, error) {
	thingPath := reader.GetModelFileName(classname)
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
	if err = reader.RegisterPixels(skinName, int(md1.Header.SkinWidth), int(md1.Header.SkinHeight), skin.Data, false, 255, false); err != nil {
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

	thingCfg := p.createConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// createThingBSP constructs a Thing instance using external BSP model data, applying positions, textures, and materials.
func (p *Builder) createThingBSP(bspPath string, position geometry.XYZ, classname string, arc lumps.IArchive, parentReader lumps.IBSPReader) (*config.Thing, error) {
	reader, err := lumps.NewBSPReader(arc, bspPath)
	if err != nil {
		return nil, err
	}
	if err = reader.Setup(arc); err != nil {
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
			_ = parentReader.RegisterPixelsRGBA(texName, tw, th, pixels, false)
		}
		rawTriangles := p.triangulateConvex3d(bspFace.Points)
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
	thingCfg := p.createConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates a Thing configuration object with properties like position, model, animation, and physics.
func (p *Builder) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.MD1, angle, mass, radius, height, speed float64) *config.Thing {
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

// triangulateConvex3d generates a triangle fan from a convex 3D polygon defined by a list of vertices.
// It returns a slice of slices, each containing exactly three vertices representing a single triangle.
func (p *Builder) triangulateConvex3d(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil // Degenerate polygon
	}
	if pLen == 3 {
		return [][]geometry.XYZ{{pts[0], pts[1], pts[2]}}
	}
	output := make([][]geometry.XYZ, 0, pLen-2)
	// Triangle Fan anchored to pts[0]
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
		// INVERTED: from (0, 1, 2) to (0, 2, 1)
		return [][]geometry.XYZ{{pts[0], pts[2], pts[1]}}
	}

	output := make([][]geometry.XYZ, 0, pLen-2)
	for i := 1; i < pLen-1; i++ {
		// INVERTED: pts[i+1] comes BEFORE pts[i]
		output = append(output, []geometry.XYZ{pts[0], pts[i+1], pts[i]})
	}
	return output
}

// parseVector extracts 3 floats from a Quake-style string (e.g. "1.0 0.5 0.0").
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

// calcDirection converts Quake angles (yaw, pitch) into a normalized direction vector.
func (p *Builder) calcDirection(yaw, pitch float64) (float64, float64, float64) {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0
	dirX := math.Cos(pitchRad) * math.Cos(yawRad)
	dirY := math.Sin(pitchRad)
	dirZ := math.Cos(pitchRad) * math.Sin(yawRad)
	return dirX, dirY, dirZ
}
