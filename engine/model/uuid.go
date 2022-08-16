package model

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync/atomic"
)

var _generator *Generator

func init() {
	_generator = MustNewGenerator()
}

type Generator struct {
	seed    [24]byte
	counter uint64
}

func NewGenerator() (*Generator, error) {
	var g Generator
	_, err := rand.Read(g.seed[:])
	if err != nil {
		return nil, errors.New("cannot generate random seed: " + err.Error())
	}
	g.counter = binary.LittleEndian.Uint64(g.seed[:8])
	return &g, nil
}

func MustNewGenerator() *Generator {
	g, err := NewGenerator()
	if err != nil {
		panic(err)
	}
	return g
}

func (g *Generator) Next() [24]byte {
	x := atomic.AddUint64(&g.counter, 1)
	uuid := g.seed
	binary.LittleEndian.PutUint64(uuid[:8], x)
	return uuid
}

func (g *Generator) Hex128() string {
	return Hex128(g.Next())
}

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

func isValidHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

func NextUUId() string {
	id := _generator.Next()
	return hex.EncodeToString(id[:])
}
