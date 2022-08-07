package main

import (
	"fmt"
	"github.com/markel1974/godoom/engine/config"
	"github.com/markel1974/godoom/engine/world"
	"github.com/markel1974/godoom/pixels"
	"image/color"
	"math"
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
	mainSurface *pixels.PictureRGBA
	mainMatrix  pixels.Matrix
	mainSprite  *pixels.Sprite
	world       *World
	enableClear bool
	viewMode    int
	cfg         *config.Config
	stubIdx     int
}

func NewGame() *Game {
	return &Game{}
}

func (g *Game) Setup(c pixels.Vec) {
	var err error
	g.viewMode = -1
	g.enableClear = false //true
	if g.enableClear {
		fmt.Println("WIN CLEAR IS ENABLE - DISABLE WHEN COMPLETE!!!!!!!!!!")
	}

	g.mainSurface = pixels.NewPictureRGBA(pixels.R(float64(0), float64(0), float64(_W), float64(_H)))

	g.mainSprite = pixels.NewSprite()
	g.mainSprite.SetCached(pixels.CacheModeUpdate)
	g.mainSprite.Set(g.mainSurface, g.mainSurface.Bounds())
	g.mainMatrix = pixels.IM.Moved(c).Scaled(c, _scale)

	//g.cfg, err = legacy.ParseLegacyData(legacy.StubOld2)
	//wb := wad.NewBuilder()
	//wadFile := "resources" + string(os.PathSeparator) + "wad"+ string(os.PathSeparator) + "DOOM.WAD"
	//g.cfg, err = wb.Setup(wadFile, 1)
	g.cfg, err = world.Generate(16, 16)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	g.world = NewWorld(_W, _H, _MaxQueue, g.viewMode)
	if err := g.world.Setup(g.cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (g *Game) Run() {
	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(_W)*_scale, float64(_H)*_scale),
		VSync:       true,
		Undecorated: false,
		Smooth:      false,
	}

	win, err := pixels.NewGLWindow(cfg)
	if err != nil {
		panic(err)
	}

	g.Setup(win.Bounds().Center())

	var currentTimer float64
	var lastTimer float64
	mouseConnected := true

	//text := pixel.NewText(pixel.V(10, 10), pixel.Atlas7x13)
	//_, _ = text.WriteString("test")

	for !win.Closed() {
		currentTimer = pixels.GLGetTime()
		if currentTimer-lastTimer >= 1.0/framerate {
			lastTimer = currentTimer
			g.Update(win)
		}

		if mouseConnected && win.MouseInsideWindow() {
			mousePos := win.MousePosition()
			mousePrevPos := win.MousePreviousPosition()
			if mousePos.X != mousePrevPos.X || mousePos.Y != mousePrevPos.Y {
				mouseX := mousePos.X - mousePrevPos.X
				mouseY := mousePos.Y - mousePrevPos.Y
				g.world.DoPlayerMouseMove(mouseX, mouseY)
			}
		}

		var up, down, left, right, slow bool

		scroll := win.MouseScroll()
		if scroll.Y != 0 {
			if scroll.Y > 0 {
				//g.world.DoZoom(1)
				up = true
			} else {
				//g.world.DoZoom(-1)
				down = true
			}
		}

		for v := range win.KeysPressed() {
			switch v {
			case pixels.KeyEscape:
				return
			case pixels.KeyUp:
				up = true
			case pixels.KeyW:
				up = true
				slow = true
			case pixels.KeyDown:
				down = true
			case pixels.KeyS:
				down = true
				slow = true
			case pixels.KeyLeft:
				left = true
			case pixels.KeyRight:
				right = true
			}
		}
		g.world.DoPlayerMoves(up, down, left, right, slow)
		if win.JustPressed(pixels.KeyC) {
			g.enableClear = true
			g.world.DebugMoveSectorToggle()
		}
		if win.JustPressed(pixels.KeyZ) {
			g.world.DebugMoveSector(true)
		}
		if win.JustPressed(pixels.KeyX) {
			g.world.DebugMoveSector(false)
		}
		if win.JustPressed(pixels.KeyTab) || win.Pressed(pixels.MouseButton2) {
			g.world.DoPlayerDuckingToggle()
		}
		if win.JustPressed(pixels.KeySpace) || win.Pressed(pixels.MouseButton1) {
			g.world.DoPlayerJump()
		}
		if win.JustPressed(pixels.Key0) {
			g.stubIdx++
		}
		if win.JustPressed(pixels.Key9) {
			g.stubIdx--
		}
		if win.JustPressed(pixels.KeyM) {
			mouseConnected = !mouseConnected
		}

		win.Update()

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
	//drawStub(g.mainSurface, g.cfg.Sectors[g.stubIdx])
	g.mainSprite.Draw(win, g.mainMatrix)
}

func main() {
	g := NewGame()
	pixels.GLRun(g.Run)
}


func drawStub(surface *pixels.PictureRGBA, sector *config.Sector) {
	t  := make([]XYZ, len(sector.Neighbors))
	maxX := 0.0
	maxY := 0.0
	for idx := 0; idx < len(sector.Neighbors); idx++ {
		x := math.Abs(sector.Neighbors[idx].X)
		y := math.Abs(sector.Neighbors[idx].Y)
		if x > maxX { maxX = x }
		if y > maxY { maxY = y }
	}

	if maxX > 300 { maxX = -(300 - maxX)}
	if maxY > 400 { maxY = -(400 - maxY)}

	if maxX < 30 { maxX -= 30}
	if maxY < 30 { maxY -= 30}

	for idx := 0; idx < len(sector.Neighbors); idx++ {
		x := math.Abs(sector.Neighbors[idx].X) - maxX
		y := math.Abs(sector.Neighbors[idx].Y) - maxY
		t[idx].X = x
		t[idx].Y = y
	}

	//t = t[0:5]
	dp := NewDrawPolygon(640, 480)
	dp.Setup(surface, t, len(t), 0x00ff00, 1.0, 1.0)
	dp.DrawPoints(10)
	dp.color = 0xff0000
	dp.DrawLines()
}