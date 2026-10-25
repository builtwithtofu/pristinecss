package pristinecss

import "math"

// CSS Color 4 binary search with local MINDE, JND 0.02 and epsilon 0.0001:
// https://www.w3.org/TR/css-color-4/#gamut-mapping
// Matrices: https://raw.githubusercontent.com/w3c/csswg-drafts/main/css-color-4/conversions.js
func gamutMap(lab [3]float64, p3 bool) (rgb [3]float64, mapped bool) {
	// Already-neutral endpoints need no fidelity loss, but return exact channels.
	if lab[0] >= 1 {
		return [3]float64{1, 1, 1}, lab != [3]float64{1, 0, 0}
	}
	if lab[0] <= 0 {
		return [3]float64{}, lab != [3]float64{}
	}
	rgb = oklabRGB(lab[0], lab[1], lab[2], p3)
	if rgbInGamut(rgb) {
		return rgb, false
	}
	const jnd, epsilon = .02, .0001
	clipped := clipRGB(rgb)
	if deltaOK(rgbOKLab(clipped, p3), lab) < jnd {
		return clipped, true
	}
	hue := math.Atan2(lab[2], lab[1])
	cos, sin := math.Cos(hue), math.Sin(hue)
	low, high := 0.0, min(math.Hypot(lab[1], lab[2]), math.MaxFloat64)
	minInGamut := true
	for high-low > epsilon {
		chroma := low + (high-low)/2
		current := [3]float64{lab[0], chroma * cos, chroma * sin}
		rgb = oklabRGB(current[0], current[1], current[2], p3)
		if minInGamut && rgbInGamut(rgb) {
			low = chroma
			continue
		}
		clipped = clipRGB(rgb)
		delta := deltaOK(rgbOKLab(clipped, p3), current)
		if delta < jnd {
			if jnd-delta < epsilon {
				return clipped, true
			}
			minInGamut = false
			low = chroma
		} else {
			high = chroma
		}
	}
	return clipped, true
}

func rgbInGamut(rgb [3]float64) bool {
	for _, c := range rgb {
		if !(c >= 0 && c <= 1) {
			return false
		}
	}
	return true
}

func clipRGB(rgb [3]float64) [3]float64 {
	for i, c := range rgb {
		rgb[i] = max(0, min(1, c))
	}
	return rgb
}

func deltaOK(a, b [3]float64) float64 {
	return math.Hypot(math.Hypot(a[0]-b[0], a[1]-b[1]), a[2]-b[2])
}

func oklabRGB(L, a, b float64, p3 bool) (rgb [3]float64) {
	l := L + .3963377773761749*a + .2158037573099136*b
	m := L - .1055613458156586*a - .0638541728258133*b
	s := L - .0894841775298119*a - 1.2914855480194092*b
	l, m, s = l*l*l, m*m*m, s*s*s
	x := 1.2268798758459243*l - .5578149944602171*m + .2813910456659647*s
	y := -.0405757452148008*l + 1.1122868032803170*m - .0717110580655164*s
	z := -.0763729366746601*l - .4214933324022432*m + 1.5869240198367816*s
	if p3 {
		rgb = [3]float64{446124.0/178915*x - 333277.0/357830*y - 72051.0/178915*z, -14852.0/17905*x + 63121.0/35810*y + 423.0/17905*z, 11844.0/330415*x - 50337.0/660830*y + 316169.0/330415*z}
	} else {
		rgb = [3]float64{12831.0/3959*x - 329.0/214*y - 1974.0/3959*z, -851781.0/878810*x + 1648619.0/878810*y + 36519.0/878810*z, 705.0/12673*x - 2585.0/12673*y + 705.0/667*z}
	}
	// Neutral OKLab has equal linear RGB channels in both D65 spaces.
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

func rgbOKLab(rgb [3]float64, p3 bool) [3]float64 {
	for i, c := range rgb {
		if math.Abs(c) <= .04045 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Copysign(math.Pow((math.Abs(c)+.055)/1.055, 2.4), c)
		}
	}
	r, g, b := rgb[0], rgb[1], rgb[2]
	var x, y, z float64
	if p3 {
		x = 608311.0/1250200*r + 189793.0/714400*g + 198249.0/1000160*b
		y = 35783.0/156275*r + 247089.0/357200*g + 198249.0/2500400*b
		z = 32229.0/714400*g + 5220557.0/5000800*b
	} else {
		x = 506752.0/1228815*r + 87881.0/245763*g + 12673.0/70218*b
		y = 87098.0/409605*r + 175762.0/245763*g + 12673.0/175545*b
		z = 7918.0/409605*r + 87881.0/737289*g + 1001167.0/1053270*b
	}
	l := math.Cbrt(.8190224379967030*x + .3619062600528904*y - .1288737815209879*z)
	m := math.Cbrt(.0329836539323885*x + .9292868615863434*y + .0361446663506424*z)
	s := math.Cbrt(.0481771893596242*x + .2642395317527308*y + .6335478284694309*z)
	return [3]float64{
		.2104542683093140*l + .7936177747023054*m - .0040720430116193*s,
		1.9779985324311684*l - 2.4285922420485799*m + .4505937096174110*s,
		.0259040424655478*l + .7827717124575296*m - .8086757549230774*s,
	}
}
