package doom3

import (
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/generators/quake/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const GForce = 9.8 * 50

const JumpForce = GForce * 8

// Builder manages the construction and handling of graphical assets for Doom 3 (id Tech 4).
type Builder struct {
}

// NewBuilder initializes and returns a pointer to a new Builder instance for Doom 3.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the game environment by loading and processing Doom 3 maps and assets from a .pk4 file.
func (p *Builder) Setup(res common.IResources, pk4Path string, lev int) (*config.Root, error) {
	if lev < 1 {
		lev = 1
	}

	arc := proc.NewPk4(res)

	if err := arc.Setup(pk4Path); err != nil {
		return nil, fmt.Errorf("failed to open pk4 archive: %w", err)
	}

	// Doom 3 maps use the .map and .proc extensions.
	// The .proc file contains the precalculated geometry, portals and shadow hulls.
	maps, _ := arc.ReadDirFilter("maps", "\\.proc$")

	if len(maps) == 0 {
		return nil, fmt.Errorf("no .proc maps found in archive")
	}

	levelIndex := lev - 1
	if levelIndex >= len(maps) {
		return nil, fmt.Errorf("level %d out of range for available maps", levelIndex)
	}

	procPath := "maps" + proc.Pak4Separator + maps[levelIndex]
	fmt.Printf("Selected Doom 3 map: %s\n", procPath)

	f, err := arc.Open(procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open proc file %s: %w", procPath, err)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read proc file %s: %w", procPath, err)
	}

	parser := proc.NewParser(data)
	procFile, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse proc file %s: %w", procPath, err)
	}
	fmt.Printf("Parsed %d models, %d portals from .proc\n", len(procFile.Models), len(procFile.Portals))

	materials := NewMaterials()
	_ = materials.LoadFromArchive(arc)
	fmt.Printf("Loaded %d materials from .mtr shaders\n", len(materials.defs))

	texManager := textures.NewTextures()
	imgLoader := NewImageLoader(arc, texManager, materials)
	for _, model := range procFile.Models {
		if !strings.HasPrefix(model.Name, "_area") {
			continue
		}
		for _, surface := range model.Surfaces {
			texNameLC := strings.ToLower(surface.Material)
			if strings.HasPrefix(texNameLC, "textures/decals/") || strings.HasSuffix(texNameLC, "_shadow") {
				continue
			}
			if def := materials.Resolve(surface.Material); def != nil && def.IsDecal {
				continue
			}
			imgLoader.LoadMaterialMaps(surface.Material)
		}
	}
	fmt.Printf("Loaded textures for world geometry\n")

	const chunkSize = float64(512)
	vol := NewVolumes(procFile, chunkSize, materials)
	volumes, err := vol.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build volumes: %w", err)
	}
	fmt.Printf("Built %d volumes\n", len(volumes))

	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.0
	cal.ZFarFront = 2048
	cal.ZFarRear = 512

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, texManager)
	root.Volumes = volumes

	playerPos := geometry.XYZ{X: 0, Y: 0, Z: 64}
	playerAngle := 0.0

	// Parse corresponding .map file to extract player start and dynamic lights
	mapName := strings.TrimSuffix(maps[levelIndex], ".proc")
	mapPath := "maps" + proc.Pak4Separator + mapName + ".map"
	if mf, err := arc.Open(mapPath); err == nil {
		if mapData, err := io.ReadAll(mf); err == nil {
			mapParser := NewMapParser(mapData)
			if entities, err := mapParser.Parse(); err == nil {
				for _, ent := range entities {
					classname, _ := ent.GetProperty("classname")
					base := strings.Split(classname, "_")[0]
					switch base {
					case "info":
						if classname == "info_player_start" {
							if origin, err := ent.GetOrigin(); err == nil {
								playerPos = origin
								playerAngle = ent.GetAngle()
							}
						}
					case "light":
						if origin, err := ent.GetOrigin(); err == nil {
							r, g, b := ent.GetColor()
							radius := ent.GetLightRadius()
							light := config.NewConfigLight(origin, 1.0, config.LightKindAmbient, radius)
							light.R = r
							light.G = g
							light.B = b
							root.Lights = append(root.Lights, light)
						}
					case "func": //IMPLEMENT
					case "trigger": //IMPLEMENT
					case "speaker": //IMPLEMENT
					case "item": //IMPLEMENT
					case "path": //IMPLEMENT
					case "admin": //IMPLEMENT
					case "monster": //IMPLEMENT
					case "ammo": //IMPLEMENT
					case "weapon": //IMPLEMENT
					case "moveable": //IMPLEMENT
					case "target": //IMPLEMENT
					case "ai": //IMPLEMENT
					case "env": //IMPLEMENT
					case "worldspawn": //IMPLEMENT
					default:
						fmt.Printf("Unhandled entity: %s\n", classname)
					}
				}
			}
		}
	}

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 15, 50)
	_ = common.NewPlayer(root.Player)
	root.Player.GForce = GForce
	root.Player.JumpForce = JumpForce
	root.Player.Impulse = 0.18 //mod

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Intensity = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1
	root.Player.Bobbing.SwayScale = 1.0 //mod
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
