package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// glTexture represents a texture layer index within a 3D texture array.
// layer specifies the index of the layer in the 3D texture array, ready for use in the Vertex Buffer.
type glTexture struct {
	layer float32 // Indice del layer nell'Array 3D (float32 pronto per il Vertex Buffer)
}

// Textures provides methods and structures for managing a collection of 2D texture layers used in rendering operations.
// It organizes textures in a map and handles creation and allocation of texture arrays for diffuse, normal, and emissive data.
type Textures struct {
	textures      map[*textures.Texture]glTexture
	diffuseArray  uint32
	normalArray   uint32
	emissiveArray uint32
}

// NewTextures creates and initializes a new instance of Textures with an empty texture-to-glTexture mapping.
func NewTextures() *Textures {
	return &Textures{
		textures: make(map[*textures.Texture]glTexture),
	}
}

// Get retrieves the texture layer associated with the given texture and returns a boolean indicating its presence.
func (tx *Textures) Get(tex *textures.Texture) (float32, bool) {
	t, ok := tx.textures[tex]
	return t.layer, ok
}

// GetDiffuseArray returns the OpenGL texture ID of the diffuse texture array stored in the Textures instance.
func (tx *Textures) GetDiffuseArray() uint32 { return tx.diffuseArray }

// GetNormalArray returns the texture array ID used for storing normal maps as a uint32.
func (tx *Textures) GetNormalArray() uint32 { return tx.normalArray }

// GetEmissiveArray retrieves the OpenGL ID of the emissive texture 2D array used for rendering.
func (tx *Textures) GetEmissiveArray() uint32 { return tx.emissiveArray }

// Setup initializes the texture arrays and maps, allocates memory for texture storage, and uploads texture data to the GPU.
func (tx *Textures) Setup(t textures.ITextures) error {
	tx.textures = make(map[*textures.Texture]glTexture)
	names := t.GetNames()
	layerCount := int32(len(names))

	if layerCount == 0 {
		return nil
	}

	// 1. TROVA LE DIMENSIONI MASSIME (o imposta un 256x256 fisso)
	var maxWidth, maxHeight int
	for _, id := range names {
		tn := t.Get([]string{id})
		if tn != nil && len(tn) > 0 {
			w, h := tn[0].Size()
			if w > maxWidth {
				maxWidth = w
			}
			if h > maxHeight {
				maxHeight = h
			}
		}
	}

	// Evita dimensioni strane, forza potenze di due per sicurezza mipmap
	maxWidth = nextPowerOfTwo(maxWidth)
	maxHeight = nextPowerOfTwo(maxHeight)

	// 2. ALLOCAZIONE VRAM DEI 3 ARRAY
	tx.diffuseArray = createTextureArray(maxWidth, maxHeight, layerCount)
	tx.normalArray = createTextureArray(maxWidth, maxHeight, layerCount)
	tx.emissiveArray = createTextureArray(maxWidth, maxHeight, layerCount)

	blackPixels := make([]uint8, maxWidth*maxHeight*4)
	for i := 0; i < len(blackPixels); i += 4 {
		blackPixels[i] = 0
		blackPixels[i+1] = 0
		blackPixels[i+2] = 0
		blackPixels[i+3] = 255
	}

	// 3. POPOLAMENTO LAYER
	layerIndex := int32(0)
	for _, id := range names {
		tn := t.Get([]string{id})
		if tn == nil || len(tn) == 0 {
			continue
		}
		tex := tn[0]
		origW, origH, pixels := tex.RGBA()

		// Resize dei pixel in CPU se la texture è più piccola del Layer
		diffusePixels := resizePixels(pixels, origW, origH, maxWidth, maxHeight)

		// Upload Diffuse Layer
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.diffuseArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layerIndex, int32(maxWidth), int32(maxHeight), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(diffusePixels))

		// Generazione Normali e Upload Normal Layer
		normalPixels := generateNormalMap(diffusePixels, maxWidth, maxHeight, 3.0)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.normalArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layerIndex, int32(maxWidth), int32(maxHeight), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(normalPixels))

		// Upload Emissive Layer (Nero di default)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.emissiveArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layerIndex, int32(maxWidth), int32(maxHeight), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(blackPixels))

		// Salva l'indice!
		tx.textures[tex] = glTexture{layer: float32(layerIndex)}
		layerIndex++
	}

	// 4. GENERAZIONE MIPMAP GLOBALI (Ora sicura)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.diffuseArray)
	gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)

	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.normalArray)
	gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)

	return nil
}

// -- Helper Functions --

// createTextureArray initializes and returns a 2D texture array with specified dimensions and number of layers.
func createTextureArray(width, height int, layers int32) uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tex)

	// Calcolo dei livelli MipMap necessari
	maxDim := float64(width)
	if height > width {
		maxDim = float64(height)
	}
	mipCount := int32(math.Floor(math.Log2(maxDim))) + 1

	// Fissa i limiti dei MipMap PRIMA dell'allocazione
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_BASE_LEVEL, 0)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAX_LEVEL, mipCount-1)

	// Allocazione esplicita "Immutable-Style" (Compatibilità OpenGL 3.3)
	for i := int32(0); i < mipCount; i++ {
		w := int32(width >> i)
		h := int32(height >> i)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		gl.TexImage3D(gl.TEXTURE_2D_ARRAY, i, gl.RGBA8, w, h, layers, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	}

	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_T, gl.REPEAT)
	// Anisotropic filtering
	gl.TexParameterf(gl.TEXTURE_2D_ARRAY, 0x84FE, 8.0)
	return tex
}

// generateNormalMap generates a normal map from the input pixel data using the specified width, height, and strength factor.
func generateNormalMap(pixels []uint8, width, height int, strength float64) []uint8 {
	normPixels := make([]uint8, len(pixels))
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
	return normPixels
}

// resizePixels resizes a source RGBA pixel array to a new width and height using nearest-neighbor scaling.
func resizePixels(src []uint8, oldW, oldH, newW, newH int) []uint8 {
	if oldW == newW && oldH == newH {
		return src
	}
	dst := make([]uint8, newW*newH*4)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := (x * oldW) / newW
			srcY := (y * oldH) / newH
			srcIdx := (srcY*oldW + srcX) * 4
			dstIdx := (y*newW + x) * 4
			copy(dst[dstIdx:dstIdx+4], src[srcIdx:srcIdx+4])
		}
	}
	return dst
}

// nextPowerOfTwo returns the smallest power of two greater than or equal to the given integer v.
func nextPowerOfTwo(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
