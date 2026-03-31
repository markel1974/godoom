package open_gl

// DrawCommand represents a rendering command with a starting index and the number of indices to be drawn.
type DrawCommand struct {
	firstIndex int32
	indexCount int32
}

// DrawCommands manages a collection of draw command objects used to define rendering batches in a graphics pipeline.
type DrawCommands struct {
	commands             []*DrawCommand
	freezeLen            int
	len                  int
	freezeLastIndexCount int32
}

// NewDrawCommands creates and initializes a new DrawCommands instance with a preallocated slice of the given size.
func NewDrawCommands(s int) *DrawCommands {
	dc := &DrawCommands{
		commands: make([]*DrawCommand, s),
	}
	for i := range dc.commands {
		dc.commands[i] = &DrawCommand{}
	}
	return dc
}

// Compute updates or creates a draw command using the provided start and current index values.
func (w *DrawCommands) Compute(startIndices, currentIndices int32) {
	var cmd *DrawCommand

	// Se abbiamo già un comando e la nuova geometria inizia esattamente
	// dove finisce la precedente, fondiamo tutto in un singolo batch gigante!
	if w.len > 0 && (w.commands[w.len-1].firstIndex+w.commands[w.len-1].indexCount) == startIndices {
		cmd = w.commands[w.len-1]
	} else {
		// Crea un nuovo comando solo in caso di discontinuità di memoria
		if w.len >= len(w.commands) {
			w.Grow()
		}
		cmd = w.commands[w.len]
		cmd.firstIndex = startIndices
		cmd.indexCount = 0
		w.len++
	}

	cmd.indexCount += currentIndices - startIndices
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
		w.commands[w.len-1].indexCount = w.freezeLastIndexCount
	}
}

// Freeze locks the current state of the draw commands, saving the length and index count for future resets.
func (w *DrawCommands) Freeze() {
	w.freezeLen = w.len
	if w.len > 0 {
		w.freezeLastIndexCount = w.commands[w.len-1].indexCount
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
		newCommands[i] = &DrawCommand{}
	}
	w.commands = newCommands
}
