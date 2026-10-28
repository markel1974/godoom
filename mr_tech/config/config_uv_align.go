package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// UVAlign represents UV alignment properties for texture mapping, including transformations, scaling, and wall alignment.
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

// NewUVAlignWall creates a new UVAlign instance configured for wall alignment with specified transformation options.
func NewUVAlignWall(alignWall bool, start geometry.XY, end geometry.XY, swapXY bool, flipX bool, flipY bool, panning geometry.XY) *UVAlign {
	return &UVAlign{
		FixUV:       true,
		AlignWall:   alignWall,
		AlignBottom: false,
		Start:       start,
		End:         end,
		SwapXY:      swapXY,
		FlipX:       flipX,
		FlipY:       flipY,
		Panning:     panning,
		Length:      0,
		ScaleY:      0,
	}
}

// NewUVAlignBottom creates and returns a new UVAlign instance with specified alignment, flipping, panning, length, and scaling.
// alignBottom determines whether alignment is anchored at the bottom (true) or not.
// flipX specifies whether the UV mapping is flipped horizontally.
// flipY specifies whether the UV mapping is flipped vertically.
// panning defines the XY offset to adjust the UV mapping position.
// length sets the length for the UV mapping, with 0 implying no defined length.
// scaleY specifies the vertical scaling factor applied to the UV mapping.
func NewUVAlignBottom(alignBottom bool, flipX bool, flipY bool, panning geometry.XY, length float64, scaleY float64) *UVAlign {
	return &UVAlign{
		FixUV:       true,
		AlignWall:   false,
		AlignBottom: alignBottom,
		Start:       geometry.XY{},
		End:         geometry.XY{},
		SwapXY:      false,
		FlipX:       flipX,
		FlipY:       flipY,
		Panning:     panning,
		Length:      length,
		ScaleY:      scaleY,
	}
}
