package lumps

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Texture represents a texture structure containing header information and a list of patches.
type Texture struct {
	Header  *TextureHeader
	Patches []*Patch
}

// TextureHeader represents metadata for a texture, including its name, dimensions, masking, and patch information.
type TextureHeader struct {
	TexName         string
	Masked          int32
	Width           int16
	Height          int16
	ColumnDirectory int32
	NumPatches      int16
}

// Patch represents a single patch entry in a texture definition file.
// It specifies the offset, patch number, step direction, and colormap index.
type Patch struct {
	XOffset     int16
	YOffset     int16
	PNameNumber int16
	StepDir     int16
	ColorMap    int16
}

// NewPatchNames reads a lump containing patch names from the file and returns the names as a slice of strings.
func NewPatchNames(f *os.File, info *LumpInfo) ([]string, error) {
	if err := Seek(f, info.Filepos); err != nil {
		return nil, err
	}
	var count uint32
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	p := make([][8]byte, count, count)
	if err := binary.Read(f, binary.LittleEndian, p); err != nil {
		return nil, err
	}
	pNames := make([]string, count, count)
	for idx, p := range p {
		pNames[idx] = ToString(p)
	}
	return pNames, nil
}

// NewTextures reads texture data from the given file and lump information, returning a slice of textures or an error.
func NewTextures(f *os.File, lumpInfo *LumpInfo) ([]*Texture, error) {
	if err := Seek(f, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	var count uint32
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	fmt.Printf("Loading %d textures ...\n", count)
	offsets := make([]int32, count, count)
	if err := binary.Read(f, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}

	var textures []*Texture

	for _, offset := range offsets {
		if err := Seek(f, lumpInfo.Filepos+int64(offset)); err != nil {
			return nil, err
		}
		type PrivateTextureHeader struct {
			TexName         [8]byte
			Masked          int32
			Width           int16
			Height          int16
			ColumnDirectory int32
			NumPatches      int16
		}
		pHeader := &PrivateTextureHeader{}
		if err := binary.Read(f, binary.LittleEndian, pHeader); err != nil {
			return nil, err
		}
		name := ToString(pHeader.TexName)
		pPatches := make([]Patch, pHeader.NumPatches, pHeader.NumPatches)
		if err := binary.Read(f, binary.LittleEndian, pPatches); err != nil {
			return nil, err
		}
		header := &TextureHeader{
			TexName:         name,
			Masked:          pHeader.Masked,
			Width:           pHeader.Width,
			Height:          pHeader.Height,
			ColumnDirectory: pHeader.ColumnDirectory,
			NumPatches:      pHeader.NumPatches,
		}
		patches := make([]*Patch, pHeader.NumPatches, pHeader.NumPatches)
		for idx, p := range pPatches {
			patches[idx] = &Patch{
				XOffset:     p.XOffset,
				YOffset:     p.YOffset,
				PNameNumber: p.PNameNumber,
				StepDir:     p.StepDir,
				ColorMap:    p.ColorMap,
			}
		}
		textures = append(textures, &Texture{Header: header, Patches: patches})
	}
	return textures, nil
}
