package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// glTexture represents an OpenGL texture with its associated IDs for diffuse and normal maps.
// texId is the OpenGL ID of the primary texture.
// normTexId is the OpenGL ID of the normal map texture.
type glTexture struct {
	texId     uint32
	normTexId uint32
}

// Textures represents a collection of textures mapped to their corresponding OpenGL texture identifiers.
type Textures struct {
	textures map[*textures.Texture]glTexture
}

// NewTextures initializes and returns a new Textures instance with an empty map to store texture mappings.
func NewTextures() *Textures {
	return &Textures{
		textures: make(map[*textures.Texture]glTexture),
	}
}

// Get retrieves the OpenGL texture ID, normal texture ID, and existence status for the given texture.
func (w *Textures) Get(tex *textures.Texture) (uint32, uint32, bool) {
	t, ok := w.textures[tex]
	return t.texId, t.normTexId, ok
}

// Setup initializes the textures by uploading diffuse and normal maps to the GPU using OpenGL and stores them in the Textures map.
func (w *Textures) Setup(t textures.ITextures) error {
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
