package open_gl

import (
	"fmt"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	// "github.com/go-gl/gl/v4.1-core/gl" // Da decommentare per i binding nativi
)

// RenderOpenGL gestisce la pipeline hardware, sfruttando la topologia
// strict-sector e il view-window clipping lato CPU per iniettare nella GPU
// esclusivamente la geometria visibile tramite VBO dinamici.
type RenderOpenGL struct {
	screenWidth      int
	screenHeight     int
	textures         textures.ITextures
	sectorsMaxHeight float64

	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string

	// Hardware Context
	vao            uint32
	vbo            uint32
	bufferCapacity int
}

// NewOpenGLRender inizializza il renderer hardware allocando la VRAM necessaria
// per lo stream dinamico dei vertici frame-by-frame.
func NewOpenGLRender(screenWidth int, screenHeight int, textures textures.ITextures, sectorsMaxHeight float64) *RenderOpenGL {
	r := &RenderOpenGL{
		screenWidth:      screenWidth,
		screenHeight:     screenHeight,
		textures:         textures,
		sectorsMaxHeight: sectorsMaxHeight,
		targetSectors:    map[int]bool{0: true},
		bufferCapacity:   65536 * 32, // Dimensione preallocata per il Vertex Buffer (es. Float32 * attributi)
	}
	r.initGL()
	return r
}

func (r *RenderOpenGL) initGL() {
	// Setup VAO e VBO in modalità GL_DYNAMIC_DRAW
	// gl.GenVertexArrays(1, &r.vao)
	// gl.BindVertexArray(r.vao)
	// gl.GenBuffers(1, &r.vbo)
	// gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	// gl.BufferData(gl.ARRAY_BUFFER, r.bufferCapacity, nil, gl.DYNAMIC_DRAW)

	// Layout: Layout 0 (Pos 3D), Layout 1 (UV), Layout 2 (Light/Normals)
}

// Render processa il grafo di visibilità calcolato dal portal walker.
func (r *RenderOpenGL) Render(vi *model.ViewItem, css []*model.CompiledSector, compiled int) {
	r.targetLastCompiled = compiled
	if compiled < 1 {
		return
	}

	// 1. Setup Uniforms (Matrici di View e Projection dal vi)
	r.updateCameraUniforms(vi)

	// 2. Traversal e allocazione buffer
	r.streamRender(css, compiled, vi)
}

func (r *RenderOpenGL) updateCameraUniforms(vi *model.ViewItem) {
	// Trasmissione al Vertex Shader della Posizione (vi.Where),
	// Pitch (vi.Yaw), e Yaw (vi.AngleSin, vi.AngleCos) come matrici 4x4.
}

// streamRender mappa i poligoni logici su array di vertici, raggruppandoli
// per texture ID per minimizzare le transizioni di stato (Context Switch) in OpenGL.
func (r *RenderOpenGL) streamRender(css []*model.CompiledSector, compiled int, vi *model.ViewItem) {
	// batchMap := make(map[string][]float32)

	for idx := compiled - 1; idx >= 0; idx-- {
		if r.targetEnabled {
			if f, _ := r.targetSectors[idx]; f && r.targetId != css[idx].Sector.Id {
				r.targetId = css[idx].Sector.Id
				fmt.Println("GL Active Sector:", r.targetId)
			}
		}

		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]

			// lightAmbientDist := vi.LightDistance
			// lightDist := cp.Sector.LightDistance

			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				// Costruzione vertici verticali usando p.X1, p.X2 e quote cp.Sector.Floor/Ceil
				// textureId := cp.Texture
				// append(batchMap[textureId], cp.Points3D...)

			case model.IdCeil, model.IdFloor:
				// Costruzione vertici planari topologici
				// tex := cp.Sector.TextureFloor // (o Ceil)
				// append(batchMap[tex], cp.Points3D...)
			}
		}
	}

	// Flush dei batch sulla pipeline hardware
	// gl.BindVertexArray(r.vao)
	// for texID, vertices := range batchMap {
	//    gl.BindTexture(gl.TEXTURE_2D, texID)
	//    gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))
	//    gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)/vertexSize))
	// }
}
