package doom3

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	common2 "github.com/markel1974/godoom/mr_tech/generators/quake/textures"
)

type imageDriver struct {
	ext string
	fn  func(io.Reader) (image.Image, string, error)
}

var _doom3ImageDrivers = []imageDriver{
	{ext: ".tga", fn: common.DecodeTGA},
	{ext: ".jpg", fn: image.Decode},
	{ext: ".png", fn: image.Decode},
}

// ImageLoader is a utility for loading texture maps from Doom 3 PK4 archives into the Godoom texture manager.
type ImageLoader struct {
	arc       IPkArchive
	textures  *common2.Textures
	materials *Materials
}

// NewImageLoader creates a new ImageLoader instance.
func NewImageLoader(arc IPkArchive, textures *common2.Textures, materials *Materials) *ImageLoader {
	return &ImageLoader{
		arc:       arc,
		textures:  textures,
		materials: materials,
	}
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

// FallbackNormalImage generates and returns a 2x2 flat normal map (0, 0, 1) mapped to RGBA (128, 128, 255, 255).
func (il *ImageLoader) FallbackNormalImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	flatNormal := color.RGBA{R: 128, G: 128, B: 255, A: 255}
	fallbackImg.Set(0, 0, flatNormal)
	fallbackImg.Set(1, 1, flatNormal)
	fallbackImg.Set(1, 0, flatNormal)
	fallbackImg.Set(0, 1, flatNormal)
	return fallbackImg
}

// FallbackSpecularImage generates a 2x2 neutral dark specular map.
func (il *ImageLoader) FallbackSpecularImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	flatSpec := color.RGBA{R: 32, G: 32, B: 32, A: 255}
	fallbackImg.Set(0, 0, flatSpec)
	fallbackImg.Set(1, 1, flatSpec)
	fallbackImg.Set(1, 0, flatSpec)
	fallbackImg.Set(0, 1, flatSpec)
	return fallbackImg
}

// Load attempts to load an image by name from the archive and registers it in the texture manager.
func (il *ImageLoader) Load(texName string, forceOpaque bool) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.textures.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
		return nil
	}

	img, err := il.retrieve(texNameLC, "")
	if err != nil {
		img = il.FallbackImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if forceOpaque {
		for i := 3; i < len(rgba.Pix); i += 4 {
			rgba.Pix[i] = 255
		}
	}

	if err = il.textures.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering texture %s: %w", texNameLC, err)
	}
	return nil
}

// LoadNormal loads a bump/normal map texture.
func (il *ImageLoader) LoadNormal(texName string) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.textures.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
		return nil
	}

	img, err := il.retrieve(texNameLC, "_local")
	if err != nil {
		img = il.FallbackNormalImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if err = il.textures.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering normal map %s: %w", texNameLC, err)
	}
	return nil
}

// LoadSpecular loads a specular map texture.
func (il *ImageLoader) LoadSpecular(texName string) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.textures.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
		return nil
	}

	img, err := il.retrieve(texNameLC, "_s")
	if err != nil {
		img = il.FallbackSpecularImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if err = il.textures.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering specular map %s: %w", texNameLC, err)
	}
	return nil
}

// LoadMaterialMaps resolves a material and loads all its associated maps (diffuse, normal, specular).
func (il *ImageLoader) LoadMaterialMaps(matName string) {
	if il.materials == nil {
		_ = il.Load(matName, false)
		return
	}
	def := il.materials.Resolve(matName)
	if def == nil {
		_ = il.Load(matName, false)
		return
	}

	if def.DiffuseMap != "" {
		_ = il.Load(def.DiffuseMap, false)
	}
	if def.NormalMap != "" {
		_ = il.LoadNormal(def.NormalMap)
	}
	if def.SpecularMap != "" {
		_ = il.LoadSpecular(def.SpecularMap)
	}
}

func (il *ImageLoader) retrieve(texName string, suffixHint string) (image.Image, error) {
	ext := filepath.Ext(texName)
	if ext != "" {
		if img, err := il.doLoad(texName); err == nil {
			return img, nil
		}
	}

	// Try without extension + various extensions
	baseName := strings.TrimSuffix(texName, ext)
	candidates := []string{baseName}

	if suffixHint != "" && !strings.HasSuffix(baseName, suffixHint) {
		candidates = append(candidates, baseName+suffixHint)
	}

	// In Doom 3, diffuse maps often have _d suffix
	if suffixHint == "" && !strings.HasSuffix(baseName, "_d") {
		candidates = append(candidates, baseName+"_d")
	}

	for _, cand := range candidates {
		for _, driver := range _doom3ImageDrivers {
			testPath := cand + driver.ext
			if img, err := il.doLoad(testPath); err == nil {
				return img, nil
			}
		}
	}

	return nil, errors.New("image not found: " + texName)
}

func (il *ImageLoader) doLoad(fullPath string) (image.Image, error) {
	cleanPath := strings.TrimPrefix(fullPath, "/")
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	f, err := il.arc.Open(cleanPath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(strings.ToLower(cleanPath), ".tga") {
		img, _, err := common.DecodeTGA(f)
		return img, err
	}

	img, _, err := image.Decode(f)
	return img, err
}
