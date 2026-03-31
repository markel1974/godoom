package open_gl

// DrawCommand represents a single rendering command with associated textures and index information for rendering.
type DrawCommand struct {
	texId         uint32
	normTexId     uint32
	emissiveTexId uint32
	firstIndex    int32
	indexCount    int32
}

// DrawCommands holds a collection of DrawCommand objects and provides methods to manage, grow, and manipulate them efficiently.
type DrawCommands struct {
	commands             []*DrawCommand
	freezeLen            int
	len                  int
	freezeLastIndexCount int32 // Backup essenziale per prevenire la corruzione
}

// NewDrawCommands initializes and returns a new DrawCommands instance with preallocated memory for the specified size.
func NewDrawCommands(s int) *DrawCommands {
	dc := &DrawCommands{
		commands: make([]*DrawCommand, s),
	}
	for i := range dc.commands {
		dc.commands[i] = &DrawCommand{}
	}
	return dc
}

// Compute updates or creates a draw command with provided texture IDs and manages index ranges for rendering.
func (w *DrawCommands) Compute(texId, normTexId, emissiveTexId uint32, startIndices, currentIndices int32) {
	var cmd *DrawCommand
	if w.len > 0 && w.commands[w.len-1].texId == texId && w.commands[w.len-1].normTexId == normTexId && w.commands[w.len-1].emissiveTexId == emissiveTexId {
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

// DeepReset clears all freeze-related state and resets the commands list to its initial state by invoking the Reset method.
func (w *DrawCommands) DeepReset() {
	w.freezeLen = 0
	w.freezeLastIndexCount = 0
	w.Reset()
}

// Reset reverts the DrawCommands object to its frozen state by restoring length and updating the last command's index count.
func (w *DrawCommands) Reset() {
	w.len = w.freezeLen
	if w.len > 0 {
		w.commands[w.len-1].indexCount = w.freezeLastIndexCount
	}
}

// Freeze finalizes the current state of draw commands, storing the count and last index count for future resets.
func (w *DrawCommands) Freeze() {
	w.freezeLen = w.len
	if w.len > 0 {
		w.freezeLastIndexCount = w.commands[w.len-1].indexCount
	}
}

// Get retrieves the list of DrawCommand objects up to the current length of commands.
func (w *DrawCommands) Get() []*DrawCommand {
	return w.commands[0:w.len]
}

// GetDrawCommands returns the slice of DrawCommand objects currently stored, limited to the active length of the collection.
func (w *DrawCommands) GetDrawCommands() []*DrawCommand {
	return w.Get()
}

// Grow dynamically increases the size of the commands slice, doubling its capacity or initializing it to 128 if empty.
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
