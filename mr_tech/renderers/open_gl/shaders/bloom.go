package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// BloomLoc represents a type used to define specific locations in a Bloom filter system.
type BloomLoc int

// BloomLocImage represents the image location for a bloom effect.
// BloomLocHorizontal represents the horizontal location for a bloom effect.
// BloomLocLast represents the last location for a bloom effect.
const (
	BloomLocImage = BloomLoc(iota)
	BloomLocHorizontal
	BloomLocPassage
	BloomLocLast
)

// Bloom is a struct that encapsulates data and methods for managing bloom post-processing effects in a graphics engine.
type Bloom struct {
	prg              uint32
	table            [BloomLocLast]int32
	pingPongFbo      [2]uint32
	pingPongTex      [2]uint32
	width            int32
	height           int32
	hvPassages       int32
	internalPassages int32
	vao              uint32
	vbo              uint32
}

// NewBloom creates and returns a new instance of the Bloom structure.
func NewBloom() *Bloom {
	return &Bloom{
		hvPassages:       5, //passaggi orizzontali e verticali
		internalPassages: 3,
	}
}

// Setup initializes the Bloom structure with the specified width and height values.
func (s *Bloom) Setup(width, height int32) {
	s.width = width
	s.height = height
}

// SetupSamplers initializes the VAO and VBO for rendering a full-screen quad and configures vertex attribute pointers.
func (s *Bloom) SetupSamplers() {
	gl.GenVertexArrays(1, &s.vao)
	gl.BindVertexArray(s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)
}

// Compile initializes the bloom effect by compiling shaders, creating a shader program, and setting up framebuffer resources.
func (s *Bloom) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("post.vert", "bloom.frag")
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(vSh, fSh)
	if err != nil {
		return err
	}

	s.table[BloomLocImage] = gl.GetUniformLocation(s.prg, gl.Str("image\x00"))
	s.table[BloomLocHorizontal] = gl.GetUniformLocation(s.prg, gl.Str("horizontal\x00"))
	s.table[BloomLocPassage] = gl.GetUniformLocation(s.prg, gl.Str("u_passages\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}

	gl.GenFramebuffers(2, &s.pingPongFbo[0])
	gl.GenTextures(2, &s.pingPongTex[0])
	for i := 0; i < 2; i++ {
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.pingPongFbo[i])
		gl.BindTexture(gl.TEXTURE_2D, s.pingPongTex[i])
		// Downsample a metà risoluzione (massimizza banda e diffusione)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, s.width/2, s.height/2, 0, gl.RGBA, gl.FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.pingPongTex[i], 0)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return nil
}

// Render applies a multi-pass Gaussian blur to the bright regions of the texture and returns the final blurred texture ID.
func (s *Bloom) Render(brightTex uint32) uint32 {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.table[BloomLocImage], 0)
	gl.BindVertexArray(s.vao)
	gl.Disable(gl.DEPTH_TEST)
	gl.Viewport(0, 0, s.width/2, s.height/2)

	horizontal := true
	firstIteration := true

	for i := int32(0); i < s.hvPassages; i++ {
		idx := 0
		if !horizontal {
			idx = 1
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.pingPongFbo[idx])

		val := int32(0)
		if horizontal {
			val = 1
		}
		gl.Uniform1i(s.table[BloomLocHorizontal], val)
		gl.Uniform1i(s.table[BloomLocPassage], s.internalPassages)

		gl.ActiveTexture(gl.TEXTURE0)
		if firstIteration {
			gl.BindTexture(gl.TEXTURE_2D, brightTex)
			firstIteration = false
		} else {
			prevIdx := 1
			if !horizontal {
				prevIdx = 0
			}
			gl.BindTexture(gl.TEXTURE_2D, s.pingPongTex[prevIdx])
		}

		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		horizontal = !horizontal
	}

	gl.Viewport(0, 0, s.width, s.height)
	gl.Enable(gl.DEPTH_TEST)
	return s.pingPongTex[1] // L'ultima scrittura cade sull'indice 1
}
