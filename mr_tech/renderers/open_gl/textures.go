package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures provides methods and structures for managing a collection of 2D texture layers used in rendering operations.
// It organizes textures in a map and handles creation and allocation of texture arrays for diffuse, normal, and emissive data.
type Textures struct {
	textures      map[*textures.Texture]float32
	diffuseArray  uint32
	normalArray   uint32
	emissiveArray uint32
}

// NewTextures creates and initializes a new instance of Textures with an empty texture-to-glTexture mapping.
func NewTextures() *Textures {
	return &Textures{
		textures: make(map[*textures.Texture]float32),
	}
}

// Get retrieves the texture layer associated with the given texture and returns a boolean indicating its presence.
func (tx *Textures) Get(tex *textures.Texture) (float32, bool) {
	t, ok := tx.textures[tex]
	return t, ok
}

// GetDiffuseArray returns the OpenGL texture ID of the diffuse texture array stored in the Textures instance.
func (tx *Textures) GetDiffuseArray() uint32 { return tx.diffuseArray }

// GetNormalArray returns the texture array ID used for storing normal maps as a uint32.
func (tx *Textures) GetNormalArray() uint32 { return tx.normalArray }

// GetEmissiveArray retrieves the OpenGL ID of the emissive texture 2D array used for rendering.
func (tx *Textures) GetEmissiveArray() uint32 { return tx.emissiveArray }

// Setup initializes the texture arrays and maps, allocates memory for texture storage, and uploads texture data to the GPU.
func (tx *Textures) Setup(t textures.ITextures) error {
	const stride = 4
	tx.textures = make(map[*textures.Texture]float32)
	names := t.GetNames()
	layerCount := int32(len(names))

	if layerCount == 0 {
		return nil
	}

	// 1. FIND MAXIMUM DIMENSIONS (or set a fixed 256x256)
	var maxW, maxH int
	for _, id := range names {
		tn := t.Get([]string{id})
		if tn != nil && len(tn) > 0 {
			w, h := tn[0].Size()
			if w > maxW {
				maxW = w
			}
			if h > maxH {
				maxH = h
			}
		}
	}

	// Avoid odd dimensions, force power of two for mipmap safety
	maxW = nextPowerOfTwo(maxW)
	maxH = nextPowerOfTwo(maxH)

	// 2. VRAM ALLOCATION OF THE 3 ARRAYS
	tx.diffuseArray = createTextureArray(maxW, maxH, layerCount)
	tx.normalArray = createTextureArray(maxW, maxH, layerCount)
	tx.emissiveArray = createTextureArray(maxW, maxH, layerCount)

	blackPixels := createBlackPixels(maxW, maxH, stride)

	// 3. LAYER POPULATION
	layer := int32(0)
	for _, id := range names {
		tn := t.Get([]string{id})
		if tn == nil || len(tn) == 0 {
			continue
		}
		tex := tn[0]
		origW, origH, pixels := tex.RGBA()
		// CPU pixel resize if texture is smaller than the layer
		resizedPixels := resizePixelsFixedPoint(pixels, origW, origH, maxW, maxH, stride)
		// Normal generation
		normalPixels := generateNormalMap(resizedPixels, maxW, maxH, stride, 3.0)
		// Upload Diffuse Layer
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.diffuseArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(maxW), int32(maxH), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(resizedPixels))
		// Upload Normal Layer
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.normalArray)
		gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(maxW), int32(maxH), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(normalPixels))
		// Upload Emissive Layer (Black by default)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.emissiveArray)

		if len(id) > 0 && (id[0] == '*' || id[0] == '+') {
			// Inietta i pixel diffuse direttamente nel canale emissivo
			gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(maxW), int32(maxH), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(resizedPixels))
		} else {
			// Default (Nero)
			gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(maxW), int32(maxH), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(blackPixels))
		}

		//gl.TexSubImage3D(gl.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(maxW), int32(maxH), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(blackPixels))

		tx.textures[tex] = float32(layer)
		layer++
	}

	// 4. GLOBAL MIPMAP GENERATION (Now safe)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.diffuseArray)
	gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)

	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tx.normalArray)
	gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)

	return nil
}

// createTextureArray initializes and returns a 2D texture array with specified dimensions and number of layers.
func createTextureArray(width, height int, layers int32) uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, tex)

	mipCount := computeMipMapLevel(width, height)

	// Set MipMap limits BEFORE allocation
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_BASE_LEVEL, 0)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAX_LEVEL, mipCount-1)

	// Explicit "Immutable-Style" allocation (OpenGL 3.3 compatibility)
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

// createBlackPixels generates an array of black RGBA pixel data with specified width, height, and stride.
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

// computeMipMapLevel calculates the number of mipmap levels based on the maximum dimension of a texture (width or height).
func computeMipMapLevel(width, height int) int32 {
	maxDim := float64(width)
	if height > width {
		maxDim = float64(height)
	}
	mipCount := int32(math.Floor(math.Log2(maxDim))) + 1
	return mipCount
}

// resizePixels resizes a source RGBA pixel array to a new width and height using nearest-neighbor scaling.
func resizePixelsNearest(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
	if oldW == newW && oldH == newH {
		return src
	}
	dst := make([]uint8, newW*newH*stride)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := (x * oldW) / newW
			srcY := (y * oldH) / newH
			srcIdx := (srcY*oldW + srcX) * stride
			dstIdx := (y*newW + x) * stride
			copy(dst[dstIdx:dstIdx+stride], src[srcIdx:srcIdx+stride])
		}
	}
	return dst
}

// resizePixels esegue un Bilinear Filtering ottimizzato in aritmetica intera (Fixed-Point 8.8).
func resizePixelsFixedPoint(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
	if oldW == newW && oldH == newH {
		return src
	}
	dst := make([]uint8, newW*newH*stride)

	xRatio := (oldW << 8) / newW
	yRatio := (oldH << 8) / newH

	for y := 0; y < newH; y++ {
		yPos := y * yRatio
		yi := yPos >> 8
		yDiff := yPos & 0xFF
		yInv := 256 - yDiff

		y1 := yi + 1
		if y1 >= oldH {
			y1 = oldH - 1
		}

		row0 := yi * oldW * stride
		row1 := y1 * oldW * stride
		dstRow := y * newW * stride

		for x := 0; x < newW; x++ {
			xPos := x * xRatio
			xi := xPos >> 8
			xDiff := xPos & 0xFF
			xInv := 256 - xDiff

			x1 := xi + 1
			if x1 >= oldW {
				x1 = oldW - 1
			}

			idx00 := row0 + xi*stride
			idx10 := row0 + x1*stride
			idx01 := row1 + xi*stride
			idx11 := row1 + x1*stride
			dstIdx := dstRow + x*stride

			// Pesi precalcolati per l'interpolazione
			w00 := (xInv * yInv) >> 8
			w10 := (xDiff * yInv) >> 8
			w01 := (xInv * yDiff) >> 8
			w11 := (xDiff * yDiff) >> 8

			for c := 0; c < stride; c++ {
				a := int(src[idx00+c])
				b := int(src[idx10+c])
				cVal := int(src[idx01+c])
				d := int(src[idx11+c])

				val := (a*w00 + b*w10 + cVal*w01 + d*w11) >> 8
				dst[dstIdx+c] = uint8(val)
			}
		}
	}
	return dst
}

// resizePixels resizes a source RGBA pixel array to a new width and height using Bilinear Interpolation.
func resizePixelsLinear(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
	if oldW == newW && oldH == newH {
		return src
	}
	dst := make([]uint8, newW*newH*stride)

	xRatio := float64(oldW) / float64(newW)
	yRatio := float64(oldH) / float64(newH)

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			// Center the sample to faithfully emulate OpenGL hardware sampling
			px := (float64(x)+0.5)*xRatio - 0.5
			py := (float64(y)+0.5)*yRatio - 0.5

			if px < 0 {
				px = 0
			}
			if py < 0 {
				py = 0
			}

			xi := int(px)
			yi := int(py)

			xDiff := px - float64(xi)
			yDiff := py - float64(yi)

			x1 := xi + 1
			if x1 >= oldW {
				x1 = oldW - 1
			}
			y1 := yi + 1
			if y1 >= oldH {
				y1 = oldH - 1
			}

			idx00 := (yi*oldW + xi) * stride
			idx10 := (yi*oldW + x1) * stride
			idx01 := (y1*oldW + xi) * stride
			idx11 := (y1*oldW + x1) * stride

			dstIdx := (y*newW + x) * stride

			for c := 0; c < stride; c++ {
				a := float64(src[idx00+c])
				b := float64(src[idx10+c])
				cVal := float64(src[idx01+c])
				d := float64(src[idx11+c])

				// Bilinear interpolation formula
				pixel := a*(1.0-xDiff)*(1.0-yDiff) + b*xDiff*(1.0-yDiff) + cVal*(1.0-xDiff)*yDiff + d*xDiff*yDiff

				dst[dstIdx+c] = uint8(pixel)
			}
		}
	}
	return dst
}

// resizePixelsBicubic applies Bicubic Interpolation (Catmull-Rom spline) for high-quality texture scaling.
func resizePixelsBicubic(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
	if oldW == newW && oldH == newH {
		return src
	}
	dst := make([]uint8, newW*newH*stride)

	xRatio := float64(oldW) / float64(newW)
	yRatio := float64(oldH) / float64(newH)

	// Catmull-Rom cubic interpolation
	cubic := func(p0, p1, p2, p3, x float64) float64 {
		return p1 + 0.5*x*(p2-p0+x*(2.0*p0-5.0*p1+4.0*p2-p3+x*(3.0*(p1-p2)+p3-p0)))
	}

	getPixel := func(x, y, c int) float64 {
		if x < 0 {
			x = 0
		} else if x >= oldW {
			x = oldW - 1
		}
		if y < 0 {
			y = 0
		} else if y >= oldH {
			y = oldH - 1
		}
		return float64(src[(y*oldW+x)*stride+c])
	}

	for y := 0; y < newH; y++ {
		py := (float64(y)+0.5)*yRatio - 0.5
		yi := int(math.Floor(py))
		yDiff := py - float64(yi)

		for x := 0; x < newW; x++ {
			px := (float64(x)+0.5)*xRatio - 0.5
			xi := int(math.Floor(px))
			xDiff := px - float64(xi)

			dstIdx := (y*newW + x) * stride

			for c := 0; c < stride; c++ {
				var col [4]float64
				for j := -1; j <= 2; j++ {
					p0 := getPixel(xi-1, yi+j, c)
					p1 := getPixel(xi, yi+j, c)
					p2 := getPixel(xi+1, yi+j, c)
					p3 := getPixel(xi+2, yi+j, c)
					col[j+1] = cubic(p0, p1, p2, p3, xDiff)
				}

				val := cubic(col[0], col[1], col[2], col[3], yDiff)

				// Clamp hardware-level prevention
				if val < 0.0 {
					val = 0.0
				} else if val > 255.0 {
					val = 255.0
				}

				dst[dstIdx+c] = uint8(val)
			}
		}
	}
	return dst
}
