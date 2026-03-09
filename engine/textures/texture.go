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
	z := &Texture{
		name: name, id: id, w: w, h: h, // Nessun w-1 o h-1!
		data: make([][]int, w),
	}
	for i := range z.data {
		z.data[i] = make([]int, h)
	}
	return z
}

func (t *Texture) Get(x int, y int) int {
	wrapX := (x%t.w + t.w) % t.w
	wrapY := (y%t.h + t.h) % t.h
	return t.data[wrapX][wrapY]
}

func (t *Texture) Set(x int, y int, color int) {
	wrapX := (x%t.w + t.w) % t.w
	wrapY := (y%t.h + t.h) % t.h
	t.data[wrapX][wrapY] = color
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

func (t *Texture) RGBA() (int, int, []uint8) {
	width, height := t.Size()
	width = width + 1
	height = height + 1
	pixels := make([]uint8, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := t.Get(x, y)
			idx := (y*width + x) * 4

			if c == -1 {
				pixels[idx] = 0
				pixels[idx+1] = 0
				pixels[idx+2] = 0
				pixels[idx+3] = 0
			} else {
				pixels[idx] = uint8(c >> 16)
				pixels[idx+1] = uint8(c >> 8)
				pixels[idx+2] = uint8(c)
				pixels[idx+3] = 255
			}
		}
	}
	return width, height, pixels
}
