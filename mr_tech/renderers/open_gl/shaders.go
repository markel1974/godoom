package open_gl

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// IShader defines an interface for managing shader operations in a rendering pipeline.
// Setup initializes shader-specific configurations with the given width and height.
// SetupSamplers configures the required texture samplers for the shader program.
// Compile compiles the shader using provided assets and returns an error if compilation fails.
type IShader interface {
	Setup(width int32, height int32)

	SetupSamplers()

	Compile(a IAssets) error
}

// ShaderCompile compiles a shader of the given type using the provided source code and returns the shader or an error.
func ShaderCompile(source string, shaderType uint32) (uint32, error) {
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
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}

// ShaderCreateProgram links vertex and fragment shaders into a program, activates it, and cleans up shader objects.
func ShaderCreateProgram(vertexShader uint32, fragmentShader uint32) (uint32, error) {
	shaderProgram := gl.CreateProgram()
	gl.AttachShader(shaderProgram, vertexShader)
	gl.AttachShader(shaderProgram, fragmentShader)
	gl.LinkProgram(shaderProgram)
	var status int32
	gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		return 0, fmt.Errorf("failed to link shader prg")
	}
	gl.UseProgram(shaderProgram)
	gl.DeleteShader(fragmentShader)
	gl.DeleteShader(vertexShader)

	return shaderProgram, nil
}
