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

// ExtLevel is the file extension used to identify level files in the archive.
const (
	ExtLevel = ".LEV"
)

// GobHeader represents the header of a GOB archive file, containing metadata like magic bytes and master table offset.
type GobHeader struct {
	Magic     [4]byte
	MasterOfs int32
}

// GobEntry represents a single entry in a GOB archive with its offset, size, and name.
type GobEntry struct {
	Offset int32
	Size   int32
	Name   [13]byte
}

// Gob represents a structure linking a file reader and an entry within the file.
type Gob struct {
	file  io.ReaderAt
	entry *GobEntry
}

// NewGob creates a new Gob instance using the provided reader and GobEntry for managing file data.
func NewGob(file io.ReaderAt, entry *GobEntry) *Gob {
	return &Gob{file: file, entry: entry}
}

// Read retrieves the byte data from the file based on the entry's offset and size. Returns an error if the read fails.
func (g *Gob) Read() ([]byte, error) {
	data := make([]byte, g.entry.Size)
	// ReadAt esegue una lettura thread-safe all'offset specificato
	if _, err := g.file.ReadAt(data, int64(g.entry.Offset)); err != nil {
		// ReadAt può ritornare io.EOF se legge esattamente fino alla fine, gestiscilo se necessario
		if err != io.EOF {
			return nil, err
		}
	}
	return data, nil
}

// ArchiveGob represents a collection of game assets, including entries, files, levels, textures, and entities.
type ArchiveGob struct {
	entries  map[string]*Gob
	files    []*os.File
	levels   []string
	level    *Level
	bm       *BM
	textures *Textures
	colorPal [256]color.RGBA
	entities *Entities
}

// NewArchiveGob initializes and returns a new ArchiveGob instance with default maps and textures.
func NewArchiveGob() *ArchiveGob {
	return &ArchiveGob{
		entries:  make(map[string]*Gob),
		textures: NewTextures(),
	}
}

// Parse scans the specified directory for .GOB files, parses them, and adds their entries to the ArchiveGob instance.
// Returns an error if the directory cannot be read or if any parsing operation fails.
func (g *ArchiveGob) Parse(dirPath string) error {
	entries, dErr := os.ReadDir(dirPath)
	if dErr != nil {
		return dErr
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pName := strings.TrimSpace(strings.ToUpper(entry.Name()))
		if !strings.HasSuffix(pName, ".GOB") {
			continue
		}
		if err := g.add(dirPath + string(os.PathSeparator) + entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// add reads a GOB file, parses its header and entries, and adds them to the ArchiveGob instance.
func (g *ArchiveGob) add(filename string) error {
	f, fErr := os.Open(filename)
	if fErr != nil {
		return fErr
	}
	g.files = append(g.files, f)
	var header GobHeader
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return err
	}
	if string(header.Magic[:]) != "GOB\n" {
		return fmt.Errorf("invalid magic")
	}
	if _, err := f.Seek(int64(header.MasterOfs), io.SeekStart); err != nil {
		return err
	}
	var numEntries int32
	if err := binary.Read(f, binary.LittleEndian, &numEntries); err != nil {
		return err
	}
	for i := int32(0); i < numEntries; i++ {
		entry := &GobEntry{}
		if err := binary.Read(f, binary.LittleEndian, entry); err != nil {
			return err
		}
		nameLen := bytes.IndexByte(entry.Name[:], 0)
		if nameLen == -1 {
			nameLen = len(entry.Name)
		}
		cleanName := strings.ToUpper(string(entry.Name[:nameLen]))
		g.entries[cleanName] = NewGob(f, entry)
		if pos := strings.Index(cleanName, ExtLevel); pos > 0 {
			g.levels = append(g.levels, cleanName[:pos])
		}
	}
	return nil
}

// GetLevels returns a slice of level names available in the ArchiveGob.
func (g *ArchiveGob) GetLevels() []string {
	return g.levels
}

// GetPayload retrieves the payload data for a given entry name. Returns an error if the entry is not found or unreadable.
func (g *ArchiveGob) GetPayload(name string) ([]byte, error) {
	gob, ok := g.entries[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	return gob.Read()
}

// Close releases all open file handles and cleans up associated resources within the ArchiveGob instance.
func (g *ArchiveGob) Close() error {
	for _, f := range g.files {
		if f != nil {
			f.Close()
		}
	}
	g.files = nil
	return nil
}

// SetLevel loads the specified level and its associated components, including entities and palette, into the archive.
func (g *ArchiveGob) SetLevel(levelNumber int) error {
	if levelNumber < 0 || levelNumber > len(g.levels) {
		return fmt.Errorf("invalid level number %d", levelNumber)
	}
	baseName := g.levels[levelNumber]
	levelName := baseName + ExtLevel
	levelData, err := g.GetPayload(levelName)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", levelName, err)
	}

	g.level = NewLevel()
	if err = g.level.Parse(bytes.NewReader(levelData)); err != nil {
		return fmt.Errorf("syntax error in %s: %w", levelName, err)
	}
	entitiesName := baseName + ".O"
	entitiesData, err := g.GetPayload(entitiesName)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", entitiesName, err)
	}
	g.entities = NewEntities()
	if err = g.entities.Parse(bytes.NewReader(entitiesData)); err != nil {
		return err
	}
	palData, err := g.GetPayload(g.entities.LevelName + ".PAL")
	if err != nil {
		palData, err = g.GetPayload("SECBASE.PAL")
		if err != nil {
			return fmt.Errorf("master palette non found: %w", err)
		}
	}
	palette := NewPalette()
	g.colorPal, err = palette.Parse(bytes.NewReader(palData))
	if err != nil {
		return fmt.Errorf("error while parsing palette: %w", err)
	}
	g.bm = NewBM()

	return nil
}

// GetLevel returns the current level object associated with the ArchiveGob instance.
func (g *ArchiveGob) GetLevel() *Level {
	return g.level
}

// GetEntities retrieves the Entities object associated with the ArchiveGob instance.
func (g *ArchiveGob) GetEntities() *Entities {
	return g.entities
}

// GetTextures retrieves the Textures instance associated with the ArchiveGob, which manages 2D texture resources.
func (g *ArchiveGob) GetTextures() *Textures {
	return g.textures
}

// AddTexture adds a texture by its name, parses the texture data, and registers it in the textures collection.
func (g *ArchiveGob) AddTexture(texName string) ([]string, error) {
	bmData, err := g.GetPayload(texName)
	if err != nil {
		fmt.Printf("payload %s not found: %v\n", texName, err)
		return nil, err
	}
	images, err := g.bm.Parse(bytes.NewReader(bmData), g.colorPal)
	if err != nil {
		return nil, err
	}
	out := g.textures.AddTexture(texName, images)
	return out, nil
}

// AddRawTexture adds a raw texture to the texture manager using indexed pixel data and a color palette.
func (g *ArchiveGob) AddRawTexture(texName string, sizeX int, sizeY int, pixels []byte) {
	g.textures.AddRawTexture(texName, sizeX, sizeY, pixels, g.colorPal)
}
