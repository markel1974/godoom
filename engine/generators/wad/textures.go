package wad

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/markel1974/godoom/engine/textures"
)

func CreateTextureId(id string) string {
	const textureId = "__TEXTURE__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return textureId + id
}

func CreateFlatId(id string) string {
	const flatId = "__FLAT__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return flatId + id
}

// Textures manages a collection of textures using a map with string identifiers as keys.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures initializes and returns a Textures object by loading texture data from the specified base path. Returns an error if loading fails.
func NewTextures() (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t, nil
}

// Add inserts a new texture into the Textures map using the given id and image, returning a pointer to the created texture.
func (t *Textures) Add(srcId string, src *image.RGBA) *textures.Texture {
	size := src.Bounds().Size()
	//dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	//draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	dst := src

	texture := textures.NewTexture(size.X, size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			rgba := dst.At(x, y).(color.RGBA)
			c := int(rgba.R)*65536 + int(rgba.G)*256 + int(rgba.B)
			texture.Set(x, y, c)
		}
	}

	t.resources[srcId] = texture
	return texture
}

// Get retrieves a texture from the resources map by its cleaned identifier. Returns nil if not found or the ID is invalid.
func (t *Textures) Get(id string) *textures.Texture {
	if len(id) == 0 {
		return nil
	}
	x, ok := t.resources[id]
	if !ok {
		fmt.Println("CAN'T FIND TEXTURE", id)
		return nil
	}
	return x
}

// cleanId normalizes the input string by trimming whitespace and converting it to uppercase.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}

/*
func (t * Textures) initialize() error {
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	files, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() {
			data, err := t.load(basePath + f.Name())
			if err == nil || err == io.EOF {
				t.resources[f.Name()] = data
			} else {
				return err
			}
		}
	}
	return nil
}

// load reads texture data from the specified file and initializes a 1024x1024 texture with pixel color values.
func (t *Textures) load(filename string) (*textures.Texture, error) {
	var texture = textures.NewTexture(1024, 1024)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.Seek(0x11, io.SeekStart); err != nil {
		return nil, err
	}
	br := bufio.NewReader(file)
	var r byte
	var g byte
	var b byte
	for {
		for y := 0; y < 1024; y++ {
			for x := 0; x < 1024; x++ {
				if r, err = br.ReadByte(); err != nil {
					return texture, err
				}
				if g, err = br.ReadByte(); err != nil {
					return texture, err
				}
				if b, err = br.ReadByte(); err != nil {
					return texture, err
				}
				texture.Set(x, y, int(r)*65536+int(g)*256+int(b))
			}
		}
	}
}

func (t *Textures) Add(id string, img *image.RGBA) *textures.Texture {
	if img.Rect.Dx() > 1024 || img.Rect.Dy() > 1024 {
		panic("Immagine deve essere 1024x1024")
	}
	texture := textures.NewTexture()
	for y := 0; y < 1024; y++ {
		for x := 0; x < 1024; x++ {
			rgba := img.At(x, y).(color.RGBA)
			texture.Set(x, y, int(rgba.R)*65536+int(rgba.G)*256+int(rgba.B))
		}
	}
	t.resources[id] = texture
	return texture
}
*/
