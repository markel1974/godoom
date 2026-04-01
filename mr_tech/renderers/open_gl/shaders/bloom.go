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
	hvPassages       int32
	internalPassages int32
	vao              uint32
	vbo              uint32
	w                int32
	h                int32
}

// NewBloom creates and returns a new instance of the Bloom structure.
func NewBloom() *Bloom {
	return &Bloom{
		hvPassages:       5, //passaggi orizzontali e verticali
		internalPassages: 3,
	}
}

func (s *Bloom) GetBloomTexture() uint32 {
	return s.pingPongTex[1] // L'ultima scrittura cade sull'indice 1
}

// Init initializes the Bloom effect by setting up its resources and ensuring it is ready for rendering operations.
func (s *Bloom) Init() error {
	return nil
}

// SetupSamplers initializes the VAO and VBO for rendering a full-screen quad and configures vertex attribute pointers.
func (s *Bloom) SetupSamplers() error {
	gl.GenVertexArrays(1, &s.vao)
	gl.BindVertexArray(s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)
	return nil
}

// Compile initializes the bloom effect by compiling shaders, creating a shader program, and setting up framebuffer resources.
func (s *Bloom) Compile(a IAssets) error {
	const vertId = "post.vert"
	const fragId = "bloom.frag"
	vSrc, fSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(vertId, string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(fragId, string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram("bloom", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[BloomLocImage] = gl.GetUniformLocation(s.prg, gl.Str("image\x00"))
	s.table[BloomLocHorizontal] = gl.GetUniformLocation(s.prg, gl.Str("horizontal\x00"))
	s.table[BloomLocPassage] = gl.GetUniformLocation(s.prg, gl.Str("u_passages\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in bloom: %d", idx)
		}
	}
	return nil
}

// Render applies a multi-pass Gaussian blur to the bright regions of the texture and returns the final blurred texture ID.
func (s *Bloom) Render(brightTex uint32, fbw, fbh int32) {
	if fbw != s.w || fbh != s.h {
		s.allocate(fbw, fbh)
	}

	gl.UseProgram(s.prg)
	gl.Uniform1i(s.table[BloomLocImage], 0)
	gl.BindVertexArray(s.vao)
	gl.Disable(gl.DEPTH_TEST)
	gl.Viewport(0, 0, fbw/2, fbh/2)

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

	gl.Viewport(0, 0, fbw, fbh)
	gl.Enable(gl.DEPTH_TEST)
}

// allocate resizes the bloom effect textures and framebuffers to match the specified width and height values.
func (s *Bloom) allocate(width, height int32) {
	s.w = width
	s.h = height

	// Prevenzione memory leak al ridimensionamento
	if s.pingPongFbo[0] != 0 {
		gl.DeleteFramebuffers(2, &s.pingPongFbo[0])
		gl.DeleteTextures(2, &s.pingPongTex[0])
	}

	gl.GenFramebuffers(2, &s.pingPongFbo[0])
	gl.GenTextures(2, &s.pingPongTex[0])

	mipW := s.w / 2
	mipH := s.h / 2

	for i := 0; i < 2; i++ {
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.pingPongFbo[i])
		gl.BindTexture(gl.TEXTURE_2D, s.pingPongTex[i])

		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, mipW, mipH, 0, gl.RGBA, gl.FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.pingPongTex[i], 0)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}
