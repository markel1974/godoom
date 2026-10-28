package textures

// Texture represents a 2D texture with metadata and pixel data.
// name is the identifier name of the texture.
// id is the unique identifier for the texture.
// w and h define the width and height of the texture in pixels.
// data holds the actual pixel information as a 2D array of integers.
type Texture struct {
	name       string
	id         uint32
	w          int
	h          int
	data       [][]int
	emissive   bool
	identifier float32
}

// NewTexture creates a new Texture instance with the given name, ID, width, and height.
func NewTexture(name string, id uint32, w int, h int, emissive bool) *Texture {
	z := &Texture{
		name:       name,
		id:         id,
		w:          w,
		h:          h,
		data:       make([][]int, w),
		emissive:   emissive,
		identifier: -1,
	}
	for i := range z.data {
		z.data[i] = make([]int, h)
	}
	//z.SetScaleFactor(1.0, 1.0)
	return z
}

// GetIdentifier retrieves the unique floating-point identifier associated with the Texture instance.
func (t *Texture) GetIdentifier() float32 {
	return t.identifier
}

// SetIdentifier assigns a new float32 value to the identifier property of the Texture instance.
func (t *Texture) SetIdentifier(identifier float32) {
	t.identifier = identifier
}

// GetSize returns the width and height of the texture as two float64 values.
func (t *Texture) GetSize() (float64, float64) {
	return float64(t.w), float64(t.h)
}

// Get retrieves the color value at the specified wrapped texture coordinates (x, y).
func (t *Texture) Get(x int, y int) int {
	wrapX := (x%t.w + t.w) % t.w
	wrapY := (y%t.h + t.h) % t.h
	return t.data[wrapX][wrapY]
}

// Set updates the color of the pixel at the specified coordinates (x, y) after wrapping them within texture bounds.
func (t *Texture) Set(x int, y int, color int) {
	wrapX := (x%t.w + t.w) % t.w
	wrapY := (y%t.h + t.h) % t.h
	t.data[wrapX][wrapY] = color
}

// BeginX returns the starting X coordinate for the texture, typically used as an offset in texture mapping.
func (t *Texture) BeginX() int {
	return 0
}

// BeginY returns the starting Y coordinate for the texture, typically used for texture mapping or rendering operations.
func (t *Texture) BeginY() int {
	return 0
}

// Size returns the width and height of the texture as two integer values.
func (t *Texture) Size() (int, int) {
	return t.w, t.h
}

// GetId returns the unique identifier (id) of the Texture instance as a uint32.
func (t *Texture) GetId() uint32 {
	return t.id
}

// GetName returns the name of the texture as a string.
func (t *Texture) GetName() string {
	return t.name
}

// IsEmissive determines if the texture is marked as emissive and returns true if it is, otherwise false.
func (t *Texture) IsEmissive() bool {
	return t.emissive
}

// RGBA returns the width, height, and a pixel array representing the texture in RGBA format.
func (t *Texture) RGBA() (int, int, []uint8) {
	const stride = 4
	width, height := t.Size()
	pixels := make([]uint8, width*height*stride)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := t.Get(x, y)
			idx := (y*width + x) * stride
			pixels[idx] = uint8(c >> 24)   // R
			pixels[idx+1] = uint8(c >> 16) // G
			pixels[idx+2] = uint8(c >> 8)  // B
			pixels[idx+3] = uint8(c)       // A
		}
	}
	return width, height, pixels
}
