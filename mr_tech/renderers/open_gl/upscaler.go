package open_gl

import "math"

// __srgbToLin is a lookup table for converting sRGB values (0-255) to linear color space values in the range [0.0, 1.0].
var __srgbToLin [256]float64

// init initializes the __srgbToLin array with sRGB to linear conversion values for the range [0, 255].
func init() {
	for i := 0; i < 256; i++ {
		v := float64(i) / 255.0
		if v <= 0.04045 {
			__srgbToLin[i] = v / 12.92
		} else {
			__srgbToLin[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
}

// UpscaleNearest resizes an image buffer using nearest-neighbor scaling from old dimensions to new dimensions.
func UpscaleNearest(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
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

// UpscaleLinear performs bilinear interpolation to scale an image from old dimensions to new dimensions.
// src is the source pixel data, oldW and oldH are the original width and height, newW and newH are the target dimensions.
// stride specifies the number of color channels per pixel. The function returns the upscaled pixel data as a slice of uint8.
func UpscaleLinear(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
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

// UpscaleFixedPoint performs bilinear image upscaling from old dimensions to new dimensions with fixed-point arithmetic.
// src is the source byte slice containing pixel data.
// oldW and oldH represent the width and height of the original image, respectively.
// newW and newH represent the width and height of the resized image, respectively.
// stride is the number of bytes per pixel (e.g., 3 for RGB, 4 for RGBA).
// Returns a byte slice containing the resized image's pixel data.
func UpscaleFixedPoint(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
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

			// Precomputed weights for interpolation
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

// UpscaleBicubic resizes an image using bicubic interpolation for smoother scaling.
// src is the source pixel data array.
// oldW and oldH are the width and height of the original image.
// newW and newH are the desired width and height of the resulting image.
// stride represents the number of color components per pixel (e.g., 3 for RGB, 4 for RGBA).
// Returns a new slice containing the resized image pixel data.
func UpscaleBicubic(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
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

// UpscaleLanczosSeparable performs image upscaling using a separable Lanczos-3 convolution algorithm in two passes.
// It resizes an input image from dimensions (oldW x oldH) to new dimensions (newW x newH) with a specified stride.
// The function preserves color accuracy by operating in linear color space and converting back to sRGB after processing.
// If the new dimensions match the old dimensions, the original source image is returned unchanged.
func UpscaleLanczosSeparable(src []uint8, oldW, oldH, newW, newH, stride int) []uint8 {
	linToSRGB := func(c float64) uint8 {
		if c <= 0.0 {
			return 0
		}
		if c >= 1.0 {
			return 255
		}
		var v float64
		if c <= 0.0031308 {
			v = c * 12.92
		} else {
			v = 1.055*math.Pow(c, 1/2.4) - 0.055
		}
		return uint8(math.Round(v * 255.0))
	}
	sinc := func(x float64) float64 {
		if x == 0 {
			return 1.0
		}
		x *= math.Pi
		return math.Sin(x) / x
	}
	lanczos3 := func(x float64) float64 {
		if x < 0 {
			x = -x
		}
		if x >= 3.0 {
			return 0.0
		}
		return sinc(x) * sinc(x/3.0)
	}

	if oldW == newW && oldH == newH {
		return src
	}

	// Intermediate buffer in linear space (float64 to preserve gradient precision)
	temp := make([]float64, newW*oldH*stride)
	xRatio := float64(oldW) / float64(newW)

	// Horizontal Convolution (src -> temp)
	for y := 0; y < oldH; y++ {
		for x := 0; x < newW; x++ {
			cx := (float64(x)+0.5)*xRatio - 0.5
			ix := int(math.Floor(cx))

			for c := 0; c < stride; c++ {
				var sum, wSum float64
				minV, maxV := 1.0, 0.0

				// Radius 3 -> 6 tap (-2 to +3)
				for j := -2; j <= 3; j++ {
					sx := ix + j
					if sx < 0 {
						sx = 0
					} else if sx >= oldW {
						sx = oldW - 1
					}

					val := __srgbToLin[src[(y*oldW+sx)*stride+c]]

					// Local Extrema Tracking (on the 2 texels adjacent to the center)
					if j == 0 || j == 1 {
						if val < minV {
							minV = val
						}
						if val > maxV {
							maxV = val
						}
					}

					w := lanczos3(cx - float64(ix+j))
					sum += val * w
					wSum += w
				}

				res := sum / wSum
				if res < minV {
					res = minV
				} else if res > maxV {
					res = maxV
				}
				temp[(y*newW+x)*stride+c] = res
			}
		}
	}

	dst := make([]uint8, newW*newH*stride)
	yRatio := float64(oldH) / float64(newH)

	// Vertical Convolution (temp -> dst)
	for x := 0; x < newW; x++ {
		for y := 0; y < newH; y++ {
			cy := (float64(y)+0.5)*yRatio - 0.5
			iy := int(math.Floor(cy))
			for c := 0; c < stride; c++ {
				var sum, wSum float64
				minV, maxV := 1.0, 0.0
				for j := -2; j <= 3; j++ {
					sy := iy + j
					if sy < 0 {
						sy = 0
					} else if sy >= oldH {
						sy = oldH - 1
					}
					val := temp[(sy*newW+x)*stride+c]
					if j == 0 || j == 1 {
						if val < minV {
							minV = val
						}
						if val > maxV {
							maxV = val
						}
					}
					w := lanczos3(cy - float64(iy+j))
					sum += val * w
					wSum += w
				}
				res := sum / wSum
				if res < minV {
					res = minV
				} else if res > maxV {
					res = maxV
				}
				// Return to gamma-compressed sRGB space
				dst[(y*newW+x)*stride+c] = linToSRGB(res)
			}
		}
	}
	return dst
}

func PadPixels(pixels []uint8, srcW, srcH, dstSize, stride int) []uint8 {
	if srcW == dstSize && srcH == dstSize {
		return pixels
	}
	padded := make([]uint8, dstSize*dstSize*stride)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			srcIdx := (y*srcW + x) * stride
			dstIdx := (y*dstSize + x) * stride
			copy(padded[dstIdx:dstIdx+stride], pixels[srcIdx:srcIdx+stride])
		}
	}
	return padded
}
