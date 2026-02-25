package renderers

const (
	HFov = 0.73
	VFov = 0.2

	NearZ    = 1e-4
	NearSide = 1e-5
	FarZ     = 5.0
	FarSide  = 20.0
)

func Yaw(y float64, z float64, yaw float64) float64 {
	return y + (z * yaw)
}

func ToRGB(rgb int, light float64) (r uint8, g uint8, b uint8) {
	fr := float64(uint8((rgb>>16)&255)) * light
	fg := float64(uint8((rgb>>8)&255)) * light
	fb := float64(uint8(rgb&255)) * light
	return uint8(fr), uint8(fg), uint8(fb)
}

/*
func ScreenCoordsToMapCoords(mapY float64, screenX float64, screenY float64, yaw float64, angleSin float64, angleCos float64, whereX float64, whereY float64, screenW float64, screenH float64) (float64, float64) {
	pVFov := vFov
	//TODO TEST
	pHFov := hFov * 0.78
	//pHFov := hFov
	z := (mapY * screenH * pVFov) / (((screenH / 2) - screenY) - (yaw * screenH * pVFov))
	x := z * ((screenW / 2) - screenX) / (screenW * pHFov)
	return RelativeToAbsoluteMap(x, z, angleSin, angleCos, whereX, whereY)
}

func RelativeToAbsoluteMap(x float64, z float64, angleSin float64, angleCos float64, whereX float64, whereY float64) (float64, float64) {
	rtx := (z * angleCos) + (x * angleSin)
	rtz := (z * angleSin) - (x * angleCos)
	x = rtx + whereX
	z = rtz + whereY
	return x, z
}
*/
