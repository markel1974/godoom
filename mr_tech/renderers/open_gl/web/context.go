//go:build js && wasm

package web

import (
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const (
	glNone             = uint32(0)
	glColorAttachment0 = uint32(0x8CE0)
)

// Context represents a WebGL rendering context, managing WebGL resources and interactions with the underlying JS environment.
type Context struct {
	win *Window
	gl  js.Value

	buffers  ResourceTracker
	textures ResourceTracker
	programs ResourceTracker
	shaders  ResourceTracker
	vaos     ResourceTracker
	fbos     ResourceTracker
	rbos     ResourceTracker

	uniforms []js.Value

	ptrMap   map[uintptr]interface{}
	ptrMutex sync.Mutex

	jsBuffer js.Value
}

// NewContextWeb initializes and returns a new WebGL rendering context for the provided JavaScript WebGL context.
func NewContext(width int, height int) *Context {
	cfg := WindowConfig{
		Width:  width,
		Height: height,
		VSync:  true,
	}
	ctx := &Context{
		buffers:  NewResourceTracker(),
		textures: NewResourceTracker(),
		programs: NewResourceTracker(),
		shaders:  NewResourceTracker(),
		vaos:     NewResourceTracker(),
		fbos:     NewResourceTracker(),
		rbos:     NewResourceTracker(),
		uniforms: []js.Value{js.Null()},
		ptrMap:   make(map[uintptr]interface{}),
		jsBuffer: js.Global().Get("Uint8Array"),
	}
	ctx.win = NewGLWindow(ctx, cfg)
	return ctx
}

func (d *Context) Setup(r api.IRender) error {
	if err := d.win.Setup(r); err != nil {
		return err
	}
	return nil
}

func (d *Context) Start() {
	d.win.Start()
}

// getSliceBytes converts various slice types (e.g., []float32, []uint32, []uint8, []int32) to a byte slice representation.
func (d *Context) getSliceBytes(data interface{}) []byte {
	switch v := data.(type) {
	case []float32:
		if len(v) == 0 {
			return nil
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*4)
	case []uint32:
		if len(v) == 0 {
			return nil
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*4)
	case []uint8:
		return v
	case []int32:
		if len(v) == 0 {
			return nil
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*4)
	}
	return nil
}

// ActiveTexture selects the active texture unit for subsequent texture state modifications.
func (d *Context) ActiveTexture(texture uint32) {
	d.gl.Call("activeTexture", texture)
}

// AttachShader associates a compiled shader object to a program object for linking in a graphics rendering pipeline.
func (d *Context) AttachShader(program uint32, shader uint32) {
	programObj := d.programs.Get(program)
	shaderObj := d.shaders.Get(shader)
	shaderType := d.gl.Call("getShaderParameter", shaderObj, 0x8B4F) // GL_SHADER_TYPE

	fmt.Printf(
		"AttachShader: program=%d shader=%d type=%d program=%s shader=%s\n",
		program,
		shader,
		shaderType.Int(),
		programObj.String(),
		shaderObj.String(),
	)

	//d.gl.Call("attachShader", programObj, shaderObj)

	d.gl.Call("attachShader", d.programs.Get(program), d.shaders.Get(shader))
}

// BindBuffer binds a given buffer object to a specified target for subsequent GPU operations.
func (d *Context) BindBuffer(target uint32, buffer uint32) {
	d.gl.Call("bindBuffer", target, d.buffers.Get(buffer))
}

// BindBufferBase binds a buffer object to an indexed buffer target.
func (d *Context) BindBufferBase(target uint32, index uint32, buffer uint32) {
	d.gl.Call("bindBufferBase", target, index, d.buffers.Get(buffer))
}

// BindFramebuffer binds the specified framebuffer to the given target within the WebGL rendering context.
func (d *Context) BindFramebuffer(target uint32, framebuffer uint32) {
	d.gl.Call("bindFramebuffer", target, d.fbos.Get(framebuffer))
}

// BindRenderbuffer binds a renderbuffer to the specified renderbuffer target within the current graphics context.
func (d *Context) BindRenderbuffer(target uint32, renderbuffer uint32) {
	d.gl.Call("bindRenderbuffer", target, d.rbos.Get(renderbuffer))
}

// BindTexture binds a named texture to a specified target in the WebGL rendering context.
func (d *Context) BindTexture(target uint32, texture uint32) {
	if target == 0x9100 { // TEXTURE_2D_MULTISAMPLE
		target = 0x0DE1 // TEXTURE_2D
	}
	d.gl.Call("bindTexture", target, d.textures.Get(texture))
}

// BindVertexArray binds the specified vertex array object, enabling its state for subsequent rendering operations.
func (d *Context) BindVertexArray(array uint32) {
	d.gl.Call("bindVertexArray", d.vaos.Get(array))
}

// BlendEquation sets the blend equation mode, controlling how source and destination colors are combined.
func (d *Context) BlendEquation(mode uint32) {
	d.gl.Call("blendEquation", mode)
}

// BlendFunc sets the blending factors for source and destination in the rendering pipeline.
func (d *Context) BlendFunc(sfactor uint32, dfactor uint32) {
	d.gl.Call("blendFunc", sfactor, dfactor)
}

// BlitFramebuffer transfers a block of pixels from one framebuffer to another with specified source and destination bounds.
// srcX0, srcY0, srcX1, srcY1 define the source rectangle bounds in the framebuffer.
// dstX0, dstY0, dstX1, dstY1 define the destination rectangle bounds in the framebuffer.
// mask specifies which buffers to copy, such as color, depth, or stencil buffers.
// filter specifies the interpolation to apply during scaling, such as nearest or linear filtering.
// This method is commonly used for resolving multisampled framebuffers or copying pixels between framebuffers.
func (d *Context) BlitFramebuffer(srcX0 int32, srcY0 int32, srcX1 int32, srcY1 int32, dstX0 int32, dstY0 int32, dstX1 int32, dstY1 int32, mask uint32, filter uint32) {
	d.gl.Call("blitFramebuffer", srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter)
}

// BufferData uploads data to a buffer object for the specified target with the given usage pattern.
func (d *Context) BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	if data == nil || uintptr(data) == 0 {
		d.gl.Call("bufferData", target, size, usage)
		return
	}
	d.ptrMutex.Lock()
	dataVal, ok := d.ptrMap[uintptr(data)]
	if ok {
		delete(d.ptrMap, uintptr(data))
	}
	d.ptrMutex.Unlock()

	if !ok || dataVal == nil {
		d.gl.Call("bufferData", target, size, usage)
		return
	}

	bytes := d.getSliceBytes(dataVal)
	if len(bytes) > size {
		bytes = bytes[:size]
	}
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	d.gl.Call("bufferData", target, jsArr, usage)
}

// BufferSubData updates a subset of a buffer object's data store with new data from a specified source in memory.
// target specifies the target buffer object.
// offset specifies the offset into the buffer object's data store where to begin replacement.
// size specifies the size in bytes of the data store region being replaced.
// data specifies a pointer to the source data in memory.
func (d *Context) BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	if data == nil || uintptr(data) == 0 {
		return
	}
	d.ptrMutex.Lock()
	dataVal, ok := d.ptrMap[uintptr(data)]
	if ok {
		delete(d.ptrMap, uintptr(data))
	}
	d.ptrMutex.Unlock()

	if !ok || dataVal == nil {
		return
	}

	bytes := d.getSliceBytes(dataVal)
	if len(bytes) > size {
		bytes = bytes[:size]
	}
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	d.gl.Call("bufferSubData", target, offset, jsArr)
}

// CheckFramebufferStatus checks the completeness status of a framebuffer object for the given target.
//func (d *Context) CheckFramebufferStatus(target uint32) uint32 {
//	return api.FRAMEBUFFER_COMPLETE
//}

func (d *Context) CheckFramebufferStatus(target uint32) uint32 {
	status := d.gl.Call("checkFramebufferStatus", int(target))
	return uint32(status.Int())
}

// Clear resets the specified bits in the context state based on the provided mask.
func (d *Context) Clear(mask uint32) {
	d.gl.Call("clear", mask)
}

// ClearColor sets the color used to clear the color buffer with specified red, green, blue, and alpha values.
func (d *Context) ClearColor(red float32, green float32, blue float32, alpha float32) {
	d.gl.Call("clearColor", red, green, blue, alpha)
}

// CompileShader compiles the specified shader within the current graphics context.
func (d *Context) CompileShader(shader uint32) {
	d.gl.Call("compileShader", d.shaders.Get(shader))
}

// CreateProgram creates a new shader program, adds it to the internal program manager, and returns its unique ID.
func (d *Context) CreateProgram() uint32 {
	obj := d.gl.Call("createProgram")
	id := d.programs.Add(obj)
	fmt.Printf(
		"CreateProgram: id=%d JS=%s\n",
		id,
		obj.String(),
	)
	return id
}

// CreateShader creates a shader of the specified type and returns its unique identifier from the shader registry.
func (d *Context) CreateShader(xtype uint32) uint32 {
	obj := d.gl.Call("createShader", xtype)
	id := d.shaders.Add(obj)
	shaderType := d.gl.Call("getShaderParameter", obj, 0x8B4F) // GL_SHADER_TYPE
	fmt.Printf(
		"CreateShader: GL type=%d id=%d JS type=%d\n",
		xtype,
		id,
		shaderType.Int(),
	)
	return id
}

// DeleteFramebuffers deletes framebuffer objects referenced by the IDs in the provided array.
// n specifies the number of framebuffers to delete, and framebuffers points to the IDs of the framebuffers to delete.
func (d *Context) DeleteFramebuffers(n int32, framebuffers *uint32) {
	arr := unsafe.Slice(framebuffers, n)
	for i := int32(0); i < n; i++ {
		d.gl.Call("deleteFramebuffer", d.fbos.Get(arr[i]))
		d.fbos.Remove(arr[i])
	}
}

// DeleteRenderbuffers deletes renderbuffer objects identified by the specified slices of renderbuffer names.
// The parameter `n` specifies the number of renderbuffers to delete, and `renderbuffers` points to the list of names.
func (d *Context) DeleteRenderbuffers(n int32, renderbuffers *uint32) {
	arr := unsafe.Slice(renderbuffers, n)
	for i := int32(0); i < n; i++ {
		d.gl.Call("deleteRenderbuffer", d.rbos.Get(arr[i]))
		d.rbos.Remove(arr[i])
	}
}

// DeleteShader deletes a compiled shader object given its ID, freeing associated resources.
//func (d *Context) DeleteShader(shader uint32) {
// TODO: Check implementation
//	panic("DeleteShader not implemented")
//}

func (d *Context) DeleteShader(shader uint32) {
	shaderObj := d.shaders.Get(shader)
	if shaderObj.IsNull() || shaderObj.IsUndefined() {
		return
	}
	d.gl.Call("deleteShader", shaderObj)
	// Do not remove the shader from the tracker or recycle its ID here.
	// OpenGL allows a deleted shader to remain attached to programs, and
	// the engine may reuse the same shader handle for subsequent programs.
	// Keeping the WebGLShader object associated with the virtual handle
	// preserves this OpenGL lifetime semantics.
	//d.shaders.Remove(shader)
}

// DeleteTextures deletes a number of textures specified by the `n` parameter, referenced by the `textures` pointer.
func (d *Context) DeleteTextures(n int32, textures *uint32) {
	arr := unsafe.Slice(textures, n)
	for i := int32(0); i < n; i++ {
		d.gl.Call("deleteTexture", d.textures.Get(arr[i]))
		d.textures.Remove(arr[i])
	}
}

// DepthFunc sets the function used for depth buffer comparisons to control whether a fragment passes the depth test.
func (d *Context) DepthFunc(xfunc uint32) {
	d.gl.Call("depthFunc", xfunc)
}

// DepthMask sets whether writing into the depth buffer is enabled or disabled based on the given boolean flag.
func (d *Context) DepthMask(flag bool) {
	d.gl.Call("depthMask", flag)
}

// Disable disables a specific capability for the given context based on the provided capability identifier.
func (d *Context) Disable(cap uint32) {
	if cap == 0x864F { // DEPTH_CLAMP
		return
	}
	d.gl.Call("disable", cap)
}

// DrawArrays renders primitives from array data based on the specified mode, starting index, and number of vertices.
func (d *Context) DrawArrays(mode uint32, first int32, count int32) {
	d.gl.Call("drawArrays", mode, first, count)
}

// DrawBuffer sets the destination buffer for rendering operations.
func (d *Context) setDrawBuffers(bufs []uint32) {
	if len(bufs) == 0 {
		arr := js.Global().Get("Uint32Array").New(0)
		d.gl.Call("drawBuffers", arr)
		return
	}
	maxIndex := 0
	for _, buf := range bufs {
		if buf < glColorAttachment0 {
			continue
		}

		index := int(buf - glColorAttachment0)
		if index > maxIndex {
			maxIndex = index
		}
	}

	arr := js.Global().Get("Uint32Array").New(maxIndex + 1)
	for i := 0; i <= maxIndex; i++ {
		arr.SetIndex(i, int(glNone))
	}
	for _, buf := range bufs {
		if buf == glNone {
			continue
		}
		if buf < glColorAttachment0 {
			continue
		}
		index := int(buf - glColorAttachment0)
		arr.SetIndex(index, int(buf))
	}
	d.gl.Call("drawBuffers", arr)
}

func (d *Context) DrawBuffer(buf uint32) {
	d.setDrawBuffers([]uint32{buf})
}

func (d *Context) DrawBuffers(n int32, bufs *uint32) {
	slice := unsafe.Slice(bufs, n)
	d.setDrawBuffers(slice)
}

// Enable enables the specified capability for the current context, identified by the provided cap parameter.
func (d *Context) Enable(cap uint32) {
	if cap == 0x864F || cap == 0x809D { // DEPTH_CLAMP, MULTISAMPLE
		return
	}
	d.gl.Call("enable", cap)
}

// EnableVertexAttribArray enables a generic vertex attribute array at the specified index.
func (d *Context) EnableVertexAttribArray(index uint32) {
	d.gl.Call("enableVertexAttribArray", index)
}

// FramebufferRenderbuffer attaches a renderbuffer to a framebuffer object at a specific attachment point.
func (d *Context) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	d.gl.Call("framebufferRenderbuffer", target, attachment, renderbuffertarget, d.rbos.Get(renderbuffer))
}

// FramebufferTexture2D attaches a texture image to a framebuffer at the specified attachment point and mipmap level.
func (d *Context) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	if textarget == 0x9100 { // TEXTURE_2D_MULTISAMPLE
		textarget = 0x0DE1 // TEXTURE_2D
	}
	d.gl.Call("framebufferTexture2D", target, attachment, textarget, d.textures.Get(texture), level)
}

// GenBuffers generates n buffer object names and stores them in the provided buffers pointer.
func (d *Context) GenBuffers(n int32, buffers *uint32) {
	arr := unsafe.Slice(buffers, n)
	for i := int32(0); i < n; i++ {
		arr[i] = d.buffers.Add(d.gl.Call("createBuffer"))
	}
}

// GenFramebuffers generates n framebuffer object names and stores them in the memory pointed to by framebuffers.
func (d *Context) GenFramebuffers(n int32, framebuffers *uint32) {
	arr := unsafe.Slice(framebuffers, n)
	for i := int32(0); i < n; i++ {
		arr[i] = d.fbos.Add(d.gl.Call("createFramebuffer"))
	}
}

// GenRenderbuffers generates n renderbuffer object names and stores them in the memory pointed to by renderbuffers.
func (d *Context) GenRenderbuffers(n int32, renderbuffers *uint32) {
	arr := unsafe.Slice(renderbuffers, n)
	for i := int32(0); i < n; i++ {
		arr[i] = d.rbos.Add(d.gl.Call("createRenderbuffer"))
	}
}

// GenTextures generates texture objects and stores their identifiers in the provided textures pointer.
func (d *Context) GenTextures(n int32, textures *uint32) {
	arr := unsafe.Slice(textures, n)
	for i := int32(0); i < n; i++ {
		arr[i] = d.textures.Add(d.gl.Call("createTexture"))
	}
}

// GenVertexArrays generates `n` vertex array objects and stores their IDs in the memory pointed to by `arrays`.
func (d *Context) GenVertexArrays(n int32, arrays *uint32) {
	arr := unsafe.Slice(arrays, n)
	for i := int32(0); i < n; i++ {
		arr[i] = d.vaos.Add(d.gl.Call("createVertexArray"))
	}
}

// GenerateMipmap generates mipmaps for the specified texture target to enhance rendering performance and visual quality.
func (d *Context) GenerateMipmap(target uint32) {
	d.gl.Call("generateMipmap", target)
}

// GetFloatv retrieves the value or values of a specified floating-point state variable.
// pname specifies the state variable to query, and data is a pointer where the result is stored.
func (d *Context) GetFloatv(pname uint32, data *float32) {
	if pname == 0x84FF {
		ext := d.gl.Call("getExtension", "EXT_texture_filter_anisotropic")
		if ext.IsNull() || ext.IsUndefined() {
			ext = d.gl.Call("getExtension", "MOZ_EXT_texture_filter_anisotropic")
		}
		if ext.IsNull() || ext.IsUndefined() {
			ext = d.gl.Call("getExtension", "WEBKIT_EXT_texture_filter_anisotropic")
		}
		if !ext.IsNull() && !ext.IsUndefined() {
			maxAniso := d.gl.Call("getParameter", ext.Get("MAX_TEXTURE_MAX_ANISOTROPY_EXT"))
			if !maxAniso.IsUndefined() {
				*data = float32(maxAniso.Float())
				return
			}
		}
		*data = 1.0
		return
	}
	v := d.gl.Call("getParameter", pname)
	if !v.IsUndefined() && !v.IsNull() {
		if v.Type() == js.TypeNumber {
			*data = float32(v.Float())
		}
	}
}

// GetProgramiv retrieves a parameter from a program object, such as its link status or active attribute count.
func (d *Context) GetProgramiv(program uint32, pname uint32, params *int32) {
	val := d.gl.Call("getProgramParameter", d.programs.Get(program), pname)
	if val.Type() == js.TypeBoolean {
		if val.Bool() {
			*params = 1
		} else {
			*params = 0
			if pname == 0x8B82 { // LINK_STATUS
				log := d.gl.Call("getProgramInfoLog", d.programs.Get(program))
				if !log.IsUndefined() && !log.IsNull() {
					fmt.Println("CRITICAL LINK ERROR:", log.String())
				}
			}
		}
	} else if val.Type() == js.TypeNumber {
		*params = int32(val.Int())
	} else {
		*params = 0
	}
}

// GetShaderInfoLog retrieves the information log for a shader object, including messages from the shader compilation process.
func (d *Context) GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8) {
	log := d.gl.Call("getShaderInfoLog", d.shaders.Get(shader))
	if !log.IsUndefined() && !log.IsNull() {
		str := log.String()
		if str != "" {
			fmt.Println("Shader Compile Error:", str)
		}
	}
}

// GetShaderiv retrieves a parameter value from a shader object.
// The parameter is specified by pname, and the result is stored in params.
// Shader is the name of the shader object to query.
func (d *Context) GetShaderiv(shader uint32, pname uint32, params *int32) {
	val := d.gl.Call("getShaderParameter", d.shaders.Get(shader), pname)
	if val.Type() == js.TypeBoolean {
		if val.Bool() {
			*params = 1
		} else {
			*params = 0
			if pname == 0x8B81 { // COMPILE_STATUS
				log := d.gl.Call("getShaderInfoLog", d.shaders.Get(shader))
				if !log.IsUndefined() && !log.IsNull() {
					js.Global().Get("console").Call("error", "Shader Compile Error:", log.String())
				}
			}
		}
	} else if val.Type() == js.TypeNumber {
		*params = int32(val.Int())
	} else {
		*params = 0
	}
}

// GetUniformBlockIndex retrieves the index of a uniform block within a program by its name.
func (d *Context) GetUniformBlockIndex(program uint32, uniformBlockName *uint8) uint32 {
	d.ptrMutex.Lock()
	val, ok := d.ptrMap[uintptr(unsafe.Pointer(uniformBlockName))]
	d.ptrMutex.Unlock()

	strName := ""
	if ok && val != nil {
		strName = val.(string)
	}
	strName = strings.ReplaceAll(strName, "\x00", "")

	loc := d.gl.Call("getUniformBlockIndex", d.programs.Get(program), strName)
	if loc.Type() == js.TypeNumber {
		return uint32(loc.Int())
	}
	return 0xFFFFFFFF // GL_INVALID_INDEX
}

// GetUniformLocation retrieves the location of a uniform variable within a given WebGL program object.
// Returns the index of the uniform or -1 if the uniform does not exist.
func (d *Context) GetUniformLocation(program uint32, name *uint8) int32 {
	d.ptrMutex.Lock()
	val, ok := d.ptrMap[uintptr(unsafe.Pointer(name))]
	d.ptrMutex.Unlock()

	strName := ""
	if ok && val != nil {
		strName = val.(string)
	}
	strName = strings.ReplaceAll(strName, "\x00", "")

	loc := d.gl.Call("getUniformLocation", d.programs.Get(program), strName)
	if loc.IsNull() || loc.IsUndefined() {
		return -1
	}
	d.uniforms = append(d.uniforms, loc)
	return int32(len(d.uniforms) - 1)
}

// Init initializes the Context and prepares it for use, returning an error if the initialization fails.
func (d *Context) Init() error {
	return nil
}

// LinkProgram links a compiled shader program, making it executable within the current OpenGL rendering context.
func (d *Context) LinkProgram(program uint32) {
	d.gl.Call("linkProgram", d.programs.Get(program))
}

// MultiDrawElements renders multiple sets of primitives by specifying multiple indices, counts, and modes in a single call.
// mode specifies the kind of primitives to render.
// count is a pointer to an array specifying the number of elements to render per draw call.
// xtype specifies the type of data in the indices array (e.g., GL_UNSIGNED_BYTE, GL_UNSIGNED_SHORT, GL_UNSIGNED_INT).
// indices is a pointer to the starting point of each index array.
// drawcount specifies the number of draw calls to execute.
func (d *Context) MultiDrawElements(mode uint32, count *int32, xtype uint32, indices *unsafe.Pointer, drawcount int32) {
	c_arr := unsafe.Slice(count, drawcount)
	i_arr := unsafe.Slice(indices, drawcount)
	for i := int32(0); i < drawcount; i++ {
		d.gl.Call("drawElements", mode, c_arr[i], xtype, uintptr(i_arr[i]))
	}
}

// PolygonOffset sets the scale and units used to calculate depth offset for polygons to avoid depth-fighting.
func (d *Context) PolygonOffset(factor float32, units float32) {
	d.gl.Call("polygonOffset", factor, units)
}

// Ptr registers the given data and returns an unsafe.Pointer that can be used to reference it in memory.
func (d *Context) Ptr(data interface{}) unsafe.Pointer {
	d.ptrMutex.Lock()
	defer d.ptrMutex.Unlock()
	b := new(byte)
	ptr := unsafe.Pointer(b)
	d.ptrMap[uintptr(ptr)] = data
	return ptr
}

// PtrOffset returns a pointer calculated by applying the given offset to the base address of the context.
func (d *Context) PtrOffset(offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(offset))
}

// ReadBuffer binds a buffer object to a source target for the current context.
func (d *Context) ReadBuffer(src uint32) {
	d.gl.Call("readBuffer", src)
}

// RenderbufferStorage defines storage parameters for a renderbuffer object currently bound to the specified target.
// target specifies the target of the operation, typically gl.RENDERBUFFER.
// internalformat specifies the internal format to use for the renderbuffer's storage.
// width specifies the width of the renderbuffer in pixels.
// height specifies the height of the renderbuffer in pixels.
func (d *Context) RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	d.gl.Call("renderbufferStorage", target, internalformat, width, height)
}

// RenderbufferStorageMultisample specifies storage format and dimensions for a multisample renderbuffer object.
func (d *Context) RenderbufferStorageMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32) {
	// Downgrade MSAA renderbuffers to standard renderbuffers to match downgraded MSAA textures
	d.gl.Call("renderbufferStorage", target, internalformat, width, height)
}

// ShaderSource sets the source code in a shader object to the specified string array.
func (d *Context) ShaderSource(shader uint32, count int32, xstring **uint8, length *int32) {
	const version = "#version 300 es"

	d.ptrMutex.Lock()
	val, ok := d.ptrMap[uintptr(unsafe.Pointer(xstring))]
	d.ptrMutex.Unlock()

	source := ""
	if ok && val != nil {
		source = val.(string)
	}

	// Remove embedded NUL characters.
	source = strings.ReplaceAll(source, "\x00", "")

	// Convert GLSL version from desktop OpenGL to OpenGL ES.
	if !strings.Contains(source, version) {
		source = strings.ReplaceAll(source, "#version 330 core", version)
	}

	// Collect missing precision specifiers.
	precisionSpecifiers := []string{
		"precision highp sampler2DShadow;",
		"precision highp sampler2D;",
		"precision highp sampler2DArray;",
		"precision highp float;",
	}

	var missing []string

	for _, specifier := range precisionSpecifiers {
		if !strings.Contains(source, specifier) {
			missing = append(missing, specifier)
		}
	}

	// Inject all missing precision specifiers at once.
	if len(missing) > 0 {
		source = strings.Replace(source, version, version+"\n"+strings.Join(missing, "\n"), 1)
	}

	//js.Global().Get("console").Call("info", source)

	d.gl.Call("shaderSource", d.shaders.Get(shader), source)
}

// Str converts the provided string into a pointer to a uint8 and returns it, enabling compatibility with C-style strings.
func (d *Context) Str(str string) *uint8 {
	d.ptrMutex.Lock()
	defer d.ptrMutex.Unlock()
	b := new(uint8)
	ptr := (*uint8)(unsafe.Pointer(b))
	d.ptrMap[uintptr(unsafe.Pointer(ptr))] = str
	return ptr
}

// Strs converts a variadic list of strings into a C-style string array and returns a pointer and a cleanup function.
func (d *Context) Strs(strs ...string) (cstrs **uint8, free func()) {
	d.ptrMutex.Lock()
	defer d.ptrMutex.Unlock()
	b := new(uint8)
	ptr := (**uint8)(unsafe.Pointer(&b))
	d.ptrMap[uintptr(unsafe.Pointer(ptr))] = strs[0]
	return ptr, func() {}
}

// TexImage2D defines a two-dimensional texture image in the current WebGL rendering context.
func (d *Context) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	// Fallback RGBA16F to standard RGBA8 (sized)
	if internalformat == 0x881A {
		internalformat = 0x8058 // RGBA8
		format = 0x1908         // RGBA
		xtype = 0x1401          // UNSIGNED_BYTE
	}

	// Fallback RED+FLOAT to R8+UNSIGNED_BYTE
	if internalformat == 0x1903 && xtype == 0x1406 {
		internalformat = 0x8229 // R8
		format = 0x1903         // RED
		xtype = 0x1401          // UNSIGNED_BYTE
	}

	// Fallback DEPTH_COMPONENT+FLOAT to DEPTH_COMPONENT24
	if internalformat == 0x1902 && xtype == 0x1406 {
		internalformat = 0x81A6 // DEPTH_COMPONENT24
		format = 0x1902         // DEPTH_COMPONENT
		xtype = 0x1405          // UNSIGNED_INT
	}

	if pixels == nil || uintptr(pixels) == 0 {
		d.gl.Call("texImage2D", target, level, internalformat, width, height, border, format, xtype, js.Null())
		return
	}
	d.ptrMutex.Lock()
	dataVal, ok := d.ptrMap[uintptr(pixels)]
	if ok {
		delete(d.ptrMap, uintptr(pixels))
	}
	d.ptrMutex.Unlock()

	if !ok || dataVal == nil {
		d.gl.Call("texImage2D", target, level, internalformat, width, height, border, format, xtype, js.Null())
		return
	}

	bytes := d.getSliceBytes(dataVal)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	d.gl.Call("texImage2D", target, level, internalformat, width, height, border, format, xtype, getJSView(jsArr, xtype))
}

// TexImage2DMultisample specifies storage for a 2D multisample texture.
func (d *Context) TexImage2DMultisample(target uint32, samples int32, internalformat uint32, width int32, height int32, fixedsamplelocations bool) {
	if target == 0x9100 { // TEXTURE_2D_MULTISAMPLE
		target = 0x0DE1 // TEXTURE_2D
	}

	format := uint32(0x1908) // RGBA
	xtype := uint32(0x1401)  // UNSIGNED_BYTE

	if internalformat == 0x881A {
		internalformat = 0x8058 // RGBA8
	}

	d.gl.Call("texImage2D", target, 0, internalformat, width, height, 0, format, xtype, js.Null())

	// CRITICAL: WebGL defaults to NEAREST_MIPMAP_LINEAR for TEXTURE_2D.
	// Since we downgraded an MSAA texture (which doesn't have mipmaps) to TEXTURE_2D,
	// we MUST force NEAREST or LINEAR filtering to make the texture "mipmap complete".
	// Otherwise CheckFramebufferStatus will return FRAMEBUFFER_INCOMPLETE_ATTACHMENT.
	d.gl.Call("texParameteri", target, 0x2801 /* TEXTURE_MIN_FILTER */, 0x2600 /* NEAREST */)
	d.gl.Call("texParameteri", target, 0x2800 /* TEXTURE_MAG_FILTER */, 0x2600 /* NEAREST */)
	d.gl.Call("texParameteri", target, 0x2802 /* TEXTURE_WRAP_S */, 0x812F /* CLAMP_TO_EDGE */)
	d.gl.Call("texParameteri", target, 0x2803 /* TEXTURE_WRAP_T */, 0x812F /* CLAMP_TO_EDGE */)
}

// TexImage3D specifies a three-dimensional texture image for a target texture.
func (d *Context) TexImage3D(target uint32, level int32, internalformat int32, width int32, height int32, depth int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	if pixels == nil || uintptr(pixels) == 0 {
		d.gl.Call("texImage3D", target, level, internalformat, width, height, depth, border, format, xtype, js.Null())
		return
	}
	d.ptrMutex.Lock()
	dataVal, ok := d.ptrMap[uintptr(pixels)]
	if ok {
		delete(d.ptrMap, uintptr(pixels))
	}
	d.ptrMutex.Unlock()

	if !ok || dataVal == nil {
		d.gl.Call("texImage3D", target, level, internalformat, width, height, depth, border, format, xtype, js.Null())
		return
	}

	bytes := d.getSliceBytes(dataVal)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	d.gl.Call("texImage3D", target, level, internalformat, width, height, depth, border, format, xtype, getJSView(jsArr, xtype))
}

// TexParameterf sets the float parameter for a specific texture target and property.
func (d *Context) TexParameterf(target uint32, pname uint32, param float32) {
	d.gl.Call("texParameterf", target, pname, param)
}

// TexParameterfv sets float parameters for a texture target, specified by target, pname, and the pointer params.
func (d *Context) TexParameterfv(target uint32, pname uint32, params *float32) {
	// WebGL2 does not support texParameterfv and TEXTURE_BORDER_COLOR natively.
	// CLAMP_TO_BORDER is already downgraded to CLAMP_TO_EDGE in TexParameteri.
}

// TexParameteri sets parameters for a texture object, specified by target, pname, and param values.
func (d *Context) TexParameteri(target uint32, pname uint32, param int32) {
	if param == 0x812D { // CLAMP_TO_BORDER
		param = 0x812F // CLAMP_TO_EDGE
	}
	d.gl.Call("texParameteri", target, pname, param)
}

// TexSubImage3D updates a portion of a 3D texture with new pixel data for the specified level and offset coordinates.
func (d *Context) TexSubImage3D(target uint32, level int32, xoffset int32, yoffset int32, zoffset int32, width int32, height int32, depth int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	if pixels == nil || uintptr(pixels) == 0 {
		return
	}
	d.ptrMutex.Lock()
	dataVal, ok := d.ptrMap[uintptr(pixels)]
	if ok {
		delete(d.ptrMap, uintptr(pixels))
	}
	d.ptrMutex.Unlock()

	if !ok || dataVal == nil {
		return
	}

	bytes := d.getSliceBytes(dataVal)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	d.gl.Call("texSubImage3D", target, level, xoffset, yoffset, zoffset, width, height, depth, format, xtype, getJSView(jsArr, xtype))
}

// Uniform1f sets the value of a float uniform variable at the given location in the WebGL program context.
func (d *Context) Uniform1f(location int32, v0 float32) {
	if location == -1 {
		return
	}
	d.gl.Call("uniform1f", d.uniforms[location], v0)
}

// Uniform1i specifies the integer value of a uniform variable for the current shader program.
func (d *Context) Uniform1i(location int32, v0 int32) {
	if location == -1 {
		return
	}
	d.gl.Call("uniform1i", d.uniforms[location], v0)
}

// Uniform1iv sets the value of a uniform variable array in the active shader program as a slice of integers.
func (d *Context) Uniform1iv(location int32, count int32, value *int32) {
	if location == -1 {
		return
	}
	loc := d.uniforms[location]
	slice := unsafe.Slice(value, count)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&slice[0])), count*4)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	i32Arr := js.Global().Get("Int32Array").New(jsArr.Get("buffer"))
	d.gl.Call("uniform1iv", loc, i32Arr)
}

// Uniform2f sets the values of a 2-component floating-point uniform variable for the current shader program.
func (d *Context) Uniform2f(location int32, v0 float32, v1 float32) {
	if location == -1 {
		return
	}
	d.gl.Call("uniform2f", d.uniforms[location], v0, v1)
}

// Uniform3f sets the values of a 3-component floating-point uniform variable for the current shader program.
func (d *Context) Uniform3f(location int32, v0 float32, v1 float32, v2 float32) {
	if location == -1 {
		return
	}
	d.gl.Call("uniform3f", d.uniforms[location], v0, v1, v2)
}

// Uniform3fv sets the value of a 3-component floating point uniform variable or an array of such variables in a program.
func (d *Context) Uniform3fv(location int32, count int32, value *float32) {
	if location == -1 {
		return
	}
	loc := d.uniforms[location]
	total := count * 3
	slice := unsafe.Slice(value, total)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&slice[0])), total*4)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	f32Arr := js.Global().Get("Float32Array").New(jsArr.Get("buffer"))
	d.gl.Call("uniform3fv", loc, f32Arr)
}

// UniformBlockBinding assigns a binding point to a uniform block within the specified program's shader.
func (d *Context) UniformBlockBinding(program uint32, uniformBlockIndex uint32, uniformBlockBinding uint32) {
	d.gl.Call("uniformBlockBinding", d.programs.Get(program), uniformBlockIndex, uniformBlockBinding)
}

// UniformMatrix4fv sets the value of a 4x4 floating-point matrix uniform variable in the current WebGL program.
// location specifies the location of the uniform variable.
// count specifies the number of matrices to be passed.
// transpose indicates whether the matrix should be transposed when transferred.
// value is a pointer to the first element of the matrix data.
func (d *Context) UniformMatrix4fv(location int32, count int32, transpose bool, value *float32) {
	if location == -1 {
		return
	}
	loc := d.uniforms[location]
	total := count * 16
	slice := unsafe.Slice(value, total)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&slice[0])), total*4)
	jsArr := d.jsBuffer.New(len(bytes))
	js.CopyBytesToJS(jsArr, bytes)
	f32Arr := js.Global().Get("Float32Array").New(jsArr.Get("buffer"))
	d.gl.Call("uniformMatrix4fv", loc, transpose, f32Arr)
}

// UseProgram sets the active shader program to the one specified by the given program ID.
func (d *Context) UseProgram(program uint32) {
	d.gl.Call("useProgram", d.programs.Get(program))
}

// VertexAttribPointer specifies the location and data format of the array of generic vertex attributes.
func (d *Context) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer unsafe.Pointer) {
	offset := uintptr(pointer)
	d.gl.Call("vertexAttribPointer", index, size, xtype, normalized, stride, offset)
}

// Viewport sets the viewport dimensions and position using specified x, y, width, and height parameters.
func (d *Context) Viewport(x int32, y int32, width int32, height int32) {
	d.gl.Call("viewport", x, y, width, height)
}

func getJSView(buffer js.Value, xtype uint32) js.Value {
	switch xtype {
	case 0x1400: // BYTE
		return js.Global().Get("Int8Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength"))
	case 0x1401: // UNSIGNED_BYTE
		return js.Global().Get("Uint8Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength"))
	case 0x1402: // SHORT
		return js.Global().Get("Int16Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength").Int()/2)
	case 0x1403: // UNSIGNED_SHORT
		return js.Global().Get("Uint16Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength").Int()/2)
	case 0x1404: // INT
		return js.Global().Get("Int32Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength").Int()/4)
	case 0x1405: // UNSIGNED_INT
		return js.Global().Get("Uint32Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength").Int()/4)
	case 0x1406: // FLOAT
		return js.Global().Get("Float32Array").New(buffer.Get("buffer"), buffer.Get("byteOffset"), buffer.Get("byteLength").Int()/4)
	}
	return buffer
}
