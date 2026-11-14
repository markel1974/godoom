package doom3

import (
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/textures"
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
func (br *Builder) Setup(res common.IResources, pk4Path string, lev int) (*config.Root, error) {
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

	materials := proc.NewMaterials(arc)
	if err = materials.Load(); err != nil {
		return nil, fmt.Errorf("failed to load materials: %w", err)
	}
	fmt.Printf("Loaded %d materials from .mtr shaders\n", len(materials.Defs))

	defs := proc.NewDefs(arc)
	if err = defs.Load(); err != nil {
		return nil, fmt.Errorf("failed to load entityDefs and models: %w", err)
	}
	fmt.Printf("Loaded %d entityDefs, %d models from .def files\n", defs.EntityLen(), defs.ModelLen())

	// Parse corresponding .map file to extract entities (lights, player start, func_static, items)
	mapName := strings.TrimSuffix(maps[levelIndex], ".proc")
	mapPath := "maps" + proc.Pak4Separator + mapName + ".map"
	var entities []*proc.MapEntity
	if mf, err := arc.Open(mapPath); err == nil {
		if mapData, err := io.ReadAll(mf); err == nil {
			mapParser := proc.NewMapParser(mapData)
			if parsedEntities, err := mapParser.Parse(); err == nil {
				entities = parsedEntities
			}
		}
	}
	fmt.Printf("Parsed %d entities from .map\n", len(entities))

	texManager := textures.NewTextures()
	imgLoader := proc.NewImageLoader(arc, texManager, materials)

	entitiesMap := make(map[string]*proc.MapEntity, len(entities))
	for _, ent := range entities {
		if name, ok := ent.GetProperty("name"); ok && len(name) > 0 {
			entitiesMap[name] = ent
		}
		if model, ok := ent.GetProperty("model"); ok && len(model) > 0 {
			entitiesMap[model] = ent
		}
	}

	for _, model := range procFile.Models {
		ent := entitiesMap[model.Name]
		for _, surface := range model.Surfaces {
			texNameLC := strings.ToLower(surface.Material)
			if strings.HasPrefix(texNameLC, "textures/decals/") || strings.HasSuffix(texNameLC, "_shadow") {
				continue
			}
			if def := materials.Resolve(surface.Material); def != nil && def.IsDecal {
				continue
			}
			imgLoader.LoadMaterialMaps(surface.Material)
			if guiBg := ResolveGUIBackgroundForSurface(arc, materials, surface.Material, ent); len(guiBg) > 0 {
				imgLoader.LoadMaterialMaps(guiBg)
			}
		}
	}

	for _, ent := range entities {
		if modelPath, ok := ent.GetProperty("model"); ok && strings.HasSuffix(strings.ToLower(modelPath), ".lwo") {
			cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(modelPath, "\\", "/")))
			if lf, err := arc.Open(cleanPath); err == nil {
				lwd := proc.NewLWO()
				if err := lwd.Parse(lf); err == nil {
					for _, tag := range lwd.Tags {
						imgLoader.LoadMaterialMaps(tag)
						if guiBg := ResolveGUIBackgroundForSurface(arc, materials, tag, ent); len(guiBg) > 0 {
							imgLoader.LoadMaterialMaps(guiBg)
						}
					}
				}
			}
		}
	}
	imgLoader.LoadMaterialMaps(DefaultMonitorTexture)
	fmt.Printf("Loaded textures for world geometry\n")

	const volumesCount = float64(128)
	vol := NewVolumes(procFile, volumesCount, materials, entities, arc)
	volumes, err := vol.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build volumes: %w", err)
	}
	fmt.Printf("Built %d volumes\n", len(volumes))

	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.0

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, texManager)
	root.Volumes = volumes

	playerPos := geometry.XYZ{X: 0, Y: 0, Z: 64}
	playerAngle := 0.0
	monsters := 0

	thingsManager := NewThings(arc, materials, defs, texManager, imgLoader)
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
				kind := br.ToLightKind(ent.GetLightKind())
				light := config.NewConfigLight(origin, 1.0, kind, radius)
				light.R, light.G, light.B = r, g, b
				if kind == config.LightKindSpot {
					light.Intensity *= 5
					dir, cutOff, outerCutOff := ent.GetSpotlightParams()
					light.DirX, light.DirY, light.DirZ = dir.X, dir.Y, dir.Z
					light.CutOff, light.OuterCutOff = cutOff, outerCutOff
				}
				root.Lights = append(root.Lights, light)
			}
		case "func": //IMPLEMENT
		case "trigger": //IMPLEMENT
		case "speaker": //IMPLEMENT
		case "sound": //IMPLEMENT
		case "item":
			if thing, err := thingsManager.Create(ent); err == nil {
				root.Things = append(root.Things, thing)
			}
		case "path": //IMPLEMENT
		case "admin": //IMPLEMENT
		case "powerup": //IMPLEMENT
		case "monster":
			if thing, err := thingsManager.Create(ent); err == nil {
				monsters++
				root.Things = append(root.Things, thing)
			}
		case "ammo":
			if thing, err := thingsManager.Create(ent); err == nil {
				root.Things = append(root.Things, thing)
			}
		case "weapon":
			if thing, err := thingsManager.Create(ent); err == nil {
				root.Things = append(root.Things, thing)
			}
		case "moveable":
			if thing, err := thingsManager.Create(ent); err == nil {
				root.Things = append(root.Things, thing)
			}
		case "target": //IMPLEMENT
		case "ai": //IMPLEMENT
		case "env": //IMPLEMENT
		case "char": //IMPLEMENT
		case "worldspawn": //IMPLEMENT
		case "alphalabs2": //IMPLEMENT
		case "alphalabs1": //IMPLEMENT
		case "hellhole": //IMPLEMENT
		case "marscity": //IMPLEMENT
		case "marscity2": //IMPLEMENT
		case "ending": //IMPLEMENT
		default:
			fmt.Printf("Unhandled entity: %s\n", classname)
		}
	}

	fmt.Printf("MONSTERS: %d\n", monsters)

	crosshair := common.NewSimpleCrosshair()
	texManager.AddDirect(crosshair.GetTexture())
	root.AddUIElement(crosshair.Create())

	root.Player = br.createPlayer(playerPos, playerAngle)

	return root, nil
}

// ToLightKind determines the LightKind based on texture name, ambient flag, and target string conditions.
func (br *Builder) ToLightKind(tex, ambient, target string) config.LightKind {
	texLower := strings.ToLower(tex)
	if ambient == "1" || strings.Contains(texLower, "ambient") {
		return config.LightKindAmbient
	}
	if len(target) > 0 {
		return config.LightKindSpot
	}
	if strings.Contains(texLower, "spot") {
		return config.LightKindSpot
	}
	return config.LightKindDiffuse
}

// createPlayer initializes and returns a Player instance with specified position, angle, and predefined configurations.
func (br *Builder) createPlayer(playerPos geometry.XYZ, playerAngle float64) *config.Player {
	player := config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 20, 60)
	_ = common.NewPlayer(player)
	player.GForce = GForce
	player.JumpForce = JumpForce
	player.Impulse = 0.18 //mod

	player.Flash.ZFar = 8192
	player.Flash.Intensity = 0.02
	player.Flash.Falloff = 2000
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 1.0 //mod
	player.Bobbing.SwayOffsetX = 50
	player.Bobbing.SwayOffsetY = -0.9
	player.Bobbing.MaxAmplitudeX = 5.0 // MAXIMUM EXCURSION: 12 units (approx. 20% of player height)
	player.Bobbing.MaxAmplitudeY = 5.5
	player.Bobbing.StrideLength = 0.07 //mod //0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	// Breathing
	player.Bobbing.IdleAmpX = 0.9  //mod 0.9
	player.Bobbing.IdleAmpY = 0.9  //mod 0.9
	player.Bobbing.IdleDrift = 3.0 //2.0 - 0.01

	return player
}
