package open_gl_legacy

// DrawCommand represents a rendering command consisting of texture ID, starting vertex index, and vertex count.
type DrawCommand struct {
	texId         uint32
	normTexId     uint32
	emissiveTexId uint32
	firstVertex   int32
	vertexCount   int32
}

// DrawCommands manages a collection of draw commands, each representing rendering instructions for a specific texture or geometry.
type DrawCommands struct {
	commands []*DrawCommand
	len      int
}

// NewDrawCommands initializes and returns a new DrawCommands instance with a preallocated capacity for the command slice.
func NewDrawCommands(s int) *DrawCommands {
	dc := &DrawCommands{
		commands: make([]*DrawCommand, s),
	}
	for i := range dc.commands {
		dc.commands[i] = &DrawCommand{}
	}
	return dc
}

// Compute updates the vertex count of a draw command by calculating the difference between lengths and aligning it.
func (w *DrawCommands) Compute(texId, normTexId, emissiveTexId uint32, startLen, currentLen, alignment int32) {
	var cmd *DrawCommand
	if w.len > 0 && w.commands[w.len-1].texId == texId {
		cmd = w.commands[w.len-1]
	} else {
		if w.len >= len(w.commands) {
			w.Grow()
		}

		cmd = w.commands[w.len]
		cmd.texId = texId
		cmd.normTexId = normTexId
		cmd.emissiveTexId = emissiveTexId
		cmd.firstVertex = startLen / alignment
		cmd.vertexCount = 0
		w.len++
	}
	cmd.vertexCount += (currentLen - startLen) / alignment
}

// Reset clears all draw commands by resetting the slice to an empty state without deallocating its memory.
func (w *DrawCommands) Reset() {
	w.len = 0
}

// Get retrieves all draw commands stored in the DrawCommands structure.
func (w *DrawCommands) Get() []*DrawCommand {
	return w.commands[0:w.len]
}

// Grow increases the capacity of the commands slice, doubling its size or setting an initial size if empty.
func (w *DrawCommands) Grow() {
	oldLen := len(w.commands)
	newSize := oldLen * 2
	if newSize == 0 {
		newSize = 128 // Rimossa la dipendenza da vertexAlignment (qui non c'entra)
	}
	newCommands := make([]*DrawCommand, newSize)
	copy(newCommands, w.commands)
	for i := oldLen; i < newSize; i++ {
		newCommands[i] = &DrawCommand{}
	}
	w.commands = newCommands
}
