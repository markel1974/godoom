package config

import (
	"github.com/markel1974/godoom/mr_tech/utils"
)

type MaterialKind int

const (
	MaterialKindNone MaterialKind = iota
	MaterialKindSolid
	MaterialKindSky
	MaterialKindLiquidImpenetrable
	MaterialKindLiquid
	MaterialKindLatest
)

const (
	BlendModeOpaque = iota
	BlendModeAdditive
	BlendModeLiquid
	BlendModeLatest
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
	Name      string       `json:"name"`
	Shader    string       `json:"shader"`
	Textures  []string     `json:"textures"`
	Kind      MaterialKind `json:"kind"`
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
	// Multi-layer / Terrain Blend (WorldVertexTransition)
	DiffuseMap2 string `json:"diffuseMap2"`
	NormalMap2  string `json:"normalMap2"`
	// Detail Texture Properties
	DetailScale  float32 `json:"detailScale"`
	DetailFactor float32 `json:"detailFactor"`
	// Environment / Cubemap Reflections & Tint
	EnvMap     string    `json:"envMap"`
	EnvMapMask string    `json:"envMapMask"`
	EnvMapTint []float32 `json:"envMapTint"`
	ColorTint  []float32 `json:"colorTint"`
	// UV Transformations & Animations
	ScaleU  float32 `json:"scaleU"`
	ScaleV  float32 `json:"scaleV"`
	ScrollU float32 `json:"scrollU"`
	ScrollV float32 `json:"scrollV"`
	Rotate  float32 `json:"rotate"`
	// UV Clamping Modes
	ClampU bool `json:"clampU"`
	ClampV bool `json:"clampV"`
	// Pipeline Flags
	VertexLitGeneric bool `json:"vertexLitGeneric"`
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
	IsBacksided    bool            `json:"isBacksided"`
	SpecularBoost  float64         `json:"specularBoost"`
	Shininess      float64         `json:"shininess"`
	IsSolid        bool            `json:"isSolid"`
	Liquid         *Liquid         `json:"liquid"`
}

// NewConfigMaterial creates and initializes a new Material instance with the provided animation and kind values.
func NewConfigMaterial(name string, textures []string, kind MaterialKind) *Material {
	m := &Material{
		Name:       name,
		Id:         utils.NextUUId(),
		Textures:   textures,
		DepthWrite: true, // Default to true for opaque materials
		CullMode:   CullFront,
		BlendMode:  BlendModeOpaque,
		IsSolid:    true,
		Liquid:     nil,
	}
	m.SetKind(kind)
	m.SpecularBoost = 0.1
	m.Shininess = 40.0
	m.DetailScale = 1.0
	m.DetailFactor = 1.0
	m.ScaleU = 1.0
	m.ScaleV = 1.0
	return m
}

// SetKind sets the material's kind and updates related properties like blend mode, depth write, solidity, and liquid type.
func (m *Material) SetKind(kind MaterialKind) {
	m.Kind = kind
	if m.Kind == MaterialKindLiquid || m.Kind == MaterialKindLiquidImpenetrable {
		m.BlendMode = BlendModeLiquid
		m.DepthWrite = false // Liquids non scrivono nel depth buffer opaco
		m.Liquid = NewLiquid(LiquidTypeWater)
		m.IsSolid = false
		if kind == MaterialKindLiquidImpenetrable {
			m.IsSolid = true
		}
	}
}

// HashKey generates a unique key for the Material instance based on its Id field.
func (m *Material) HashKey() string {
	return m.Id
}
