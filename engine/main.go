package main

import (
	"fmt"
	"github.com/markel1974/godoom/engine/geometry"
	"github.com/markel1974/godoom/engine/legacy"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/wad"
	"github.com/markel1974/godoom/engine/world"
	"github.com/markel1974/godoom/pixels"
	"image/color"
	"os"
)

const (
	framerate = 30
	_W        = 640 //1980 / 2
	_H        = 480 //1080 / 2
	_MaxQueue = 32
	_scale    = 1
)


type Game struct {
	win        *pixels.GLWindow
	mainSurface *pixels.PictureRGBA
	mainMatrix  pixels.Matrix
	mainSprite  *pixels.Sprite
	world       *World
	enableClear bool
	viewMode    int
	cfg         *model.Input
}

func NewGame() *Game {
	return &Game{}
}

func (g *Game) Setup() {
	var err error
	g.viewMode = 1
	g.enableClear = true //true
	m := 2

	switch m {
	case 0:
		g.cfg, err = legacy.ParseLegacyData(legacy.StubOld2)
	case 1:
		g.cfg, err = world.Generate(16, 16)
	case 2:
		wb := wad.NewBuilder()
		wadFile := "resources" + string(os.PathSeparator) + "wad"+ string(os.PathSeparator) + "DOOM.WAD"
		g.cfg, err = wb.Setup(wadFile, 1)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	if g.enableClear {
		fmt.Println("WIN CLEAR IS ENABLE - DISABLE WHEN COMPLETE!!!!!!!!!!")
	}

	g.world = NewWorld(_W, _H, _MaxQueue, g.viewMode)
	if err := g.world.Setup(g.cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(_W)*_scale, float64(_H)*_scale),
		VSync:       true,
		Undecorated: false,
		Smooth:      false,
	}

	g.win, err = pixels.NewGLWindow(cfg)
	if err != nil { panic(err) }
	center := g.win.Bounds().Center()

	g.mainSurface = pixels.NewPictureRGBA(pixels.R(float64(0), float64(0), float64(_W), float64(_H)))

	g.mainSprite = pixels.NewSprite()
	g.mainSprite.SetCached(pixels.CacheModeUpdate)
	g.mainSprite.Set(g.mainSurface, g.mainSurface.Bounds())
	g.mainMatrix = pixels.IM.Moved(center).Scaled(center, _scale)
}

func (g *Game) Run() {
	g.Setup()

	var currentTimer float64
	var lastTimer float64
	mouseConnected := true

	//text := pixel.NewText(pixel.V(10, 10), pixel.Atlas7x13)
	//_, _ = text.WriteString("test")

	for !g.win.Closed() {
		currentTimer = pixels.GLGetTime()
		if currentTimer - lastTimer >= 1.0 / framerate {
			lastTimer = currentTimer
			g.Update(g.win)
		}

		if mouseConnected && g.win.MouseInsideWindow() {
			mousePos := g.win.MousePosition()
			mousePrevPos := g.win.MousePreviousPosition()
			if mousePos.X != mousePrevPos.X || mousePos.Y != mousePrevPos.Y {
				mouseX := mousePos.X - mousePrevPos.X
				mouseY := mousePos.Y - mousePrevPos.Y
				g.world.DoPlayerMouseMove(mouseX, mouseY)
			}
		}

		var up, down, left, right, slow bool

		scroll := g.win.MouseScroll()
		if scroll.Y != 0 {
			if scroll.Y > 0 {
				//g.world.DoZoom(1)
				up = true
			} else {
				//g.world.DoZoom(-1)
				down = true
			}
		}

		for v := range g.win.KeysPressed() {
			switch v {
				case pixels.KeyEscape: return
				case pixels.KeyUp: up = true
				case pixels.KeyW: up = true; slow = true
				case pixels.KeyDown: down = true
				case pixels.KeyS: down = true; slow = true
				case pixels.KeyLeft: left = true
				case pixels.KeyRight: right = true
				case pixels.KeyV: g.world.DebugMoveSector(true)
				case pixels.KeyB: g.world.DebugMoveSector(false)
			}
		}
		g.world.DoPlayerMoves(up, down, left, right, slow)
		if g.win.JustPressed(pixels.KeyC) {
			g.enableClear = true
			g.world.DebugMoveSectorToggle()
		}
		if g.win.JustPressed(pixels.KeyZ) {
			g.world.DebugMoveSector(true)
		}
		if g.win.JustPressed(pixels.KeyX) {
			g.world.DebugMoveSector(false)
		}
		if g.win.JustPressed(pixels.KeyTab) || g.win.Pressed(pixels.MouseButton2) {
			g.world.DoPlayerDuckingToggle()
		}
		if g.win.JustPressed(pixels.KeySpace) || g.win.Pressed(pixels.MouseButton1) {
			g.world.DoPlayerJump()
		}
		if g.win.JustPressed(pixels.Key8) {
			g.world.DoDebug(0)
		}
		if g.win.JustPressed(pixels.Key0) {
			g.world.DoDebug(1)
		}
		if g.win.JustPressed(pixels.Key9) {
			g.world.DoDebug(-1)
		}
		if g.win.JustPressed(pixels.KeyM) {
			mouseConnected = !mouseConnected
		}
		g.win.Update()
		//text.Draw(win, g.mainMatrix)
	}
}

func (g *Game) Update(win *pixels.GLWindow) {
	if g.enableClear {
		win.Clear(color.Black)
		g.mainSurface = pixels.NewPictureRGBA(pixels.R(float64(0), float64(0), float64(_W), float64(_H)))
		g.mainSprite.Set(g.mainSurface, g.mainSurface.Bounds())
	}
	g.world.Update(g.mainSurface)
	//g.drawStub(g.mainSurface, g.stubIdx)
	g.mainSprite.Draw(win, g.mainMatrix)
}

func main() {

	//geometryTest()

	g := NewGame()
	pixels.GLRun(g.Run)
}


func geometryTest() (model.XY, model.XY, bool) {
	//refStart := geometry.Point{X: 0, Y: 10}
	//refEnd := geometry.Point{X: 20, Y: 10}
	//currStart := geometry.Point{X: 5, Y: 10}
	//currEnd := geometry.Point{X: 10, Y: 10}

	/*
	refStart := geometry.Point{X: 0, Y: 5}
	refEnd := geometry.Point{X: 5, Y: 5}

	currStart := geometry.Point{X: 1, Y: 5}
	currEnd := geometry.Point{X: 2, Y: 5}
	*/



	//BASE: 1664 2552 1664 2432
	refStart := geometry.Point{X: 1664, Y: 2552}
	refEnd := geometry.Point{X: 1664, Y: 2432}

	//0 - 1664 2552 1664 2432 real: 1664 2432 1664 2560
	currStart := geometry.Point{X: 1664, Y: 2432}
	currEnd := geometry.Point{X: 1664, Y: 2560}

	//15 - 1664 2552 1664 2432 real: 1664 2552 1664 2312 [ERRORE]
	//currStart := geometry.Point{X: 1664, Y: 2552}
	//currEnd := geometry.Point{X: 1664, Y: 2312}

	var p1[]model.XY
	if _, sa0, _ := geometry.PointLine(refStart, currStart, currEnd);
		sa0.Has(geometry.OnPoint0Segment) || sa0.Has(geometry.OnPoint1Segment) {
		//TODO RESIZE
		p1 = append(p1, model.XY{X: refStart.X, Y: refStart.Y})
	}
	if _, sb0, _ := geometry.PointLine(refEnd, currStart, currEnd);
		sb0.Has(geometry.OnPoint0Segment) || sb0.Has(geometry.OnPoint1Segment) {
		//TODO RESIZE
		p1 = append(p1, model.XY{X: refStart.X, Y: refStart.Y})
	}
	if len(p1) == 2 {
		fmt.Println("OK", p1[0], p1[1])
		os.Exit(-1)
		return p1[0], p1[1], true
	}
	var p2[]model.XY
	if _, sc0, _ := geometry.PointLine(currStart, refStart, refEnd);
		sc0.Has(geometry.OnPoint0Segment) || sc0.Has(geometry.OnPoint1Segment) {
		//TODO RESIZE
		p2 = append(p2, model.XY{X: currStart.X, Y: currStart.Y})
	}
	if _, sd0, _ := geometry.PointLine(currEnd, refStart, refEnd);
		sd0.Has(geometry.OnPoint0Segment) || sd0.Has(geometry.OnPoint1Segment) {
		//TODO RESIZE
		p2 = append(p2, model.XY{X: currEnd.X, Y: currEnd.Y})
	}
	if len(p2) >= 2 {
		fmt.Println("OK", p2[0], p2[1])
		os.Exit(-1)
		return p2[0], p2[1], true
	}

	if len(p1) > 0 && len(p2) > 0 {
		fmt.Println("OK", p1[0], p2[0])
		os.Exit(-1)
		return p1[0], p2[0], true
	}

	fmt.Println("NOK")
	os.Exit(-1)

	return model.XY{}, model.XY{}, false
}