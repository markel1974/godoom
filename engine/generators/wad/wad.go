package wad

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/textures"
)

// WAD represents a parsed Doom-engine WAD file containing lumps, levels, textures, flats, patches, and play palettes.
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

// New initializes and returns a new WAD instance with preallocated maps and a default transparent palette index.
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

// Load initializes the WAD object by reading data from the specified file and loading necessary resources.
func (w *WAD) Load(filename string) error {
	var err error
	if w.file, err = os.Open(filename); err != nil {
		return err
	}
	if err = w.loadInfoTables(); err != nil {
		return err
	}
	if err = w.loadPlayPals(); err != nil {
		return err
	}
	if err = w.loadPatches(); err != nil {
		return err
	}
	if err = w.loadTextures(); err != nil {
		return err
	}
	if err = w.loadFlats(); err != nil {
		return err
	}
	return nil
}

// loadInfoTables parses information tables from a WAD file and populates lump and level maps for quick access.
func (w *WAD) loadInfoTables() error {
	var err error
	w.lumpInfos, err = lumps.NewLumpInfos(w.file)
	if err != nil {
		return err
	}
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

// loadPlayPals loads the PLAYPAL lump data from the WAD file into a PlayPal structure, returning an error if not found.
func (w *WAD) loadPlayPals() error {
	var err error
	playPalLump, ok := w.lumps["PLAYPAL"]
	if !ok {
		return fmt.Errorf("PLAYPAL not found")
	}
	lumpInfo := w.lumpInfos[playPalLump]
	if w.playPal, err = lumps.NewPlayPal(w.file, lumpInfo); err != nil {
		return err
	}
	return nil
}

// loadPatches reads and loads patch images from the WAD file, using the patch names defined in the PNAMES lump.
func (w *WAD) loadPatches() error {
	var err error
	pNamesLump, ok := w.lumps["PNAMES"]
	if !ok {
		return fmt.Errorf("PNAMES not found")
	}
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
		if err != nil {
			return err
		}
	}
	return nil
}

// loadTextures loads texture data from the WAD file and initializes the texture map with processed texture entries.
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
		ts, err := lumps.NewTextures(w.file, lumpInfo)
		if err != nil {
			return err
		}
		for _, t := range ts {
			w.textures[lumps.FixName(t.Header.TexName)] = t
		}
	}
	return nil
}

// loadFlats loads flat texture data from the WAD file into the flats map by parsing lumps between F_START and F_END.
func (w *WAD) loadFlats() error {
	start, ok := w.lumps["F_START"]
	if !ok {
		return fmt.Errorf("F_START not found")
	}
	end, ok := w.lumps["F_END"]
	if !ok {
		return fmt.Errorf("F_END not found")
	}
	for i := start; i < end; i++ {
		var err error
		lumpInfo := w.lumpInfos[i]
		w.flats[lumpInfo.Name], err = lumps.NewFlat(w.file, lumpInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTexture retrieves a texture by name from the WAD file, returning the texture and a boolean indicating success.
func (w *WAD) GetTexture(name string) (*lumps.Texture, bool) {
	texture, ok := w.textures[lumps.FixName(name)]
	return texture, ok
}

func (w *WAD) GetTextures() textures.ITextures {
	t, _ := NewTextures()
	for name := range w.textures {
		if data, err := w.GetTextureImage(name); err == nil {
			t.Add(CreateTextureId(name), data)
		}
	}
	for name := range w.flats {
		if data, err := w.GetFlatImage(name); err == nil {
			t.Add(CreateFlatId(name), data)
		}
	}
	return t
}

// GetImage retrieves an image by its patch name index and returns the image and a boolean indicating success.
func (w *WAD) GetImage(pNameNumber int16) (*lumps.Image, bool) {
	img, ok := w.patches[w.pNames[pNameNumber]]
	return img, ok
}

// GetFlat retrieves a flat texture from the WAD by its name. Returns the flat and true if found, otherwise false.
func (w *WAD) GetFlat(flatName string) (*lumps.Flat, bool) {
	flat, ok := w.flats[flatName]
	return flat, ok
}

// GetLevels returns a sorted slice of level names extracted from the WAD file's internal data structures.
func (w *WAD) GetLevels() []string {
	var result []string
	for name := range w.levels {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// GetLevel retrieves a Level by its name, loading its component data from the WAD file, and returns the Level or an error.
func (w *WAD) GetLevel(levelName string) (*Level, error) {
	var err error
	level := &Level{}
	levelIdx := w.levels[levelName]
	for i := levelIdx + 1; i < levelIdx+11; i++ {
		lumpInfo := w.lumpInfos[i]
		if err := lumps.Seek(w.file, lumpInfo.Filepos); err != nil {
			return nil, err
		}
		switch lumpInfo.Name {
		case "THINGS":
			if level.Things, err = lumps.NewThings(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "SIDEDEFS":
			if level.SideDefs, err = lumps.NewSideDefs(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "LINEDEFS":
			if level.LineDefs, err = lumps.NewLineDefs(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "VERTEXES":
			if level.Vertexes, err = lumps.NewVertexes(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "SEGS":
			if level.Segments, err = lumps.NewSegments(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "SSECTORS":
			if level.SubSectors, err = lumps.NewSubSectors(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "NODES":
			if level.Nodes, err = lumps.NewNodes(w.file, lumpInfo); err != nil {
				return nil, err
			}
		case "SECTORS":
			if level.Sectors, err = lumps.NewSectors(w.file, lumpInfo); err != nil {
				return nil, err
			}
		default:
			fmt.Printf("Unhandled lump %s\n", lumpInfo.Name)
		}
	}
	return level, nil
}

func (w *WAD) GetTextureImage(textureName string) (*image.RGBA, error) {
	texture, tOk := w.GetTexture(textureName)
	if !tOk || texture.Header == nil {
		return nil, fmt.Errorf("invalid texture %s", textureName)
	}

	texW := int(texture.Header.Width)
	texH := int(texture.Header.Height)
	rgba := image.NewRGBA(image.Rect(0, 0, texW, texH))

	for _, patch := range texture.Patches {
		img, iOk := w.GetImage(patch.PNameNumber)
		if !iOk {
			return nil, fmt.Errorf("invalid patch %d", patch.PNameNumber)
		}

		for x, col := range img.Columns {
			// Wrap-around orizzontale nativo
			drawX := (int(patch.XOffset) + x) % texW
			if drawX < 0 {
				drawX += texW
			}

			for _, post := range col.Posts {
				for y, pixel := range post.Pixels {
					drawY := int(patch.YOffset) + post.RowStart + y
					if drawY < 0 || drawY >= texH {
						continue
					}

					rgb := w.playPal.Palettes[0].Table[pixel]
					rgba.SetRGBA(drawX, drawY, color.RGBA{R: rgb.Red, G: rgb.Green, B: rgb.Blue, A: 255})
				}
			}
		}
	}
	return rgba, nil
}

func (w *WAD) GetFlatImage(pName string) (*image.RGBA, error) {
	z, ok := w.flats[pName]
	if !ok {
		return nil, fmt.Errorf("unknown patch %s", pName)
	}
	width := 64
	height := 64
	bounds := image.Rect(0, 0, width, height)
	rgba := image.NewRGBA(bounds)
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride " + pName)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := z.Data[y*width+x]
			// I flat di Doom non hanno mai zone trasparenti
			rgb := w.playPal.Palettes[0].Table[pixel]
			rgba.SetRGBA(x, y, color.RGBA{R: rgb.Red, G: rgb.Green, B: rgb.Blue, A: 255})
		}
	}
	return rgba, nil
}
