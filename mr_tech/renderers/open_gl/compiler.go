package open_gl

import (
	"embed"
	"fmt"
	"io/fs"
	"math"
	rnd "math/rand"
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
	shaderMain     = 0
	shaderSky      = 1
	shaderSSAO     = 2
	shaderBlur     = 3
	shaderGeometry = 4
	shaderLatest   = 64 // unused
)

const (
	shaderMainView = iota
	shaderMainProj
	shaderMainAmbientLight
	shaderMainProjection
	shaderMainScreenResolution
	shaderMainFlashDir
	shaderMainTexture
	shaderMainNormalMap
	shaderMainSSAO

	shaderSkyProjection
	shaderSkyView
	shaderSkySky

	shaderBlurSSAOInput

	shaderSSAOGPosition
	shaderSSAOGNormal
	shaderSSAOTexNoise
	shaderSSAOSamples
	shaderSSAOProjection

	shaderGeometryTexture
	shaderGeometryView
	shaderGeometryProjection

	shaderLast
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
	shaderPrograms  []uint32
	textures        map[*textures.Texture]glTexture
	ssaoNoiseTex    uint32    // Texture di rumore 4x4
	ssaoKernel      []float32 // 64 campioni vec3
	gBufferFbo      uint32
	gPositionDepth  uint32
	gNormal         uint32
	ssaoFbo         uint32
	ssaoColorBuffer uint32
	ssaoBlurFbo     uint32
	ssaoBlurTexture uint32
	table           [shaderLast]int32
}

// NewCompiler initializes and returns a new instance of Compiler with default shaderPrograms allocation.
func NewCompiler() *Compiler {
	return &Compiler{
		shaderPrograms: make([]uint32, shaderLatest),
	}
}

func (w *Compiler) Setup(width, height int32) {
	w.setupSSAOBuffers(width, height)
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

func (w *Compiler) GetSSAOBlurTexture() uint32 {
	return w.ssaoBlurTexture
}

func (w *Compiler) GetGBufferTextures() (uint32, uint32) {
	return w.gPositionDepth, w.gNormal
}

// GetSSAOResources restituisce la texture di rumore OpenGL e il kernel di campionamento (64 campioni vec3).
func (w *Compiler) GetSSAOResources() (uint32, []float32) {
	return w.ssaoNoiseTex, w.ssaoKernel
}

// Compile compiles all shaders and textures, returning an error if any of the compilation steps fail.
func (w *Compiler) Compile(t textures.ITextures) error {
	if err := w.compileShaders(); err != nil {
		return err
	}
	if err := w.compileTextures(t); err != nil {
		return err
	}

	sm := w.GetShaderProgram(shaderMain)
	w.table[shaderMainView] = gl.GetUniformLocation(sm, gl.Str("u_view\x00"))
	w.table[shaderMainProj] = gl.GetUniformLocation(sm, gl.Str("u_projection\x00"))
	w.table[shaderMainAmbientLight] = gl.GetUniformLocation(sm, gl.Str("u_ambient_light\x00"))
	w.table[shaderMainProjection] = gl.GetUniformLocation(sm, gl.Str("u_projection\x00"))
	w.table[shaderMainScreenResolution] = gl.GetUniformLocation(sm, gl.Str("u_screenResolution\x00"))
	w.table[shaderMainFlashDir] = gl.GetUniformLocation(sm, gl.Str("u_flashDir\x00"))
	w.table[shaderMainTexture] = gl.GetUniformLocation(sm, gl.Str("u_texture\x00"))
	w.table[shaderMainNormalMap] = gl.GetUniformLocation(sm, gl.Str("u_normalMap\x00"))
	w.table[shaderMainSSAO] = gl.GetUniformLocation(sm, gl.Str("u_ssao\x00"))
	w.table[shaderMainTexture] = gl.GetUniformLocation(sm, gl.Str("u_texture\x00"))
	w.table[shaderMainNormalMap] = gl.GetUniformLocation(sm, gl.Str("u_normalMap\x00"))

	ss := w.GetShaderProgram(shaderSky)
	w.table[shaderSkyProjection] = gl.GetUniformLocation(ss, gl.Str("u_projection\x00"))
	w.table[shaderSkyView] = gl.GetUniformLocation(ss, gl.Str("u_view\x00"))
	w.table[shaderSkySky] = gl.GetUniformLocation(ss, gl.Str("u_sky\x00"))

	sb := w.GetShaderProgram(shaderBlur)
	w.table[shaderBlurSSAOInput] = gl.GetUniformLocation(sb, gl.Str("ssaoInput\x00"))

	progSSAO := w.GetShaderProgram(shaderSSAO)
	w.table[shaderSSAOGPosition] = gl.GetUniformLocation(progSSAO, gl.Str("gPosition\x00"))
	w.table[shaderSSAOGNormal] = gl.GetUniformLocation(progSSAO, gl.Str("gNormal\x00"))
	w.table[shaderSSAOTexNoise] = gl.GetUniformLocation(progSSAO, gl.Str("texNoise\x00"))
	w.table[shaderSSAOSamples] = gl.GetUniformLocation(progSSAO, gl.Str("samples\x00"))
	w.table[shaderSSAOProjection] = gl.GetUniformLocation(progSSAO, gl.Str("projection\x00"))

	programGeometry := w.GetShaderProgram(shaderGeometry)
	w.table[shaderGeometryTexture] = gl.GetUniformLocation(programGeometry, gl.Str("u_texture\x00"))
	w.table[shaderGeometryView] = gl.GetUniformLocation(programGeometry, gl.Str("u_view\x00"))
	w.table[shaderGeometryProjection] = gl.GetUniformLocation(programGeometry, gl.Str("u_projection\x00"))

	return nil
}

// compileShaders compiles and links vertex and fragment shaders, storing the resulting programs in shaderPrograms.
func (w *Compiler) compileShaders() error {
	bp := func(s string) string {
		return "assets/" + s
	}
	programs := map[int]ShaderProgram{
		shaderMain:     {vPath: bp("shader_vertex.vert"), fPath: bp("shader_fragment.vert")},
		shaderSky:      {vPath: bp("sky_vertex.vert"), fPath: bp("sky_fragment.vert")},
		shaderSSAO:     {vPath: bp("ssao_vertex.vert"), fPath: bp("ssao.frag")},
		shaderBlur:     {vPath: bp("ssao_vertex.vert"), fPath: bp("ssao_blur.frag")},
		shaderGeometry: {vPath: bp("shader_vertex.vert"), fPath: bp("geometry.frag")}, // Nuovo
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

	// --- 4. SSAO: GENERAZIONE KERNEL (64 campioni) ---
	w.ssaoKernel = make([]float32, 64*3)
	for i := 0; i < 64; i++ {
		sample := [3]float32{
			rnd.Float32()*2.0 - 1.0,
			rnd.Float32()*2.0 - 1.0,
			rnd.Float32(), // Emisfero orientato verso Z+
		}
		// Normalizzazione
		mag := float32(math.Sqrt(float64(sample[0]*sample[0] + sample[1]*sample[1] + sample[2]*sample[2])))
		s := float32(i) / 64.0
		scale := 0.1 + (s*s)*(1.0-0.1) // Lerp per concentrare i campioni vicino all'origine

		w.ssaoKernel[i*3] = (sample[0] / mag) * scale
		w.ssaoKernel[i*3+1] = (sample[1] / mag) * scale
		w.ssaoKernel[i*3+2] = (sample[2] / mag) * scale
	}

	// --- 5. SSAO: GENERAZIONE NOISE TEXTURE (4x4) ---
	noiseData := make([]float32, 16*3)
	for i := 0; i < 16; i++ {
		noiseData[i*3] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+1] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+2] = 0.0
	}

	gl.GenTextures(1, &w.ssaoNoiseTex)
	gl.BindTexture(gl.TEXTURE_2D, w.ssaoNoiseTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB32F, 4, 4, 0, gl.RGB, gl.FLOAT, gl.Ptr(noiseData))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	return nil
}

// SetupSSAOBuffers crea i buffer necessari in base alla risoluzione.
func (w *Compiler) setupSSAOBuffers(width, height int32) {
	// 1. G-Buffer
	gl.GenFramebuffers(1, &w.gBufferFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.gBufferFbo)

	// Position + Depth (RGBA16F per precisione spaziale)
	gl.GenTextures(1, &w.gPositionDepth)
	gl.BindTexture(gl.TEXTURE_2D, w.gPositionDepth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, w.gPositionDepth, 0)

	// Normals
	gl.GenTextures(1, &w.gNormal)
	gl.BindTexture(gl.TEXTURE_2D, w.gNormal)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT1, gl.TEXTURE_2D, w.gNormal, 0)

	// Aggiungi il Depth Renderbuffer
	var rboDepth uint32
	gl.GenRenderbuffers(1, &rboDepth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, rboDepth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, width, height)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, rboDepth)

	attachments := []uint32{gl.COLOR_ATTACHMENT0, gl.COLOR_ATTACHMENT1}
	gl.DrawBuffers(2, &attachments[0])

	// 2. SSAO FBO
	gl.GenFramebuffers(1, &w.ssaoFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.ssaoFbo)
	gl.GenTextures(1, &w.ssaoColorBuffer)
	gl.BindTexture(gl.TEXTURE_2D, w.ssaoColorBuffer)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, width, height, 0, gl.RED, gl.FLOAT, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, w.ssaoColorBuffer, 0)

	// 3. SSAO Blur FBO
	gl.GenFramebuffers(1, &w.ssaoBlurFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.ssaoBlurFbo)
	gl.GenTextures(1, &w.ssaoBlurTexture)
	gl.BindTexture(gl.TEXTURE_2D, w.ssaoBlurTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, width, height, 0, gl.RED, gl.FLOAT, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, w.ssaoBlurTexture, 0)

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}
