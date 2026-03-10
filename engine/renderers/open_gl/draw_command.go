package open_gl

// DrawCommand represents a rendering command consisting of texture ID, starting vertex index, and vertex count.
type DrawCommand struct {
	texId       uint32
	firstVertex int32
	vertexCount int32
}

// DrawCommands manages a collection of draw commands, each representing rendering instructions for a specific texture or geometry.
type DrawCommands struct {
	commands []*DrawCommand
}

// NewDrawCommands initializes and returns a new DrawCommands instance with a preallocated capacity for the command slice.
func NewDrawCommands(s int) *DrawCommands {
	return &DrawCommands{
		commands: make([]*DrawCommand, 0, s),
	}
}

// Compute updates the vertex count of a draw command by calculating the difference between lengths and aligning it.
func (w *DrawCommands) Compute(texId uint32, startLen int32, currentLen int32, alignment int32) {
	var cmd *DrawCommand
	if n := len(w.commands); n > 0 && w.commands[n-1].texId == texId {
		cmd = w.commands[n-1]
	} else {
		w.commands = append(w.commands, &DrawCommand{texId: texId, firstVertex: startLen / alignment, vertexCount: 0})
		cmd = w.commands[len(w.commands)-1]
	}
	cmd.vertexCount += (currentLen - startLen) / alignment
}

// Reset clears all draw commands by resetting the slice to an empty state without deallocating its memory.
func (w *DrawCommands) Reset() {
	w.commands = w.commands[:0]
}

// Get retrieves all draw commands stored in the DrawCommands structure.
func (w *DrawCommands) Get() []*DrawCommand {
	return w.commands
}
