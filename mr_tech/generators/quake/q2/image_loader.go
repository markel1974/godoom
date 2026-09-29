package q2

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

// ImageLoader handles image loading, texture management, and color palette application in rendering workflows.
type ImageLoader struct {
	arc      interfaces.IArchive
	textures *lumps.Textures
	palette  [256]color.RGBA
}

// NewImageLoader initializes a new ImageLoader with the given IArchive, Textures, and color palette.
func NewImageLoader(arc interfaces.IArchive, textures *lumps.Textures, palette [256]color.RGBA) *ImageLoader {
	return &ImageLoader{arc: arc, textures: textures, palette: palette}
}

// fallbackImage generates and returns a 2x2 placeholder image with a pink and black checkerboard pattern.
func (il *ImageLoader) fallbackImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pink := color.RGBA{R: 255, B: 255, A: 255}
	black := color.RGBA{A: 255}
	fallbackImg.Set(0, 0, pink)
	fallbackImg.Set(1, 1, pink)
	fallbackImg.Set(1, 0, black)
	fallbackImg.Set(0, 1, black)
	return fallbackImg
}

// Load attempts to load a texture by its name, registers it if successful, and uses a fallback image if loading fails.
func (il *ImageLoader) Load(id string, fileName string) error {
	if texes := il.textures.Get([]string{id}); len(texes) > 0 && texes[0] != nil {
		return nil // Already loaded
	}

	img, err := il.doLoad(fileName)
	if err != nil {
		fmt.Printf("[warning] using fallback for %s [%s]: %s\n", id, fileName, err)
		img = il.fallbackImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if err = il.textures.RegisterPixelsRGBA(id, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		fmt.Printf("warning: registering picture %s [%s]: %s\n", id, fileName, err.Error())
	}
	return nil
}

// doLoad attempts to load an image by its name from the archive. Returns the loaded image or an error if unsuccessful.
func (il *ImageLoader) doLoad(texName string) (image.Image, error) {
	if f, rErr := il.arc.Open(texName); rErr == nil {
		if strings.HasSuffix(strings.ToLower(texName), ".pcx") {
			pcx := common.NewPCX()
			img, err := pcx.Parse(f, il.palette)
			return img, err
		}
		img, _, err := image.Decode(f)
		return img, err
	}
	return nil, errors.New("no image found")
}
