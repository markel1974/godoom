package duke3d

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// AspectRatio defines the default aspect ratio for the application's display.
// ScaleSectorH determines the vertical scaling factor for sectors.
// ScaleXY specifies a uniform scaling factor for both x and y dimensions.
// PlayerHeight calculates the height of the player based on ScaleSectorH.
// PlayerRadius defines the radius of the player character.
// PlayerSpeed specifies the movement speed of the player in units per second.
// PlayerMass defines the mass of the player for physics calculations.
// GForce represents the gravitational force multiplied by a fixed mass.
// JumpForce calculates the jump force as a multiple of GForce.
const (
	AspectRatio  = 1.333
	ScaleSectorH = 1.0
	ScaleXY      = 1.0 / 16.0
	PlayerHeight = 56.0 * ScaleSectorH
	PlayerRadius = 16.0
	PlayerSpeed  = 1800.0
	PlayerMass   = 50.0

	GForce    = 9.8 * 50
	JumpForce = GForce * 8
)

// Builder is used to configure and construct game level structures with adjustable scaling properties.
type Builder struct {
	scaleSectorH float64
	scaleXY      float64
}

// NewBuilder creates and initializes a new Builder instance with default scale values.
func NewBuilder() *Builder {
	return &Builder{
		scaleSectorH: ScaleSectorH,
		scaleXY:      ScaleXY,
	}
}

// Build constructs and initializes the game world configuration based on the provided resources, target path, and level number.
func (b *Builder) Build(res common.IResources, target string, levelNumber int) (*config.Root, error) {
	if res == nil {
		return nil, fmt.Errorf("duke3d: resources interface cannot be nil")
	}

	var reader io.ReadCloser
	tex := lumps.NewTextures()

	// Check if target is a GRP file directly
	if strings.HasSuffix(strings.ToLower(target), ".grp") {
		grpReader, err := b.createGrpReader(res, target, levelNumber, tex)
		if err != nil {
			return nil, fmt.Errorf("duke3d: failed to open grp %s: %w", target, err)
		}
		reader = grpReader
	} else {
		// Target might be a directory or a .map file
		entries, eErr := res.ReadDir(target)
		if eErr != nil {
			return nil, fmt.Errorf("duke3d: failed to read directory %s: %w", target, eErr)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("duke3d: no files found in directory %s", target)
		}
		// Check if there is a .GRP file in the directory first
		var grpName string
		var mapFiles []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			nameLower := strings.ToLower(e.Name())
			if strings.HasSuffix(nameLower, ".grp") {
				grpName = e.Name()
			} else if strings.HasSuffix(nameLower, ".map") {
				mapFiles = append(mapFiles, e.Name())
			}
		}

		if len(grpName) > 0 {
			grpPath := target
			if strings.HasSuffix(grpPath, "/") {
				grpPath += grpName
			} else {
				grpPath += "/" + grpName
			}
			grpReader, err := b.createGrpReader(res, grpPath, levelNumber, tex)
			if err != nil {
				return nil, fmt.Errorf("duke3d: failed to open grp %s: %w", grpPath, err)
			}
			reader = grpReader
		} else if len(mapFiles) > 0 {
			sort.Strings(mapFiles)
			idx := levelNumber - 1
			if idx < 0 {
				idx = 0
			}
			if idx >= len(mapFiles) {
				idx = len(mapFiles) - 1
			}
			if strings.HasSuffix(target, "/") {
				target = target + mapFiles[idx]
			} else {
				target = target + "/" + mapFiles[idx]
			}
			r, err := res.Open(target)
			if err != nil {
				return nil, fmt.Errorf("duke3d: failed to open map %s: %w", target, err)
			}
			reader = r
		} else {
			return nil, fmt.Errorf("duke3d: no map found for level %d", levelNumber)
		}
	}
	defer reader.Close()

	level := lumps.NewLevel()

	if err := level.Parse(reader); err != nil {
		return nil, fmt.Errorf("duke3d: failed to load map %s: %w", target, err)
	}

	world := NewWorld(level, tex)
	world.SetScales(b.scaleSectorH, b.scaleXY, true)

	cSectors, gVertices, volumes, lights, err := world.Build()
	if err != nil {
		return nil, fmt.Errorf("duke3d: failed to build world: %w", err)
	}

	pX := float64(level.Header.PosX) * b.scaleXY
	pY := -float64(level.Header.PosY) * b.scaleXY // flip Y to Cartesian up

	// Build angle: 0 to 2047. In Build: 0=East (+X), 512=South (+Y).
	// With Y flipped, 0=East, 512=South (-Y), 1536=North (+Y).
	// In Cartesian degrees: (2048 - ang) * 360 / 2048.
	pAngle := (2048.0 - float64(level.Header.Ang)) * 360.0 / 2048.0
	for pAngle >= 360.0 {
		pAngle -= 360.0
	}
	for pAngle < 0.0 {
		pAngle += 360.0
	}

	playerPos := geometry.XYZ{X: pX, Y: pY, Z: 0}
	player := config.NewConfigPlayer(playerPos, pAngle, PlayerMass, PlayerSpeed, PlayerRadius, PlayerHeight)
	_ = common.NewPlayer(player)
	player.GForce = GForce
	player.JumpForce = JumpForce
	player.Impulse = 0.1 //mod

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
	//breathing
	player.Bobbing.IdleAmpX = 0.9  //mod 0.9
	player.Bobbing.IdleAmpY = 0.9  //mod 0.9
	player.Bobbing.IdleDrift = 3.0 //2.0 - 0.01

	player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	player.Bobbing.AmpLerp = 0.20
	player.Bobbing.ImpactMax = 1000.0
	player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
	player.Bobbing.SpringDamping = 0.80
	player.Bobbing.TiltAmp = 0.05

	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio

	crosshair := common.NewSimpleCrosshair()
	tex.AddDirect(crosshair.GetTexture())

	scaleFactor := geometry.XYZ{X: 1.0, Y: 1.0, Z: 1.0}
	cr := config.NewConfigRoot(cal, cSectors, player, nil, scaleFactor, tex)
	cr.Vertices = gVertices
	cr.Volumes = volumes
	cr.Lights = lights
	cr.Crosshair = crosshair.Create()

	return cr, nil
}

// createGrpReader parses a .GRP file, loads textures and level data, and returns a reader for the specified level map.
func (b *Builder) createGrpReader(res common.IResources, target string, levelNumber int, tex *lumps.Textures) (io.ReadCloser, error) {
	r, rErr := res.Open(target)
	if rErr != nil {
		return nil, fmt.Errorf("duke3d: failed to open grp %s: %w", target, rErr)
	}
	defer r.Close()
	arc := lumps.NewArchiveGrp()
	if pErr := arc.Parse(r); pErr != nil {
		return nil, fmt.Errorf("duke3d: failed to parse grp %s: %w", target, pErr)
	}
	defer arc.Close()
	palData, pErr := arc.GetPayload("PALETTE.DAT")
	if pErr != nil {
		return nil, fmt.Errorf("duke3d: failed to get palette data from grp %s: %w", target, pErr)
	}
	pal := lumps.NewPalette()
	if err := pal.Parse(palData); err != nil {
		return nil, fmt.Errorf("duke3d: failed to parse palette data from grp %s: %w", target, err)
	}
	tiles, tErr := arc.Match("TILES\\d{3}.ART", false)
	if tErr != nil {
		return nil, fmt.Errorf("duke3d: failed to match TILES files in grp %s: %w", target, tErr)
	}
	arts := lumps.NewArts(pal.GetData())
	tileCount := 0
	for _, artName := range tiles {
		if artData, err := arc.GetPayload(artName); err == nil {
			_ = arts.Load(artData)
			tileCount++
		}
	}
	for tId, tImg := range arts.GetTiles() {
		tex.AddDepot(tId, tImg)
	}
	fmt.Printf("Loaded %d ART files from GRP into texture manager\n", tileCount)
	maps, err := arc.Match(".*\\.MAP", true)
	if err != nil {
		return nil, fmt.Errorf("duke3d: failed to extract map files from grp: %w", err)
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("duke3d: no .MAP files found in %s", target)
	}
	idx := levelNumber - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(maps) {
		idx = len(maps) - 1
	}
	fmt.Printf("Loading Duke3D level %d (%s) from GRP (total %d maps)\n", idx+1, maps[idx], len(maps))
	mapReader, err := arc.GetReader(maps[idx])
	if err != nil {
		return nil, fmt.Errorf("duke3d: failed to extract map %s from grp: %w", maps[idx], err)
	}
	return mapReader, nil
}
