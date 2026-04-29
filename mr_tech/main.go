package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/dungeon"
	"github.com/markel1974/godoom/mr_tech/generators/jedi"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/generators/script"
	"github.com/markel1974/godoom/mr_tech/generators/wad"
	"github.com/markel1974/godoom/mr_tech/generators/wolfstein"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl"
	"github.com/markel1974/godoom/mr_tech/renderers/software"
	"github.com/markel1974/godoom/mr_tech/version"
)

type IRender interface {
	Setup(engine *engine.Engine) error
	Start()
}

func main() {
	var cfg *config.Root
	var err error
	var showHelp bool
	var showVersion bool
	var softwareRender bool
	var full3d bool
	var mode int
	var level int
	var width int
	var height int
	var maxQueue int

	flag.BoolVar(&showHelp, "h", false, "show this help")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&softwareRender, "s", false, "enable software renderer")
	flag.BoolVar(&full3d, "d", false, "show this help")
	flag.IntVar(&mode, "m", 2, "mode 0 = legacy, 1 = Generate, 2 = Doom")
	flag.IntVar(&level, "l", 1, "level number")
	flag.IntVar(&width, "width", 640, "width")
	flag.IntVar(&height, "height", 480, "height")
	flag.IntVar(&maxQueue, "queue", 32, "max queue size")
	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version.AppName, version.AppVersion)
		return
	}

	switch mode {
	case 0:
		p := script.NewBuilder()
		cfg, err = p.Build(script.StubOld2)
	case 1:
		db := dungeon.NewBuilder()
		cfg, err = db.Build(level)
	case 2:
		wb := wolfstein.NewBuilder()
		cfg, err = wb.Build(level)
	case 3:
		wadFile := "resources" + string(os.PathSeparator) + "wad" + string(os.PathSeparator) + "DOOM.WAD"
		wb := wad.NewBuilder()
		cfg, err = wb.Build(wadFile, level)
	case 4:
		jFile := "resources" + string(os.PathSeparator) + "jedi"
		jf := jedi.NewBuilder()
		cfg, err = jf.Build(jFile, level)
	case 5:
		quakeFile := "resources" + string(os.PathSeparator) + "quake" + string(os.PathSeparator) + "PAK0.PAK"
		wb := quake.NewBuilder()
		cfg, err = wb.Setup(quakeFile, level)
	default:
		db := dungeon.NewBuilder()
		cfg, err = db.Build(level)
	}

	if err != nil {
		fmt.Println(err)
		return
	}
	if full3d {
		cfg.Calibration.Full3d = true
	}
	en := engine.NewEngine(maxQueue, 3.0)
	if err = en.Setup(cfg); err != nil {
		fmt.Println(err)
		return
	}

	var render IRender
	if softwareRender {
		render = software.NewRender(int32(width), int32(height))
	} else {
		render = open_gl.NewRender(int32(width), int32(height))
	}
	if err = render.Setup(en); err != nil {
		fmt.Println(err)
		return
	}
	render.Start()
}
