package pristinecss

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

// MINDE reference results (not the three-algorithm averages) from WPT:
// https://github.com/web-platform-tests/wpt/blob/master/css/css-color/canvas-srgb-gamut-mapping.html
// https://github.com/web-platform-tests/wpt/blob/master/css/css-color/canvas-display-p3-gamut-mapping.html
func TestGamutMappingReferenceVectors(t *testing.T) {
	for _, tc := range []struct {
		l, c, h float64
		p3      bool
		want    [3]float64 // 8-bit channel scale, before rounding.
	}{
		{.9, .35, 0, false, [3]float64{255, 200.23, 219.68}},
		{.8, .35, 30, false, [3]float64{255, 153.55, 135.55}},
		{.8, .35, 0, false, [3]float64{255, 145.71, 185.67}},
		{.8, .35, 300, false, [3]float64{204.30, 167.06, 255}},
		{.9, .3, 240, false, [3]float64{175.12, 229.75, 255}},
		{.4, .35, 210, false, [3]float64{0, 83.62, 97.90}},
		{.6, .35, 60, false, [3]float64{195.29, 98.41, 0}},
		{.4, .35, 90, false, [3]float64{92.60, 67.69, 0}},
		{.5, .35, 210, false, [3]float64{0, 114.49, 132.39}},
		{.5, .3, 60, false, [3]float64{153.88, 74.54, 0}},
		{.9, .3, 300, false, [3]float64{229.95, 210.38, 255}},
		{.9, .3, 240, true, [3]float64{183.85, 228.53, 255}},
		{.6, .35, 60, true, [3]float64{188.72, 98.96, 0}},
		{.9, .35, 30, true, [3]float64{255, 202.78, 189.01}},
		{.9, .35, 300, true, [3]float64{227.04, 210.11, 255}},
		{.5, .35, 60, true, [3]float64{148.39, 75.09, 0}},
		{.8, .35, 300, true, [3]float64{199.17, 166.21, 255}},
		{.8, .35, 30, true, [3]float64{255, 148.92, 127.06}},
		{.3, .35, 210, true, [3]float64{0, 54.75, 68.49}},
		{.3, .3, 90, true, [3]float64{59.90, 42.83, 0}},
		{.3, .35, 240, true, [3]float64{0, 47.44, 93.90}},
		{.8, .35, 0, true, [3]float64{255, 138.44, 183.70}},
		{.5, .35, 210, true, [3]float64{0, 115.10, 138.38}},
		{.9, .25, 60, true, [3]float64{255, 207.43, 153.66}},
		{.9, .35, 270, true, [3]float64{205.94, 219.95, 255}},
		{.4, .35, 90, true, [3]float64{90.88, 67.87, 0}},
	} {
		t.Run(fmt.Sprintf("%g_%g_%g_p3=%v", tc.l, tc.c, tc.h, tc.p3), func(t *testing.T) {
			h := tc.h * math.Pi / 180
			lab := [3]float64{tc.l, tc.c * math.Cos(h), tc.c * math.Sin(h)}
			got, mapped := gamutMap(lab, tc.p3)
			if !mapped || !rgbInGamut(got) {
				t.Fatalf("not gamut mapped: %v, %v", got, mapped)
			}
			for i, want := range tc.want {
				if math.Abs(got[i]*255-want) > .02 {
					t.Fatalf("channel %d = %.5f, want %.2f ± .02", i, got[i]*255, want)
				}
			}
			// Also exercise the public builder and serializer against these vectors.
			target := Browser(Chrome, "49")
			if tc.p3 {
				target = Browser(Safari, "12")
			}
			src := []byte(fmt.Sprintf("a{color:oklch(%g %g %g)}", tc.l, tc.c, tc.h))
			s, report := buildLower(t, src, Options{Lower: []Target{{Caps: target}}})
			if len(report.Gaps) != 0 || len(report.Mappings) != 1 || len(report.Changes) != 0 {
				t.Fatalf("mapping report: %+v", report)
			}
			if !tc.p3 {
				want := fmt.Sprintf("a{color:rgb(%d,%d,%d)}", int(math.Round(tc.want[0])), int(math.Round(tc.want[1])), int(math.Round(tc.want[2])))
				if out := string(s.CSS(nil, target)); out != want {
					t.Fatalf("got %s; want %s", out, want)
				}
			} else {
				out := string(s.CSS(nil, target))
				coords := strings.TrimSuffix(strings.TrimPrefix(out, "a{color:color(display-p3 "), ")}")
				fields := strings.Fields(coords)
				if len(fields) != 3 {
					t.Fatalf("not display-p3: %s", out)
				}
				for i, field := range fields {
					value, err := strconv.ParseFloat(field, 64)
					if err != nil || math.Abs(value*255-tc.want[i]) > .02 {
						t.Fatalf("P3 channel %d: %s; want %.2f on 8-bit scale", i, field, tc.want[i])
					}
				}
			}
		})
	}
}

func TestLowerP3MiddleStep(t *testing.T) {
	const source = "a{color:oklch(.7 .2 30 / .4)}"
	old, p3 := Browser(Chrome, "49"), Browser(Safari, "12")
	s, report := buildLower(t, []byte(source), Options{Lower: []Target{{Caps: old}, {Caps: p3}, {Caps: old}}})
	if report.Variants != 3 || len(report.Gaps) != 0 || len(report.Mappings) != 1 || len(report.Changes) != 0 {
		t.Fatalf("report: %+v", report)
	}
	mapping := report.Mappings[0]
	if mapping.Target != (Target{Caps: old}) || mapping.Gamut != "srgb" || source[mapping.Lo:mapping.Hi] != "oklch(.7 .2 30 / .4)" {
		t.Fatalf("mapping lost target or position: %+v", mapping)
	}
	wide := string(s.CSS(nil, p3))
	narrow := string(s.CSS(nil, old))
	if !strings.Contains(wide, "color(display-p3 ") || !strings.Contains(wide, "/ 0.4") || !strings.Contains(narrow, "rgba(") {
		t.Fatalf("middle step: p3=%s, srgb=%s", wide, narrow)
	}
	if string(s.CSS(nil, p3, Pretty)) == string(s.CSS(nil, old, Pretty)) || s.Key(p3) == s.Key(old) {
		t.Fatal("distinct gamuts collapsed")
	}
	if string(s.CSS(nil, Modern)) != minCSS(source) || string(s.CSS(nil, Caps{})) != narrow || s.Key(Caps{}) != s.Key(old) {
		t.Fatal("modern output or fallback changed")
	}
	for _, caps := range []Caps{old, p3, Modern} {
		if _, errs := Parse(s.CSS(nil, caps)); len(errs) != 0 {
			t.Fatal(errs)
		}
	}
	withGap := []byte("a{color:oklch(.7 .2 30 / .4);background:oklch(var(--l) .2 30)}")
	s, report = buildLower(t, withGap, Options{Lower: []Target{{Caps: old}, {Caps: p3}}})
	if len(report.Gaps) != 2 || len(report.Mappings) != 1 || !strings.Contains(string(s.CSS(nil, p3)), "color(display-p3 ") {
		t.Fatalf("unresolved colour vetoed P3 lowering: %s, %+v", s.CSS(nil, p3), report)
	}
}

func TestGamutMappingEndsAndPreservesInGamut(t *testing.T) {
	for _, p3 := range []bool{false, true} {
		for _, lab := range [][3]float64{{.5, 0, 0}, {.6, .05, -.05}} {
			want := oklabRGB(lab[0], lab[1], lab[2], p3)
			got, mapped := gamutMap(lab, p3)
			if mapped || got != want {
				t.Fatalf("in-gamut colour changed: %v -> %v", want, got)
			}
		}
		for _, lab := range [][3]float64{{0, .4, .4}, {1, .4, .4}, {.5, 1e308, 1e308}, {.5, math.MaxFloat64, math.MaxFloat64}, {.5, 0, .000001}} {
			got, _ := gamutMap(lab, p3)
			if !rgbInGamut(got) {
				t.Fatalf("mapping failed to finish in gamut: %v -> %v", lab, got)
			}
		}
	}
}

func TestLowerSelectionPrefersP3RegardlessOfTargetOrder(t *testing.T) {
	const source = "a{inset:0;color:oklch(.7 .2 30)}"
	const want = "a{top:0;right:0;bottom:0;left:0;color:color(display-p3 0.940720 0.423212 0.337924)}"
	chrome, safari := Browser(Chrome, "87"), Browser(Safari, "12")
	request := Browser(Safari, "14.1")
	for _, fidelity := range []Fidelity{0, Identical} {
		var key Key
		for i, targets := range [][]Target{{{Caps: chrome}, {Caps: safari}}, {{Caps: safari}, {Caps: chrome}}} {
			s, _ := buildLower(t, []byte(source), Options{Lower: targets, MinFidelity: fidelity})
			if got := string(s.CSS(nil, request)); got != want {
				t.Errorf("fidelity=%d, order=%d: got %s; want %s", fidelity, i, got, want)
			}
			if got := minCSS(string(s.CSS(nil, request, Pretty))); got != want {
				t.Errorf("pretty fidelity=%d, order=%d: got %s; want %s", fidelity, i, got, want)
			}
			if s.Key(request) != s.Key(safari) {
				t.Errorf("fidelity=%d, order=%d: request did not select the P3 variant", fidelity, i)
			}
			if i == 0 {
				key = s.Key(request)
			} else if key != s.Key(request) {
				t.Errorf("fidelity=%d: target order changed Key", fidelity)
			}
		}
	}
}

func TestLowerSelectionPrefersNativeColourOnRichnessTie(t *testing.T) {
	const source = "a{inset:0;color:oklch(.7 .2 30);background:oklab(1 0 0)}"
	const want = "a{top:0;right:0;bottom:0;left:0;color:oklch(.7 .2 30);background:rgb(255,255,255)}"
	inset := Browser(Chrome, "87")
	lch := CapsFromBits([4]uint64{1 << CapOklch})
	request := CapsFromBits([4]uint64{1<<CapInset | 1<<CapOklch})
	for _, targets := range [][]Target{{{Caps: inset}, {Caps: lch}}, {{Caps: lch}, {Caps: inset}}} {
		s, _ := buildLower(t, []byte(source), Options{Lower: targets})
		if got := string(s.CSS(nil, request)); got != want {
			t.Errorf("got %s; want %s", got, want)
		}
	}
}

func TestGamutMappingLightnessEndpoints(t *testing.T) {
	for _, p3 := range []bool{false, true} {
		for _, tc := range []struct {
			lab, want [3]float64
			mapped    bool
		}{
			{[3]float64{1, 0, 0}, [3]float64{1, 1, 1}, false},
			{[3]float64{0, 0, 0}, [3]float64{}, false},
			{[3]float64{1, .4, .4}, [3]float64{1, 1, 1}, true},
			{[3]float64{0, .4, .4}, [3]float64{}, true},
			{[3]float64{1.1, 0, 0}, [3]float64{1, 1, 1}, true},
			{[3]float64{-.1, 0, 0}, [3]float64{}, true},
		} {
			got, mapped := gamutMap(tc.lab, p3)
			if got != tc.want || mapped != tc.mapped {
				t.Errorf("p3=%v, %v: got %v mapped=%v; want %v mapped=%v", p3, tc.lab, got, mapped, tc.want, tc.mapped)
			}
		}
	}
}
