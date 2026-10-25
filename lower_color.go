package pristinecss

import (
	"bytes"
	"math"
	"strconv"
)

func (w *lowering) color(raw []byte, lo, hi uint32, nameLen int) {
	values, ok := colorValues(raw[nameLen+1:len(raw)-1], w.cap == CapOklch)
	if !ok {
		w.cannot(lo, hi, "cannot lower colour: needs literal numeric components (no none/calc/var or relative syntax)")
		return
	}
	lab := [3]float64{values[0], values[1], values[2]}
	for _, component := range lab {
		if math.IsNaN(component) {
			w.cannot(lo, hi, "cannot lower colour: numeric conversion overflow")
			return
		}
	}
	p3 := w.target.Has(CapDisplayP3)
	rgb, mapped := gamutMap(lab, p3)
	if mapped {
		if w.fidelity < Close {
			w.fidelityGap = true
			w.cannot(lo, hi, "gamut mapping requires Close fidelity")
			return
		}
		gamut := "srgb"
		if p3 {
			gamut = "display-p3"
		}
		w.mappings = append(w.mappings, Mapping{Lo: lo, Hi: hi, Cap: w.cap, Gamut: gamut})
	}
	start := len(w.text)
	if p3 {
		w.text = append(w.text, "color(display-p3"...)
		for _, c := range rgb {
			w.text = append(w.text, ' ')
			w.text = strconv.AppendFloat(w.text, c, 'f', 6, 64)
		}
		if values[3] != 1 {
			w.text = append(w.text, " / "...)
			w.text = strconv.AppendFloat(w.text, values[3], 'f', -1, 64)
		}
	} else {
		w.text = append(w.text, "rgb("...)
		for i, v := range rgb {
			if i != 0 {
				w.text = append(w.text, ',')
			}
			w.text = strconv.AppendInt(w.text, int64(math.Round(v*255)), 10)
		}
		// Legacy comma syntax needs the rgba alias for translucent colours.
		if values[3] != 1 {
			w.text = append(w.text, 0)
			copy(w.text[start+4:], w.text[start+3:len(w.text)-1])
			w.text[start+3] = 'a'
			w.text = append(w.text, ',')
			w.text = strconv.AppendFloat(w.text, values[3], 'f', -1, 64)
		}
	}
	w.text = append(w.text, ')')
	w.edits = append(w.edits, lowerEdit{lo, hi, start, len(w.text)})
}

func colorValues(b []byte, polar bool) ([4]float64, bool) {
	v := [4]float64{0, 0, 0, 1}
	p := 0
	skip := func() bool {
		before := p
		for p < len(b) {
			if isCSSSpace(b[p]) {
				p++
				continue
			}
			if p+1 < len(b) && b[p] == '/' && b[p+1] == '*' {
				end := bytes.Index(b[p+2:], []byte("*/"))
				if end < 0 {
					return false
				}
				p += end + 4
				continue
			}
			break
		}
		return p > before
	}
	for i := range 4 {
		separated := skip()
		if i == 3 {
			if p == len(b) {
				break
			}
			if b[p] != '/' {
				return v, false
			}
			p++
			skip()
		} else if i > 0 && !separated {
			return v, false
		}
		start := p
		if p < len(b) && (b[p] == '+' || b[p] == '-') {
			p++
		}
		digits := 0
		for p < len(b) && b[p] >= '0' && b[p] <= '9' {
			p++
			digits++
		}
		if p < len(b) && b[p] == '.' {
			p++
			for p < len(b) && b[p] >= '0' && b[p] <= '9' {
				p++
				digits++
			}
		}
		if digits == 0 {
			return v, false
		}
		if p < len(b) && (b[p] == 'e' || b[p] == 'E') {
			p++
			if p < len(b) && (b[p] == '+' || b[p] == '-') {
				p++
			}
			for p < len(b) && b[p] >= '0' && b[p] <= '9' {
				p++
			}
		}
		x, err := strconv.ParseFloat(string(b[start:p]), 64)
		if err != nil {
			return v, false
		}
		if p < len(b) && b[p] == '%' {
			if polar && i == 2 {
				return v, false
			}
			p++
			scale := .01
			if i == 1 || i == 2 {
				scale = .004
			}
			x *= scale
		} else if polar && i == 2 {
			unitStart := p
			for p < len(b) && ((b[p] >= 'a' && b[p] <= 'z') || (b[p] >= 'A' && b[p] <= 'Z')) {
				p++
			}
			unit := b[unitStart:p]
			switch {
			case len(unit) == 0, bytes.EqualFold(unit, []byte("deg")):
			case bytes.EqualFold(unit, []byte("grad")):
				x *= .9
			case bytes.EqualFold(unit, []byte("rad")):
				x *= 180 / math.Pi
			case bytes.EqualFold(unit, []byte("turn")):
				x *= 360
			default:
				return v, false
			}
		}
		v[i] = x
	}
	skip()
	if p != len(b) {
		return v, false
	}
	v[0] = max(0, min(1, v[0]))
	v[3] = max(0, min(1, v[3]))
	if polar {
		c := max(0, v[1])
		h := math.Mod(v[2], 360) * math.Pi / 180
		v[1], v[2] = c*math.Cos(h), c*math.Sin(h)
	}
	return v, true
}
