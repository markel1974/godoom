package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"unicode"
)

func CleanKey(in string) string {
	var out []rune
	for _, r := range in {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out = append(out, unicode.ToUpper(r))
		}
	}
	return string(out)
}

func GetTokenIntAt(tokens []string, index int) (int, error) {
	if index < 0 || index >= len(tokens) {
		return 0, fmt.Errorf("index out of range")
	}
	count, err := strconv.Atoi(tokens[index])
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTokenFloatAt(tokens []string, index int) (float64, error) {
	if index < 0 || index >= len(tokens) {
		return 0, fmt.Errorf("index out of range")
	}
	count, err := strconv.ParseFloat(tokens[index], 64)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTokenStringAt(tokens []string, index int) (string, error) {
	if index < 0 || index >= len(tokens) {
		return "", fmt.Errorf("index out of range")
	}
	return tokens[index], nil
}

// DecompressPayload controlla se i dati sono in chiaro o compressi.
// Nel Jedi Engine, se i primi byte non sono una firma ASCII nota ("WAX ", "FME ", "3DO "),
// i primi 4 byte rappresentano la dimensione decompressa e il resto è uno stream LZSS.
func DecompressPayload(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil
	}
	// 1. Controllo Magic Number binari (WAX Version)
	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic == 0x00010001 || magic == 0x00011000 {
		return data, nil
	}
	// 2. Controllo header ASCII (per 3DO, MSG, etc.)
	if bytes.HasPrefix(data, []byte("FME ")) ||
		bytes.HasPrefix(data, []byte("3DO ")) ||
		bytes.HasPrefix(data, []byte("MSG ")) ||
		bytes.HasPrefix(data, []byte("WAXF")) {
		return data, nil
	}
	// 3. Altrimenti, è uno stream LZSS (i primi 4 byte sono decompressedSize)
	return decodeJediCompression(data)
}

// decodeJediCompression decomprime il payload usando la variante Okumura LZSS del Jedi Engine.
func decodeJediCompression(inData []byte) ([]byte, error) {
	if len(inData) < 4 {
		return nil, fmt.Errorf("data too short")
	}

	decompressedSize := binary.LittleEndian.Uint32(inData[0:4])

	// 1. Sanity check (es. max 16MB per asset Jedi Engine)
	const maxDecompressedSize = 16 * 1024 * 1024
	if decompressedSize > maxDecompressedSize {
		return nil, fmt.Errorf("excessive decompressed size: %d", decompressedSize)
	}

	// 2. Allocazione statica
	outData := make([]byte, decompressedSize)
	outPos := uint32(0)

	window := make([]byte, 4096)
	windowPos := 0

	inPos := 4
	var flags uint8
	var flagBits int

	for outPos < decompressedSize {
		if flagBits == 0 {
			if inPos >= len(inData) {
				break
			}
			flags = inData[inPos]
			inPos++
			flagBits = 8
		}

		if (flags & 1) != 0 {
			// --- LITERAL ---
			if inPos >= len(inData) {
				break
			}
			b := inData[inPos]
			inPos++

			outData[outPos] = b
			outPos++
			window[windowPos] = b
			// 3. Bitwise AND al posto del modulo
			windowPos = (windowPos + 1) & 0xFFF
		} else {
			// --- MATCH ---
			if inPos+1 >= len(inData) {
				break
			}
			b1 := int(inData[inPos])
			b2 := int(inData[inPos+1])
			inPos += 2

			offset := b1 | ((b2 & 0x0F) << 8)
			length := ((b2 & 0xF0) >> 4) + 3

			for i := 0; i < length; i++ {
				if outPos >= decompressedSize {
					break
				}

				readPos := (offset + i) & 0xFFF
				b := window[readPos]

				outData[outPos] = b
				outPos++
				window[windowPos] = b
				windowPos = (windowPos + 1) & 0xFFF
			}
		}

		flags >>= 1
		flagBits--
	}

	if outPos != decompressedSize {
		return outData[:outPos], fmt.Errorf("truncated decompression: expected %d, got %d", decompressedSize, outPos)
	}

	return outData, nil
}
