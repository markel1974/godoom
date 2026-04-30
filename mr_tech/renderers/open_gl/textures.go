package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Bucket represents a structure used for storing texture-related arrays and metadata for rendering operations.
type Bucket struct {
	Size          int
	Count         int32
	Layer         int32
	DiffuseArray  uint32
	NormalArray   uint32
	EmissiveArray uint32
}

// Textures manages a collection of 2D textures and their associated VRAM buckets for efficient rendering and sampling.
type Textures struct {
	textures map[*textures.Texture]float32
	buckets  []Bucket
}

// NewTextures creates and returns a new instance of Textures with an initialized map to store texture references.
func NewTextures() *Textures {
	return &Textures{
		textures: make(map[*textures.Texture]float32),
	}
}

// Get retrieves the layer value associated with the given texture and returns a boolean indicating if the texture exists.
func (tx *Textures) Get(tex *textures.Texture) (float32, bool) {
	t, ok := tx.textures[tex]
	return t, ok
}

// GetBucketsLen returns the number of buckets in the Textures structure.
func (tx *Textures) GetBucketsLen() int {
	return len(tx.buckets)
}

// GetBucket retrieves the DiffuseArray, NormalArray, and EmissiveArray from the specified bucket index b.
func (tx *Textures) GetBucket(b int) (uint32, uint32, uint32) {
	return tx.buckets[b].DiffuseArray, tx.buckets[b].NormalArray, tx.buckets[b].EmissiveArray
}

// Setup initializes texture buckets, allocates memory, and processes textures for VRAM usage and mipmap generation.
func (tx *Textures) Setup(t textures.ITextures) error {
	const stride = 4
	tx.textures = make(map[*textures.Texture]float32)
	names := t.GetNames()

	if len(names) == 0 {
		return nil
	}

	slots := []int{64, 128, 256, 1024}

	idxByDim := func(maxDim int) int {
		bIdx := len(slots) - 1
		for x := 0; x < bIdx; x++ {
			if maxDim <= slots[x] {
				return x
			}
		}
		return bIdx
	}

	tx.buckets = make([]Bucket, len(slots))
	for i := range slots {
		tx.buckets[i] = Bucket{Size: slots[i], Count: 0}
	}

	for _, id := range names {
		tn := t.Get([]string{id})
		if len(tn) == 0 {
			continue
		}
		w, h := tn[0].Size()
		maxDim := w
		if h > maxDim {
			maxDim = h
		}
		idx := idxByDim(maxDim)
		tx.buckets[idx].Count++
	}

	// 3. Allocazione VRAM selettiva
	for i := range tx.buckets {
		if tx.buckets[i].Count > 0 {
			tx.buckets[i].DiffuseArray = createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].NormalArray = createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].EmissiveArray = createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
		}
	}

	// 4. Popolamento e Packing
	for _, id := range names {
		tn := t.Get([]string{id})
		if len(tn) == 0 {
			continue
		}
		tex := tn[0]
		w, h, pixels := tex.RGBA()
		maxDim := w
		if h > maxDim {
			maxDim = h
		}
		bIdx := idxByDim(maxDim)
		size := tx.buckets[bIdx].Size
		layer := tx.buckets[bIdx].Layer
		//UpscaleFixedPoint UpscaleBicubic UpscaleLanczosSeparable
		resizedPixels := UpscaleBicubic(pixels, w, h, size, size, stride)
		//normalPixels := generateNormalMap(resizedPixels, size, size, stride, 3.0)
		normalPixels := generateNormalMapScharr(resizedPixels, size, size, stride, 7.0)
		blackPixels := createBlackPixels(size, size, stride)

		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[bIdx].DiffuseArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(resizedPixels))

		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[bIdx].NormalArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(normalPixels))

		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[bIdx].EmissiveArray)
		if len(id) > 0 && (id[0] == '*' || id[0] == '+') {
			gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(resizedPixels))
		} else {
			gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(blackPixels))
		}
		//Impacchettiamo Bucket e Layer in un solo Float
		packedValue := float32(bIdx*1000) + float32(layer)
		tx.textures[tex] = packedValue
		tx.buckets[bIdx].Layer++
	}

	// 5. Generazione MipMaps globali (aggiunto l'emissivo!)
	for i, l := range tx.buckets {
		if l.Layer > 0 {
			gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[i].DiffuseArray)
			gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)
			gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[i].NormalArray)
			gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)
			gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.buckets[i].EmissiveArray)
			gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)
		}
	}
	return nil
}

// createTextureArray creates a texture array with specified width, height, and number of layers, and sets up mipmaps and filtering.
func createTextureArray(width, height int, layers int32) uint32 {
	// computeMipMapLevel calculates the number of mipmap levels based on the maximum dimension of a texture (width or height).
	computeMipMapLevel := func(width, height int) int32 {
		maxDim := float64(width)
		if height > width {
			maxDim = float64(height)
		}
		mipCount := int32(math.Floor(math.Log2(maxDim))) + 1
		return mipCount
	}
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tex)

	mipCount := computeMipMapLevel(width, height)

	// Set MipMap limits BEFORE allocation
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_BASE_LEVEL, 0)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAX_LEVEL, mipCount-1)

	// Explicit "Immutable-Style" allocation
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

	//gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	//gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_T, gl.REPEAT)
	// Anisotropic filtering
	var maxAniso float32
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &maxAniso)
	gl.TexParameterf(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAX_ANISOTROPY, maxAniso)

	return tex
}

// generateNormalMap creates a normal map from image pixel data, applying luminance-based calculations and strength scaling.
func generateNormalMap(pixels []uint8, width, height, stride int, strength float64) []uint8 {
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
		idx := (y*width + x) * stride
		return (0.299*float64(pixels[idx]) + 0.587*float64(pixels[idx+1]) + 0.114*float64(pixels[idx+2])) / 255.0
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dX := (luma(x+1, y) - luma(x-1, y)) * strength
			dY := (luma(x, y+1) - luma(x, y-1)) * strength
			invLen := 1.0 / math.Sqrt(dX*dX+dY*dY+1.0)
			idx := (y*width + x) * stride
			normPixels[idx] = uint8(((dX * invLen) + 1.0) * 127.5)
			normPixels[idx+1] = uint8(((-dY * invLen) + 1.0) * 127.5)
			normPixels[idx+2] = uint8((invLen + 1.0) * 127.5)
			normPixels[idx+3] = 255
		}
	}
	return normPixels
}

// generateNormalMapScharr generates a normal map from an image using the Scharr operator for edge detection.
// pixels is the input RGBA pixel data of the image.
// width and height define the dimensions of the input image.
// stride is the number of bytes per pixel; typically 4 for RGBA data.
// strength controls the influence of the detected edges on the resulting normal map.
// Returns the RGBA pixel data for the generated normal map.
func generateNormalMapScharr(pixels []uint8, width, height, stride int, strength float64) []uint8 {
	size := width * height
	luma := make([]float64, size)

	// Pass 1: Pre-calcolo della mappa di luminanza (O(N) invece di O(N*9))
	for i := 0; i < size; i++ {
		idx := i * stride
		luma[i] = (0.299*float64(pixels[idx]) + 0.587*float64(pixels[idx+1]) + 0.114*float64(pixels[idx+2])) / 255.0
	}

	normPixels := make([]uint8, len(pixels))

	getLuma := func(x, y int) float64 {
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
		return luma[y*width+x]
	}

	// Fattore di normalizzazione per i pesi di Scharr (3 + 10 + 3 = 16 -> span 32)
	weightNorm := strength / 32.0

	// Pass 2: Convoluzione di Scharr 3x3
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tl := getLuma(x-1, y-1)
			tc := getLuma(x, y-1)
			tr := getLuma(x+1, y-1)

			l := getLuma(x-1, y)
			r := getLuma(x+1, y)

			bl := getLuma(x-1, y+1)
			bc := getLuma(x, y+1)
			br := getLuma(x+1, y+1)

			// Operatore Scharr X e Y
			dX := ((tr + 10.0*r + br) - (tl + 10.0*l + bl)) * weightNorm
			dY := ((bl + 10.0*bc + br) - (tl + 10.0*tc + tr)) * weightNorm

			invLen := 1.0 / math.Sqrt(dX*dX+dY*dY+1.0)
			idx := (y*width + x) * stride

			// Packing nel range [0, 255]
			normPixels[idx] = uint8(((dX * invLen) + 1.0) * 127.5)
			normPixels[idx+1] = uint8(((-dY * invLen) + 1.0) * 127.5)
			normPixels[idx+2] = uint8((invLen + 1.0) * 127.5)

			if stride >= 4 {
				normPixels[idx+3] = 255
			}
		}
	}
	return normPixels
}

// createBlackPixels generates an RGBA pixel array of black color with dimensions w x h and given stride.
func createBlackPixels(w, h, stride int) []uint8 {
	s := w * h * stride
	blackPixels := make([]uint8, s)
	for i := 0; i < len(blackPixels); i += stride {
		blackPixels[i] = 0
		blackPixels[i+1] = 0
		blackPixels[i+2] = 0
		blackPixels[i+3] = 255
	}
	return blackPixels
}

/*
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
*/
