package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

type LABHeader struct {
	Magic           [4]byte
	Version         uint32
	NumFiles        uint32
	StringTableSize uint32
}
type LABEntry struct {
	NameOffset uint32
	DataOffset uint32
	DataSize   uint32
	FourCC     [4]byte
}

// LabFile represents a segment of a file with a specific offset and size for thread-safe read access.
type LabFile struct {
	file   io.ReaderAt
	offset int
	size   int
}

// NewLabFile initializes and returns a new LabFile instance with the specified file, offset, and size.
func NewLabFile(file io.ReaderAt, offset int, size int) *LabFile {
	return &LabFile{
		file:   file,
		offset: offset,
		size:   size,
	}
}

// Read reads the data from the file at the specified offset and returns a byte slice and an error if any occurs.
func (g *LabFile) Read() ([]byte, error) {
	data := make([]byte, g.size)
	if _, err := g.file.ReadAt(data, int64(g.offset)); err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	return data, nil
}

// ArchiveLab represents a structure for managing LAB file archives and their extracted data.
type ArchiveLab struct {
	fs        common.IResources
	container map[string]*LabFile
	files     []common.IReader
	levels    []string
	level     *Level
	bm        *BM
	textures  *Textures
	palIndex  int
	palettes  *Palettes
	entities  *Entities
	atxs      *ATXHandler
}

// NewArchiveLab initializes and returns a new instance of ArchiveLab with an empty container map.
func NewArchiveLab(fs common.IResources) *ArchiveLab {
	al := &ArchiveLab{
		fs:        fs,
		container: make(map[string]*LabFile),
		textures:  NewTextures(),
		palIndex:  0,
	}
	return al
}

// GetLevels retrieves the list of level names parsed from the archive.
func (al *ArchiveLab) GetLevels() []string {
	return al.levels
}

func (al *ArchiveLab) GetContainerKeys() []string {
	keys := make([]string, 0, len(al.container))
	for k := range al.container {
		keys = append(keys, k)
	}
	return keys
}

// Parse scans the specified directory for `.LAB` files, processes them, and adds their data to the instance container.
func (al *ArchiveLab) Parse(dirPath string) error {
	entries, dErr := al.fs.ReadDir(dirPath)
	if dErr != nil {
		return dErr
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pName := strings.TrimSpace(strings.ToUpper(entry.Name()))
		if !strings.HasSuffix(pName, ".LAB") {
			continue
		}
		var tFile string
		if len(dirPath) > 0 {
			tFile = dirPath + string(os.PathSeparator) + entry.Name()
		} else {
			tFile = entry.Name()
		}
		if err := al.add(tFile); err != nil {
			return err
		}
	}
	return nil
}

// add processes a .LAB archive file at the specified path and adds its contents to the ArchiveLab instance.
func (al *ArchiveLab) add(path string) error {
	file, err := al.fs.Open(path)
	if err != nil {
		return err
	}
	al.files = append(al.files, file)

	var header LABHeader
	if err = binary.Read(file, binary.LittleEndian, &header); err != nil {
		return err
	}
	if string(header.Magic[:]) != "LABN" {
		return fmt.Errorf("invalid LAB magic")
	}

	entries := make([]LABEntry, header.NumFiles)
	if err = binary.Read(file, binary.LittleEndian, entries); err != nil {
		return err
	}

	stringTable := make([]byte, header.StringTableSize)
	if err = binary.Read(file, binary.LittleEndian, stringTable); err != nil {
		return err
	}

	for _, e := range entries {
		start := e.NameOffset
		if start >= uint32(len(stringTable)) {
			fmt.Printf("Warning: invalid name offset %d in LAB archive\n", start)
			continue
		}
		end := start
		for end < uint32(len(stringTable)) && stringTable[end] != 0 {
			end++
		}
		fileName := string(stringTable[start:end])
		cleanName := CleanName(fileName)
		//fmt.Printf("Adding %s\n", cleanName)
		al.container[cleanName] = NewLabFile(file, int(e.DataOffset), int(e.DataSize))
		if pos := strings.Index(cleanName, LVTSuffix); pos > 0 {
			al.levels = append(al.levels, cleanName[:pos])
		}
	}
	return nil
}

// GetPayload retrieves the payload data associated with the given name from the container.
func (al *ArchiveLab) GetPayload(name string) ([]byte, error) {
	gob, ok := al.container[CleanName(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	return gob.Read()
}

// SetLevel loads the specified level topology, entities, and palette into the archive memory.
func (al *ArchiveLab) SetLevel(levelNumber int) error {
	if levelNumber < 0 || levelNumber >= len(al.levels) {
		return fmt.Errorf("invalid level number %d", levelNumber)
	}
	baseName := al.levels[levelNumber]
	levelName := baseName + LVTSuffix
	levelData, lErr := al.GetPayload(levelName)
	if lErr != nil {
		return fmt.Errorf("error reading %s: %w", levelName, lErr)
	}
	al.level = NewLevel()
	if err := al.level.Parse(bytes.NewReader(levelData)); err != nil {
		return fmt.Errorf("syntax error in %s: %w", levelName, err)
	}
	entitiesName := baseName + OBTSuffix
	entitiesData, pErr := al.GetPayload(entitiesName)
	if pErr != nil {
		return fmt.Errorf("error reading %s: %w", entitiesName, pErr)
	}
	al.entities = NewEntities()
	if err := al.entities.Parse(bytes.NewReader(entitiesData)); err != nil {
		return err
	}
	if len(al.level.Palettes) == 0 {
		return fmt.Errorf("no palettes declared in LVT")
	}
	al.palettes = NewPalettes(al)
	if err := al.palettes.Setup(al.level.Palettes); err != nil {
		return err
	}
	al.bm = NewBM()
	al.atxs = NewATXS(al, al.textures, al.palettes)
	for name := range al.container {
		if strings.HasSuffix(name, ATXSuffix) {
			if err := al.atxs.AddScript(name); err != nil {
				fmt.Printf("Error adding %s to ATX: %v\n", name, err)
			}
		}
	}
	return nil
}

// GetLevel returns the current parsed Level topology object.
func (al *ArchiveLab) GetLevel() *Level {
	return al.level
}

// GetEntities retrieves the parsed Entities object for the active level.
func (al *ArchiveLab) GetEntities() *Entities {
	return al.entities
}

// GetTextures retrieves the Textures manager associated with this archive.
func (al *ArchiveLab) GetTextures() *Textures {
	return al.textures
}

// AddTexture adds a texture to the texture manager by its name and returns a list of associated texture names or an error.
func (al *ArchiveLab) AddTexture(texName string) ([]string, error) {
	cleanName := CleanName(texName)
	if cached, ok := al.textures.GetFromCache(cleanName); ok {
		return cached, nil
	}
	if strings.HasSuffix(cleanName, ATXSuffix) {
		return al.atxs.CompileScriptTexture(cleanName, al.palIndex)
	}
	if frames, ok := al.atxs.CyclicFrame(cleanName, al.palIndex); ok {
		return frames, nil
	}

	data, dErr := al.GetPayload(cleanName)
	if dErr != nil {
		return nil, dErr
	}
	var images []*image.RGBA
	colorPal, cErr := al.palettes.Get(al.palIndex)
	if cErr != nil {
		return nil, cErr
	}
	var err error
	if strings.HasSuffix(cleanName, PCXSuffix) {
		parser := common.NewPCX()
		img, iErr := parser.Parse(bytes.NewReader(data), colorPal)
		if iErr != nil {
			return nil, iErr
		}
		images = append(images, img)
	} else {
		images, err = al.bm.Parse(bytes.NewReader(data), colorPal)
	}

	if err != nil {
		return nil, err
	}
	out := al.textures.AddTexture(cleanName, images)
	return out, nil
}

// AddRawTexture directly maps an indexed raw pixel array to the global texture manager using the active palette.
func (al *ArchiveLab) AddRawTexture(texName string, sizeX int, sizeY int, pixels []byte, dump bool) {
	colorPal, err := al.palettes.Get(al.palIndex)
	if err != nil {
		fmt.Printf("invalid palette index %s\n", err.Error())
		return
	}
	al.textures.AddRawTexture(texName, sizeX, sizeY, pixels, colorPal, dump)
}

// Close releases all open file handles associated with the ArchiveLab instance and resets its file list to nil.
func (al *ArchiveLab) Close() error {
	for _, f := range al.files {
		if f != nil {
			f.Close()
		}
	}
	al.files = nil
	return nil
}
