package q3

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

// BaseName removes the file extension from the given string and returns the base name of the file.
func BaseName(in string) string {
	p := filepath.Ext(in)
	if len(p) == 0 {
		return in
	}
	return in[:len(in)-len(p)]
}

// ImageLoader is a utility for loading image assets from an archive implementing the IArchive interface.
type ImageLoader struct {
	arc      interfaces.IArchive
	textures *lumps.Textures
}

// NewImageLoader creates a new ImageLoader instance using the provided IArchive for file access and directory operations.
func NewImageLoader(arc interfaces.IArchive, textures *lumps.Textures) *ImageLoader {
	return &ImageLoader{arc: arc, textures: textures}
}

// FallbackImage generates and returns a 2x2 fallback image with a pink and black checkerboard pattern.
func (il *ImageLoader) FallbackImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pink := color.RGBA{R: 255, B: 255, A: 255}
	black := color.RGBA{A: 255}
	fallbackImg.Set(0, 0, pink)
	fallbackImg.Set(1, 1, pink)
	fallbackImg.Set(1, 0, black)
	fallbackImg.Set(0, 1, black)
	return fallbackImg
}

// Load attempts to load an image by its name and returns it; falls back to a default image on failure.
func (il *ImageLoader) Load(texName string) error {
	if texName == "noshader" || len(texName) == 0 {
		return nil
	}
	if texes := il.textures.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
		return nil // Already loaded
	}
	img, err := il.retrieve(texName)
	if err != nil {
		fmt.Printf("[warning] using fallback for %s: %s\n", texName, err)

		/*
			dir := path.Dir(texName)
			if dir != "." && dir != "" {
				files, errDir := arc.ReadDir(dir)
				if errDir == nil {
					for _, f := range files {
						ext := strings.ToLower(path.Ext(f))
						if ext == ".jpg" || ext == ".tga" || ext == ".png" {
							smartPath := path.Join(dir, f)
							if smartImg, errSmart := doLoadImage(smartPath, arc); errSmart == nil {
								fmt.Printf("[info] smart fallback found: %s\n", smartPath)
								return smartImg, nil
							}
						}
					}
				}
			}
		*/

		img = il.FallbackImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	if err = il.textures.RegisterPixelsRGBA(texName, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		fmt.Printf("warning: registering picture %s: %s\n", texName, err.Error())
	}
	return nil
}

// retrieve attempts to load an image by its name, with fallback mechanisms if the primary load fails.
func (il *ImageLoader) retrieve(texName string) (image.Image, error) {
	if img, err := il.doLoad(texName, il.arc); err == nil {
		return img, nil
	}
	baseName := BaseName(texName)
	if img, err := il.doBaseLoad(baseName, il.arc); err == nil {
		return img, err
	}

	fallbackName, ok := _q3ShaderFallback[baseName]
	if !ok {
		fallbackName, ok = _q3ShaderFallback[texName]
	}
	if ok && len(fallbackName) > 0 {
		img, err := il.doBaseLoad(fallbackName, il.arc)
		if err == nil {
			return img, err
		}
		return nil, fmt.Errorf("missing asset %s (.jpg/.tga) (fallback failed)", texName)
	}
	return nil, fmt.Errorf("missing asset %s (.jpg/.tga)", texName)
}

// doBaseLoad attempts to load an image by trying .jpg, .tga, and .png file extensions in the given archive.
// Returns the loaded image or an error if no matching file is found.
func (il *ImageLoader) doBaseLoad(baseName string, arc interfaces.IArchive) (image.Image, error) {
	if fileJpg, errJpg := arc.Open(baseName + ".jpg"); errJpg == nil {
		img, _, err := image.Decode(fileJpg)
		return img, err
	}
	if fileTga, errTga := arc.Open(baseName + ".tga"); errTga == nil {
		return common.DecodeTGA(fileTga)
	}
	if filePng, errPng := arc.Open(baseName + ".png"); errPng == nil {
		img, _, err := image.Decode(filePng)
		return img, err
	}
	return nil, errors.New("no image found")
}

// doLoad attempts to load an image from the archive using its file name and format, returning the decoded image or an error.
func (il *ImageLoader) doLoad(texName string, arc interfaces.IArchive) (image.Image, error) {
	if f, err := arc.Open(texName); err == nil {
		if strings.HasSuffix(strings.ToLower(texName), ".tga") {
			return common.DecodeTGA(f)
		}
		img, _, errDec := image.Decode(f)
		return img, errDec
	}
	return nil, errors.New("no image found")
}
