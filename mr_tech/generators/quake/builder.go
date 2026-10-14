package quake

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
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
func (p *Builder) Setup(res common.IFileSystem, pakPath string, lev int) (*config.Root, error) {
	if lev < 1 {
		lev = 1
	}
	levelIndex := lev - 1
	//bpsPath := "maps" + lumps.PakSeparator + "e1m" + strconv.Itoa(level) + ".bsp"

	arc, aErr := NewArchive(res, pakPath)
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

	reader, bErr := NewBSPReader(arc, bpsPath)
	if bErr != nil {
		return nil, bErr
	}
	if err := reader.Setup(); err != nil {
		return nil, err
	}
	root, rErr := reader.Retrieve()
	if rErr != nil {
		return nil, rErr
	}
	//fmt.Println("TODO REACTIVATE ROOT THINGS!")
	//root.Things = nil
	return root, nil
}
