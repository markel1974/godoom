package api

import "unsafe"

// IContext represents an interface for OpenGL-like graphics operations.
type IContext interface {
	ActiveTexture(texture uint32)

	AttachShader(program uint32, shader uint32)

	BindBuffer(target uint32, buffer uint32)

	BindBufferBase(target uint32, index uint32, buffer uint32)

	BindFramebuffer(target uint32, framebuffer uint32)

	BindRenderbuffer(target uint32, renderbuffer uint32)

	BindTexture(target uint32, texture uint32)

	BindVertexArray(array uint32)

	BlendEquation(mode uint32)

	BlendFunc(sfactor uint32, dfactor uint32)

	BlitFramebuffer(srcX0 int32, srcY0 int32, srcX1 int32, srcY1 int32, dstX0 int32, dstY0 int32, dstX1 int32, dstY1 int32, mask uint32, filter uint32)

	BufferData(target uint32, size int, data unsafe.Pointer, usage uint32)

	BufferSubData(target uint32, offset int, size int, data unsafe.Pointer)

	CheckFramebufferStatus(target uint32) uint32

	Clear(mask uint32)

	ClearColor(red float32, green float32, blue float32, alpha float32)

	CompileShader(shader uint32)

	CreateProgram() uint32

	CreateShader(xtype uint32) uint32

	DeleteFramebuffers(n int32, framebuffers *uint32)

	DeleteRenderbuffers(n int32, renderbuffers *uint32)

	DeleteShader(shader uint32)

	DeleteTextures(n int32, textures *uint32)

	DepthFunc(xfunc uint32)

	DepthMask(flag bool)

	Disable(cap uint32)

	DrawArrays(mode uint32, first int32, count int32)

	DrawBuffer(buf uint32)

	DrawBuffers(n int32, bufs *uint32)

	Enable(cap uint32)

	EnableVertexAttribArray(index uint32)

	FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32)

	FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32)

	GenBuffers(n int32, buffers *uint32)

	GenFramebuffers(n int32, framebuffers *uint32)

	GenRenderbuffers(n int32, renderbuffers *uint32)

	GenTextures(n int32, textures *uint32)

	GenVertexArrays(n int32, arrays *uint32)

	GenerateMipmap(target uint32)

	GetFloatv(pname uint32, data *float32)

	GetProgramiv(program uint32, pname uint32, params *int32)

	GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8)

	GetShaderiv(shader uint32, pname uint32, params *int32)

	GetUniformBlockIndex(program uint32, uniformBlockName *uint8) uint32

	GetUniformLocation(program uint32, name *uint8) int32

	Init() error

	LinkProgram(program uint32)

	MultiDrawElements(mode uint32, count *int32, xtype uint32, indices *unsafe.Pointer, drawcount int32)

	PolygonOffset(factor float32, units float32)

	Ptr(data interface{}) unsafe.Pointer

	PtrOffset(offset int) unsafe.Pointer

	ReadBuffer(src uint32)

	RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32)

	RenderbufferStorageMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32)

	ShaderSource(shader uint32, count int32, xstring **uint8, length *int32)

	Str(str string) *uint8

	Strs(strs ...string) (cstrs **uint8, free func())

	TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer)

	TexImage2DMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32, fixedsamplelocations bool)

	TexImage3D(target uint32, level int32, internalformat int32, width int32, height int32, depth int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer)

	TexParameterf(target uint32, pname uint32, param float32)

	TexParameterfv(target uint32, pname uint32, params *float32)

	TexParameteri(target uint32, pname uint32, param int32)

	TexSubImage3D(target uint32, level int32, xoffset int32, yoffset int32, zoffset int32, width int32, height int32, depth int32, format uint32, xtype uint32, pixels unsafe.Pointer)

	Uniform1f(location int32, v0 float32)

	Uniform1i(location int32, v0 int32)

	Uniform1iv(location int32, count int32, value *int32)

	Uniform2f(location int32, v0 float32, v1 float32)

	Uniform3f(location int32, v0 float32, v1 float32, v2 float32)

	Uniform3fv(location int32, count int32, value *float32)

	UniformBlockBinding(program uint32, uniformBlockIndex uint32, uniformBlockBinding uint32)

	UniformMatrix4fv(location int32, count int32, transpose bool, value *float32)

	UseProgram(program uint32)

	VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer unsafe.Pointer)

	Viewport(x int32, y int32, width int32, height int32)
}
