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
// RegisterPixels registers texture data given its parameters, returning an error if registration fails.
// RegisterPixelsRGBA registers RGBA-based texture data, returning an error if registration fails.
// GetRawFaces retrieves raw face data based on the specified model index and returns an error if retrieval fails.
// GetExternalBModelFileName retrieves the external brush model file name for a given classname.
// GetModelFileName retrieves the file name for a model associated with a given classname.
type IBSPReader interface {
	Setup() error

	GetArchive() IArchive

	Build(root *config.Root) error

	GetPlayerInfo() (float64, geometry.XYZ)

	GetEntities() ([]*lumps.Entity, error)

	GetModels() ([]*lumps.Model, error)

	GetTextures() *lumps.Textures

	RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error

	RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error

	GetRawFaces(modelIdx int) ([]*lumps.RawFace, error)

	GetExternalBModelFileName(classname string) string

	GetModelFileName(classname string) string
}
