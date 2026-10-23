package duke3d

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	AspectRatio   = 1.333
	ScaleSectorH  = 1.0
	ScaleXY       = 1.0
	ScaleTextureW = 1.0
	ScaleTextureH = 1.0

	PlayerHeight = 40.0 * ScaleSectorH
	PlayerRadius = 18.0
	PlayerSpeed  = 1800.0
	PlayerMass   = 50.0

	GForce    = 9.8 * 50
	JumpForce = GForce * 8
)

type Builder struct {
	scaleSectorH float64
	scaleXY      float64
}

func NewBuilder() *Builder {
	return &Builder{
		scaleSectorH: ScaleSectorH,
		scaleXY:      ScaleXY,
	}
}

func (b *Builder) Build(res common.IResources, target string, levelNumber int) (*config.Root, error) {
	mapPath := target

	// If target is a directory or empty, scan for .map files
	isDir := false
	if fi, err := os.Stat(target); err == nil && fi.IsDir() {
		isDir = true
	}
	if isDir || (!strings.HasSuffix(strings.ToLower(target), ".map") && len(target) > 0) {
		var entries []os.DirEntry
		if res != nil {
			if e, err := res.ReadDir(target); err == nil {
				entries = e
			}
		}
		if entries == nil {
			entries, _ = os.ReadDir(target)
		}
		var mapFiles []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".map") {
				mapFiles = append(mapFiles, e.Name())
			}
		}
		if len(mapFiles) > 0 {
			sort.Strings(mapFiles)
			idx := levelNumber - 1
			if idx < 0 {
				idx = 0
			}
			if idx >= len(mapFiles) {
				idx = len(mapFiles) - 1
			}
			sep := string(os.PathSeparator)
			if strings.HasSuffix(target, sep) || strings.HasSuffix(target, "/") {
				mapPath = target + mapFiles[idx]
			} else {
				mapPath = target + sep + mapFiles[idx]
			}
		}
	}

	var reader io.ReadCloser
	if res != nil {
		r, err := res.Open(mapPath)
		if err == nil {
			reader = r
		}
	}
	if reader == nil {
		f, err := os.Open(mapPath)
		if err != nil {
			return nil, fmt.Errorf("duke3d: failed to open map %s: %w", mapPath, err)
		}
		reader = f
	}
	defer reader.Close()

	level, err := lumps.LoadMap(reader)
	if err != nil {
		return nil, fmt.Errorf("duke3d: failed to load map %s: %w", mapPath, err)
	}

	tex := lumps.NewTextures()
	world := NewWorld(level, tex)
	world.SetScales(b.scaleSectorH, b.scaleXY, ScaleTextureW, ScaleTextureH, true)

	cSectors, gVertices, lights, err := world.Build()
	if err != nil {
		return nil, fmt.Errorf("duke3d: failed to build world: %w", err)
	}

	pX := float64(level.Header.PosX) * b.scaleXY
	pY := -float64(level.Header.PosY) * b.scaleXY // flip Y to Cartesian up
	pZ := -float64(level.Header.PosZ) / 16.0 * b.scaleSectorH

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

	playerPos := geometry.XYZ{X: pX, Y: pY, Z: pZ}
	player := config.NewConfigPlayer(playerPos, pAngle, PlayerMass, PlayerSpeed, PlayerRadius, PlayerHeight)
	_ = common.NewPlayer(player)
	player.GForce = GForce
	player.JumpForce = JumpForce

	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio

	crosshair := common.NewSimpleCrosshair()
	tex.AddDirect(crosshair.GetTexture())

	scaleFactor := geometry.XYZ{X: b.scaleXY, Y: b.scaleXY, Z: b.scaleSectorH}
	cr := config.NewConfigRoot(cal, cSectors, player, nil, scaleFactor, tex)
	cr.Vertices = gVertices
	cr.Lights = lights
	cr.Crosshair = crosshair.Create()

	return cr, nil
}
