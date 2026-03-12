package open_gl

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/engine/textures"
)

// assets is an embedded file system containing resources such as shaders and textures, provided via the embed.FS package.
//
//go:embed assets
var assets embed.FS

// shaderMain represents the identifier for the main shader.
// shaderSky represents the identifier for the sky shader.
const (
	shaderMain = 0
	shaderSky  = 1
)

// ShaderProgram represents a shader program with associated vertex and fragment shader file paths and an ID.
type ShaderProgram struct {
	id    int
	vPath string
	fPath string
}

// Compiler manages the compilation of shader programs and OpenGL textures for rendering operations.
type Compiler struct {
	shaderPrograms []uint32
	textures       map[*textures.Texture]uint32
}

// NewCompiler initializes and returns a new instance of Compiler with default shaderPrograms allocation.
func NewCompiler() *Compiler {
	return &Compiler{
		shaderPrograms: make([]uint32, 2),
	}
}

// GetShaderProgram returns the shader program ID associated with the given shader ID.
func (w *Compiler) GetShaderProgram(shaderId int) uint32 {
	return w.shaderPrograms[shaderId]
}

// GetTexture retrieves the OpenGL texture ID associated with the given texture, returning false if not found.
func (w *Compiler) GetTexture(tex *textures.Texture) (uint32, bool) {
	t, ok := w.textures[tex]
	return t, ok
}

// Compile compiles all shaders and textures, returning an error if any of the compilation steps fail.
func (w *Compiler) Compile(t textures.ITextures) error {
	if err := w.compileShaders(); err != nil {
		return err
	}
	if err := w.compileTextures(t); err != nil {
		return err
	}
	return nil
}

// compileShaders compiles and links vertex and fragment shaders, storing the resulting programs in shaderPrograms.
func (w *Compiler) compileShaders() error {
	bp := func(s string) string {
		return "assets/" + s
	}
	programs := map[int]ShaderProgram{
		shaderMain: {vPath: bp("shader_vertex.vert"), fPath: bp("shader_fragment.vert")},
		shaderSky:  {vPath: bp("sky_vertex.vert"), fPath: bp("sky_fragment.vert")},
	}

	for shaderId, data := range programs {
		vertexSrc, err := fs.ReadFile(assets, data.vPath)
		if err != nil {
			return err
		}
		fragmentSrc, err := fs.ReadFile(assets, data.fPath)
		if err != nil {
			return err
		}
		vertexShader, err := w.compileShader(string(vertexSrc), gl.VERTEX_SHADER)
		if err != nil {
			return err
		}
		fragmentShader, err := w.compileShader(string(fragmentSrc), gl.FRAGMENT_SHADER)
		if err != nil {
			gl.DeleteShader(vertexShader)
			return err
		}
		shaderProgram := gl.CreateProgram()
		gl.AttachShader(shaderProgram, vertexShader)
		gl.AttachShader(shaderProgram, fragmentShader)
		gl.LinkProgram(shaderProgram)
		var status int32
		gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
		if status == gl.FALSE {
			return fmt.Errorf("failed to link shader program")
		}
		gl.UseProgram(shaderProgram)
		w.shaderPrograms[shaderId] = shaderProgram
		gl.DeleteShader(fragmentShader)
		gl.DeleteShader(vertexShader)
	}
	return nil
}

// compileShader compiles a shader from source code and shader type, returning the compiled shader ID or an error.
func (w *Compiler) compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	cSources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, cSources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}

// compileTextures processes and uploads 2D texture data to the GPU, generating OpenGL texture objects for rendering.
func (w *Compiler) compileTextures(t textures.ITextures) error {
	w.textures = make(map[*textures.Texture]uint32)
	for _, id := range t.GetNames() {
		tn := t.Get([]string{id})
		if tn == nil {
			continue
		}
		tex := tn[0]
		glTex := uint32(0)
		width, height, glPixels := tex.RGBA()
		gl.GenTextures(1, &glTex)
		gl.BindTexture(gl.TEXTURE_2D, glTex)
		w.textures[tex] = glTex

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(glPixels))

		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

		// 2. Tenta il filtro anisotropico via brute-force (valore 0x84FE)
		// Questo rimuove il blur "fangoso" sulle texture dei muri viste radenti
		gl.TexParameterf(gl.TEXTURE_2D, 0x84FE, 4.0)

		//var maxAnisotropy float32
		//gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY_EXT, &maxAnisotropy)
		//gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY_EXT, maxAnisotropy)
	}
	return nil
}
