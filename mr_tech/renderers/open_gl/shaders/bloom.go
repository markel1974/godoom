package shaders

import (
	"fmt"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
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
	ctx              api.IContext
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
	handler          *Handler
	post             *Post
	quad             []float32
	quadPtr          unsafe.Pointer
}

// NewBloom creates and returns a new instance of the Bloom structure.
func NewBloom() *Bloom {
	b := &Bloom{}
	return b
}

// Setup initializes the Bloom instance by assigning the rendering context from the provided Handler.
func (s *Bloom) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.post = h.GetPost()
	s.hvPassages = h.GetHvPassages()
	s.internalPassages = h.GetInternalPassages()

	s.quad = []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.quadPtr = s.ctx.Ptr(s.quad)
	return nil
}

// GetBloomTexture retrieves the OpenGL texture ID of the final bloom-texture used in the bloom post-processing effect.
func (s *Bloom) GetBloomTexture() uint32 {
	return s.pingPongTex[1] // L'ultima scrittura cade sull'indice 1
}

// Init initializes the Bloom effect by setting up its resources and ensuring it is ready for rendering operations.
func (s *Bloom) Init() error {
	return nil
}

// SetupSamplers initializes the VAO and VBO for rendering a full-screen quad and configures vertex attribute pointers.
func (s *Bloom) SetupSamplers() error {
	s.ctx.GenVertexArrays(1, &s.vao)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.GenBuffers(1, &s.vbo)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.vbo)

	s.ctx.BufferData(api.ARRAY_BUFFER, len(s.quad)*4, s.quadPtr, api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
	s.ctx.EnableVertexAttribArray(0)
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

	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(s.ctx, "bloom", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[BloomLocImage] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("image\x00"))
	s.table[BloomLocHorizontal] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("horizontal\x00"))
	s.table[BloomLocPassage] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_passages\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in bloom: %d", idx)
		}
	}
	return nil
}

// Render applies a multi-pass Gaussian blur to the bright regions of the texture and returns the final blurred texture ID.
func (s *Bloom) Render() {
	fbW, fbH := s.handler.GetSize()
	brightTex := s.post.GetBrightBuffer()

	if fbW != s.w || fbH != s.h {
		s.allocate(fbW, fbH)
	}

	if s.handler.GetBloomIntensity() <= 0.001 || s.hvPassages <= 0 {
		return
	}

	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.table[BloomLocImage], 0)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.Disable(api.DEPTH_TEST)
	s.ctx.Viewport(0, 0, fbW/2, fbH/2)

	horizontal := true
	firstIteration := true

	for i := int32(0); i < s.hvPassages; i++ {
		idx := 0
		if !horizontal {
			idx = 1
		}
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.pingPongFbo[idx])

		val := int32(0)
		if horizontal {
			val = 1
		}
		s.ctx.Uniform1i(s.table[BloomLocHorizontal], val)
		s.ctx.Uniform1i(s.table[BloomLocPassage], s.internalPassages)

		s.ctx.ActiveTexture(api.TEXTURE0)
		if firstIteration {
			s.ctx.BindTexture(api.TEXTURE_2D, brightTex)
			firstIteration = false
		} else {
			prevIdx := 1
			if !horizontal {
				prevIdx = 0
			}
			s.ctx.BindTexture(api.TEXTURE_2D, s.pingPongTex[prevIdx])
		}

		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
		horizontal = !horizontal
	}

	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.Enable(api.DEPTH_TEST)
}

// allocate resizes the bloom effect textures and framebuffers to match the specified width and height values.
func (s *Bloom) allocate(width, height int32) {
	s.w = width
	s.h = height

	// Prevenzione memory leak al ridimensionamento
	if s.pingPongFbo[0] != 0 {
		s.ctx.DeleteFramebuffers(2, &s.pingPongFbo[0])
		s.ctx.DeleteTextures(2, &s.pingPongTex[0])
	}

	s.ctx.GenFramebuffers(2, &s.pingPongFbo[0])
	s.ctx.GenTextures(2, &s.pingPongTex[0])

	mipW := s.w / 2
	mipH := s.h / 2

	for i := 0; i < 2; i++ {
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.pingPongFbo[i])
		s.ctx.BindTexture(api.TEXTURE_2D, s.pingPongTex[i])

		s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, mipW, mipH, 0, api.RGBA, api.FLOAT, nil)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
		s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.pingPongTex[i], 0)
	}
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}
