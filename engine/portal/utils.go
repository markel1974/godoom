package portal

func Yaw(y float64, z float64, yaw float64) float64 {
	return y + (z * yaw)
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
