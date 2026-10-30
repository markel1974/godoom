package q3

import (
	"fmt"
	_ "image/jpeg"
	"io"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	common2 "github.com/markel1974/godoom/mr_tech/generators/quake/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// BSPVersionQ3 represents the BSP version number used for Quake 3 map files.
const BSPVersionQ3 int = 46

const GForce = 9.8 * 50

const JumpForce = GForce * 8

// BSPReader provides functionality to parse and read Quake 3 BSP (Binary Space Partitioning) map files.
type BSPReader struct {
	arc         interfaces.IArchive
	headers     lumps.Headers3
	rs          io.ReadSeeker
	texManager  *common2.Textures
	playerAngle float64
	playerPos   geometry.XYZ
	shaders     *Shaders
}

// NewQ3BSPReader creates a new instance of BSPReader with the provided archive and ReadSeeker.
func NewQ3BSPReader(arc interfaces.IArchive, rs io.ReadSeeker) *BSPReader {
	q3 := &BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: common2.NewTextures(),
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
	q3.headers, err = lumps.NewHeader3(q3.rs)
	if string(q3.headers.Magic[:]) != "IBSP" || q3.headers.Version != 46 {
		return fmt.Errorf("formato Quake 3 non valido (Magic: %s, Versione: %d)", string(q3.headers.Magic[:]), q3.headers.Version)
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

// GetTextures retrieves the texture manager containing the loaded textures for the current Q3 BSP file.
func (q3 *BSPReader) GetTextures() *common2.Textures {
	return q3.texManager
}

// GetHeaders retrieves the header structure (lumps.Headers3) of the Quake 3 BSP file contained in the BSPReader.
func (q3 *BSPReader) GetHeaders() lumps.Headers3 {
	return q3.headers
}

func (q3 *BSPReader) GetRawFaces(modelIdx int) ([]*lumps.RawFace3, error) {
	noDraws := q3.shaders.NoDraws()
	rawFaces, err := lumps.NewRawFaces3(q3.rs, q3.headers, modelIdx, noDraws)
	if err != nil {
		return nil, err
	}
	q3.compileTextures(rawFaces)
	return rawFaces, nil
}

// compileTextures loads and registers unique textures from a list of faces, supporting JPEG and TGA formats.
func (q3 *BSPReader) compileTextures(faces []*lumps.RawFace3) {
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
			if diffMap := q3.shaders.GetDiffuseMap(texNameLC); len(diffMap) > 0 {
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
	entities, eErr := lumps.NewEntities3(q3.rs, q3.headers)
	if eErr != nil {
		return eErr
	}
	things := NewThings(q3.arc, q3.shaders, q3.texManager)
	lights := NewLights(entities)
	volumes := NewVolumes(q3.shaders)

	playersIdx := 0
	var playersBot []string
	const playersPath = "models/players/"
	if ps, _ := q3.arc.ReadDir(playersPath); len(ps) > 0 {
		for _, player := range ps {
			playersBot = append(playersBot, playersPath+player+"/")
		}
	}

	playerSpawned := false
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		baseClass := classname
		if z := strings.Split(classname, "_"); len(z) > 1 {
			baseClass, _ = z[0], z[1]
		}
		var pos geometry.XYZ
		if origin, ok := ent.GetProperty("origin"); ok {
			x, y, z, _ := lumps.ParseVector(origin)
			pos = lumps.CreateXYZ(x, y, z)
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
					q3.playerPos, q3.playerAngle = pos, 0.0
					if a, ok := ent.GetProperty("angle"); ok {
						angle, _ := lumps.ParseFloat(a)
						q3.playerAngle = angle * (math.Pi / 180.0)
					}
					playerSpawned = true
				} else {
					// We use remaining spawn points as Bot spawners
					if len(playersBot) == 0 {
						continue
					}
					playerPath := playersBot[playersIdx%len(playersBot)]
					playersIdx++
					classname = "enemy_bot"

					//thingPath := q3.GetModelFileName(enemyClass)

					var cThing *config.Thing
					var err error
					if strings.HasSuffix(playerPath, "/") {
						cThing, err = things.CreatePlayer(playerPath, pos, classname)
					} else {
						cThing, err = things.Create(playerPath, pos, classname)
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
			if cLight, err := lights.Create(ent, pos); err != nil {
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
	if sun := q3.shaders.GetSun(); sun != nil {
		root.Lights = append(root.Lights, sun)
	}
	return nil
}

// Retrieve constructs and returns a populated config.Root object based on BSPReader state or an error if the process fails.
func (q3 *BSPReader) Retrieve() (*config.Root, error) {
	texManager := q3.GetTextures()
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	//cal.Auto = false
	//cal.OrthoSize = 32092
	//cal.LightCamY = 8000
	//cal.ZNearRoom = 0.1
	//cal.ZFarRoom = 16000
	cal.AspectRatio = 1.0

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, texManager)

	if err := q3.Build(root); err != nil {
		return nil, err
	}

	crosshair := common.NewSimpleCrosshair()
	texManager.AddDirect(crosshair.GetTexture())
	root.AddUIElement(crosshair.Create())

	playerAngle, playerPos := q3.GetPlayerInfo()

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 15, 50)
	_ = common.NewPlayer(root.Player)
	root.Player.GForce = GForce
	root.Player.JumpForce = JumpForce
	root.Player.Impulse = 0.13 //mod

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Intensity = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1
	root.Player.Bobbing.SwayScale = 1.0 //mod 2.0
	root.Player.Bobbing.SwayOffsetX = 50
	root.Player.Bobbing.SwayOffsetY = -0.9
	root.Player.Bobbing.MaxAmplitudeX = 5.0 // MAXIMUM EXCURSION: 12 units (approx. 20% of player height)
	root.Player.Bobbing.MaxAmplitudeY = 5.5
	root.Player.Bobbing.StrideLength = 0.07 //mod //0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	// Breathing
	root.Player.Bobbing.IdleAmpX = 0.9  //mod 0.9
	root.Player.Bobbing.IdleAmpY = 0.9  //mod 0.9
	root.Player.Bobbing.IdleDrift = 3.0 //2.0 - 0.01

	root.Player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	root.Player.Bobbing.AmpLerp = 0.20
	root.Player.Bobbing.ImpactMax = 1000.0
	root.Player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	root.Player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
	root.Player.Bobbing.SpringDamping = 0.80
	root.Player.Bobbing.TiltAmp = 0.05

	return root, nil
}
