package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const ExtLevelLFD = ".LFD"

// LabFile represents a section of a file with a specified offset and size for reading.
type LabFile struct {
	file   io.ReaderAt
	offset int
	size   int
}

// NewLabFile creates a new LabFile instance with the given file, offset, and size parameters.
func NewLabFile(file io.ReaderAt, offset int, size int) *LabFile {
	return &LabFile{
		file:   file,
		offset: offset,
		size:   size,
	}
}

// Read reads data from the LabFile starting at the specified offset and returns the data or an error if the operation fails.
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

// ArchiveLab represents a handler for managing LAB archive files, allowing parsing, retrieving, and cleanup operations.
type ArchiveLab struct {
	container map[string]*LabFile
	files     []*os.File
	levels    []string
}

// NewArchiveLab initializes and returns a new instance of ArchiveLab with an empty container.
func NewArchiveLab() *ArchiveLab {
	return &ArchiveLab{
		container: make(map[string]*LabFile),
	}
}

// Parse reads a LAB archive from the specified file path, extracting its metadata and initializing the file container.
func (pk *ArchiveLab) Parse(dirPath string) error {
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
		if err := pk.add(entryPath); err != nil {
			return err
		}
	}
	return nil
}

func (pk *ArchiveLab) add(path string) error {
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
	pk.files = append(pk.files, file)
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
		cleanName := pk.cleanName(fileName)
		pk.container[cleanName] = NewLabFile(file, int(e.DataOffset), int(e.DataSize))

		fmt.Printf("Found file: %s\n", cleanName)
		if pos := strings.Index(cleanName, ExtLevelLFD); pos > 0 {
			pk.levels = append(pk.levels, cleanName[:pos])
		}
	}
	return nil
}

// GetPayload retrieves the data associated with the specified name from the ArchiveLab. It returns an error if not found.
func (pk *ArchiveLab) GetPayload(name string) ([]byte, error) {
	gob, ok := pk.container[pk.cleanName(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	return gob.Read()
}

// Close releases all open file handles and clears the file list in the ArchiveLab instance.
func (pk *ArchiveLab) Close() error {
	for _, f := range pk.files {
		if f != nil {
			f.Close()
		}
	}
	pk.files = nil
	return nil
}

// cleanName standardizes a file name by removing leading/trailing spaces and converting it to uppercase.
func (pk *ArchiveLab) cleanName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

/*
*.LVT e *.LVB (Level Text / Level Binary): Qui c'è la topologia pura. Vertici, settori, adjoins (portali), e texture dei muri. La versione T è puro ASCII, la B è lo stesso dato compilato per caricamenti rapidi.

*.OBT e *.OBB (Object Text / Object Binary): Le entità. I player spawn, i nemici, i barili, i pickup.

*.INF (Information/Scripting): Il codice logico del livello. Qui dentro ci sono le macchine a stati per porte, ascensori, chiavi e trigger. Il Jedi Engine usa un linguaggio di scripting rudimentale basato su seq (sequenze).

*.ITM (Items): Definizioni specifiche per il piazzamento e il comportamento degli oggetti nel livello.
 */
