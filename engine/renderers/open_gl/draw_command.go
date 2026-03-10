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
func (w *DrawCommands) Compute(id uint32, startLen int32, currentLen int32, alignment int32) {
	cmd := w.assign(id, startLen/alignment)
	total := currentLen - startLen
	cmd.vertexCount += total / alignment
}

// assign assigns or creates a new DrawCommand with the specified texture ID and first vertex, returning the applicable command.
func (w *DrawCommands) assign(texID uint32, firstVertex int32) *DrawCommand {
	n := len(w.commands)
	if n > 0 && w.commands[n-1].texId == texID {
		return w.commands[n-1]
	}
	w.commands = append(w.commands, &DrawCommand{
		texId:       texID,
		firstVertex: firstVertex, //int32(len(w.frameVertices) / vertexFloatsAlignment),
		vertexCount: 0,
	})
	return w.commands[len(w.commands)-1]
}

// Reset clears all draw commands by resetting the slice to an empty state without deallocating its memory.
func (w *DrawCommands) Reset() {
	w.commands = w.commands[:0]
}

/*
func (w *DrawCommands) Draw() {
	for _, cmd := range w.commands {
		if cmd.vertexCount > 0 {
			gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}

*/
