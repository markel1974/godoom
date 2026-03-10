package open_gl

import "github.com/go-gl/gl/v3.3-core/gl"

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

func (w *DrawCommands) Bind(texId uint32, vertices int32) {
	cmd := w.Retrieve(texId)
	cmd.vertexCount += vertices
}

func (w *DrawCommands) Retrieve(texId uint32) *DrawCommand {
	n := len(w.commands)
	if n > 0 && w.commands[n-1].texId == texId {
		return w.commands[n-1]
	}
	w.commands = append(w.commands, &DrawCommand{
		texId:       texId,
		firstVertex: int32(len(w.commands) / vertexFloatsAlignment),
		vertexCount: 0,
	})
	return w.commands[len(w.commands)-1]
}

func (w *DrawCommands) Reset() {
	w.commands = w.commands[:0]
}

func (w *DrawCommands) Draw() {
	for _, cmd := range w.commands {
		if cmd.vertexCount > 0 {
			gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}
