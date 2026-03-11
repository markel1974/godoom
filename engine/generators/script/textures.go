package script

import (
	"bufio"
	"io"
	"os"

	"github.com/markel1974/godoom/engine/textures"
)

// Textures is a collection of texture resources identified by unique string keys.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures loads textures from files in the specified directory and returns a Textures instance or an error.
func NewTextures(basePath string) (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	files, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for idx, f := range files {
		if !f.IsDir() {
			tex := textures.NewTexture(f.Name(), uint32(idx), 1024, 1024)
			err = t.load(tex, basePath+f.Name())
			if err == nil || err == io.EOF {
				t.resources[f.Name()] = tex
			} else {
				return nil, err
			}
		}
	}
	return t, nil
}

// load reads texture data from the specified file and populates the given Animations instance with pixel values.
func (t *Textures) load(tex *textures.Texture, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(0x11, io.SeekStart); err != nil {
		return err
	}
	br := bufio.NewReader(file)
	var r byte
	var g byte
	var b byte
	for {
		for y := 0; y < 1024; y++ {
			for x := 0; x < 1024; x++ {
				if r, err = br.ReadByte(); err != nil {
					return err
				}
				if g, err = br.ReadByte(); err != nil {
					return err
				}
				if b, err = br.ReadByte(); err != nil {
					return err
				}
				tex.Set(x, y, int(r)*65536+int(g)*256+int(b))
			}
		}
	}
}

// Get retrieves textures matching the provided `ids` from the Textures resource map. Returns nil if an id is not found.
func (t *Textures) Get(ids []string) []*textures.Texture {
	var out []*textures.Texture
	for _, id := range ids {
		x, ok := t.resources[id]
		if !ok {
			return nil
		}
		out = append(out, x)
	}
	return out
}

// GetNames returns a list of all texture names (keys) stored in the Textures' resources map.
func (t *Textures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}
