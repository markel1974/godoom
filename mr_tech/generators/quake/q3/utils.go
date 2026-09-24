package q3

import (
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

func LoadImage(texName string, arc interfaces.IArchive) (image.Image, error) {
	img, err := doLoadImage(texName, arc)
	if err != nil {
		fmt.Printf("[warning] using fallback for %s: %s\n", texName, err)
		return FallbackImage(), nil
	}
	return img, nil
}

func doLoadImage(texName string, arc interfaces.IArchive) (image.Image, error) {
	// 1. Prova PRIMA esattamente il nome fornito (utile se ha già l'estensione corretta)
	if f, err := arc.Open(texName); err == nil {
		if strings.HasSuffix(strings.ToLower(texName), ".tga") {
			return common.DecodeTGA(f)
		}
		img, _, errDec := image.Decode(f)
		return img, errDec
	}

	// 2. Se fallisce, prova con le estensioni standard (.tga, .jpg, .png)
	baseName := BaseName(texName)

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

	// 3. Fallbacks
	fallbackName, ok := _q3ShaderFallback[baseName]
	if !ok {
		fallbackName, ok = _q3ShaderFallback[texName]
	}
	if ok && len(fallbackName) > 0 {
		if fTga, e := arc.Open(fallbackName + ".tga"); e == nil {
			return common.DecodeTGA(fTga)
		}
		if fJpg, e := arc.Open(fallbackName + ".jpg"); e == nil {
			img, _, err := image.Decode(fJpg)
			return img, err
		}
		var im image.Image
		return im, fmt.Errorf("missing asset %s (.jpg/.tga) (fallback failed)", texName)
	}

	var img image.Image
	return img, fmt.Errorf("missing asset %s (.jpg/.tga)", texName)
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
