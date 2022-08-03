package executor

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type VertexSlice struct {
	va   *vertexArray
	i, j int
}

func MakeVertexSlice(shader *Shader, len, cap int) *VertexSlice {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice{
		va: newVertexArray(shader, cap),
		i:  0,
		j:  len,
	}
}

func (vs *VertexSlice) VertexFormat() AttrFormat {
	return vs.va.format
}

func (vs *VertexSlice) Stride() int {
	return vs.va.stride / 4
}

func (vs *VertexSlice) Len() int {
	return vs.j - vs.i
}

func (vs *VertexSlice) Cap() int {
	return vs.va.cap - vs.i
}

func (vs *VertexSlice) SetLen(len int) {
	vs.End()
	*vs = vs.grow(len)
	vs.Begin()
}

func (vs VertexSlice) grow(len int) VertexSlice {
	if len <= vs.Cap() {
		return VertexSlice{
			va: vs.va,
			i:  vs.i,
			j:  vs.i + len,
		}
	}

	newCap := vs.Cap()
	if newCap < 1024 {
		newCap += newCap
	} else {
		newCap += newCap / 4
	}
	if newCap < len {
		newCap = len
	}
	newVs := VertexSlice{
		va: newVertexArray(vs.va.shader, newCap),
		i:  0,
		j:  len,
	}

	newVs.Begin()
	newVs.Slice(0, vs.Len()).SetVertexData(vs.VertexData())
	newVs.End()
	return newVs
}

func (vs *VertexSlice) Slice(i, j int) *VertexSlice {
	if i < 0 || j < i || j > vs.va.cap {
		panic("failed to slice vertex slice: index out of range")
	}
	return &VertexSlice{
		va: vs.va,
		i:  vs.i + i,
		j:  vs.i + j,
	}
}

func (vs *VertexSlice) SetVertexData(data []float32) {
	if len(data)/vs.Stride() != vs.Len() {
		fmt.Println(len(data)/vs.Stride(), vs.Len())
		panic("set vertex data: wrong length of vertices")
	}
	vs.va.setVertexData(vs.i, vs.j, data)
}

func (vs *VertexSlice) VertexData() []float32 {
	return vs.va.vertexData(vs.i, vs.j)
}

func (vs *VertexSlice) Draw() {
	vs.va.draw(vs.i, vs.j)
}

func (vs *VertexSlice) Begin() {
	vs.va.begin()
}

func (vs *VertexSlice) End() {
	vs.va.end()
}

type vertexArray struct {
	vao, vbo binder
	cap      int
	format   AttrFormat
	stride   int
	offset   []int
	shader   *Shader
}

const vertexArrayMinCap = 4

func newVertexArray(shader *Shader, cap int) *vertexArray {
	if cap < vertexArrayMinCap {
		cap = vertexArrayMinCap
	}

	va := &vertexArray{
		vao: binder{
			restoreLoc: gl.VERTEX_ARRAY_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindVertexArray(obj)
			},
		},
		vbo: binder{
			restoreLoc: gl.ARRAY_BUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindBuffer(gl.ARRAY_BUFFER, obj)
			},
		},
		cap:    cap,
		format: shader.VertexFormat(),
		stride: shader.VertexFormat().Size(),
		offset: make([]int, len(shader.VertexFormat())),
		shader: shader,
	}

	offset := 0
	for i, attr := range va.format {
		switch attr.Type {
		case Float, Vec2, Vec3, Vec4:
		default:
			panic(errors.New("failed to create vertex array: invalid attribute type"))
		}
		va.offset[i] = offset
		offset += attr.Type.Size()
	}

	gl.GenVertexArrays(1, &va.vao.obj)

	va.vao.bind()

	gl.GenBuffers(1, &va.vbo.obj)
	defer va.vbo.bind().restore()

	emptyData := make([]byte, cap*va.stride)
	gl.BufferData(gl.ARRAY_BUFFER, len(emptyData), gl.Ptr(emptyData), gl.DYNAMIC_DRAW)

	for i, attr := range va.format {
		loc := gl.GetAttribLocation(shader.program.obj, gl.Str(attr.Name+"\x00"))

		var size int32
		switch attr.Type {
		case Float:
			size = 1
		case Vec2:
			size = 2
		case Vec3:
			size = 3
		case Vec4:
			size = 4
		}

		gl.VertexAttribPointerWithOffset(
			uint32(loc),
			size,
			gl.FLOAT,
			false,
			int32(va.stride),
			uintptr(va.offset[i]),
		)
		gl.EnableVertexAttribArray(uint32(loc))
	}

	va.vao.restore()

	runtime.SetFinalizer(va, (*vertexArray).delete)

	return va
}

func (va *vertexArray) delete() {
	Thread.Post(func() {
		gl.DeleteVertexArrays(1, &va.vao.obj)
		gl.DeleteBuffers(1, &va.vbo.obj)
	})
}

func (va *vertexArray) begin() {
	va.vao.bind()
	va.vbo.bind()
}

func (va *vertexArray) end() {
	va.vbo.restore()
	va.vao.restore()
}

func (va *vertexArray) draw(i, j int) {
	gl.DrawArrays(gl.TRIANGLES, int32(i), int32(j-i))
}

func (va *vertexArray) setVertexData(i, j int, data []float32) {
	if j-i == 0 {
		// avoid setting 0 bytes of buffer data
		return
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
}

func (va *vertexArray) vertexData(i, j int) []float32 {
	if j-i == 0 {
		// avoid getting 0 bytes of buffer data
		return nil
	}
	data := make([]float32, (j-i)*va.stride/4)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
	return data
}
