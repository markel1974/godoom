package geometry

// E2 function replace `math.Pow(x,2.0)`
func PowE2(x float64) float64 {
	return x * x
}

// E3 function replace `math.Pow(x,3.0)`
func PowE3(x float64) float64 {
	return x * x * x
}

// E4 function replace `math.Pow(x,4.0)`
func PowE4(x float64) float64 {
	x2 := x * x
	return x2 * x2
}

// En function replace `math.Pow(x,e)`
func PowEn(x float64, e int) float64 {
	var isNegative bool
	if e < 0 {
		isNegative = true
		e = -e
	}
	var (
		i uint8
		n uint8   = uint8(e)
		r float64 = 1.0

		// variables for iterations
		tmp float64
		c   uint8
	)

	for 1 < n {
		tmp = x
		c = 1
		for c <= n/2 {
			tmp *= tmp
			c *= 2
		}
		n = n - c
		r *= tmp
	}

	for i = 0; i < n; i++ {
		r *= x
	}
	if isNegative {
		return 1 / r
	}
	return r
}
