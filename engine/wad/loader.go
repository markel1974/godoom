package wad

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type header struct {
	Magic        [4]byte
	NumLumps     int32
	InfoTableOfs int32
}


type Loader struct {
	header *header
}

func NewLoader() * Loader {
	return &Loader{
	}
}

func (l * Loader) Setup(w * WAD) error {
	var err error
	if err = l.loadHeader(w); err != nil { return err }
	if err = l.loadInfoTables(w); err != nil { return err }
	if err = l.loadPlayPal(w); err != nil { return err }
	if err = l.loadPatchNames(w); err != nil { return err }
	if err = l.loadPatchLumps(w); err != nil { return err }
	if err = l.loadTextureLumps(w); err != nil { return err }
	if err = l.loadFlatLumps(w); err != nil { return err }
	return nil
}

func (l *Loader) loadHeader(w * WAD) error {
	l.header = &header{}
	if err := binary.Read(w.file, binary.LittleEndian, l.header); err != nil { return err }
	if string(l.header.Magic[:]) != "IWAD" { return fmt.Errorf("bad magic: %s\n", l.header.Magic) }
	return nil
}

func (l *Loader) loadInfoTables(w * WAD) error {
	if err := Seek(w.file, int64(l.header.InfoTableOfs)); err != nil {
		return err
	}
	w.lumps = map[string]int{}
	w.levels = map[string]int{}
	type PrivateLumpInfo struct {
		Filepos int32
		Size    int32
		Name    [8]byte
	}

	w.lumpInfos = make([]*LumpInfo, l.header.NumLumps, l.header.NumLumps)
	for i := int32(0); i < l.header.NumLumps; i++ {
		pLumpInfo := &PrivateLumpInfo{}
		if err := binary.Read(w.file, binary.LittleEndian, pLumpInfo); err != nil {
			return err
		}
		name := ToString(pLumpInfo.Name)
		if name == "THINGS" {
			levelIdx := int(i - 1)
			levelLump := w.lumpInfos[levelIdx]
			w.levels[levelLump.Name] = levelIdx
		}
		w.lumps[name] = int(i)
		w.lumpInfos[i] = &LumpInfo{
			Filepos: int64(pLumpInfo.Filepos),
			Size:    pLumpInfo.Size,
			Name:    name,
		}
	}
	return nil
}

func (l *Loader) loadPlayPal(w * WAD) error {
	playPalLump := w.lumps["PLAYPAL"]
	lumpInfo := w.lumpInfos[playPalLump]
	if err := Seek(w.file, int64(lumpInfo.Filepos)); err != nil {
		return err
	}
	fmt.Printf("Loading palette ...\n")
	w.playPal = &PlayPal{}
	if err := binary.Read(w.file, binary.LittleEndian, w.playPal); err != nil {
		return err
	}
	return nil
}

func (l *Loader) loadPatchNames(w * WAD) error {
	pNamesLump := w.lumps["PNAMES"]
	lumpInfo := w.lumpInfos[pNamesLump]
	if err := Seek(w.file, int64(lumpInfo.Filepos)); err != nil {
		return err
	}
	var count uint32
	if err := binary.Read(w.file, binary.LittleEndian, &count); err != nil {
		return err
	}
	fmt.Printf("Loading %d patches ...\n", count)
	pNames := make([][8]byte, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pNames); err != nil {
		return err
	}
	w.pNames = make([]string, count, count)
	for idx, p := range pNames {
		w.pNames[idx] = ToString(p)
	}
	return nil
}

func (l *Loader) loadPatchLumps(w * WAD) error {
	w.patches = make(map[string]*Image)
	for _, pName := range w.pNames {
		lumpInfo := w.lumpInfos[w.lumps[pName]]
		if err := Seek(w.file, lumpInfo.Filepos); err != nil {
			return err
		}
		lump := make([]byte, lumpInfo.Size, lumpInfo.Size)
		n, err := w.file.Read(lump)
		if err != nil {
			return err
		}
		if n != int(lumpInfo.Size) {
			return fmt.Errorf("Truncated lump")
		}
		reader := bytes.NewBuffer(lump[0:])
		var header PictureHeader
		if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
			return err
		}
		if header.Width > 4096 || header.Height > 4096 {
			continue
		}
		offsets := make([]int32, header.Width, header.Width)
		if err := binary.Read(reader, binary.LittleEndian, offsets); err != nil {
			return err
		}
		size := int(header.Width) * int(header.Height)
		pixels := make([]byte, size, size)
		for y := 0; y < int(header.Height); y++ {
			for x := 0; x < int(header.Width); x++ {
				pixels[y*int(header.Width)+x] = l.transparentPaletteIndex
			}
		}
		for columnIndex, offset := range offsets {
			for {
				rowStart := lump[offset]
				offset += 1
				if rowStart == 255 {
					break
				}
				numPixels := lump[offset]
				offset += 1
				offset += 1 /* Padding */
				for i := 0; i < int(numPixels); i++ {
					pixelOffset := (int(rowStart)+i)*int(header.Width) + columnIndex
					pixels[pixelOffset] = lump[offset]
					offset += 1
				}
				offset += 1 /* Padding */
			}
		}
		w.patches[pName] = &Image{Width: int(header.Width), Height: int(header.Height), Pixels: pixels}
	}
	return nil
}

func (l *Loader) loadTextureLumps(w * WAD) error {
	textureLumps := make([]int, 0, 2)
	if lump, ok := w.lumps["TEXTURE1"]; ok {
		textureLumps = append(textureLumps, lump)
	}
	if lump, ok := w.lumps["TEXTURE2"]; ok {
		textureLumps = append(textureLumps, lump)
	}
	w.textures = make(map[string]*Texture)
	for _, i := range textureLumps {
		lumpInfo := w.lumpInfos[i]
		if err := Seek(w.file, lumpInfo.Filepos); err != nil {
			return err
		}
		var count uint32
		if err := binary.Read(w.file, binary.LittleEndian, &count); err != nil {
			return err
		}
		fmt.Printf("Loading %d textures ...\n", count)
		offsets := make([]int32, count, count)
		if err := binary.Read(w.file, binary.LittleEndian, offsets); err != nil {
			return err
		}
		for _, offset := range offsets {
			if err := Seek(w.file, lumpInfo.Filepos + int64(offset)); err != nil {
				return err
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
			if err := binary.Read(w.file, binary.LittleEndian, pHeader); err != nil {
				return err
			}
			name := ToString(pHeader.TexName)
			patches := make([]Patch, pHeader.NumPatches, pHeader.NumPatches)
			if err := binary.Read(w.file, binary.LittleEndian, patches); err != nil {
				return err
			}
			header := &TextureHeader{
				TexName:         name,
				Masked:          pHeader.Masked,
				Width:           pHeader.Width,
				Height:          pHeader.Height,
				ColumnDirectory: pHeader.ColumnDirectory,
				NumPatches:      pHeader.NumPatches,
			}
			w.textures[name] = &Texture{Header: header, Patches: patches}
		}
	}
	return nil
}

func (l *Loader) loadFlatLumps(w * WAD) error {
	w.flats = make(map[string]*Flat)
	startLump, ok := w.lumps["F_START"]
	if !ok {
		return fmt.Errorf("F_START not found")
	}
	endLump, ok := w.lumps["F_END"]
	if !ok {
		return fmt.Errorf("F_END not found")
	}
	for i := startLump; i < endLump; i++ {
		lumpInfo := w.lumpInfos[i]
		if err := Seek(w.file, int64(lumpInfo.Filepos)); err != nil {
			return err
		}
		size := 4096
		data := make([]byte, size, size)
		if err := binary.Read(w.file, binary.LittleEndian, data); err != nil {
			return err
		}
		w.flats[lumpInfo.Name] = &Flat{Data: data}
	}
	return nil
}
