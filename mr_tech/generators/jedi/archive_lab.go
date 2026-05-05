package jedi

import (
	"encoding/binary"
	"fmt"
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
	// ReadAt esegue una lettura thread-safe all'offset specificato
	if _, err := g.file.ReadAt(data, int64(g.offset)); err != nil {
		// ReadAt può ritornare io.EOF se legge esattamente fino alla fine, gestiscilo se necessario
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
}

// NewArchiveLab initializes and returns a new instance of ArchiveLab with an empty container map.
func NewArchiveLab() *ArchiveLab {
	return &ArchiveLab{
		container: make(map[string]*LabFile),
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
		Magic           [4]byte // Deve essere "LABN"
		Version         uint32  // Solitamente 65536 (ovvero 1.0 in fixed-point 16.16)
		NumFiles        uint32  // Numero totale di file nell'archivio
		StringTableSize uint32  // Dimensione in byte del blocco contenente i nomi
	}
	type LABEntry struct {
		NameOffset uint32  // Offset del nome del file (relativo all'inizio della String Table)
		DataOffset uint32  // Offset assoluto nel file .LAB dove iniziano i dati del file
		DataSize   uint32  // Dimensione in byte del file
		FourCC     [4]byte // Tipo di file (opzionale, spesso 4 caratteri ASCII identificativi)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	//defer file.Close()
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
		// Sicurezza contro i file corrotti o offset sballati
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

		fmt.Println("ADDING ", cleanName)
		if pos := strings.Index(cleanName, ExtLevelLVT); pos > 0 {
			al.levels = append(al.levels, cleanName[:pos])
		}
	}
	//Found file: TOWN.LVT Level Toplogy. Contiene i vertici 2D, le altezze dei settori, le texture applicate ai muri, i piani inclinati (slopes)
	//Found file: TOWN.OBT È la popolazione del livello. Definisce le coordinate X, Y, Z, il Pitch/Yaw/Roll e la classe di tutto ciò che è dinamico: il punto di spawn del giocatore (PLAYER), i nemici, le
	//Found file: TOWN.INF linguaggio di scripting proprietario
	//Found file: TOWN.ITM Definisce proprietà aggiuntive e comportamenti specifici per gli oggetti interattivi presenti nell'OBT
	//Found file: TOWN.MSC Contiene le direttive per la colonna sonora interattiva (i file audio IMUSE)
	return nil
}

// GetPayload retrieves the payload data associated with the given name from the container.
// Returns the data as a byte slice or an error if the name is not found or the read operation fails.
func (al *ArchiveLab) GetPayload(name string) ([]byte, error) {
	gob, ok := al.container[al.cleanName(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	return gob.Read()
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
