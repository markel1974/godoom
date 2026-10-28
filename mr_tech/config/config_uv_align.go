package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// UVAlign defines UV alignment and transformation parameters for sector floors, ceilings, and walls.
type UVAlign struct {
	FixUV       bool        `json:"fixUV"`
	AlignWall   bool        `json:"alignWall"`
	Start       geometry.XY `json:"start"`
	End         geometry.XY `json:"end"`
	SwapXY      bool        `json:"swapXY"`
	FlipX       bool        `json:"flipX"`
	FlipY       bool        `json:"flipY"`
	Panning     geometry.XY `json:"panning"`
	Length      float64     `json:"length,omitempty"`
	AlignBottom bool        `json:"alignBottom,omitempty"`
	ScaleY      float64     `json:"scaleY,omitempty"`
}
