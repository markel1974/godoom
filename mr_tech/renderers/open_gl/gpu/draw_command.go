package gpu

import "github.com/markel1974/godoom/mr_tech/textures"

// DrawCommand represents a rendering command with a starting index and the number of indices to be drawn.
type DrawCommand struct {
	firstIndex int32
	indexCount int32
	material   *textures.Material
	centroidX  float32
	centroidY  float32
	centroidZ  float32
}

// NewDrawCommand initializes and returns a pointer to a new DrawCommand instance with default values.
func NewDrawCommand() *DrawCommand {
	return &DrawCommand{}
}

// SetMaterial assigns a new material to the DrawCommand for rendering.
func (w *DrawCommand) SetMaterial(material *textures.Material) {
	w.material = material
}

// SetFirstIndex sets the starting index for the draw command.
func (w *DrawCommand) SetFirstIndex(firstIndex int32) {
	w.firstIndex = firstIndex
}

// SetIndexCount sets the count of indices to be drawn for the draw command.
func (w *DrawCommand) SetIndexCount(indexCount int32) {
	w.indexCount = indexCount
}

// SetCentroid sets the approximate GL-space centroid for transparent depth sorting.
func (w *DrawCommand) SetCentroid(cx, cy, cz float32) {
	w.centroidX = cx
	w.centroidY = cy
	w.centroidZ = cz
}

// GetCentroid returns the stored GL-space centroid.
func (w *DrawCommand) GetCentroid() (float32, float32, float32) {
	return w.centroidX, w.centroidY, w.centroidZ
}

// IncIndexCount increments the index count of the draw command by the specified amount.
func (w *DrawCommand) IncIndexCount(count int32) {
	w.indexCount += count
}

// GetMaterial retrieves the material associated with the DrawCommand.
func (w *DrawCommand) GetMaterial() *textures.Material {
	return w.material
}

// GetFirstIndex returns the starting index of the draw command used for rendering.
func (w *DrawCommand) GetFirstIndex() int32 {
	return w.firstIndex
}

// GetIndexCount returns the total number of indices to be drawn in this draw command.
func (w *DrawCommand) GetIndexCount() int32 {
	return w.indexCount
}
