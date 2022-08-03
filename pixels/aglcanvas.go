package pixels

import (
	"errors"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/markel1974/godoom/pixels/executor"
	"image/color"
)

// GLCanvas is an off-screen rectangular IBasicTarget and IPicture at the same time, that you can draw onto.
//
// It supports ITrianglesPosition, ITrianglesColor, ITrianglesPicture and IPictureColor.
type GLCanvas struct {
	gf     *GLFrame
	shader *GLShader
	cmp    ComposeMethod
	mat    mgl32.Mat3
	col    mgl32.Vec4
	smooth bool
	sprite *Sprite
}

var _ ComposeTarget = (*GLCanvas)(nil)

// NewGLCanvas creates a new empty, fully transparent GLCanvas with given bounds.
func NewGLCanvas(bounds Rect, smooth bool) *GLCanvas {
	c := &GLCanvas{
		gf:     NewGLFrame(bounds),
		mat:    mgl32.Ident3(),
		col:    mgl32.Vec4{1, 1, 1, 1},
		smooth: smooth,
	}

	c.shader = NewGLShader(baseCanvasFragmentShader)
	c.SetBounds(bounds)

	return c
}

// SetUniform will update the named uniform with the value of any supported underlying
// attribute variable. If the uniform already exists, including defaults, they will be reassigned
// to the new value. The value can be a pointer.
func (c *GLCanvas) SetUniform(name string, value interface{}) {
	c.shader.SetUniform(name, value)
}

// SetFragmentShader allows you to set a new fragment shader on the underlying
// framebuffer. Argument "src" is the GLSL source, not a filename.
func (c *GLCanvas) SetFragmentShader(src string) {
	c.shader.fs = src
	c.shader.Update()
}

// MakeTriangles creates a specialized copy of the supplied ITriangles that draws onto this GLCanvas.
//
// ITrianglesPosition, ITrianglesColor and ITrianglesPicture are supported.
func (c *GLCanvas) MakeTriangles(t ITriangles) ITargetTriangles {
	if gt, ok := t.(*GLTriangles); ok {
		return &canvasTriangles{
			GLTriangles: gt,
			dst:         c,
		}
	}
	return &canvasTriangles{
		GLTriangles: NewGLTriangles(c.shader, t),
		dst:         c,
	}
}

// MakePicture create a specialized copy of the supplied IPicture that draws onto this GLCanvas.
//
// IPictureColor is supported.
func (c *GLCanvas) MakePicture(p IPicture) ITargetPicture {
	if cp, ok := p.(*canvasPicture); ok {
		return &canvasPicture{
			GLPicture: cp.GLPicture,
			dst:       c,
		}
	}
	if gp, ok := p.(GLPicture); ok {
		return &canvasPicture{
			GLPicture: gp,
			dst:       c,
		}
	}
	return &canvasPicture{
		GLPicture: NewGLPicture(p),
		dst:       c,
	}
}

// SetMatrix sets a Matrix that every point will be projected by.
func (c *GLCanvas) SetMatrix(m Matrix) {
	// pixel.Matrix is 3x2 with an implicit 0, 0, 1 row after it. So
	// [0] [2] [4]    [0] [3] [6]
	// [1] [3] [5] => [1] [4] [7]
	//  0   0   1      0   0   1
	// since all matrix ops are affine, the last row never changes, and we don't need to copy it
	for i, j := range [...]int{0, 1, 3, 4, 6, 7} {
		c.mat[j] = float32(m[i])
	}
}

// SetColorMask sets a color that every color in triangles or a picture will be multiplied by.
func (c *GLCanvas) SetColorMask(col color.Color) {
	rgba := Alpha(1)
	if col != nil {
		rgba = ToRGBA(col)
	}
	c.col = mgl32.Vec4{
		float32(rgba.R),
		float32(rgba.G),
		float32(rgba.B),
		float32(rgba.A),
	}
}

// SetComposeMethod sets a Porter-Duff composition method to be used in the following draws onto
// this GLCanvas.
func (c *GLCanvas) SetComposeMethod(cmp ComposeMethod) {
	c.cmp = cmp
}

// SetBounds resizes the GLCanvas to the new bounds. Old content will be preserved.
func (c *GLCanvas) SetBounds(bounds Rect) {
	c.gf.SetBounds(bounds)
	if c.sprite == nil {
		c.sprite = NewSprite()
	}
	c.sprite.Set(c, c.Bounds())
	// c.sprite.SetMatrix(pixel.IM.Moved(c.Bounds().Center()))
}

// Bounds returns the rectangular bounds of the GLCanvas.
func (c *GLCanvas) Bounds() Rect {
	return c.gf.Bounds()
}

// SetSmooth sets whether stretched Pictures drawn onto this GLCanvas should be drawn smooth or
// pixely.
func (c *GLCanvas) SetSmooth(smooth bool) {
	c.smooth = smooth
}

// Smooth returns whether stretched Pictures drawn onto this GLCanvas are set to be drawn smooth or
// pixely.
func (c *GLCanvas) Smooth() bool {
	return c.smooth
}

// must be manually called inside mainthread
func (c *GLCanvas) setGlhfBounds() {
	_, _, bw, bh := intBounds(c.gf.Bounds())
	executor.Bounds(0, 0, bw, bh)
}

// must be manually called inside mainthread
func setBlendFunc(cmp ComposeMethod) {
	switch cmp {
	case ComposeOver:
		executor.BlendFunc(executor.One, executor.OneMinusSrcAlpha)
	case ComposeIn:
		executor.BlendFunc(executor.DstAlpha, executor.Zero)
	case ComposeOut:
		executor.BlendFunc(executor.OneMinusDstAlpha, executor.Zero)
	case ComposeAtop:
		executor.BlendFunc(executor.DstAlpha, executor.OneMinusSrcAlpha)
	case ComposeRover:
		executor.BlendFunc(executor.OneMinusDstAlpha, executor.One)
	case ComposeRin:
		executor.BlendFunc(executor.Zero, executor.SrcAlpha)
	case ComposeRout:
		executor.BlendFunc(executor.Zero, executor.OneMinusSrcAlpha)
	case ComposeRatop:
		executor.BlendFunc(executor.OneMinusDstAlpha, executor.SrcAlpha)
	case ComposeXor:
		executor.BlendFunc(executor.OneMinusDstAlpha, executor.OneMinusSrcAlpha)
	case ComposePlus:
		executor.BlendFunc(executor.One, executor.One)
	case ComposeCopy:
		executor.BlendFunc(executor.One, executor.Zero)
	default:
		panic(errors.New("GLCanvas: invalid compose method"))
	}
}

// Clear fills the whole GLCanvas with a single color.
func (c *GLCanvas) Clear(color color.Color) {
	c.gf.Dirty()

	rgba := ToRGBA(color)

	// color masking
	rgba = rgba.Mul(RGBA{
		R: float64(c.col[0]),
		G: float64(c.col[1]),
		B: float64(c.col[2]),
		A: float64(c.col[3]),
	})

	executor.Thread.Post(func() {
		c.setGlhfBounds()
		c.gf.Frame().Begin()
		executor.Clear(
			float32(rgba.R),
			float32(rgba.G),
			float32(rgba.B),
			float32(rgba.A),
		)
		c.gf.Frame().End()
	})
}

// Color returns the color of the pixel over the given position inside the GLCanvas.
func (c *GLCanvas) Color(at Vec) RGBA {
	return c.gf.Color(at)
}

// Texture returns the underlying OpenGL Texture of this GLCanvas.
//
// Implements GLPicture interface.
func (c *GLCanvas) Texture() *executor.Texture {
	return c.gf.Texture()
}

// Frame returns the underlying OpenGL Frame of this GLCanvas.
func (c *GLCanvas) Frame() *executor.Frame {
	return c.gf.frame
}

// SetPixels replaces the content of the GLCanvas with the provided pixels. The provided slice must be
// an alpha-premultiplied RGBA sequence of correct length (4 * width * height).
func (c *GLCanvas) SetPixels(pixels []uint8) {
	c.gf.Dirty()

	executor.Thread.Call(func() {
		tex := c.Texture()
		tex.Begin()
		tex.SetPixels(0, 0, tex.Width(), tex.Height(), pixels)
		tex.End()
	})
}

// Pixels returns an alpha-premultiplied RGBA sequence of the content of the GLCanvas.
func (c *GLCanvas) Pixels() []uint8 {
	var pixels []uint8

	executor.Thread.Call(func() {
		tex := c.Texture()
		tex.Begin()
		pixels = tex.Pixels(0, 0, tex.Width(), tex.Height())
		tex.End()
	})

	return pixels
}

// Draw draws the content of the GLCanvas onto another ITarget, transformed by the given Matrix, just
// like if it was a Sprite containing the whole GLCanvas.
func (c *GLCanvas) Draw(t ITarget, matrix Matrix) {
	c.sprite.Draw(t, matrix)
}

// DrawColorMask draws the content of the GLCanvas onto another ITarget, transformed by the given
// Matrix and multiplied by the given mask, just like if it was a Sprite containing the whole GLCanvas.
//
// If the color mask is nil, a fully opaque white mask will be used causing no effect.
func (c *GLCanvas) DrawColorMask(t ITarget, matrix Matrix, mask color.Color) {
	c.sprite.DrawColorMask(t, matrix, mask)
}

type canvasTriangles struct {
	*GLTriangles
	dst *GLCanvas
}

func (ct *canvasTriangles) draw(tex *executor.Texture, bounds Rect) {
	ct.dst.gf.Dirty()

	// save the current state vars to avoid race condition
	cmp := ct.dst.cmp
	smt := ct.dst.smooth
	mat := ct.dst.mat
	col := ct.dst.col

	executor.Thread.Post(func() {
		ct.dst.setGlhfBounds()
		setBlendFunc(cmp)

		frame := ct.dst.gf.Frame()
		shader := ct.shader.s

		frame.Begin()
		shader.Begin()

		ct.shader.uniformDefaults.transform = mat
		ct.shader.uniformDefaults.colormask = col
		dstBounds := ct.dst.Bounds()
		ct.shader.uniformDefaults.bounds = mgl32.Vec4{
			float32(dstBounds.Min.X),
			float32(dstBounds.Min.Y),
			float32(dstBounds.W()),
			float32(dstBounds.H()),
		}

		bx, by, bw, bh := intBounds(bounds)
		ct.shader.uniformDefaults.texbounds = mgl32.Vec4{
			float32(bx),
			float32(by),
			float32(bw),
			float32(bh),
		}

		for loc, u := range ct.shader.uniforms {
			_, _ = ct.shader.s.SetUniformAttr(loc, u.Value())
		}

		if tex == nil {
			ct.vs.Begin()
			ct.vs.Draw()
			ct.vs.End()
		} else {
			tex.Begin()

			if tex.Smooth() != smt {
				tex.SetSmooth(smt)
			}

			ct.vs.Begin()
			ct.vs.Draw()
			ct.vs.End()

			tex.End()
		}

		shader.End()
		frame.End()
	})
}

func (ct *canvasTriangles) Draw() {
	ct.draw(nil, Rect{})
}

type canvasPicture struct {
	GLPicture
	dst *GLCanvas
}

func (cp *canvasPicture) Draw(t ITargetTriangles) {
	ct := t.(*canvasTriangles)
	if cp.dst != ct.dst {
		panic(fmt.Errorf("(%T).Draw: ITargetTriangles generated by different GLCanvas", cp))
	}
	ct.draw(cp.GLPicture.Texture(), cp.GLPicture.Bounds())
}

func (cp *canvasPicture) Update(p IPicture) {
	cp.GLPicture.Update(p)
}
