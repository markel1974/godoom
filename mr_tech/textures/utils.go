package textures

import "math"

// CreateBlackPixels generates an RGBA pixel array of black color with dimensions w x h and given stride.
func CreateBlackPixels(w, h, stride int) []uint8 {
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

// GenerateNormalMapScharr generates a normal map from an image using the Scharr operator for edge detection.
// pixels is the input RGBA pixel data of the image.
// width and height define the dimensions of the input image.
// stride is the number of bytes per pixel; typically 4 for RGBA data.
// strength controls the influence of the detected edges on the resulting normal map.
// aspectRatio is the ratio of original width to original height (wOrig / hOrig) to preserve isotropic normals when non-square textures are upscaled into square layers.
// Returns the RGBA pixel data for the generated normal map.
func GenerateNormalMapScharr(pixels []uint8, width, height, stride int, strength float64, sCenterWeight float64, sSideWeight float64, aspectRatio float64) []uint8 {
	//const sCenterWeight = 10.0
	//const sSideWeight = 3.0
	sPositive := 2.0*sSideWeight + sCenterWeight
	sRange := 2.0 * sPositive

	size := width * height
	luma := make([]float64, size)

	// Pre-calcolo della mappa di luminanza (O(N) invece di O(N*9))
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

	// Fattore di normalizzazione in base ai pesi del kernel.
	// Il range teorico è il doppio della somma dei pesi positivi.
	weightNormX := strength / sRange
	weightNormY := weightNormX * aspectRatio

	// Convoluzione di Scharr 3x3
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
			dX := ((sSideWeight*tr + sCenterWeight*r + sSideWeight*br) - (sSideWeight*tl + sCenterWeight*l + sSideWeight*bl)) * weightNormX
			dY := ((sSideWeight*bl + sCenterWeight*bc + sSideWeight*br) - (sSideWeight*tl + sCenterWeight*tc + sSideWeight*tr)) * weightNormY

			invLen := 1.0 / math.Sqrt(dX*dX+dY*dY+1.0)
			idx := (y*width + x) * stride

			// Packing nel range [0, 255]
			normPixels[idx] = uint8(((dX*invLen)+1.0)*127.5 + 0.5)
			normPixels[idx+1] = uint8(((-dY*invLen)+1.0)*127.5 + 0.5)
			normPixels[idx+2] = uint8((invLen+1.0)*127.5 + 0.5)

			if stride >= 4 {
				normPixels[idx+3] = 255
			}
		}
	}
	return normPixels
}

// GenerateNormalMap creates a normal map from image pixel data, applying luminance-based calculations and strength scaling.
func GenerateNormalMap(pixels []uint8, width, height, stride int, strength float64) []uint8 {
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
