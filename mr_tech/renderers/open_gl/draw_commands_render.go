package open_gl

import (
	"github.com/go-gl/gl/v3.3-core/gl"
)

// DrawCommandsRender manages batched GPU drawing commands, storing index counts and memory offsets for rendering.
type DrawCommandsRender struct {
	commands []*DrawCommand
}

// NewDrawCommandsRender initializes and returns a new instance of DrawCommandsRender with preallocated internal arrays.
func NewDrawCommandsRender() *DrawCommandsRender {
	return &DrawCommandsRender{}
}

// Prepare initializes the draw command buffers and sets up data for rendering based on the provided draw commands.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.commands = dc
}

// Render executes the rendering process for the prepared draw commands using multi-draw elements in OpenGL.
func (w *DrawCommandsRender) Render() {
	w.renderInternal(false)
}

// RenderAdditive executes the rendering process for the additive draw commands without forcing depth write.
func (w *DrawCommandsRender) RenderAdditive() {
	w.renderInternal(true)
}

func (w *DrawCommandsRender) renderInternal(isAdditive bool) {
	if len(w.commands) == 0 {
		return
	}

	// Default state variables to avoid redundant calls
	//currentCullMode := -1
	currentDepthWrite := -1 // -1 means unknown

	for _, cmd := range w.commands {
		if cmd.indexCount == 0 {
			continue
		}

		if cmd.material != nil {
			// Cull Mode (0=CullFront, 1=CullNone, 2=CullBack)
			// Note: Q3's "cull front" default actually corresponds to culling the BACK faces in our OpenGL coordinate space.
			// TEMPORARILY DISABLED: The engine's BSP/Face winding order appears to be mixed,
			// causing random valid walls to be culled regardless of CullFront/CullBack.
			/*
				cullMode := cmd.material.GetCullMode()
				if cullMode != currentCullMode {
					if cullMode == 1 { // CullNone
						gl.Disable(gl.CULL_FACE)
					} else if cullMode == 0 { // CullFront (Q3 default) -> cull BACK
						gl.Enable(gl.CULL_FACE)
						gl.CullFace(gl.BACK)
					} else if cullMode == 2 { // CullBack -> cull FRONT
						gl.Enable(gl.CULL_FACE)
						gl.CullFace(gl.FRONT)
					}
					currentCullMode = cullMode
				}
			*/

			// Depth Write
			var depthWrite int
			if isAdditive {
				depthWrite = 0 // Additive passes never write depth in this engine
			} else if cmd.material.GetDepthWrite() {
				depthWrite = 1
			} else {
				depthWrite = 0
			}

			if depthWrite != currentDepthWrite {
				if depthWrite == 1 {
					gl.DepthMask(true)
				} else {
					gl.DepthMask(false)
				}
				currentDepthWrite = depthWrite
			}

			// Polygon Offset (Decals)
			if cmd.material.GetPolygonOffset() {
				gl.Enable(gl.POLYGON_OFFSET_FILL)
				gl.PolygonOffset(-1.0, -1.0)
			} else {
				gl.Disable(gl.POLYGON_OFFSET_FILL)
			}
		}

		offset := gl.PtrOffset(int(cmd.firstIndex * 4))
		gl.DrawElements(gl.TRIANGLES, cmd.indexCount, gl.UNSIGNED_INT, offset)
	}

	// Restore default state
	/*
		if currentCullMode != 0 {
			gl.Enable(gl.CULL_FACE)
			gl.CullFace(gl.BACK)
		}
	*/
	if currentDepthWrite != 1 && !isAdditive {
		gl.DepthMask(true) // Restore depth mask if we changed it, unless we're in additive pass (where it stays false)
	}
	gl.Disable(gl.POLYGON_OFFSET_FILL)
}
