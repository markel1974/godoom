package executor

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Shader struct {
	program    binder
	vertexFmt  AttrFormat
	uniformFmt AttrFormat
	uniformLoc []int32
}

func NewShader(vertexFmt, uniformFmt AttrFormat, vertexShader, fragmentShader string) (*Shader, error) {
	shader := &Shader{
		program: binder{
			restoreLoc: gl.CURRENT_PROGRAM,
			bindFunc: func(obj uint32) {
				gl.UseProgram(obj)
			},
		},
		vertexFmt:  vertexFmt,
		uniformFmt: uniformFmt,
		uniformLoc: make([]int32, len(uniformFmt)),
	}

	var vShader, fShader uint32

	{
		vShader = gl.CreateShader(gl.VERTEX_SHADER)
		src, free := gl.Strs(vertexShader)
		defer free()
		length := int32(len(vertexShader))
		gl.ShaderSource(vShader, 1, src, &length)
		gl.CompileShader(vShader)

		var success int32
		gl.GetShaderiv(vShader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(vShader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(vShader, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error compiling vertex shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(vShader)
	}

	{
		fShader = gl.CreateShader(gl.FRAGMENT_SHADER)
		src, free := gl.Strs(fragmentShader)
		defer free()
		length := int32(len(fragmentShader))
		gl.ShaderSource(fShader, 1, src, &length)
		gl.CompileShader(fShader)

		var success int32
		gl.GetShaderiv(fShader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(fShader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(fShader, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error compiling fragment shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(fShader)
	}

	{
		shader.program.obj = gl.CreateProgram()
		gl.AttachShader(shader.program.obj, vShader)
		gl.AttachShader(shader.program.obj, fShader)
		gl.LinkProgram(shader.program.obj)

		var success int32
		gl.GetProgramiv(shader.program.obj, gl.LINK_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetProgramiv(shader.program.obj, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetProgramInfoLog(shader.program.obj, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error linking shader program: %s", string(infoLog))
		}
	}

	for i, uniform := range uniformFmt {
		loc := gl.GetUniformLocation(shader.program.obj, gl.Str(uniform.Name+"\x00"))
		shader.uniformLoc[i] = loc
	}

	runtime.SetFinalizer(shader, (*Shader).delete)

	return shader, nil
}

func (s *Shader) delete() {
	Thread.Post(func() {
		gl.DeleteProgram(s.program.obj)
	})
}

func (s *Shader) ID() uint32 {
	return s.program.obj
}

func (s *Shader) VertexFormat() AttrFormat {
	return s.vertexFmt
}

func (s *Shader) UniformFormat() AttrFormat {
	return s.uniformFmt
}

func (s *Shader) SetUniformAttr(uniform int, value interface{}) (bool, error) {
	if s.uniformLoc[uniform] < 0 {
		return false, nil
	}

	switch s.uniformFmt[uniform].Type {
	case Int:
		value := value.(int32)
		gl.Uniform1iv(s.uniformLoc[uniform], 1, &value)
	case Float:
		value := value.(float32)
		gl.Uniform1fv(s.uniformLoc[uniform], 1, &value)
	case Vec2:
		value := value.(mgl32.Vec2)
		gl.Uniform2fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec3:
		value := value.(mgl32.Vec3)
		gl.Uniform3fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec4:
		value := value.(mgl32.Vec4)
		gl.Uniform4fv(s.uniformLoc[uniform], 1, &value[0])
	case Mat2:
		value := value.(mgl32.Mat2)
		gl.UniformMatrix2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat23:
		value := value.(mgl32.Mat2x3)
		gl.UniformMatrix2x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat24:
		value := value.(mgl32.Mat2x4)
		gl.UniformMatrix2x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat3:
		value := value.(mgl32.Mat3)
		gl.UniformMatrix3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat32:
		value := value.(mgl32.Mat3x2)
		gl.UniformMatrix3x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat34:
		value := value.(mgl32.Mat3x4)
		gl.UniformMatrix3x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat4:
		value := value.(mgl32.Mat4)
		gl.UniformMatrix4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat42:
		value := value.(mgl32.Mat4x2)
		gl.UniformMatrix4x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat43:
		value := value.(mgl32.Mat4x3)
		gl.UniformMatrix4x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	default:
		return false, errors.New("set uniform attr: invalid attribute type")
	}
	return true, nil
}

func (s *Shader) Begin() {
	s.program.bind()
}

func (s *Shader) End() {
	s.program.restore()
}
