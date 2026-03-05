package textures

// TextureSize defines the size of the texture and must be a power of two.
// TextureBegin represents the starting index of the texture.
// TextureEnd represents the last index of the texture, calculated as TextureSize - 1.
const (
	//TextureSize must be a power of two
	TextureSize  = 1024
	TextureBegin = 0
	TextureEnd   = TextureSize - 1
)

// Texture represents a 2D grid of color data organized as a square with dimensions TextureSize x TextureSize.
type Texture struct {
	data [TextureSize][TextureSize]int
}

func NewTexture() *Texture {
	return &Texture{}
}

// Get retrieves the color value at the specified coordinates (x, y) in the texture, using wrap-around behavior.
func (t *Texture) Get(x uint, y uint) int {
	//return t.data[x % TextureSize][y % TextureSize]
	//TextureSize (1024) is a power of 2, we can use bitwise operator
	return t.data[x&TextureEnd][y&TextureEnd]
}

// Set updates the color of a pixel at the specified x and y coordinates in the texture.
func (t *Texture) Set(x uint, y uint, color int) {
	t.data[x&TextureEnd][y&TextureEnd] = color
}
