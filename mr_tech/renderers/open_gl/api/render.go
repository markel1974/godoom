package api

type IRender interface {
	RenderPrepare() error

	RenderStart(fbW, fbH, winW, winH int, advance bool)

	RenderPlayerMouseMove(mouseX float64, mouseY float64)

	RenderPlayerMoves(impulse float64, up bool, down bool, left bool, right bool)

	RenderToggleShadows()

	RenderEnableClear()

	RenderPlayerThrow()

	RenderPlayerFire()

	RenderPlayerDuckingToggle()

	RenderPlayerJump(multi bool)

	RenderIncreaseFlashFactor()

	RenderDecreaseFlashFactor()
}
