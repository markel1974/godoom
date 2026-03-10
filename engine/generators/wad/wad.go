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

// WAD represents a data structure for manipulating Doom-engine WAD files, including textures, levels, and graphics.
type WAD struct {
	file                    *os.File
	lumpInfos               []*lumps.LumpInfo
	playPal                 *lumps.PlayPal
	patches                 map[string]*lumps.Image
	textures                map[string]*lumps.Texture
	flats                   map[string]*lumps.Flat
	levels                  map[string]int
	lumps                   map[string]int
	patchNames              []string
	transparentPaletteIndex byte
}

// New creates a new instance of the WAD structure with initialized maps for lumps, levels, textures, flats, and patches.
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

// Load opens the WAD file, loads its internal structures, and initializes its resources.
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

// loadInfoTables loads lump information from the WAD file and organizes it into levels and lumps for easier access.
// It reads lump details using lumps.NewLumpInfos and maps level and lump names to their respective indices.
// Returns an error if lump information parsing fails or if any discrepancies occur.
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

// loadPlayPals loads the PLAYPAL lump from the WAD file and initializes the PlayPal structure. Returns an error if loading fails.
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

// loadPatches loads patch data from the WAD file by reading the PNAMES lump and mapping each patch to its graphic.
func (w *WAD) loadPatches() error {
	var err error
	pLumpIdx, ok := w.lumps["PNAMES"]
	if !ok {
		return fmt.Errorf("PNAMES not found")
	}
	if pLumpIdx < 0 || pLumpIdx >= len(w.lumpInfos) {
		return fmt.Errorf("invalid PNAMES index")
	}
	pLumpInfo := w.lumpInfos[pLumpIdx]
	w.patchNames, err = lumps.NewPatchNames(w.file, pLumpInfo)
	if err != nil {
		return err
	}
	for _, pName := range w.patchNames {
		idx, found := w.lumps[pName]
		if !found {
			fmt.Println("patch not found", pName)
			continue
		}
		if idx < 0 || idx >= len(w.lumpInfos) {
			return fmt.Errorf("invalid PNAMES index")
		}
		info := w.lumpInfos[idx]
		w.patches[pName], err = lumps.NewImage(w.file, info, w.transparentPaletteIndex)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadTextures loads the texture lumps from the WAD file and populates the textures map with parsed texture data.
// It handles both TEXTURE1 and TEXTURE2 lumps if present and processes the associated texture data.
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

// loadFlats loads all flat textures (floor/ceiling textures) from the F_START to F_END range into the WAD structure.
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

// GetTexture retrieves a texture by its name from the WAD archive, returning the texture and a boolean indicating success.
func (w *WAD) GetTexture(name string) (*lumps.Texture, bool) {
	texture, ok := w.textures[lumps.FixName(name)]
	return texture, ok
}

// GetTextures retrieves all textures and flats as an ITextures object with their corresponding data added.
func (w *WAD) GetTextures() textures.ITextures {
	t, _ := NewTextures()
	for name := range w.textures {
		if data, err := w.GetTextureImage(name); err == nil {
			t.Add(TextureCreateId(name), data)
		}
	}
	for name := range w.flats {
		if data, err := w.GetFlatImage(name); err == nil {
			t.Add(FlatCreateId(name), data)
		}
	}
	return t
}

// GetImage retrieves an image from the WAD by its patch name index (pNameNumber).
// Returns the image object or an error if the index is invalid or the patch is not found.
func (w *WAD) GetImage(pNameNumber int16) (*lumps.Image, error) {
	if pNameNumber < 0 || pNameNumber >= int16(len(w.patchNames)) {
		return nil, fmt.Errorf("invalid patch name index %d", pNameNumber)
	}
	z := w.patchNames[pNameNumber]
	img, ok := w.patches[z]
	if !ok {
		return nil, fmt.Errorf("can't find patch %s in patches", z)
	}
	return img, nil
}

// GetFlat retrieves a flat texture by its name from the WAD. It returns the flat and a boolean indicating success.
func (w *WAD) GetFlat(flatName string) (*lumps.Flat, bool) {
	flat, ok := w.flats[flatName]
	return flat, ok
}

// GetLevels returns a sorted list of all level names defined in the WAD file.
func (w *WAD) GetLevels() []string {
	var result []string
	for name := range w.levels {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// GetLevel retrieves the specified level data by its name and parses its associated lumps into a Level structure.
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

// GetTextureImage generates an RGBA image for the specified texture name by rendering patches and applying the palette.
// Returns the resulting image or an error if the texture is invalid or cannot be processed.
func (w *WAD) GetTextureImage(textureName string) (*image.RGBA, error) {
	texture, tOk := w.GetTexture(textureName)
	if !tOk || texture.Header == nil {
		return nil, fmt.Errorf("invalid texture %s", textureName)
	}

	texW := int(texture.Header.Width)
	texH := int(texture.Header.Height)
	rgba := image.NewRGBA(image.Rect(0, 0, texW, texH))

	for _, patch := range texture.Patches {
		img, err := w.GetImage(patch.PNameNumber)
		if err != nil {
			fmt.Println(err.Error())
			continue
			//return nil, fmt.Errorf("invalid patch %d", patch.PNameNumber)
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

// GetFlatImage retrieves a 64x64 flat image by name from the WAD's flat textures and returns it as an *image.RGBA.
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
