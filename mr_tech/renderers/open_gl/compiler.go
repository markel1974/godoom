package open_gl

import (
	"embed"
	"fmt"
	"io/fs"
	"math"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
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

type glTexture struct {
	texId     uint32
	normTexId uint32
}

// Compiler manages the compilation of shader programs and OpenGL textures for rendering operations.
type Compiler struct {
	shaderPrograms []uint32
	textures       map[*textures.Texture]glTexture
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
func (w *Compiler) GetTexture(tex *textures.Texture) (uint32, uint32, bool) {
	t, ok := w.textures[tex]
	return t.texId, t.normTexId, ok
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
// compileTextures processes 2D texture data, uploads it to the GPU, and procedurally generates/uploads Normal Maps.
func (w *Compiler) compileTextures(t textures.ITextures) error {
	w.textures = make(map[*textures.Texture]glTexture)

	for _, id := range t.GetNames() {
		tn := t.Get([]string{id})
		if tn == nil {
			continue
		}
		tex := tn[0]
		width, height, pixels := tex.RGBA()

		// --- 1. UPLOAD DIFFUSE MAP ---
		var glTex uint32
		gl.GenTextures(1, &glTex)
		gl.BindTexture(gl.TEXTURE_2D, glTex)
		//w.textures[tex] = glTex

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, 0x84FE, 4.0) // Anisotropic filter

		// --- 2. GENERAZIONE NORMALI (Filtro Sobel inline su glPixels) ---
		normPixels := make([]uint8, len(pixels))
		const strength = 3.0 // Intensità del bump

		luma := func(x, y int) float64 {
			if x < 0 {
				x = 0
			} else if x >= width {
				x = width - 1
			}
			if y < 0 {
				y = 0
			} else if y >= height {
				y = height - 1
			}
			idx := (y*width + x) * 4
			return (0.299*float64(pixels[idx]) + 0.587*float64(pixels[idx+1]) + 0.114*float64(pixels[idx+2])) / 255.0
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dX := (luma(x+1, y) - luma(x-1, y)) * strength
				dY := (luma(x, y+1) - luma(x, y-1)) * strength
				dZ := 1.0

				invLen := 1.0 / math.Sqrt(dX*dX+dY*dY+dZ*dZ)
				nX := dX * invLen
				nY := dY * invLen
				nZ := dZ * invLen

				idx := (y*width + x) * 4
				normPixels[idx] = uint8((nX + 1.0) * 127.5)
				normPixels[idx+1] = uint8((-nY + 1.0) * 127.5) // Inversione Y per lo spazio OpenGL
				normPixels[idx+2] = uint8((nZ + 1.0) * 127.5)
				normPixels[idx+3] = 255
			}
		}

		// --- 3. UPLOAD NORMAL MAP ---
		var glNormTex uint32
		gl.GenTextures(1, &glNormTex)
		gl.BindTexture(gl.TEXTURE_2D, glNormTex)

		w.textures[tex] = glTexture{texId: glTex, normTexId: glNormTex}

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(normPixels))
		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, 0x84FE, 4.0)
	}
	return nil
}
