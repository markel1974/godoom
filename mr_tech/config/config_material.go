package config

import (
	"github.com/markel1974/godoom/mr_tech/utils"
)

type MaterialKind int

const (
	BlendModeOpaque = iota
	BlendModeAdditive
	BlendModeLiquid
	BlendModeLates
)

const (
	MaterialKindNone MaterialKind = iota
	MaterialKindLoop
	MaterialKindSky
	MaterialKindLiquid
)

// Material represents animation properties including a sequence of frames, the type of animation, and the shader.
const (
	CullFront = iota
	CullNone
	CullBack
)

// Material represents animation properties including a sequence of frames, the type of animation, and the shader.
type Material struct {
	Id        string `json:"id"`
	BlendMode int    `json:"blendMode"`
	Name      string `json:"name"`
	//Shader    string       `json:"shader"`
	Frames []string     `json:"frames"`
	Kind   MaterialKind `json:"kind"`
	U      float64      `json:"u"`
	V      float64      `json:"v"`

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

	// Legacy Engine Properties
	IsFog           bool      `json:"isFog"`
	FogColor        []float32 `json:"fogColor"`
	PolygonOffset   bool      `json:"polygonOffset"`
	Sort            int       `json:"sort"`
	NoMipmaps       bool      `json:"noMipmaps"`
	NoPicMip        bool      `json:"noPicMip"`
	EntityMergeable bool      `json:"entityMergeable"`

	// Complex/Additional
	DeformVertexes [][]string      `json:"deformVertexes"`
	SurfaceParms   map[string]bool `json:"surfaceParms"`
	SkyParms       []string        `json:"skyParms"`
	CloudParms     []string        `json:"cloudParms"`
	FogGen         []string        `json:"fogGen"`
	IsLightning    bool            `json:"isLightning"`
	IsSky          bool            `json:"isSky"`
	IsBacksided    bool            `json:"isBacksided"`
	SpecularBoost  float64         `json:"specularBoost"`
	Shininess      float64         `json:"shininess"`
}

// NewConfigMaterial creates and initializes a new Material instance with the provided animation and kind values.
func NewConfigMaterial(name string, frames []string, kind MaterialKind) *Material {
	m := &Material{
		Name:       name,
		Id:         utils.NextUUId(),
		Frames:     frames,
		Kind:       kind,
		DepthWrite: true, // Default to true for opaque materials
		CullMode:   CullFront,
		BlendMode:  BlendModeOpaque,
	}
	if kind == MaterialKindLiquid {
		m.BlendMode = BlendModeLiquid
		m.DepthWrite = false // Liquids non scrivono nel depth buffer opaco!
	}
	m.SpecularBoost = 0.1
	m.Shininess = 40.0
	return m
}

// HashKey computes a unique, deterministic hash string for a Material instance based on its properties and frames.
func (m *Material) HashKey() string {
	return m.Id
}
