package textures

// Texture represents a 2D texture with a specified width, height, and pixel data stored as a 2D integer array.
type Texture struct {
	w    int
	h    int
	data [][]int
}

// NewTexture creates and initializes a new Texture with the specified width and height. Returns a pointer to the Texture.
func NewTexture(w int, h int) *Texture {
	texWidth := w - 1
	texHeight := h - 1
	//if texWidth <= 0 {
	//	texWidth = 64
	//}
	//if texHeight <= 0 {
	//	texHeight = 64
	//}

	z := &Texture{
		w:    texWidth,
		h:    texHeight,
		data: make([][]int, w),
	}
	for i := range z.data {
		z.data[i] = make([]int, h)
	}
	return z
}

// Get retrieves the color value at the given (x, y) coordinates in the texture, applying bitwise masking for wrapping.
func (t *Texture) Get(x int, y int) int {
	//TextureSize (1024) is a power of 2, we can use bitwise operator
	return t.data[x&t.w][y&t.h]
}

// Set updates the color value at the specified (x, y) coordinates in the texture data. Coordinates are wrapped by texture dimensions.
func (t *Texture) Set(x int, y int, color int) {
	t.data[x&t.w][y&t.h] = color
}

// BeginX returns the X-coordinate offset for the texture. It is typically used for texture alignment and mapping.
func (t *Texture) BeginX() int {
	return 0
}

// BeginY returns the starting Y-coordinate for accessing texture data, typically used for offset-based texture mapping.
func (t *Texture) BeginY() int {
	return 0
}

// Size returns the width and height of the texture as a pair of integers.
func (t *Texture) Size() (int, int) {
	return t.w, t.h
}
