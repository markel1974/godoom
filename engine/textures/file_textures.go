package textures

import (
	"bufio"
	"io"
	"os"
)

// FileTextures represents a collection of textures mapped by their unique identifiers as strings.
type FileTextures struct {
	resources map[string]*Texture
}

// NewFileTextures initializes a new FileTextures instance by loading texture files from the given basePath directory.
// Returns an error if the directory cannot be read or if an error occurs when loading texture data.
func NewFileTextures(basePath string) (*FileTextures, error) {
	t := &FileTextures{
		resources: make(map[string]*Texture),
	}
	files, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for idx, f := range files {
		if !f.IsDir() {
			tex := NewTexture(f.Name(), uint32(idx), 1024, 1024)
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

// load reads texture data from a file and returns a pointer to a Texture object or an error if the operation fails.
func (t *FileTextures) load(tex *Texture, filename string) error {
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
				tex.data[x][y] = int(r)*65536 + int(g)*256 + int(b)
			}
		}
	}
}

// Get retrieves a texture from the resources map by its ID. If the ID does not exist, it returns nil.
func (t *FileTextures) Get(id string) *Texture {
	x, ok := t.resources[id]
	if !ok {
		return nil
	}
	return x
}

// GetNames returns a slice containing all the unique identifiers of the textures stored in the resources map.
func (t *FileTextures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}
