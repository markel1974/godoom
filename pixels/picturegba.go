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
	i := (y - int(s.rect.Min.Y)) * s.stride + (x - int(s.rect.Min.X)) * 4
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
