package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
)

// ExtLevelLVT represents the file extension for level topology files within a LAB archive.
const ExtLevelLVT = ".LVT"

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
	container map[string]*LabFile
	files     []*os.File
	levels    []string
	level     *Level
	bm        *BM
	textures  *Textures
	colorPal  [256]color.RGBA
	entities  *Entities
}

// NewArchiveLab initializes and returns a new instance of ArchiveLab with an empty container map.
func NewArchiveLab() *ArchiveLab {
	return &ArchiveLab{
		container: make(map[string]*LabFile),
		textures:  NewTextures(),
	}
}

// GetLevels retrieves the list of level names parsed from the archive.
func (al *ArchiveLab) GetLevels() []string {
	return al.levels
}

// Parse scans the specified directory for `.LAB` files, processes them, and adds their data to the instance container.
func (al *ArchiveLab) Parse(dirPath string) error {
	entries, dErr := os.ReadDir(dirPath)
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
		entryPath := dirPath + string(os.PathSeparator) + entry.Name()
		if err := al.add(entryPath); err != nil {
			return err
		}
	}
	return nil
}

// add processes a .LAB archive file at the specified path and adds its contents to the ArchiveLab instance.
func (al *ArchiveLab) add(path string) error {
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

	file, err := os.Open(path)
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
		cleanName := al.cleanName(fileName)
		al.container[cleanName] = NewLabFile(file, int(e.DataOffset), int(e.DataSize))

		if pos := strings.Index(cleanName, ExtLevelLVT); pos > 0 {
			al.levels = append(al.levels, cleanName[:pos])
		}
	}
	return nil
}

// GetPayload retrieves the payload data associated with the given name from the container.
func (al *ArchiveLab) GetPayload(name string) ([]byte, error) {
	gob, ok := al.container[al.cleanName(name)]
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
	levelName := baseName + ExtLevelLVT

	levelData, err := al.GetPayload(levelName)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", levelName, err)
	}

	al.level = NewLevel()
	if err = al.level.Parse(bytes.NewReader(levelData)); err != nil {
		return fmt.Errorf("syntax error in %s: %w", levelName, err)
	}

	entitiesName := baseName + ".OBT"
	entitiesData, err := al.GetPayload(entitiesName)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", entitiesName, err)
	}

	al.entities = NewEntities()
	if err = al.entities.Parse(bytes.NewReader(entitiesData)); err != nil {
		return err
	}

	palData, err := al.GetPayload(al.entities.LevelName + ".PAL")
	if err != nil {
		// Fallback principale per Outlaws
		palData, err = al.GetPayload("OLPAL.PAL")
		if err != nil {
			// Fallback generico per compatibilità
			palData, err = al.GetPayload("SECBASE.PAL")
			if err != nil {
				return fmt.Errorf("master palette not found: %w", err)
			}
		}
	}

	palette := NewPalette()
	al.colorPal, err = palette.Parse(bytes.NewReader(palData))
	if err != nil {
		return fmt.Errorf("error while parsing palette: %w", err)
	}
	al.bm = NewBM()

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

// AddTexture dynamically loads a texture by name, decodes its payload, and caches it.
func (al *ArchiveLab) AddTexture(texName string) ([]string, error) {
	bmData, err := al.GetPayload(texName)
	if err != nil {
		fmt.Printf("payload %s not found: %v\n", texName, err)
		return nil, err
	}
	images, err := al.bm.Parse(bytes.NewReader(bmData), al.colorPal)
	if err != nil {
		return nil, err
	}
	out := al.textures.AddTexture(texName, images)
	return out, nil
}

// AddRawTexture directly maps an indexed raw pixel array to the global texture manager using the active palette.
func (al *ArchiveLab) AddRawTexture(texName string, sizeX int, sizeY int, pixels []byte) {
	al.textures.AddRawTexture(texName, sizeX, sizeY, pixels, al.colorPal)
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

// cleanName standardizes a file name by trimming whitespace and converting it to uppercase.
func (al *ArchiveLab) cleanName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
