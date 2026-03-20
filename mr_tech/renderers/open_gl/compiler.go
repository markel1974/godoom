package open_gl

import (
	"embed"
	"io/fs"
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// assets represents an embedded file system containing application resources such as shaders or assets.
//
//go:embed assets
var assets embed.FS

/*
// shaderMain represents the identifier for the main shader.
// shaderSky represents the identifier for the sky shader.
const (
	shaderMain     = 0
	shaderSky      = 1
	shaderSSAO     = 2
	shaderBlur     = 3
	shaderGeometry = 4
	shaderLatest   = 64 // unused
)

*/

// ShaderProgram represents a compiled and linked GPU shader program, encapsulating vertex and fragment shaders.
type ShaderProgram struct {
	id    int
	vPath string
	fPath string
}

// glTexture represents an OpenGL texture with identifiers for diffuse and normal maps.
// texId is the OpenGL-generated identifier for the diffuse texture.
// normTexId is the OpenGL-generated identifier for the normal map texture.
type glTexture struct {
	texId     uint32
	normTexId uint32
}

// Compiler represents a structure responsible for compiling and managing shaders and textures in a rendering pipeline.
type Compiler struct {
	textures       map[*textures.Texture]glTexture
	shaderMain     *ShaderMain
	shaderSky      *ShaderSky
	shaderSSAO     *ShaderSSAO
	shaderBlur     *ShaderBlur
	shaderGeometry *ShaderGeometry
}

// NewCompiler initializes and returns a new instance of Compiler with preconfigured shader objects.
func NewCompiler() *Compiler {
	return &Compiler{
		shaderMain:     NewShaderMain(),
		shaderSky:      NewShaderSky(),
		shaderSSAO:     NewShaderSSAO(),
		shaderBlur:     NewShaderBlur(),
		shaderGeometry: NewShaderGeometry(),
	}
}

// Setup initializes all shader programs associated with the Compiler with the specified width and height.
func (w *Compiler) Setup(width, height int32) {
	w.shaderMain.Setup(width, height)
	w.shaderSky.Setup(width, height)
	w.shaderSSAO.Setup(width, height)
	w.shaderBlur.Setup(width, height)
	w.shaderGeometry.Setup(width, height)
}

// GetTexture retrieves texture and normal texture IDs for the given texture and indicates if it was found in the cache.
func (w *Compiler) GetTexture(tex *textures.Texture) (uint32, uint32, bool) {
	t, ok := w.textures[tex]
	return t.texId, t.normTexId, ok
}

// SetupSamplers configures samplers for all associated shaders in the Compiler, preparing them for rendering tasks.
func (w *Compiler) SetupSamplers() {
	w.shaderSky.SetupSamplers()
	w.shaderSSAO.SetupSamplers()
	w.shaderBlur.SetupSamplers()
	w.shaderMain.SetupSamplers()
	w.shaderGeometry.SetupSamplers()
}

// CompileShaders compiles and links all shaders required for the application, returning an error if any step fails.
func (w *Compiler) CompileShaders() error {
	mainV, mainF, err := w.read("main.vert", "main.frag")
	if err != nil {
		return err
	}
	slyV, skyF, err := w.read("sky.vert", "sky.frag")
	if err != nil {
		return err
	}
	ssaoV, ssaoF, err := w.read("ssao.vert", "ssao.frag")
	if err != nil {
		return err
	}
	blurV, blurF, err := w.read("ssao.vert", "ssao_blur.frag")
	if err != nil {
		return err
	}
	geometryV, geometryF, err := w.read("main.vert", "geometry.frag")
	if err != nil {
		return err
	}
	if err = w.shaderMain.Compile(mainV, mainF); err != nil {
		return err
	}
	if err = w.shaderSky.Compile(slyV, skyF); err != nil {
		return err
	}
	if err = w.shaderSSAO.Compile(ssaoV, ssaoF); err != nil {
		return err
	}
	if err = w.shaderBlur.Compile(blurV, blurF); err != nil {
		return err
	}
	if err = w.shaderGeometry.Compile(geometryV, geometryF); err != nil {
		return err
	}
	return nil
}

// CompileTextures uploads textures and normal maps to the GPU, generating OpenGL texture objects for rendering.
func (w *Compiler) CompileTextures(t textures.ITextures) error {
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
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, 0x84FE, 4.0)

		// --- 2. GENERAZIONE NORMALI (Sobel Filter) ---
		normPixels := make([]uint8, len(pixels))
		const strength = 3.0

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
				invLen := 1.0 / math.Sqrt(dX*dX+dY*dY+1.0)

				idx := (y*width + x) * 4
				normPixels[idx] = uint8(((dX * invLen) + 1.0) * 127.5)
				normPixels[idx+1] = uint8(((-dY * invLen) + 1.0) * 127.5)
				normPixels[idx+2] = uint8((invLen + 1.0) * 127.5)
				normPixels[idx+3] = 255
			}
		}

		// --- 3. UPLOAD NORMAL MAP ---
		var glNormTex uint32
		gl.GenTextures(1, &glNormTex)
		gl.BindTexture(gl.TEXTURE_2D, glNormTex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(normPixels))
		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

		w.textures[tex] = glTexture{texId: glTex, normTexId: glNormTex}
	}

	return nil
}

// read reads the contents of vertex and fragment shader files from the asset file system and returns them as strings.
// It takes the paths to the vertex and fragment shaders as input and returns any errors encountered during file reading.
func (w *Compiler) read(vPath, fPath string) (string, string, error) {
	bp := func(s string) string {
		return "assets/" + s
	}
	vertexSrc, err := fs.ReadFile(assets, bp(vPath))
	if err != nil {
		return "", "", err
	}
	fragmentSrc, err := fs.ReadFile(assets, bp(fPath))
	if err != nil {
		return "", "", err
	}
	return string(vertexSrc), string(fragmentSrc), nil
}
