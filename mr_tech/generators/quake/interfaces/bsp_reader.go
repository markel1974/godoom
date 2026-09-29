package interfaces

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// IBSPReader defines an interface for reading and managing BSP (Binary Space Partitioning) files.
// Setup initializes the BSP reader and returns an error if initialization fails.
// GetArchive retrieves the archive instance associated with the BSP reader.
// Build processes and builds the BSP structure using the provided root configuration.
// GetPlayerInfo retrieves the player information, returning a float value and a 3D position (XYZ).
// GetEntities returns a slice of entity definitions and an error if retrieval fails.
// GetModels returns a slice of BSP models and an error if retrieval fails.
// GetTextures retrieves the textures associated with the BSP file.
type IBSPReader interface {
	Setup() error

	GetArchive() IArchive

	Build(root *config.Root) error

	GetPlayerInfo() (float64, geometry.XYZ)

	GetTextures() *lumps.Textures
}
