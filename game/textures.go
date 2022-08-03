package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
)

const (
	TextureSize  = 1024
	TextureBegin = 0
	TextureEnd   = TextureSize - 1
)

type Texture struct {
	data [TextureSize][TextureSize]int
}

func (t *Texture) Get(x int, y int) int {
	tx := x % TextureSize
	ty := y % TextureSize
	if tx < 0 {
		tx = 0
	}
	if ty < 0 {
		ty = 0
	}
	return t.data[tx][ty]
}

type Textures struct {
	resources map[string]*Texture
	viewMode  int
}

func NewTextures(viewMode int) (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*Texture),
		viewMode:  viewMode,
	}
	if viewMode != -1 {
		return t, nil
	}
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	files, err := ioutil.ReadDir(basePath)
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

func (t *Textures) load(filename string) (*Texture, error) {
	var texture = &Texture{}
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
				texture.data[x][y] = int(r)*65536 + int(g)*256 + int(b)
			}
		}
	}
}

func (t *Textures) Get(id string) *Texture {
	x, ok := t.resources[id]
	if !ok {
		return nil
	}
	return x
}
