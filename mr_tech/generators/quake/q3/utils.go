package q3

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
)

func BaseName(in string) string {
	p := filepath.Ext(in)
	if len(p) == 0 {
		return in
	}
	return in[:len(in)-len(p)]
}

func FallbackImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pink := color.RGBA{R: 255, B: 255, A: 255}
	black := color.RGBA{A: 255}
	fallbackImg.Set(0, 0, pink)
	fallbackImg.Set(1, 1, pink)
	fallbackImg.Set(1, 0, black)
	fallbackImg.Set(0, 1, black)
	return fallbackImg
}

func LoadImage(texName string, arc interfaces.IArchive) (image.Image, error) {
	img, err := imageLoader(texName, arc)
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

		return FallbackImage(), nil
	}
	return img, nil
}

func imageLoader(texName string, arc interfaces.IArchive) (image.Image, error) {
	if img, err := doImageLoad(texName, arc); err == nil {
		return img, nil
	}
	baseName := BaseName(texName)
	if img, err := doBaseImageLoad(baseName, arc); err == nil {
		return img, err
	}

	fallbackName, ok := _q3ShaderFallback[baseName]
	if !ok {
		fallbackName, ok = _q3ShaderFallback[texName]
	}
	if ok && len(fallbackName) > 0 {
		img, err := doBaseImageLoad(fallbackName, arc)
		if err == nil {
			return img, err
		}
		return nil, fmt.Errorf("missing asset %s (.jpg/.tga) (fallback failed)", texName)
	}
	return nil, fmt.Errorf("missing asset %s (.jpg/.tga)", texName)
}

func doBaseImageLoad(baseName string, arc interfaces.IArchive) (image.Image, error) {
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

func doImageLoad(texName string, arc interfaces.IArchive) (image.Image, error) {
	if f, err := arc.Open(texName); err == nil {
		if strings.HasSuffix(strings.ToLower(texName), ".tga") {
			return common.DecodeTGA(f)
		}
		img, _, errDec := image.Decode(f)
		return img, errDec
	}
	return nil, errors.New("no image found")
}
