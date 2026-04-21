package textures

// Texture represents a 2D texture with metadata and pixel data.
// name is the identifier name of the texture.
// id is the unique identifier for the texture.
// w and h define the width and height of the texture in pixels.
// data holds the actual pixel information as a 2D array of integers.
type Texture struct {
	name    string
	id      uint32
	w       int
	h       int
	data    [][]int
	gScale  float64
	scaleW  float64
	scaleH  float64
	wScaled float64
	hScaled float64
}

// NewTexture creates a new Texture instance with the given name, ID, width, and height.
func NewTexture(name string, id uint32, w int, h int) *Texture {
	z := &Texture{
		name: name, id: id, w: w, h: h,
		data: make([][]int, w),
	}
	for i := range z.data {
		z.data[i] = make([]int, h)
	}
	return z
}

// SetScaleFactor adjusts the global, width, and height scaling factors, recalculating the scaled dimensions of the texture.
func (t *Texture) SetScaleFactor(gScale, scaleW, scaleH float64) {
	t.gScale = gScale
	t.scaleW = scaleW
	t.scaleH = scaleH
	t.wScaled, t.hScaled = float64(t.w), float64(t.h)
	if t.gScale != 1.0 || t.scaleW != 1.0 || t.scaleH != 1.0 {
		//TODO CAPIRE PERCHE BISOGNA MOLTIPLICARE PER 5 in DOOM!!!!!
		t.gScale *= 5
		t.wScaled *= t.scaleW / t.gScale
		t.hScaled *= t.scaleH / t.gScale
	}
}

// GetSizeScaled returns the scaled width and height of the texture as two float64 values.
func (t *Texture) GetSizeScaled() (float64, float64) {
	return t.wScaled, t.hScaled
}

// GetScaleFactor returns the global scale factor and the width and height scale values of the texture as floats.
func (t *Texture) GetScaleFactor() (float64, float64, float64) {
	return t.gScale, t.scaleW, t.scaleH
}

// GetScaleFactorH retrieves the height scaling factor of the Texture as a float64.
func (t *Texture) GetScaleFactorH() float64 {
	return t.scaleH
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

// RGBA returns the width, height, and a pixel array representing the texture in RGBA format.
func (t *Texture) RGBA() (int, int, []uint8) {
	width, height := t.Size()
	width = width + 1
	height = height + 1
	pixels := make([]uint8, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := t.Get(x, y)
			idx := (y*width + x) * 4
			pixels[idx] = uint8(c >> 24)
			pixels[idx+1] = uint8(c >> 16)
			pixels[idx+2] = uint8(c >> 8)
			pixels[idx+3] = uint8(c)
		}
	}
	return width, height, pixels
}
