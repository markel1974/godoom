package shaders

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// IAssets represents an interface for handling asset operations such as resolving base paths and reading asset files.
type IAssets interface {
	BasePath(vPath string) string

	Read(p string) ([]byte, error)

	ReadMulti(a string, b string) ([]byte, []byte, error)
}

// ShaderCompile compiles a shader from source code and returns the shader ID or an error if compilation fails.
func ShaderCompile(id string, source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	cSources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, cSources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader %s: %v", id, log)
	}
	return shader, nil
}

// ShaderCreateProgram links a vertex and fragment shader into a shader program, validates it, and returns the program ID.
func ShaderCreateProgram(id string, vertexShader uint32, fragmentShader uint32) (uint32, error) {
	shaderProgram := gl.CreateProgram()
	gl.AttachShader(shaderProgram, vertexShader)
	gl.AttachShader(shaderProgram, fragmentShader)
	gl.LinkProgram(shaderProgram)
	var status int32
	gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		return 0, fmt.Errorf("failed to link shader prg: %s", id)
	}
	gl.UseProgram(shaderProgram)
	gl.DeleteShader(fragmentShader)
	gl.DeleteShader(vertexShader)

	return shaderProgram, nil
}
