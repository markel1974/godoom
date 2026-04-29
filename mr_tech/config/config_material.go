package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

type MaterialKind int

const (
	MaterialKindNone MaterialKind = iota
	MaterialKindLoop
	MaterialKindSky
)

// Material represents animation properties including a sequence of frames and the type of animation.
type Material struct {
	Id     string       `json:"id"`
	Frames []string     `json:"frames"`
	Kind   MaterialKind `json:"kind"`
	ScaleW float64      `json:"scaleW"`
	ScaleH float64      `json:"scaleH"`
	U      float64      `json:"u"`
	V      float64      `json:"v"`
}

// NewConfigMaterial creates and initializes a new Material instance with the provided animation and kind values.
func NewConfigMaterial(frames []string, kind MaterialKind, scaleW, scaleH, u, v float64) *Material {
	return &Material{
		Id:     utils.NextUUId(),
		Frames: frames,
		Kind:   kind,
		ScaleW: scaleW,
		ScaleH: scaleH,
		U:      u,
		V:      v,
	}
}

// HashKey computes a unique, deterministic hash string for a Material instance based on its properties and frames.
func (m *Material) HashKey() string {
	h := sha256.New()
	for _, f := range m.Frames {
		h.Write([]byte(f))
		h.Write([]byte{0})
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(m.ScaleW))
	h.Write(buf[:])
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(m.ScaleH))
	h.Write(buf[:])
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(m.U))
	h.Write(buf[:])
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(m.V))
	h.Write(buf[:])
	binary.LittleEndian.PutUint32(buf[:4], uint32(m.Kind))
	h.Write(buf[:4])
	var digest [32]byte
	h.Sum(digest[:0])
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
