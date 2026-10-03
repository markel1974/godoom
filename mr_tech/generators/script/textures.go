package script

import (
	"io"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures is a collection of texture resources identified by unique string keys.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures loads textures from files in the specified directory and returns a Textures instance or an error.
func NewTextures(res common.IFileSystem, basePath string) (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	files, err := res.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for idx, f := range files {
		name := f.Name()
		if !f.IsDir() && len(name) > 0 {
			emissive := false
			if name[0] == '*' || name[0] == '+' {
				emissive = true
			}
			file, err := res.Open(basePath + f.Name())
			if err != nil {
				return nil, err
			}
			tex, err := common.PPMToTexture(file, f.Name(), uint32(idx), emissive)
			file.Close()
			if err == nil || err == io.EOF {
				t.resources[f.Name()] = tex
			} else {
				return nil, err
			}
		}
	}
	return t, nil
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

func (t *Textures) AddTexture(name string, tex *textures.Texture) {
	t.resources[name] = tex
}
