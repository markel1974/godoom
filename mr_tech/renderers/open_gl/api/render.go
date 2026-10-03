package api

type IRender interface {
	RenderSetup() error

	RenderStart(fbW, fbH int)

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
