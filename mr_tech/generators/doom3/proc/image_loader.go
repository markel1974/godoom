package proc

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
	"github.com/markel1974/godoom/mr_tech/generators/quake/textures"
)

// ImageDriver represents a structure that associates a file extension with a corresponding image decoding function.
type ImageDriver struct {
	ext string
	fn  func(io.Reader) (image.Image, string, error)
}

// _doom3ImageDrivers is a collection of image drivers mapping file extensions to their respective decode functions.
var _doom3ImageDrivers = []ImageDriver{
	{ext: ".tga", fn: common.DecodeTGA},
	{ext: ".jpg", fn: image.Decode},
	{ext: ".png", fn: image.Decode},
}

// ImageLoader is responsible for loading textures and materials from a custom archive or repository. It orchestrates resource management through its dependency on IPkArchive for file access, Textures for texture handling, and Materials for storing material properties.
type ImageLoader struct {
	arc        IPkArchive
	texHandler *textures.Textures
	materials  *Materials
}

// NewImageLoader initializes and returns a new ImageLoader with the provided IPkArchive, Textures, and Materials.
func NewImageLoader(arc IPkArchive, texHandler *textures.Textures, materials *Materials) *ImageLoader {
	return &ImageLoader{
		arc:        arc,
		texHandler: texHandler,
		materials:  materials,
	}
}

// FallbackImage returns a simple 2x2 fallback image with a checkerboard pattern of pink and black colors.
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

// FallbackNormalImage generates a 2x2 fallback normal map image with a default flat normal color (128, 128, 255, 255).
func (il *ImageLoader) FallbackNormalImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	flatNormal := color.RGBA{R: 128, G: 128, B: 255, A: 255}
	fallbackImg.Set(0, 0, flatNormal)
	fallbackImg.Set(1, 1, flatNormal)
	fallbackImg.Set(1, 0, flatNormal)
	fallbackImg.Set(0, 1, flatNormal)
	return fallbackImg
}

// FallbackSpecularImage generates a default 2x2 specular image with a uniform low-intensity gray color.
func (il *ImageLoader) FallbackSpecularImage() image.Image {
	fallbackImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	flatSpec := color.RGBA{R: 32, G: 32, B: 32, A: 255}
	fallbackImg.Set(0, 0, flatSpec)
	fallbackImg.Set(1, 1, flatSpec)
	fallbackImg.Set(1, 0, flatSpec)
	fallbackImg.Set(0, 1, flatSpec)
	return fallbackImg
}

// Load retrieves and registers a texture by its name. Falls back to a default image if loading fails.
func (il *ImageLoader) Load(texName string, forceOpaque bool) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.texHandler.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
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

	if err = il.texHandler.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering texture %s: %w", texNameLC, err)
	}
	return nil
}

// LoadNormal loads a normal map texture by name, falling back to a default if unavailable, and registers it with the texture handler.
func (il *ImageLoader) LoadNormal(texName string) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.texHandler.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
		return nil
	}

	img, err := il.retrieve(texNameLC, "_local")
	if err != nil {
		img = il.FallbackNormalImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if err = il.texHandler.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering normal map %s: %w", texNameLC, err)
	}
	return nil
}

// LoadSpecular loads a specular map for the given texture name and registers it with the texture handler.
func (il *ImageLoader) LoadSpecular(texName string) error {
	if len(texName) == 0 {
		return nil
	}
	texNameLC := strings.ToLower(texName)
	if texes := il.texHandler.GetFrames([]string{texNameLC}); len(texes) > 0 && texes[0] != nil {
		return nil
	}

	img, err := il.retrieve(texNameLC, "_s")
	if err != nil {
		img = il.FallbackSpecularImage()
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if err = il.texHandler.RegisterPixelsRGBA(texNameLC, bounds.Dx(), bounds.Dy(), rgba.Pix, true); err != nil {
		return fmt.Errorf("registering specular map %s: %w", texNameLC, err)
	}
	return nil
}

// LoadMaterialMaps loads the material maps (diffuse, normal, and specular) associated with the provided material name.
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

// retrieve attempts to load an image resource by filename, trying various extensions and suffixes if needed.
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
	for _, candidate := range candidates {
		for _, driver := range _doom3ImageDrivers {
			testPath := candidate + driver.ext
			if img, err := il.doLoad(testPath); err == nil {
				return img, nil
			}
		}
	}
	return nil, errors.New("image not found: " + texName)
}

// doLoad loads an image given its full path, decoding it based on its file type. Returns the decoded image or an error.
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
