package model

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync/atomic"
)

// _generator is a shared instance of Generator used to produce unique identifiers.
var _generator *Generator

// init initializes the package-level generator instance by creating a new generator or panicking in case of an error.
func init() {
	_generator = MustNewGenerator()
}

// Generator is a type that enables the generation of unique identifiers using a random seed and an incrementing counter.
type Generator struct {
	seed    [24]byte
	counter uint64
}

// NewGenerator initializes a new Generator instance with a random seed and counter, returning it or an error if unsuccessful.
func NewGenerator() (*Generator, error) {
	var g Generator
	_, err := rand.Read(g.seed[:])
	if err != nil {
		return nil, errors.New("cannot generate random seed: " + err.Error())
	}
	g.counter = binary.LittleEndian.Uint64(g.seed[:8])
	return &g, nil
}

// MustNewGenerator creates a new Generator instance and panics if an error occurs during initialization.
func MustNewGenerator() *Generator {
	g, err := NewGenerator()
	if err != nil {
		panic(err)
	}
	return g
}

// Next generates the next unique [24]byte identifier using the generator's seed and an atomic counter.
func (g *Generator) Next() [24]byte {
	x := atomic.AddUint64(&g.counter, 1)
	uuid := g.seed
	binary.LittleEndian.PutUint64(uuid[:8], x)
	return uuid
}

// Hex128 generates a 128-bit hexadecimal string representation of a UUID.
func (g *Generator) Hex128() string {
	return Hex128(g.Next())
}

// Hex128 generates a UUIDv4 string in hexadecimal representation from a 24-byte input array.
func Hex128(uuid [24]byte) string {
	uuid[6], uuid[9] = uuid[9], uuid[6]
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = uuid[8]&0x3f | 0x80
	b := make([]byte, 36)
	hex.Encode(b[0:8], uuid[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], uuid[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], uuid[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], uuid[8:10])
	b[23] = '-'
	hex.Encode(b[24:], uuid[10:16])
	return string(b)
}

// ValidHex128 verifies if the provided string is a valid 128-bit hexadecimal UUID in the format XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX.
func ValidHex128(id string) bool {
	if len(id) != 36 {
		return false
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	return isValidHex(id[0:8]) &&
		isValidHex(id[9:13]) &&
		isValidHex(id[14:18]) &&
		isValidHex(id[19:23]) &&
		isValidHex(id[24:])
}

// isValidHex returns true if the input string contains only valid hexadecimal characters (0-9, a-f).
func isValidHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// NextUUId generates and returns a new unique identifier as a hexadecimal string.
func NextUUId() string {
	id := _generator.Next()
	return hex.EncodeToString(id[:])
}
