package pristinecss

import "math"

func rgbInGamut(rgb [3]float64) bool {
	for _, c := range rgb {
		if !(c >= 0 && c <= 1) {
			return false
		}
	}
	return true
}

func oklabRGB(L, a, b float64) (rgb [3]float64) {
	l := L + .3963377773761749*a + .2158037573099136*b
	m := L - .1055613458156586*a - .0638541728258133*b
	s := L - .0894841775298119*a - 1.2914855480194092*b
	l, m, s = l*l*l, m*m*m, s*s*s
	x := 1.2268798758459243*l - .5578149944602171*m + .2813910456659647*s
	y := -.0405757452148008*l + 1.1122868032803170*m - .0717110580655164*s
	z := -.0763729366746601*l - .4214933324022432*m + 1.5869240198367816*s
	rgb = [3]float64{12831.0/3959*x - 329.0/214*y - 1974.0/3959*z, -851781.0/878810*x + 1648619.0/878810*y + 36519.0/878810*z, 705.0/12673*x - 2585.0/12673*y + 705.0/667*z}
	// Neutral OKLab has equal linear RGB channels.
	if a == 0 && b == 0 {
		rgb = [3]float64{l, l, l}
	}
	for i, c := range rgb {
		if math.Abs(c) <= .0031308 {
			rgb[i] = 12.92 * c
		} else {
			rgb[i] = math.Copysign(1.055*math.Pow(math.Abs(c), 1/2.4)-.055, c)
		}
	}
	return rgb
}
