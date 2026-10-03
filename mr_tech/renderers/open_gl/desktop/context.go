//go:build !js

package desktop

import (
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// Context represents an abstraction for managing OpenGL state and issuing rendering commands.
type Context struct{}

// NewContext creates and returns a new instance of Context.
func NewContext() *Context {
	return &Context{}
}

// ActiveTexture selects which texture unit to make active for subsequent texture state calls.
func (d *Context) ActiveTexture(texture uint32) {
	gl.ActiveTexture(texture)
}

// AttachShader attaches a shader object to a program object, preparing it for linking.
func (d *Context) AttachShader(program uint32, shader uint32) {
	gl.AttachShader(program, shader)
}

// BindBuffer binds a buffer object to a specified target in the WebGL context.
func (d *Context) BindBuffer(target uint32, buffer uint32) {
	gl.BindBuffer(target, buffer)
}

// BindBufferBase binds a buffer object to an indexed buffer target, specifying the target, index, and buffer object.
func (d *Context) BindBufferBase(target uint32, index uint32, buffer uint32) {
	gl.BindBufferBase(target, index, buffer)
}

// BindFramebuffer binds a framebuffer object to a specific target for subsequent rendering operations.
func (d *Context) BindFramebuffer(target uint32, framebuffer uint32) {
	gl.BindFramebuffer(target, framebuffer)
}

// BindRenderbuffer binds a renderbuffer to the specified target in the current OpenGL context.
func (d *Context) BindRenderbuffer(target uint32, renderbuffer uint32) {
	gl.BindRenderbuffer(target, renderbuffer)
}

// BindTexture binds a named texture to a texturing target in the current OpenGL context.
func (d *Context) BindTexture(target uint32, texture uint32) {
	gl.BindTexture(target, texture)
}

// BindVertexArray binds the vertex array object identified by the given array ID for use in subsequent OpenGL operations.
func (d *Context) BindVertexArray(array uint32) {
	gl.BindVertexArray(array)
}

// BlendEquation sets the mode used for blending pixel arithmetic in the current OpenGL rendering context.
func (d *Context) BlendEquation(mode uint32) {
	gl.BlendEquation(mode)
}

// BlendFunc configures blending factors used when combining source and destination pixels during rendering.
func (d *Context) BlendFunc(sfactor uint32, dfactor uint32) {
	gl.BlendFunc(sfactor, dfactor)
}

// BlitFramebuffer transfers a region of pixels between source and destination framebuffers with specified mask and filter.
func (d *Context) BlitFramebuffer(srcX0 int32, srcY0 int32, srcX1 int32, srcY1 int32, dstX0 int32, dstY0 int32, dstX1 int32, dstY1 int32, mask uint32, filter uint32) {
	gl.BlitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter)
}

// BufferData updates or creates a buffer object's data store with specified target, size, data, and usage parameters.
func (d *Context) BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	gl.BufferData(target, size, data, usage)
}

// BufferSubData updates a subset of a buffer object's data store with new data, starting at the given offset.
func (d *Context) BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	gl.BufferSubData(target, offset, size, data)
}

// CheckFramebufferStatus checks the completeness status of a framebuffer object target and returns the status code.
func (d *Context) CheckFramebufferStatus(target uint32) uint32 {
	return gl.CheckFramebufferStatus(target)
}

// Clear clears buffers specified by the mask bitfield, such as color, depth, or stencil buffers.
func (d *Context) Clear(mask uint32) {
	gl.Clear(mask)
}

// ClearColor sets the clear color for the rendering context using the specified red, green, blue, and alpha values.
func (d *Context) ClearColor(red float32, green float32, blue float32, alpha float32) {
	gl.ClearColor(red, green, blue, alpha)
}

// CompileShader compiles the specified shader object in the current OpenGL context.
func (d *Context) CompileShader(shader uint32) {
	gl.CompileShader(shader)
}

// CreateProgram creates and returns a new OpenGL program object.
func (d *Context) CreateProgram() uint32 {
	return gl.CreateProgram()
}

// CreateShader creates a new shader object of the specified type (e.g., vertex or fragment) and returns its ID.
func (d *Context) CreateShader(xtype uint32) uint32 {
	return gl.CreateShader(xtype)
}

// DeleteFramebuffers deletes n framebuffers identified by the framebuffer IDs in the memory location pointed to by framebuffers.
func (d *Context) DeleteFramebuffers(n int32, framebuffers *uint32) {
	gl.DeleteFramebuffers(n, framebuffers)
}

// DeleteRenderbuffers deletes renderbuffer objects identified by the IDs in the provided slice. Use to free GPU resources.
func (d *Context) DeleteRenderbuffers(n int32, renderbuffers *uint32) {
	gl.DeleteRenderbuffers(n, renderbuffers)
}

// DeleteShader deletes a shader object, freeing its allocated resources on the GPU.
func (d *Context) DeleteShader(shader uint32) {
	gl.DeleteShader(shader)
}

// DeleteTextures deletes a specified number of texture objects identified by the textures parameter.
func (d *Context) DeleteTextures(n int32, textures *uint32) {
	gl.DeleteTextures(n, textures)
}

// DepthFunc sets the function used to compare each incoming pixel depth value with the depth buffer's value.
func (d *Context) DepthFunc(xfunc uint32) {
	gl.DepthFunc(xfunc)
}

// DepthMask sets the depth buffer writing mode based on the provided boolean flag.
func (d *Context) DepthMask(flag bool) {
	gl.DepthMask(flag)
}

// Disable deactivates a specific OpenGL capability identified by the given cap parameter.
func (d *Context) Disable(cap uint32) {
	gl.Disable(cap)
}

// DrawArrays renders primitives from array data using the specified mode, starting index, and number of vertices.
func (d *Context) DrawArrays(mode uint32, first int32, count int32) {
	gl.DrawArrays(mode, first, count)
}

// DrawBuffer sets the destination buffer for rendering operations based on the specified buffer mode.
func (d *Context) DrawBuffer(buf uint32) {
	gl.DrawBuffer(buf)
}

// DrawBuffers specifies a list of color buffers to be drawn into when rendering to a framebuffer.
func (d *Context) DrawBuffers(n int32, bufs *uint32) {
	gl.DrawBuffers(n, bufs)
}

// Enable activates a specified OpenGL capability by its unique symbolic constant identifier.
func (d *Context) Enable(cap uint32) {
	gl.Enable(cap)
}

// EnableVertexAttribArray enables a generic vertex attribute array at the specified index.
func (d *Context) EnableVertexAttribArray(index uint32) {
	gl.EnableVertexAttribArray(index)
}

// FramebufferRenderbuffer attaches a renderbuffer to a framebuffer for a specific target and attachment point.
func (d *Context) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	gl.FramebufferRenderbuffer(target, attachment, renderbuffertarget, renderbuffer)
}

// FramebufferTexture2D attaches a texture to a framebuffer object at the specified attachment point and level.
func (d *Context) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	gl.FramebufferTexture2D(target, attachment, textarget, texture, level)
}

// GenBuffers generates buffer object names and stores them in the buffers parameter. It allocates 'n' buffer object ids.
func (d *Context) GenBuffers(n int32, buffers *uint32) {
	gl.GenBuffers(n, buffers)
}

// GenFramebuffers generates n framebuffer object names and stores them in the slice pointed to by framebuffers.
func (d *Context) GenFramebuffers(n int32, framebuffers *uint32) {
	gl.GenFramebuffers(n, framebuffers)
}

// GenRenderbuffers generates one or more renderbuffer objects and stores their names in the provided slice.
func (d *Context) GenRenderbuffers(n int32, renderbuffers *uint32) {
	gl.GenRenderbuffers(n, renderbuffers)
}

// GenTextures generates n texture object names and stores them in the provided slice/textures pointer.
func (d *Context) GenTextures(n int32, textures *uint32) {
	gl.GenTextures(n, textures)
}

// GenVertexArrays generates vertex array object names for the specified count and stores them in the provided array.
func (d *Context) GenVertexArrays(n int32, arrays *uint32) {
	gl.GenVertexArrays(n, arrays)
}

// GenerateMipmap generates mipmaps for the specified texture target to improve rendering performance and quality.
func (d *Context) GenerateMipmap(target uint32) {
	gl.GenerateMipmap(target)
}

// GetFloatv retrieves the current value of a floating-point state variable specified by pname into the provided data pointer.
func (d *Context) GetFloatv(pname uint32, data *float32) {
	gl.GetFloatv(pname, data)
}

// GetProgramiv retrieves parameters of a given program object defined by pname and stores the result in params.
func (d *Context) GetProgramiv(program uint32, pname uint32, params *int32) {
	gl.GetProgramiv(program, pname, params)
}

// GetShaderInfoLog retrieves the information log for a shader object, such as warnings or compilation errors.
func (d *Context) GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8) {
	gl.GetShaderInfoLog(shader, bufSize, length, infoLog)
}

// GetShaderiv retrieves a parameter from a shader object based on the specified pname and stores it in params.
func (d *Context) GetShaderiv(shader uint32, pname uint32, params *int32) {
	gl.GetShaderiv(shader, pname, params)
}

// GetUniformBlockIndex returns the index of a named uniform block within a specified program object.
func (d *Context) GetUniformBlockIndex(program uint32, uniformBlockName *uint8) uint32 {
	return gl.GetUniformBlockIndex(program, uniformBlockName)
}

// GetUniformLocation retrieves the location of a uniform variable within a specified shader program.
func (d *Context) GetUniformLocation(program uint32, name *uint8) int32 {
	return gl.GetUniformLocation(program, name)
}

// Init initializes the graphics library and prepares the necessary context for rendering. Returns an error if initialization fails.
func (d *Context) Init() error {
	return gl.Init()
}

// LinkProgram links the specified shader program within the current OpenGL context.
func (d *Context) LinkProgram(program uint32) {
	gl.LinkProgram(program)
}

// MultiDrawElements renders multiple sets of primitives from array data using different element indices for each set.
func (d *Context) MultiDrawElements(mode uint32, count *int32, xtype uint32, indices *unsafe.Pointer, drawcount int32) {
	gl.MultiDrawElements(mode, count, xtype, indices, drawcount)
}

// PolygonOffset sets the scale and units used to calculate depth values for polygons to reduce z-fighting.
func (d *Context) PolygonOffset(factor float32, units float32) {
	gl.PolygonOffset(factor, units)
}

// Ptr converts the given data to an unsafe.Pointer, enabling low-level memory operations within the context.
func (d *Context) Ptr(data interface{}) unsafe.Pointer {
	return gl.Ptr(data)
}

// PtrOffset returns an unsafe.Pointer that represents the specified offset in memory.
func (d *Context) PtrOffset(offset int) unsafe.Pointer {
	return gl.PtrOffset(offset)
}

// ReadBuffer sets the source buffer for subsequent read operations in OpenGL.
func (d *Context) ReadBuffer(src uint32) {
	gl.ReadBuffer(src)
}

// RenderbufferStorage sets storage, format, and dimensions for a renderbuffer object within the current OpenGL context.
func (d *Context) RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	gl.RenderbufferStorage(target, internalformat, width, height)
}

// RenderbufferStorageMultisample sets the parameters of a multisample renderbuffer object's data storage.
func (d *Context) RenderbufferStorageMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32) {
	gl.RenderbufferStorageMultisample(target, samples, internalformat, width, height)
}

// ShaderSource sets the source code for a shader object, replacing any existing source code.
func (d *Context) ShaderSource(shader uint32, count int32, xstring **uint8, length *int32) {
	gl.ShaderSource(shader, count, xstring, length)
}

// Str converts a Go string into a C-style string and returns a pointer to the first byte of the resulting string.
func (d *Context) Str(str string) *uint8 {
	return gl.Str(str)
}

// Strs converts a slice of strings into a format suitable for use with OpenGL functions, returning the pointer and free function.
func (d *Context) Strs(strs ...string) (cstrs **uint8, free func()) {
	return gl.Strs(strs...)
}

// TexImage2D specifies a two-dimensional texture image for the current texture target.
func (d *Context) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	gl.TexImage2D(target, level, internalformat, width, height, border, format, xtype, pixels)
}

// TexImage2DMultisample sets up a 2D multisample texture image with the specified parameters.
func (d *Context) TexImage2DMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32, fixedsamplelocations bool) {
	gl.TexImage2DMultisample(target, samples, internalformat, width, height, fixedsamplelocations)
}

// TexImage3D specifies a three-dimensional texture image in the OpenGL context with the given parameters.
func (d *Context) TexImage3D(target uint32, level int32, internalformat int32, width int32, height int32, depth int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	gl.TexImage3D(target, level, internalformat, width, height, depth, border, format, xtype, pixels)
}

// TexParameterf sets float parameters for a texture target, specified by pname and param.
func (d *Context) TexParameterf(target uint32, pname uint32, param float32) {
	gl.TexParameterf(target, pname, param)
}

// TexParameterfv sets float vector parameters for a texture target with the specified pname and params.
func (d *Context) TexParameterfv(target uint32, pname uint32, params *float32) {
	gl.TexParameterfv(target, pname, params)
}

// TexParameteri sets texture parameters for the specified texture target.
func (d *Context) TexParameteri(target uint32, pname uint32, param int32) {
	gl.TexParameteri(target, pname, param)
}

// TexSubImage3D specifies a three-dimensional subregion of a texture image in the current OpenGL context.
func (d *Context) TexSubImage3D(target uint32, level int32, xoffset int32, yoffset int32, zoffset int32, width int32, height int32, depth int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	gl.TexSubImage3D(target, level, xoffset, yoffset, zoffset, width, height, depth, format, xtype, pixels)
}

// Uniform1f sets the value of a uniform variable for the current WebGL program.
func (d *Context) Uniform1f(location int32, v0 float32) {
	gl.Uniform1f(location, v0)
}

// Uniform1i sets the value of a uniform variable for the current WebGL program to a single integer value.
func (d *Context) Uniform1i(location int32, v0 int32) {
	gl.Uniform1i(location, v0)
}

// Uniform1iv sets the value of a uniform variable for the current shader program as an array of integers.
func (d *Context) Uniform1iv(location int32, count int32, value *int32) {
	gl.Uniform1iv(location, count, value)
}

// Uniform2f sets the values of a 2-component float uniform variable for the current shader program.
func (d *Context) Uniform2f(location int32, v0 float32, v1 float32) {
	gl.Uniform2f(location, v0, v1)
}

// Uniform3f sets the value of a vec3 uniform variable in the active shader program.
func (d *Context) Uniform3f(location int32, v0 float32, v1 float32, v2 float32) {
	gl.Uniform3f(location, v0, v1, v2)
}

// Uniform3fv sets the value of a 3-component floating-point uniform array for the specified shader program location.
func (d *Context) Uniform3fv(location int32, count int32, value *float32) {
	gl.Uniform3fv(location, count, value)
}

// UniformBlockBinding assigns a binding point to a uniform block in a specified shader program.
func (d *Context) UniformBlockBinding(program uint32, uniformBlockIndex uint32, uniformBlockBinding uint32) {
	gl.UniformBlockBinding(program, uniformBlockIndex, uniformBlockBinding)
}

// UniformMatrix4fv sets values for a 4x4 matrix uniform variable at a given location in the shader program.
func (d *Context) UniformMatrix4fv(location int32, count int32, transpose bool, value *float32) {
	gl.UniformMatrix4fv(location, count, transpose, value)
}

// UseProgram activates the specified OpenGL shader program for rendering tasks in the current context.
func (d *Context) UseProgram(program uint32) {
	gl.UseProgram(program)
}

// VertexAttribPointer specifies the location and data format of the array of generic vertex attributes.
func (d *Context) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer unsafe.Pointer) {
	gl.VertexAttribPointer(index, size, xtype, normalized, stride, pointer)
}

// Viewport sets the viewport dimensions for rendering, defined by the specified x, y coordinates, width, and height.
func (d *Context) Viewport(x int32, y int32, width int32, height int32) {
	gl.Viewport(x, y, width, height)
}
