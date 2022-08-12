package wad

import (
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"image"
	"image/color"
	"os"
	"sort"
)




type WAD struct {
	file                    *os.File
	lumpInfos               []*lumps.LumpInfo
	playPal                 *lumps.PlayPal
	patches                 map[string]*lumps.Image
	textures                map[string]*lumps.Texture
	flats                   map[string]*lumps.Flat
	levels                  map[string]int
	lumps                   map[string]int
	pNames                  []string
	transparentPaletteIndex byte
}



func New() *WAD {
	return &WAD{
		lumps:                   make(map[string]int),
		levels:                  make(map[string]int),
		textures:                make(map[string]*lumps.Texture),
		flats:                   make(map[string]*lumps.Flat),
		patches:                 make(map[string]*lumps.Image),
		transparentPaletteIndex: 255,
	}
}

func (w * WAD) Load(filename string) error {
	var err error
	if w.file, err = os.Open(filename); err != nil { return err }
	if err = w.loadInfoTables(); err != nil { return err }
	if err = w.loadPlayPals(); err != nil { return err }
	if err = w.loadPatches(); err != nil { return err }
	if err = w.loadTextures(); err != nil { return err }
	if err = w.loadFlats(); err != nil { return err }
	return nil
}

func (w *WAD) loadInfoTables() error {
	var err error
	w.lumpInfos, err = lumps.NewLumpInfos(w.file)
	if err != nil { return err }
	for i, l := range w.lumpInfos {
		if l.Name == "THINGS" {
			levelIdx := i - 1
			levelLump := w.lumpInfos[levelIdx]
			w.levels[levelLump.Name] = levelIdx
		}
		w.lumps[l.Name] = i
	}
	return nil
}

func (w *WAD) loadPlayPals() error {
	var err error
	playPalLump, ok := w.lumps["PLAYPAL"]
	if !ok { return fmt.Errorf("PLAYPAL not found") }
	lumpInfo := w.lumpInfos[playPalLump]
	if w.playPal, err = lumps.NewPlayPal(w.file, lumpInfo); err != nil {
		return err
	}
	return nil
}

func (w *WAD) loadPatches() error {
	var err error
	pNamesLump, ok := w.lumps["PNAMES"]
	if !ok { return fmt.Errorf("PNAMES not found") }
	lumpInfo := w.lumpInfos[pNamesLump]
	w.pNames, err = lumps.NewPatchNames(w.file, lumpInfo)
	if err != nil {
		return err
	}
	for _, pName := range w.pNames {
		var err error
		pNamesLump := w.lumps[pName]
		lumpInfo := w.lumpInfos[pNamesLump]
		w.patches[pName], err = lumps.NewImage(w.file, lumpInfo, w.transparentPaletteIndex)
		if err != nil { return err }
	}
	return nil
}

func (w *WAD) loadTextures() error {
	var textureLumps []int
	if lump, ok := w.lumps["TEXTURE1"]; ok {
		textureLumps = append(textureLumps, lump)
	}
	if lump, ok := w.lumps["TEXTURE2"]; ok {
		textureLumps = append(textureLumps, lump)
	}
	for _, i := range textureLumps {
		lumpInfo := w.lumpInfos[i]
		textures, err := lumps.NewTextures(w.file, lumpInfo)
		if err != nil { return err }
		for _, t := range textures {
			w.textures[lumps.FixName(t.Header.TexName)] = t
		}
	}
	return nil
}

func (w *WAD) loadFlats() error {
	start, ok := w.lumps["F_START"]
	if !ok { return fmt.Errorf("F_START not found") }
	end, ok := w.lumps["F_END"]
	if !ok { return fmt.Errorf("F_END not found") }
	for i := start; i < end; i++ {
		var err error
		lumpInfo := w.lumpInfos[i]
		w.flats[lumpInfo.Name], err = lumps.NewFlat(w.file, lumpInfo)
		if err != nil { return err }
	}
	return nil
}

func (w *WAD) GetTexture(name string) (*lumps.Texture, bool) {
	texture, ok := w.textures[lumps.FixName(name)]
	return texture, ok
}

func (w *WAD) GetImage(pNameNumber int16) (*lumps.Image, bool) {
	img, ok := w.patches[w.pNames[pNameNumber]]
	return img, ok
}

func (w *WAD) GetFlat(flatName string) (*lumps.Flat, bool) {
	flat, ok := w.flats[flatName]
	return flat, ok
}

func (w *WAD) GetLevels() []string {
	var result []string
	for name := range w.levels {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (w *WAD) GetLevel(levelName string) (*Level, error) {
	var err error
	level := &Level{}
	levelIdx := w.levels[levelName]
	for i := levelIdx + 1; i < levelIdx+11; i++ {
		lumpInfo := w.lumpInfos[i]
		if err := lumps.Seek(w.file, lumpInfo.Filepos); err != nil { return nil, err }
		switch lumpInfo.Name {
		case "THINGS": if level.Things, err = lumps.NewThings(w.file, lumpInfo); err != nil { return nil, err }
		case "SIDEDEFS": if level.SideDefs, err = lumps.NewSideDefs(w.file, lumpInfo); err != nil { return nil, err }
		case "LINEDEFS": if level.LineDefs, err = lumps.NewLineDefs(w.file, lumpInfo); err != nil { return nil, err }
		case "VERTEXES": if level.Vertexes, err = lumps.NewVertexes(w.file, lumpInfo); err != nil { return nil, err }
		case "SEGS": if level.Segments, err = lumps.NewSegments(w.file, lumpInfo); err != nil { return nil, err }
		case "SSECTORS": if level.SubSectors, err = lumps.NewSubSectors(w.file, lumpInfo); err != nil { return nil, err }
		case "NODES": if level.Nodes, err = lumps.NewNodes(w.file, lumpInfo); err != nil { return nil, err }
		case "SECTORS": if level.Sectors, err = lumps.NewSectors(w.file, lumpInfo); err != nil { return nil, err }
		default: fmt.Printf("Unhandled lump %s\n", lumpInfo.Name)
		}
	}
	return level, nil
}


func (w * WAD) GetTextureImage(textureName string) (*image.RGBA, error) {
	texture, ok := w.GetTexture(textureName)
	if !ok {
		return nil, errors.New("unknown texture " + textureName)
	}
	if texture.Header == nil {
		return nil, errors.New("nil header " + textureName)
	}
	bounds := image.Rect(0, 0, int(texture.Header.Width), int(texture.Header.Height))
	rgba := image.NewRGBA(bounds)
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride " + textureName)
	}
	for _, patch := range texture.Patches {
		img, ok := w.GetImage(patch.PNameNumber)
		if !ok {
			return nil, errors.New(fmt.Sprintf("unknown patch %d for %s", patch.PNameNumber, textureName))
		}
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				pixel := img.Pixels[y*img.Width+x]
				var alpha uint8
				if pixel == w.transparentPaletteIndex {
					alpha = 0
				} else {
					alpha = 255
				}
				rgb := w.playPal.Palettes[0].Table[pixel]
				rgba.Set(int(patch.XOffset) + x, int(patch.YOffset) + y, color.RGBA{R: rgb.Red, G: rgb.Green, B: rgb.Blue, A: alpha})
			}
		}
	}
	return rgba, nil

	/*
		var texId uint32
		gl.GenTextures(1, &texId)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texId)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
		return texId, nil
	*/
}