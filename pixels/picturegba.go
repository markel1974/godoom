package pixels

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func verticalFlip(rgba *image.RGBA) {
	bounds := rgba.Bounds()
	width := bounds.Dx()
	tmpRow := make([]uint8, width*4)
	for i, j := 0, bounds.Dy()-1; i < j; i, j = i+1, j-1 {
		iRow := rgba.Pix[i*rgba.Stride : i*rgba.Stride+width*4]
		jRow := rgba.Pix[j*rgba.Stride : j*rgba.Stride+width*4]
		copy(tmpRow, iRow)
		copy(iRow, jRow)
		copy(jRow, tmpRow)
	}
}

type PictureRGBA struct {
	rect   Rect
	stride int
	pixels []uint8
	length int
	lastY  int
}

func NewPictureRGBAFromPicture(pic IPicture) *PictureRGBA {
	if pd, ok := pic.(*PictureRGBA); ok {
		return pd
	}

	bounds := pic.Bounds()
	pd := NewPictureRGBA(bounds)

	if pic, ok := pic.(IPictureColor); ok {
		for y := math.Floor(bounds.Min.Y); y < bounds.Max.Y; y++ {
			for x := math.Floor(bounds.Min.X); x < bounds.Max.X; x++ {
				at := MakeVec(
					math.Max(x, bounds.Min.X),
					math.Max(y, bounds.Min.Y),
				)
				col := pic.Color(at)
				pd.SetRGBA(int(x), int(y), uint8(col.R*255), uint8(col.G*255), uint8(col.B*255), uint8(col.A*255))
			}
		}
	}

	return pd
}

func NewPictureRGBAFromImage(img image.Image) *PictureRGBA {
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	verticalFlip(rgba)
	pd := NewPictureRGBA(R(
		float64(rgba.Bounds().Min.X),
		float64(rgba.Bounds().Min.Y),
		float64(rgba.Bounds().Max.X),
		float64(rgba.Bounds().Max.Y),
	))
	copy(pd.pixels, rgba.Pix)
	return pd
}

func NewPictureRGBA(rect Rect) *PictureRGBA {
	w := int(math.Ceil(rect.Max.X)) - int(math.Floor(rect.Min.X))
	h := int(math.Ceil(rect.Max.Y)) - int(math.Floor(rect.Min.Y))
	l := 4 * w * h
	s := &PictureRGBA{
		stride: 4 * w,
		rect:   rect,
		pixels: make([]uint8, l),
		length: l - 4,
		lastY:  int(rect.Max.Y) - 1,
	}
	return s
}

func (s *PictureRGBA) SetColor(x int, y int, c color.RGBA) {
	s.SetRGBA(x, y, c.R, c.G, c.B, c.A)
}

func (s *PictureRGBA) SetRGBA(x int, y int, r uint8, g uint8, b uint8, a uint8) {
	//flip
	//y = (int(s.rect.Max.Y) -1) - y
	y = s.lastY - y
	i := (y-int(s.rect.Min.Y))*s.stride + (x-int(s.rect.Min.X))*4
	if i >= 0 && i < s.length {
		s.pixels[i] = r
		s.pixels[i+1] = g
		s.pixels[i+2] = b
		s.pixels[i+3] = a
	}
}

func (s *PictureRGBA) SetRGBASize(x int, y int, r uint8, g uint8, b uint8, a uint8, size int) {
	k := size / 2
	for offsetX := -k; offsetX < k; offsetX++ {
		for offsetY := -k; offsetY < k; offsetY++ {
			s.SetRGBA(x+offsetX, y+offsetY, r, g, b, a)
		}
	}
}

func (s *PictureRGBA) Bounds() Rect {
	return s.rect
}

func (s *PictureRGBA) Pixels() []uint8 {
	return s.pixels
}

func (s *PictureRGBA) Image() *image.RGBA {
	bounds := image.Rect(
		int(math.Floor(s.rect.Min.X)),
		int(math.Floor(s.rect.Min.Y)),
		int(math.Ceil(s.rect.Max.X)),
		int(math.Ceil(s.rect.Max.Y)),
	)
	rgba := image.NewRGBA(bounds)
	copy(rgba.Pix, s.pixels)
	return rgba
}

// ApplyFastAA esegue una passata di anti-aliasing (edge-smoothing) in software.
// lumaThreshold (es. 20-40) definisce il contrasto minimo per attivare il filtro.
func (s *PictureRGBA) ApplyFastAA(lumaThreshold uint8) {
	width := s.stride / 4
	height := (len(s.pixels)) / s.stride
	stride := s.stride
	pixels := s.pixels
	threshold := uint16(lumaThreshold)

	// Saltiamo i bordi per evitare out-of-bounds
	for y := 1; y < height-1; y++ {
		rowOffset := y * stride
		for x := 1; x < width-1; x++ {
			i := rowOffset + (x << 2) // x * 4

			r := uint16(pixels[i])
			g := uint16(pixels[i+1])
			b := uint16(pixels[i+2])

			// Fast Luma (approssimazione pesata: R + 2G + B / 4)
			lumaC := (r + (g << 1) + b) >> 2

			iT := i - stride
			lumaT := (uint16(pixels[iT]) + (uint16(pixels[iT+1]) << 1) + uint16(pixels[iT+2])) >> 2

			iB := i + stride
			lumaB := (uint16(pixels[iB]) + (uint16(pixels[iB+1]) << 1) + uint16(pixels[iB+2])) >> 2

			iL := i - 4
			lumaL := (uint16(pixels[iL]) + (uint16(pixels[iL+1]) << 1) + uint16(pixels[iL+2])) >> 2

			iR := i + 4
			lumaR := (uint16(pixels[iR]) + (uint16(pixels[iR+1]) << 1) + uint16(pixels[iR+2])) >> 2

			// Trova il contrasto locale
			minL := lumaC
			if lumaT < minL {
				minL = lumaT
			}
			if lumaB < minL {
				minL = lumaB
			}
			if lumaL < minL {
				minL = lumaL
			}
			if lumaR < minL {
				minL = lumaR
			}

			maxL := lumaC
			if lumaT > maxL {
				maxL = lumaT
			}
			if lumaB > maxL {
				maxL = lumaB
			}
			if lumaL > maxL {
				maxL = lumaL
			}
			if lumaR > maxL {
				maxL = lumaR
			}

			// Se c'è un gradino ad alto contrasto (aliasing)
			if (maxL - minL) > threshold {
				// Box-blur a croce ultra-veloce (divisione per 5 precalcolata o via lookup se necessario, qui usiamo div intera)
				pixels[i] = uint8((r + uint16(pixels[iT]) + uint16(pixels[iB]) + uint16(pixels[iL]) + uint16(pixels[iR])) / 5)
				pixels[i+1] = uint8((g + uint16(pixels[iT+1]) + uint16(pixels[iB+1]) + uint16(pixels[iL+1]) + uint16(pixels[iR+1])) / 5)
				pixels[i+2] = uint8((b + uint16(pixels[iT+2]) + uint16(pixels[iB+2]) + uint16(pixels[iL+2]) + uint16(pixels[iR+2])) / 5)
			}
		}
	}
}
