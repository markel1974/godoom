package open_gl

type DrawCommand struct {
	texId       uint32
	firstVertex int32
	vertexCount int32
}

type DrawCommands struct {
	commands []*DrawCommand
}

func NewDrawCommands(s int) *DrawCommands {
	return &DrawCommands{
		commands: make([]*DrawCommand, 0, s),
	}
}

//func (w *DrawCommands) Bind(texId uint32, vertices int32) {
//	cmd := w.Retrieve(texId)
//	cmd.vertexCount += vertices
//}

func (w *DrawCommands) Bind(id uint32, startLen int32, currentLen int32) {
	cmd := w.setDrawCommand(id, startLen/vertexFloatsAlignment)
	total := currentLen - startLen
	cmd.vertexCount += total / vertexFloatsAlignment
}

// setDrawCommand assigns or creates a new drawCmd for the specified texture ID and appends it to the frame commands list.
func (w *DrawCommands) setDrawCommand(texID uint32, firstVertex int32) *DrawCommand {
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
