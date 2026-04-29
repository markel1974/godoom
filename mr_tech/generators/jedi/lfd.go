package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// LfdEntry represents a single entry in an LFD file, containing metadata and access information for a resource.
type LfdEntry struct {
	Type   string // Identificatore a 4 byte (es. "BM  ", "VOC ", "ANIM")
	Name   string // Nome risorsa fino a 8 byte
	Size   uint32 // Dimensione del payload estratto
	Offset int64  // Offset assoluto del puntatore fisico al payload
}

// LFD represents a container for managing and accessing resources in an LFD file format.
type LFD struct {
	file    *os.File
	entries map[string]LfdEntry
}

// NewLFD creates and returns a new instance of the LFD struct with initialized fields.
func NewLFD() *LFD {
	return &LFD{}
}

// Parse reads the LFD file given by filename, validates its structure, and populates the entries map with resource metadata.
func (l *LFD) Parse(filename string) error {
	f, fErr := os.Open(filename)
	if fErr != nil {
		return fErr
	}
	l.file = f
	l.entries = make(map[string]LfdEntry)

	// 1. Validazione del chunk iniziale RMAP
	var rMapType [4]byte
	var rMapName [8]byte
	var rMapSize uint32

	if err := binary.Read(f, binary.LittleEndian, &rMapType); err != nil {
		return err
	}
	if string(rMapType[:]) != "RMAP" {
		return fmt.Errorf("firma LFD non valida, atteso RMAP, trovato: %s", string(rMapType[:]))
	}
	_ = binary.Read(f, binary.LittleEndian, &rMapName)
	_ = binary.Read(f, binary.LittleEndian, &rMapSize)

	// L'RMAP contiene tuple da 16 byte (Type:4 + Name:8 + Size:4)
	numEntries := rMapSize / 16

	// Il blocco dati successivo inizia immediatamente dopo i 16 byte di header del RMAP + la sua dimensione
	currentOffset := int64(16) + int64(rMapSize)

	// 2. Lettura del Table of Contents
	for i := uint32(0); i < numEntries; i++ {
		var eType [4]byte
		var eName [8]byte
		var eSize uint32

		_ = binary.Read(f, binary.LittleEndian, &eType)
		_ = binary.Read(f, binary.LittleEndian, &eName)
		_ = binary.Read(f, binary.LittleEndian, &eSize)

		cleanType := strings.TrimSpace(string(eType[:]))

		nameLen := bytes.IndexByte(eName[:], 0)
		if nameLen == -1 {
			nameLen = 8
		}
		cleanName := strings.TrimSpace(string(eName[:nameLen]))

		entry := LfdEntry{
			Type: cleanType,
			Name: cleanName,
			Size: eSize,
			// Ogni chunk fisico nel file ripete il proprio sub-header da 16 byte.
			// Puntiamo l'offset direttamente ai dati utili saltandolo.
			Offset: currentOffset + 16,
		}
		// Chiave composita TYPE_NAME (es: "BM_FONT" o "VOC_GUNFIRE")
		key := fmt.Sprintf("%s_%s", cleanType, cleanName)
		l.entries[key] = entry
		// Avanza l'offset al chunk successivo
		currentOffset += 16 + int64(eSize)
	}

	return nil
}

// GetPayload retrieves the payload data for a resource identified by resType and name.
// Returns an error if the resource is not found or an issue occurs with file operations.
func (l *LFD) GetPayload(resType, name string) ([]byte, error) {
	key := fmt.Sprintf("%s_%s", resType, name)
	entry, ok := l.entries[key]
	if !ok {
		return nil, fmt.Errorf("risorsa %s di tipo %s non trovata", name, resType)
	}

	data := make([]byte, entry.Size)
	if _, err := l.file.Seek(entry.Offset, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(l.file, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Close releases the underlying file resource if it is open and returns any error encountered during the close operation.
func (l *LFD) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
