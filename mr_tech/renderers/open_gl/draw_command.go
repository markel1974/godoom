// open_gl/draw_command.go
package open_gl

// DrawCommand represents a rendering command for indexed geometry.
type DrawCommand struct {
	texId         uint32
	normTexId     uint32
	emissiveTexId uint32
	firstIndex    int32
	indexCount    int32
}

// DrawCommands manages a collection of draw commands.
type DrawCommands struct {
	commands []*DrawCommand
	len      int
}

// NewDrawCommands initializes and returns a new DrawCommands instance.
func NewDrawCommands(s int) *DrawCommands {
	dc := &DrawCommands{
		commands: make([]*DrawCommand, s),
	}
	for i := range dc.commands {
		dc.commands[i] = &DrawCommand{}
	}
	return dc
}

// Compute updates or creates a new draw command based on texture IDs and index boundaries.
func (w *DrawCommands) Compute(texId, normTexId, emissiveTexId uint32, startIndices, currentIndices int32) {
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
		cmd.firstIndex = startIndices
		cmd.indexCount = 0
		w.len++
	}
	cmd.indexCount += currentIndices - startIndices
}

// Reset clears all draw commands logically without deallocating.
func (w *DrawCommands) Reset() {
	w.len = 0
}

// Get retrieves all active draw commands.
func (w *DrawCommands) Get() []*DrawCommand {
	return w.commands[0:w.len]
}

// GetDrawCommands retrieves the list of active draw commands for rendering the current frame.
func (w *DrawCommands) GetDrawCommands() []*DrawCommand {
	return w.Get()
}

// Grow increases the capacity of the commands slice.
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
