package wad

import (
	"bufio"
	"image"
	"io"
	"os"

	"github.com/markel1974/godoom/engine/textures"
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

func (t *Textures) Add(name string, data *image.RGBA) {
	z := textures.NewTexture()
	for k := range data.Pix {
		z.Set(uint(k%1024), uint(k/1024), int(data.Pix[k]))
	}
	t.resources[name] = z
}

// Get retrieves a texture from the resources map using the given id. Returns nil if the id is not found.
func (t *Textures) Get(id string) *textures.Texture {
	x, ok := t.resources[id]
	if !ok {
		return nil
	}
	return x
}
