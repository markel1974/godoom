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
		for _, surface := range model.Surfaces {
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
					switch classname {
					case "info_player_start":
						if origin, err := ent.GetOrigin(); err == nil {
							playerPos = origin
							playerAngle = ent.GetAngle()
						}
					case "light":
						if origin, err := ent.GetOrigin(); err == nil {
							r, g, b := ent.GetColor()
							radius := ent.GetLightRadius()
							light := config.NewConfigLight(origin, 1.0, config.LightKindDiffuse, radius)
							light.R = r
							light.G = g
							light.B = b
							root.Lights = append(root.Lights, light)
						}
					}
				}
			}
		}
	}

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 15, 50)
	_ = common.NewPlayer(root.Player)
	root.Player.GForce = 9.8
	root.Player.JumpForce = 270
	root.Player.Impulse = 0.15

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Intensity = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1

	return root, nil
}
