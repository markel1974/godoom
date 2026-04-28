package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// LfdEntry descrive i metadati di un blocco dati all'interno del container LFD.
type LfdEntry struct {
	Type   string // Identificatore a 4 byte (es. "BM  ", "VOC ", "ANIM")
	Name   string // Nome risorsa fino a 8 byte
	Size   uint32 // Dimensione del payload estratto
	Offset int64  // Offset assoluto del puntatore fisico al payload
}

// LFD espone l'accesso random-access e O(1) agli asset multimediali.
type LFD struct {
	file    *os.File
	entries map[string]LfdEntry
}

// NewLFD apre l'archivio, scansiona il chunk RMAP e mappa le posizioni fisiche di tutti i payload.
func NewLFD(filename string) (*LFD, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	lfd := &LFD{
		file:    f,
		entries: make(map[string]LfdEntry),
	}

	// 1. Validazione del chunk iniziale RMAP
	var rmapType [4]byte
	var rmapName [8]byte
	var rmapSize uint32

	if err := binary.Read(f, binary.LittleEndian, &rmapType); err != nil {
		return nil, err
	}
	if string(rmapType[:]) != "RMAP" {
		return nil, fmt.Errorf("firma LFD non valida, atteso RMAP, trovato: %s", string(rmapType[:]))
	}
	binary.Read(f, binary.LittleEndian, &rmapName)
	binary.Read(f, binary.LittleEndian, &rmapSize)

	// L'RMAP contiene tuple da 16 byte (Type:4 + Name:8 + Size:4)
	numEntries := rmapSize / 16

	// Il blocco dati successivo inizia immediatamente dopo i 16 byte di header del RMAP + la sua dimensione
	currentOffset := int64(16) + int64(rmapSize)

	// 2. Lettura del Table of Contents
	for i := uint32(0); i < numEntries; i++ {
		var eType [4]byte
		var eName [8]byte
		var eSize uint32

		binary.Read(f, binary.LittleEndian, &eType)
		binary.Read(f, binary.LittleEndian, &eName)
		binary.Read(f, binary.LittleEndian, &eSize)

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
		lfd.entries[key] = entry

		// Avanza l'offset al chunk successivo
		currentOffset += 16 + int64(eSize)
	}

	return lfd, nil
}

// GetPayload esegue l'estrazione diretta del blocco decodificato.
// resType è la stringa a 4 caratteri (es. "BM", "ANIM", "VOC"), name è l'identificativo.
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

// Close rilascia il descriptor in OS.
func (l *LFD) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
