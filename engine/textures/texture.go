package textures

// TextureSize defines the size of the texture and must be a power of two.
// TextureBegin represents the starting index of the texture.
// TextureEnd represents the last index of the texture, calculated as TextureSize - 1.
const (
// TextureSize must be a power of two
// textureWidth  = 1024
// textureHeight = 1024
// TextureBegin = 0
// TextureEnd   = TextureSize - 1
)

// Texture represents a 2D grid of color data organized as a square with dimensions TextureSize x TextureSize.
type Texture struct {
	w    int
	h    int
	data [][]int
}

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

// Get retrieves the color value at the specified coordinates (x, y) in the texture, using wrap-around behavior.
func (t *Texture) Get(x int, y int) int {
	//TextureSize (1024) is a power of 2, we can use bitwise operator
	return t.data[x&t.w][y&t.h]
}

// Set updates the color of a pixel at the specified x and y coordinates in the texture.
func (t *Texture) Set(x int, y int, color int) {
	t.data[x&t.w][y&t.h] = color
}

func (t *Texture) BeginX() int {
	return 0
}

func (t *Texture) BeginY() int {
	return 0
}

func (t *Texture) Size() (int, int) {
	return t.w, t.h
}
