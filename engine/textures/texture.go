package textures

const (
	//TextureSize must be a power of two
	TextureSize  = 1024
	TextureBegin = 0
	TextureEnd   = TextureSize - 1
)

type Texture struct {
	data [TextureSize][TextureSize]int
}

func (t *Texture) Get(x uint, y uint) int {
	//return t.data[x % TextureSize][y % TextureSize]
	//TextureSize (1024) is a power of 2, we can use bitwise operator
	return t.data[x&TextureEnd][y&TextureEnd]
}
