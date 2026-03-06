package textures

// Texture represents a 2D image resource with a unique ID and dimensions (width and height).
type Texture struct {
	name string
	id   uint32
	w    int
	h    int
	data [][]int
}

// NewTexture creates and initializes a new Texture with the specified name, ID, width, and height.
func NewTexture(name string, id uint32, w int, h int) *Texture {
	texWidth := w - 1
	texHeight := h - 1
	//if texWidth <= 0 {
	//	texWidth = 64
	//}
	//if texHeight <= 0 {
	//	texHeight = 64
	//}
	z := &Texture{
		name: name,
		id:   id,
		w:    texWidth,
		h:    texHeight,
		data: make([][]int, w),
	}
	for i := range z.data {
		z.data[i] = make([]int, h)
	}
	return z
}

// Get retrieves the color value at the specified (x, y) coordinates, applying bitwise wrapping based on texture size.
func (t *Texture) Get(x int, y int) int {
	//TextureSize (1024) is a power of 2, we can use bitwise operator
	return t.data[x&t.w][y&t.h]
}

// Set updates the color value of the pixel at the specified x and y coordinates in the texture data.
func (t *Texture) Set(x int, y int, color int) {
	t.data[x&t.w][y&t.h] = color
}

// BeginX returns the starting X-coordinate offset for the texture. Used in texture rendering operations.
func (t *Texture) BeginX() int {
	return 0
}

// BeginY returns the starting Y-coordinate offset for texture mapping.
func (t *Texture) BeginY() int {
	return 0
}

// Size returns the width and height of the texture.
func (t *Texture) Size() (int, int) {
	return t.w, t.h
}

// GetId returns the unique identifier of the texture as a uint32.
func (t *Texture) GetId() uint32 {
	return t.id
}

// GetName returns the name of the texture as a string.
func (t *Texture) GetName() string {
	return t.name
}
