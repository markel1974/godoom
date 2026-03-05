package wad

import (
	"bufio"
	"image"
	"image/color"
	"io"
	"os"

	"github.com/markel1974/godoom/engine/textures"
	"golang.org/x/image/draw"
)

// Textures is a container for managing texture resources mapped by string identifiers.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures initializes a Textures instance by loading texture files from the specified base path.
// It returns the created Textures instance or an error if loading fails.
func NewTextures(basePath string) (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	files, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !f.IsDir() {
			data, err := t.load(basePath + f.Name())
			if err == nil || err == io.EOF {
				t.resources[f.Name()] = data
			} else {
				return nil, err
			}
		}
	}
	return t, nil
}

// load reads a texture from the specified file, processes its RGB values, and populates a Texture object.
func (t *Textures) load(filename string) (*textures.Texture, error) {
	var texture = &textures.Texture{}
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
				texture.Set(uint(x), uint(y), int(r)*65536+int(g)*256+int(b))
			}
		}
	}
}

/*
func (t *Textures) Add(id string, img *image.RGBA) *textures.Texture {
	if img.Rect.Dx() > 1024 || img.Rect.Dy() > 1024 {
		panic("Immagine deve essere 1024x1024")
	}
	texture := textures.NewTexture()
	for y := 0; y < 1024; y++ {
		for x := 0; x < 1024; x++ {
			rgba := img.At(x, y).(color.RGBA)
			texture.Set(uint(x), uint(y), int(rgba.R)*65536+int(rgba.G)*256+int(rgba.B))
		}
	}
	t.resources[id] = texture
	return texture
}
*/

func (t *Textures) Add(id string, img *image.RGBA) *textures.Texture {
	dst := image.NewRGBA(image.Rect(0, 0, textures.TextureSize, textures.TextureSize))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	texture := textures.NewTexture()
	for y := 0; y < textures.TextureSize; y++ {
		for x := 0; x < textures.TextureSize; x++ {
			rgba := dst.At(x, y).(color.RGBA)
			texture.Set(uint(x), uint(y), int(rgba.R)*65536+int(rgba.G)*256+int(rgba.B))
		}
	}
	t.resources[id] = texture
	return texture
}

// Get retrieves a texture from the resources map using the given id. Returns nil if the id is not found.
func (t *Textures) Get(id string) *textures.Texture {
	if len(id) == 0 {
		return nil
	}
	if id == "-" {
		return nil
	}
	x, ok := t.resources[id]
	if !ok {
		return nil
	}
	return x
}
