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
	BlendModeOpaque = iota
	BlendModeAdditive
	BlendModeAlpha
)

const (
	MaterialKindNone MaterialKind = iota
	MaterialKindLoop
	MaterialKindSky
)

// Material represents animation properties including a sequence of frames, the type of animation, and the shader.
const (
	CullFront = iota
	CullNone
	CullBack
)

// Material represents animation properties including a sequence of frames, the type of animation, and the shader.
type Material struct {
	Id        string       `json:"id"`
	BlendMode int          `json:"blendMode"`
	Shader    string       `json:"shader"`
	Frames    []string     `json:"frames"`
	Kind      MaterialKind `json:"kind"`
	ScaleW    float64      `json:"scaleW"`
	ScaleH    float64      `json:"scaleH"`
	U         float64      `json:"u"`
	V         float64      `json:"v"`

	// Universal Modern Rendering Properties
	CullMode   int     `json:"cullMode"`
	DepthWrite bool    `json:"depthWrite"`
	AlphaTest  float32 `json:"alphaTest"`

	// Texture Slots
	DiffuseMap  string `json:"diffuseMap"`
	NormalMap   string `json:"normalMap"`
	SpecularMap string `json:"specularMap"`
	EmissionMap string `json:"emissionMap"`
	DetailMap   string `json:"detailMap"`
	BlendMap    string `json:"blendMap"`

	// UV Animations
	ScrollU float32 `json:"scrollU"`
	ScrollV float32 `json:"scrollV"`
	Rotate  float32 `json:"rotate"`
}

// NewConfigMaterial creates and initializes a new Material instance with the provided animation and kind values.
func NewConfigMaterial(frames []string, kind MaterialKind, scaleW, scaleH, u, v float64) *Material {
	return &Material{
		Id:         utils.NextUUId(),
		Frames:     frames,
		Kind:       kind,
		ScaleW:     scaleW,
		ScaleH:     scaleH,
		U:          u,
		V:          v,
		DepthWrite: true, // Default to true for opaque materials
		CullMode:   CullFront,
	}
}

// HashKey computes a unique, deterministic hash string for a Material instance based on its properties and frames.
func (m *Material) HashKey() string {
	h := sha256.New()
	h.Write([]byte(m.Shader))
	h.Write([]byte{0})
	h.Write([]byte{byte(m.BlendMode)})
	h.Write([]byte{byte(m.CullMode)})
	if m.DepthWrite {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	h.Write([]byte(m.DiffuseMap))
	h.Write([]byte(m.NormalMap))
	h.Write([]byte(m.SpecularMap))
	h.Write([]byte(m.EmissionMap))
	h.Write([]byte(m.DetailMap))
	h.Write([]byte(m.BlendMap))
	h.Write([]byte{0})
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
