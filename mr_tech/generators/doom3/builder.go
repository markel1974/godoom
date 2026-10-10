package doom3

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
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
func (p *Builder) Setup(res common.IFileSystem, pk4Path string, lev int) (*config.Root, error) {
	if lev < 1 {
		lev = 1
	}

	// Doom 3 uses .pk4 archives which are identical to zip (.pk3)
	arc := lumps.NewPk3(res)

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

	procPath := "maps" + lumps.PakSeparator + maps[levelIndex]
	fmt.Printf("Selected Doom 3 map: %s\n", procPath)

	// TODO: Implement ProcReader and MapReader

	// Create dummy root to compile
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = 1.0
	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, nil)

	return root, nil
}
