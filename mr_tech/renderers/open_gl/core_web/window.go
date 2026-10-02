//go:build js && wasm

package core_web

import (
	"syscall/js"
	"time"
)

type WindowConfig struct {
	Title      string
	Width      int
	Height     int
	Fullscreen bool
	VSync      bool
}

type XY struct {
	X float64
	Y float64
}

type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

type Button int

type Window struct {
	canvas js.Value
	gl     js.Value
	closed bool
	width  int
	height int
}

func NewGLWindow(cfg WindowConfig) (*Window, error) {
	document := js.Global().Get("document")
	canvas := document.Call("getElementById", "canvas")
	if canvas.IsUndefined() || canvas.IsNull() {
		canvas = document.Call("createElement", "canvas")
		canvas.Set("id", "canvas")
		document.Get("body").Call("appendChild", canvas)
	}

	canvas.Set("width", cfg.Width)
	canvas.Set("height", cfg.Height)

	// Richiedi il contesto WebGL2
	glArgs := js.Global().Get("Object").New()
	glArgs.Set("alpha", false)
	glArgs.Set("antialias", false)
	glArgs.Set("depth", true)
	glArgs.Set("stencil", true)

	gl := canvas.Call("getContext", "webgl2", glArgs)

	return &Window{
		canvas: canvas,
		gl:     gl,
		width:  cfg.Width,
		height: cfg.Height,
	}, nil
}

func (w *Window) Begin() {
	// ... reset input ...
}

func (w *Window) UpdateInputAndSwap() {
	// In Wasm il buffer swap e' gestito dal browser a fine frame (requestAnimationFrame).
	// Qui gestiamo l'aggiornamento dello stato dell'input.
}

func (w *Window) Closed() bool {
	return w.closed
}

func (w *Window) GetFramebufferSize() (int, int) {
	return w.width, w.height
}

// ... Gli altri metodi dell'input, clipboard, cursor, ecc. andranno mockati o implementati con addEventListener.
