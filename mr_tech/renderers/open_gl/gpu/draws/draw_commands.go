package draws

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// DrawCommands manages a collection of draw command objects used to define rendering batches in a graphics pipeline.
type DrawCommands struct {
	commands             []*DrawCommand
	freezeLen            int
	len                  int
	freezeLastIndexCount int32
	blendMode            int
}

// NewDrawCommands creates and initializes a new DrawCommands instance with a preallocated slice of the given size.
func NewDrawCommands(mode int, s int) *DrawCommands {
	dc := &DrawCommands{
		blendMode: mode,
		commands:  make([]*DrawCommand, s),
	}
	for i := range dc.commands {
		dc.commands[i] = NewDrawCommand()
	}
	return dc
}

// GetBlendMode returns the current blend mode used by the DrawCommands instance.
func (w *DrawCommands) GetBlendMode() int {
	return w.blendMode
}

// Compute updates or creates a draw command using the provided start and current index values and material.
func (w *DrawCommands) Compute(startIndices, currentIndices int32, material *textures.Material) {
	w.ComputeWithCentroid(startIndices, currentIndices, material, 0, 0, 0)
}

// ComputeWithCentroid updates or creates a draw command while recording the batch centroid for transparent sorting.
func (w *DrawCommands) ComputeWithCentroid(startIndices, currentIndices int32, material *textures.Material, cx, cy, cz float32) {
	var cmd *DrawCommand

	// Se abbiamo già un comando e la nuova geometria inizia esattamente
	// dove finisce la precedente, fondiamo tutto in un singolo batch gigante!
	// MA SOLO se lo stato OpenGL (DepthWrite, PolygonOffset, Cutout, CullMode) è identico
	// e non siamo in modalità trasparente depth-sorted!
	canMerge := false
	if w.blendMode != int(config.BlendModeTransparent) && w.len > 0 && (w.commands[w.len-1].GetFirstIndex()+w.commands[w.len-1].GetIndexCount()) == startIndices {
		prevMat := w.commands[w.len-1].GetMaterial()
		if prevMat == material {
			canMerge = true
		} else if prevMat != nil && material != nil {
			if prevMat.GetDepthWrite() == material.GetDepthWrite() &&
				prevMat.GetPolygonOffset() == material.GetPolygonOffset() &&
				prevMat.HasCutout() == material.HasCutout() &&
				prevMat.GetCullMode() == material.GetCullMode() {
				canMerge = true
			}
		} else if prevMat == nil && material == nil {
			canMerge = true
		}
	}

	if canMerge {
		cmd = w.commands[w.len-1]
	} else {
		// Crea un nuovo comando solo in caso di discontinuità di memoria o cambio di stato
		if w.len >= len(w.commands) {
			w.Grow()
		}
		cmd = w.commands[w.len]
		cmd.SetFirstIndex(startIndices)
		cmd.SetIndexCount(0)
		cmd.SetMaterial(material)
		cmd.SetCentroid(cx, cy, cz)
		w.len++
	}

	cmd.IncIndexCount(currentIndices - startIndices)
}

// DeepReset clears all commands and resets freeze-related state to initial values in the DrawCommands instance.
func (w *DrawCommands) DeepReset() {
	w.freezeLen = 0
	w.freezeLastIndexCount = 0
	w.Reset()
}

// Reset restores the DrawCommands object to a previously frozen state by adjusting its length and last index count.
func (w *DrawCommands) Reset() {
	w.len = w.freezeLen
	if w.len > 0 {
		w.commands[w.len-1].SetIndexCount(w.freezeLastIndexCount)
	}
}

// Freeze locks the current state of the draw commands, saving the length and index count for future resets.
func (w *DrawCommands) Freeze() {
	w.freezeLen = w.len
	if w.len > 0 {
		w.freezeLastIndexCount = w.commands[w.len-1].GetIndexCount()
	}
}

// Get retrieves the list of draw commands up to the current length.
func (w *DrawCommands) Get() []*DrawCommand {
	return w.commands[0:w.len]
}

// GetDrawCommands returns the current list of DrawCommand instances up to the active length.
func (w *DrawCommands) GetDrawCommands() []*DrawCommand {
	return w.Get()
}

// Grow doubles the capacity of the commands slice, initializing new elements as needed, starting with size 128 if empty.
func (w *DrawCommands) Grow() {
	oldLen := len(w.commands)
	newSize := oldLen * 2
	if newSize == 0 {
		newSize = 128
	}
	newCommands := make([]*DrawCommand, newSize)
	copy(newCommands, w.commands)
	for i := oldLen; i < newSize; i++ {
		newCommands[i] = NewDrawCommand()
	}
	w.commands = newCommands
}
