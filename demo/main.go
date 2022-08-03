package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"image"
	"image/draw"
	_ "image/png"
	"markel/mgl/pixels"
	"markel/mgl/pixels/executor"
	"os"
	"time"
)

var vertexShader = `
#version 330 core

in vec2 position;
in vec2 texture;

out vec2 Texture;

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	Texture = texture;
}
`

var fragmentShader = `
#version 330 core

in vec2 Texture;

out vec4 color;

uniform sampler2D tex;

void main() {
	color = texture(tex, Texture);
}
`

func loadImage(path string) (*image.NRGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba, nil
}

func run() {
	var win *glfw.Window
	img, err := loadImage("celebrate.png")
	if err != nil {
		panic(err)
	}

	defer func() {
		executor.Thread.Call(func() {
			glfw.Terminate()
		})
	}()

	executor.Thread.Call(func() {
		_ = glfw.Init()

		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, glfw.True)

		var err error

		win, err = glfw.CreateWindow(560, 697, "mgl", nil, nil)
		if err != nil {
			panic(err)
		}

		win.MakeContextCurrent()

		executor.Init()
	})

	var shader *executor.Shader
	var texture *executor.Texture
	var slice *executor.VertexSlice

	executor.Thread.Call(func() {
		var err error
		var vertexFormat = executor.AttrFormat{{Name: "position", Type: executor.Vec2}, {Name: "texture", Type: executor.Vec2}}

		shader, err = executor.NewShader(vertexFormat, executor.AttrFormat{}, vertexShader, fragmentShader)
		if err != nil {
			panic(err)
		}

		texture = executor.NewTexture(img.Bounds().Dx(), img.Bounds().Dy(), true, img.Pix)

		slice = executor.MakeVertexSlice(shader, 6, 6)

		slice.Begin()

		slice.SetVertexData([]float32{
			-1, -1, 0, 1,
			+1, -1, 1, 1,
			+1, +1, 1, 0,
			-1, -1, 0, 1,
			+1, +1, 1, 0,
			-1, +1, 0, 0,
		})

		slice.End()
	})

	loop := true
	for loop {
		time.Sleep(100 * time.Millisecond)

		executor.Thread.Call(func() {
			if win.ShouldClose() {
				loop = false
			}

			executor.Clear(1, 1, 1, 1)

			shader.Begin()
			texture.Begin()
			slice.Begin()
			slice.Draw()
			slice.End()
			texture.End()
			shader.End()

			win.SwapBuffers()
			glfw.PollEvents()
		})
	}
}

func main() {
	pixels.GLRun(run)
}
