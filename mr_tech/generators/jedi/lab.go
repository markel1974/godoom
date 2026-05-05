package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// LabFile represents a file in a LAB archive, providing access to its data using an offset and size.
type LabFile struct {
	file   io.ReaderAt
	offset int
	size   int
}

// NewLabFile creates and returns a pointer to a LabFile initialized with the provided file, offset, and size.
func NewLabFile(file io.ReaderAt, offset int, size int) *LabFile {
	return &LabFile{
		file:   file,
		offset: offset,
		size:   size,
	}
}

// LabArchive represents a container for working with LAB file archives, supporting operations like parsing and file retrieval.
type LabArchive struct {
}

// NewLabArchive creates and returns a new instance of LabArchive.
func NewLabArchive() *LabArchive {
	return &LabArchive{}
}

// Parse reads and processes a LAB archive file from the specified path, populating the archive's internal data structure.
func (pk *LabArchive) Parse(path string) error {
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
	defer file.Close()

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

	container := make(map[string]*LabFile)

	for _, e := range entries {
		start := e.NameOffset
		end := start
		for end < uint32(len(stringTable)) && stringTable[end] != 0 {
			end++
		}
		fileName := string(stringTable[start:end])
		container[fileName] = NewLabFile(file, int(e.DataOffset), int(e.DataSize))
	}

	fmt.Println("CONTAINER-", container)

	return nil
}
