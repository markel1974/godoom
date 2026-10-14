package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/dungeon"
	"github.com/markel1974/godoom/mr_tech/generators/jedi"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/generators/script"
	"github.com/markel1974/godoom/mr_tech/generators/wad"
	"github.com/markel1974/godoom/mr_tech/generators/wolfstein"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/version"
)

type ModeDef struct {
	Name    string
	Target  string
	Builder func(res common.IFileSystem, target string, level int) (*config.Root, error)
}

func IntToBool(i int) bool {
	return i != 0
}

var _availableModes = []ModeDef{
	{
		Name:   "script",
		Target: "",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			p := script.NewBuilder()
			return p.Build(res, script.StubOld2)
		},
	},
	{
		Name:   "generate",
		Target: "",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			db := dungeon.NewBuilder()
			return db.Build(res, level)
		},
	},
	{
		Name:   "wolfstein",
		Target: "",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			p := wolfstein.NewBuilder()
			return p.Build(level)
		},
	},
	{
		Name:   "doom",
		Target: "resources" + string(os.PathSeparator) + "wad" + string(os.PathSeparator) + "DOOM.WAD",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			wb := wad.NewBuilder()
			return wb.Build(res, target, level)
		},
	},
	{
		Name:   "jedi",
		Target: "resources" + string(os.PathSeparator) + "jedi",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			jf := jedi.NewBuilder()
			return jf.Build(res, 1, target, level)
		},
	},
	{
		Name:   "outlaws",
		Target: "resources" + string(os.PathSeparator) + "outlaws",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			jf := jedi.NewBuilder()
			return jf.Build(res, 1, target, level)
		},
	},
	{
		Name:   "quake",
		Target: "resources" + string(os.PathSeparator) + "quake" + string(os.PathSeparator) + "pak0.pak",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "quake2",
		Target: "resources" + string(os.PathSeparator) + "quake2" + string(os.PathSeparator) + "pak0.pak",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "quake3",
		Target: "resources" + string(os.PathSeparator) + "quake3arena" + string(os.PathSeparator) + "pak0.pk3",
		Builder: func(res common.IFileSystem, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
}

func main() {
	var showHelp bool
	var showVersion bool
	var mode int
	var level int
	var width int
	var height int
	var fps int
	var enableShadows int
	var enableShadowLights int

	var helpModes string
	for idx, v := range _availableModes {
		helpModes += fmt.Sprintf("[%d = %s]", idx, v.Name)
	}

	flag.BoolVar(&showHelp, "h", false, "show this help")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.IntVar(&mode, "m", 2, helpModes)
	flag.IntVar(&level, "l", 1, "level number")
	flag.IntVar(&fps, "fps", 120, "fps")
	flag.IntVar(&width, "width", 640, "width")
	flag.IntVar(&height, "height", 480, "height")
	flag.IntVar(&enableShadows, "shadows", 1, "enable shadows")
	flag.IntVar(&enableShadowLights, "shadowLights", 1, "enable shadows lights")

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version.AppName, version.AppVersion)
		return
	}

	if mode < 0 || mode >= len(_availableModes) {
		fmt.Println("Invalid mode")
		return
	}

	app := NewApp()
	if err := app.Setup(width, height, fps); err != nil {
		fmt.Println("Error setting up application:", err)
		return
	}

	textures.SetFps(uint64(fps))
	physics.SetFps(uint64(fps))

	res := app.GetResources()
	target := app.UpdatePath(_availableModes[mode].Target)
	fn := _availableModes[mode].Builder
	cfg, err := fn(res, target, level)
	if err != nil {
		fmt.Println(err)
		return
	}

	en := engine.NewEngine()
	if err = en.Setup(cfg); err != nil {
		fmt.Println(err)
		return
	}

	cal := en.GetCalibration()
	cal.EnableShadows = IntToBool(enableShadows)
	cal.EnableShadowLights = IntToBool(enableShadowLights)

	ctx := app.GetContext()
	render := open_gl.NewRender(ctx)
	if err = render.Setup(en); err != nil {
		fmt.Println(err)
		return
	}
	render.Start()
	app.WaitExit()
}
