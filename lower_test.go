package pristinecss

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func lowerTargets() []Target {
	return []Target{Chrome.At("49"), Safari.At("12"), Firefox.At("60")}
}

func denseCSS() []byte {
	const rule = ".dense{inset:1px;inset:2% auto;inset:1px 2px 3px;inset:1px 2px 3px 4px!important;color:oklch(60% .12 30);background:linear-gradient(oklab(.7 .1 -.1 / .4),oklch(.5 .2 250));--accent:oklch(.8 .1 120)}\n"
	return []byte(strings.Repeat(rule, (1_000_000+len(rule)-1)/len(rule)))
}

func lowerSource(t testing.TB, path string) []byte {
	t.Helper()
	if path == "dense-1MB" {
		return denseCSS()
	}
	return readCSS(t, path)
}

func buildLower(t testing.TB, src []byte, o Options) (*Style, Report) {
	t.Helper()
	bin, report := ToBinary(src, o)
	if len(report.Errors) != 0 {
		t.Fatal(report.Errors)
	}
	if report.Bytes != len(bin) {
		t.Fatalf("reported bytes=%d, actual=%d", report.Bytes, len(bin))
	}
	s, err := Load(bin)
	if err != nil {
		t.Fatal(err)
	}
	if s.Source() != string(src) {
		t.Fatal("changed original source")
	}
	return s, report
}

func minCSS(src string) string {
	s, _ := Parse([]byte(src))
	return string(s.Emit(nil, EmitOptions{Style: Minified}))
}

// Cases and Color.js reference vectors reused from the T1 prototype.
func TestLowerKnown(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		gaps     int
	}{
		{"a{inset:1px}", "a{top:1px;right:1px;bottom:1px;left:1px}", 0},
		{`a{in\73 et:1px}`, "a{top:1px;right:1px;bottom:1px;left:1px}", 0},
		{`a{color:o\6b lch(0 0 0)}`, "a{color:rgb(0,0,0)}", 0},
		{`a{inset:v\61 r(--edges)}`, `a{inset:v\61 r(--edges)}`, 1},
		{"a{inset:1px 2%}", "a{top:1px;right:2%;bottom:1px;left:2%}", 0},
		{"a{INSET:auto -2px 3em}", "a{top:auto;right:-2px;bottom:3em;left:-2px}", 0},
		{"a{inset:1px 2px 3px 4px!important;color:red}", "a{top:1px!important;right:2px!important;bottom:3px!important;left:4px!important;color:red}", 0},
		{"a{inset:calc(1px + 2%) auto}", "a{top:calc(1px + 2%);right:auto;bottom:calc(1px + 2%);left:auto}", 0},
		{"a{inset:1px/**/2px}", "a{top:1px;right:2px;bottom:1px;left:2px}", 0},
		{"inset{--inset:2px;inset-inline:3px;content:'inset:4px;oklab(1 0 0)'}", "inset{--inset:2px;inset-inline:3px;content:'inset:4px;oklab(1 0 0)'}", 0},
		{"a{inset:var(--edges)}", "a{inset:var(--edges)}", 1},
		{"a{inset:attr(data-edges)}", "a{inset:attr(data-edges)}", 1},
		{`a{inset:a\74 tr(data-edges)}`, `a{inset:a\74 tr(data-edges)}`, 1},
		{"a{inset:1px 2px 3px 4px 5px}", "a{inset:1px 2px 3px 4px 5px}", 1},
		{"a{color:oklab(1 0 0);--c:OKLCH(0 0 0)}", "a{color:rgb(255,255,255);--c:rgb(0,0,0)}", 0},
		{"a{color:oklab(0 0 0 / 25%)}", "a{color:rgba(0,0,0,0.25)}", 0},
		{"a{color:oklch(.6 .1 30 / .4)}", "a{color:rgba(180,104,92,0.4)}", 0},
		{"a{color:oklab(.5 1e309 0)}", "a{color:oklab(.5 1e309 0)}", 1},
		{"a{background:linear-gradient(oklab(0 0 0),oklch(none .1 30),oklab(1 0 0))}", "a{background:linear-gradient(rgb(0,0,0),oklch(none .1 30),rgb(255,255,255))}", 1},
		{"a{color:oklch(none .2 30);background:oklab(calc(.5) 0 0);border-color:oklch(.5 var(--c) 20)}", "a{color:oklch(none .2 30);background:oklab(calc(.5) 0 0);border-color:oklch(.5 var(--c) 20)}", 3},
		{"a{--c:oklch(.5 .2 calc(30));--text:'oklab(1 0 0)';--url:url(oklab(1 0 0))}", "a{--c:oklch(.5 .2 calc(30));--text:'oklab(1 0 0)';--url:url(oklab(1 0 0))}", 1},
		{"a{color:oklch(100% 0 1turn / 200%);background:oklab(-1 0 0 / -1)}", "a{color:rgb(255,255,255);background:rgba(0,0,0,0)}", 0},
		{"a{--x:oklch(0 0 0);--y:oklch(var(--l) 0 0)}", "a{--x:rgb(0,0,0);--y:oklch(var(--l) 0 0)}", 1},
	} {
		t.Run(tc.in, func(t *testing.T) {
			s, report := buildLower(t, []byte(tc.in), Options{Lower: []Target{Chrome.At("49")}, MinFidelity: Approximate})
			got := s.CSS(nil, Caps{})
			if string(got) != minCSS(tc.want) {
				t.Fatalf("got %s; want %s", got, minCSS(tc.want))
			}
			if len(report.Gaps) != tc.gaps {
				t.Fatalf("gaps: %+v", report.Gaps)
			}
			if _, errs := Parse(got); len(errs) != 0 {
				t.Fatal(errs)
			}
		})
	}
}

func TestLowerInsetValidComponents(t *testing.T) {
	for _, value := range []string{
		"auto", "0", "-0", "+.0", "0e8", "-.5em", "1e-3px", "2%", `1p\78`,
		"inherit", "initial", "unset", "revert", "revert-layer", `\69nherit`,
		"calc(1px + 2%)", "calc((1px + 2%) / 2)", "calc(-1 * (1px + 2%))", "calc(calc(1px) + 2%)",
	} {
		t.Run(value, func(t *testing.T) {
			s, report := buildLower(t, []byte("a{inset:"+value+"}"), Options{Lower: []Target{Chrome.At("49")}})
			want := "a{top:" + value + ";right:" + value + ";bottom:" + value + ";left:" + value + "}"
			if got := string(s.CSS(nil, Caps{})); got != minCSS(want) || len(report.Gaps) != 0 {
				t.Fatalf("valid component not expanded: %s, %+v", got, report.Gaps)
			}
		})
	}
}

func TestLowerInsetInvalidValues(t *testing.T) {
	for _, value := range []string{
		"inherit 1px", "inherit auto", "initial 1px", "unset auto", "revert 0", "revert-layer 2%",
		"1px !foo", "1px!important junk", "1px!important!important", "1px !important !foo",
		"10", "red", "1deg", "1px,2px", "1px / 2px", "calc(foo) auto", "calc(1px + 2) auto",
		"calc(0) auto", "calc(1px +2%) auto", "calc(1px*2px) auto", "calc(1px/0) auto",
	} {
		t.Run(value, func(t *testing.T) {
			src := "a{top:10px;inset:" + value + "}"
			s, report := buildLower(t, []byte(src), Options{Lower: []Target{Chrome.At("49")}})
			if got := string(s.CSS(nil, Caps{})); got != minCSS(src) {
				t.Errorf("invalid shorthand became active declarations: %s", got)
			}
			if len(report.Gaps) != 1 || report.Gaps[0].Cap != CapInset || report.Gaps[0].Lo != uint32(strings.Index(src, "inset")) {
				t.Errorf("missing positioned inset gap: %+v", report.Gaps)
			}
		})
	}
}

func TestLowerInsetEscapedImportant(t *testing.T) {
	for _, suffix := range []string{`!\69mportant`, `!\49MPORTANT`, `!/**/\69mportant/**/`, `! important`} {
		s, report := buildLower(t, []byte("a{inset:1px "+suffix+"}"), Options{Lower: []Target{Chrome.At("49")}})
		want := "a{top:1px!important;right:1px!important;bottom:1px!important;left:1px!important}"
		if got := string(s.CSS(nil, Caps{})); got != want || len(report.Gaps) != 0 {
			t.Errorf("priority lost for %s: %s, %+v", suffix, got, report.Gaps)
		}
	}
}

func TestLowerInsetUnterminatedComment(t *testing.T) {
	const src = "a{inset:0!important/*"
	bin, report := ToBinary([]byte(src), Options{Lower: []Target{Chrome.At("49")}})
	s, err := Load(bin)
	if err != nil {
		t.Fatal(err)
	}
	want := "a{top:0!important;right:0!important;bottom:0!important;left:0!important}"
	if got := string(s.CSS(nil, Caps{})); got != want || len(report.Gaps) != 0 || s.Source() != src {
		t.Fatalf("incomplete trailing comment: %s, %+v", got, report)
	}
}

func TestLowerColorSyntax(t *testing.T) {
	for _, pair := range [][2]string{
		{"oklch(.6 .1 90)", "oklch(60% 25% 100grad)"},
		{"oklch(.6 .1 180)", "oklch(.6 .1 3.141592653589793rad)"},
		{"oklch(.6 .1 90)", "oklch(6e-1 1e-1 .25turn)"},
		{"oklab(.6 .1 -.1)", "oklab(60% 25% -25%)"},
		{"oklab(.6 .1 -.1)", "oklab(.6/**/.1/**/-.1)"},
		{"oklch(.6 0 90)", "oklch(.6 -.1 90)"},
	} {
		var outputs [2]string
		for i, input := range pair {
			s, report := buildLower(t, []byte("a{color:"+input+"}"), Options{Lower: lowerTargets(), MinFidelity: Approximate})
			if len(report.Gaps) != 0 {
				t.Fatal(report.Gaps)
			}
			outputs[i] = string(s.CSS(nil, Caps{}))
		}
		if outputs[0] != outputs[1] {
			t.Fatal(outputs)
		}
	}
	for _, input := range []string{"none .1 30", ".6 none 30", ".6 .1 none", ".6 .1 30 / none", "var(--l) .1 30", ".6 calc(.1) 30", ".6 .1 30 / var(--a)", "from red l c h", ".6 .1 30%", ".6,.1,30", ".6 .1 30 junk"} {
		src := "a{color:oklch(" + input + ")}"
		s, report := buildLower(t, []byte(src), Options{Lower: lowerTargets(), MinFidelity: Approximate})
		if len(report.Gaps) != 3 || string(s.CSS(nil, Caps{})) != minCSS(src) {
			t.Fatalf("%s: %+v", input, report.Gaps)
		}
	}
}

func TestLowerVariantsAndKey(t *testing.T) {
	src := []byte("a{inset:1px;color:oklch(0 0 0);background:oklab(1 0 0)}")
	old, middle, newest := Browser(Chrome, "49"), Browser(Chrome, "90"), Browser(Chrome, "111")
	s, report := buildLower(t, src, Options{Lower: []Target{{Caps: old}, {Caps: middle}, {Caps: newest}}, MinFidelity: Approximate})
	if report.Variants != 3 {
		t.Fatalf("three targets: %d variants", report.Variants)
	}
	wantMiddle := "a{inset:1px;color:rgb(0,0,0);background:rgb(255,255,255)}"
	if got := string(s.CSS(nil, Browser(Chrome, "100"))); got != wantMiddle {
		t.Fatalf("between targets: %s, want %s", got, wantMiddle)
	}
	wantOld := "a{top:1px;right:1px;bottom:1px;left:1px;color:rgb(0,0,0);background:rgb(255,255,255)}"
	for _, caps := range []Caps{old, Caps{}, Browser("unknown", "1")} {
		if got := string(s.CSS(nil, caps)); got != wantOld || s.Key(caps) != s.Key(old) {
			t.Fatalf("fallback: %s, want %s", got, wantOld)
		}
	}
	if s.Key(old) == s.Key(middle) || s.Key(middle) == s.Key(Modern) {
		t.Fatal("different variants share a key")
	}
	unrelated := middle.Bits()
	unrelated[3] ^= 1 << 63
	if s.Key(middle) != s.Key(CapsFromBits(unrelated)) {
		t.Fatal("unused capability changed Key")
	}
	duplicate, dupReport := buildLower(t, src, Options{Lower: []Target{{Caps: old}, {Caps: old}, Firefox.At("60"), {Caps: middle}, {Caps: CapsFromBits(unrelated)}}, MinFidelity: Approximate})
	if dupReport.Variants != 3 || !bytes.Equal(duplicate.CSS(nil, middle), s.CSS(nil, middle)) {
		t.Fatalf("duplicates not projected and collapsed: %d", dupReport.Variants)
	}
	_, allModern := buildLower(t, src, Options{Lower: []Target{{Caps: Modern}, {Caps: newest}}})
	if allModern.Variants != 1 {
		t.Fatalf("modern duplicates: %d", allModern.Variants)
	}
	plain, plainReport := buildLower(t, src, Options{})
	if plainReport.Variants != 1 || plain.Key(Caps{}) != plain.Key(Modern) || !bytes.Equal(plain.CSS(nil, Caps{}), plain.CSS(nil, Modern)) {
		t.Fatal("unrequested lowering")
	}
}

func TestLowerGapPositionsAndFidelity(t *testing.T) {
	src := []byte("a{\n color:oklch(var(--l) .1 30);\n inset:var(--edges);\n background:oklab(calc(.5) 0 0);\n border-color:oklch(none .1 30)}")
	_, report := buildLower(t, src, Options{Lower: lowerTargets(), MinFidelity: Approximate})
	if len(report.Gaps) != 12 {
		t.Fatalf("gaps: %+v", report.Gaps)
	}
	for _, target := range lowerTargets() {
		for _, text := range []string{"oklch(var(--l) .1 30)", "inset:var(--edges)", "oklab(calc(.5) 0 0)", "oklch(none .1 30)"} {
			found := false
			for _, gap := range report.Gaps {
				if gap.Target == target && string(src[gap.Lo:gap.Hi]) == text && gap.What != "" {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing target gap for %s", text)
			}
		}
	}
}

func TestLowerSheets(t *testing.T) {
	for _, path := range append(corpusFiles(t), "dense-1MB") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src := lowerSource(t, path)
			plain, _ := buildLower(t, src, Options{})
			for _, fidelity := range []Fidelity{0, Identical} {
				s, report := buildLower(t, src, Options{Lower: lowerTargets(), MinFidelity: fidelity})
				if report.Variants > 4 {
					t.Fatalf("too many variants: %d", report.Variants)
				}
				if !bytes.Equal(plain.CSS(nil, Modern), s.CSS(nil, Modern)) {
					t.Fatal("modern output changed")
				}
				for _, target := range append(lowerTargets(), Target{}) {
					// Identical can force a fallback: check gaps for the stored
					// output actually selected, not necessarily the requested target.
					gapTarget := target
					selected := s.Key(target.Caps)
					for _, candidate := range lowerTargets() {
						if s.Key(candidate.Caps) == selected {
							gapTarget = candidate
							break
						}
					}
					for _, format := range []EmitStyle{Minified, Pretty} {
						out := s.CSS(nil, target.Caps, format)
						tree, errs := Parse(out)
						if len(errs) != 0 {
							t.Fatal(errs)
						}
						for _, cap := range []Cap{CapInset, CapOklch, CapOklab} {
							remaining := scanLower(string(out), styleSpans(tree), cap, Approximate, target.Caps)
							want := 0
							for _, gap := range report.Gaps {
								if gap.Cap == cap && gap.Target == gapTarget {
									want++
								}
							}
							if got := len(remaining.edits) + len(remaining.gaps); got != want {
								t.Fatalf("fidelity=%d variant=%d cap %d: remaining=%d, reported gaps=%d", fidelity, selected.Variant, cap, got, want)
							}
						}
						dst := out[:0]
						if allocs := testing.AllocsPerRun(3, func() { dst = s.CSS(dst[:0], target.Caps, format) }); allocs != 0 {
							t.Fatalf("warm CSS allocations=%g", allocs)
						}
					}
				}
				t.Logf("fidelity=%d variants=%d binary=%d gaps=%d", fidelity, report.Variants, report.Bytes, len(report.Gaps))
			}
		})
	}
}

func BenchmarkCSSLower(b *testing.B) {
	for _, path := range append(corpusFiles(b), "dense-1MB") {
		b.Run(filepath.Base(path), func(b *testing.B) {
			src := lowerSource(b, path)
			s, _ := buildLower(b, src, Options{Lower: lowerTargets(), MinFidelity: Approximate})
			caps := Browser(Chrome, "49")
			dst := s.CSS(nil, caps)
			b.ReportAllocs()
			b.SetBytes(int64(len(dst)))
			b.ResetTimer()
			for range b.N {
				dst = s.CSS(dst[:0], caps)
			}
			benchmarkBytes = dst
		})
	}
}

func TestLowerBinaryStructure(t *testing.T) {
	s, _ := buildLower(t, []byte("a{inset:1px}"), Options{Lower: []Target{{}}})
	base := []byte(s.blob)
	table := styleHeaderLen + int(styleU32(s.blob, 12)) + int(styleU32(s.blob, 4)) + int(styleU32(s.blob, 16)) + int(styleU32(s.blob, 20))
	for _, off := range []int{table, table + variantHeaderLen + 32, table + variantHeaderLen + 36, table + variantHeaderLen + variantEntryLen + 32, table + variantHeaderLen + variantEntryLen + 44} {
		t.Run(fmt.Sprint(off), func(t *testing.T) {
			b := bytes.Clone(base)
			binary.LittleEndian.PutUint32(b[off:], ^uint32(0))
			binary.LittleEndian.PutUint32(b[len(b)-4:], checksum(b[:len(b)-4]))
			if _, err := Load(b); err == nil {
				t.Fatal("accepted invalid variants with valid checksum")
			}
		})
	}
	for n := range len(base) {
		if _, err := Load(base[:n]); err == nil {
			t.Fatalf("accepted truncation %d", n)
		}
	}
}

func FuzzLower(f *testing.F) {
	for _, src := range []string{
		"a{inset:1px;color:oklch(.5 .1 30)}", "a{inset:1px/*", "a{inset:(}",
		"a{color:oklch(", "a{inset:var(--x);color:oklab(none 0 0)}", "a{inset:}",
		"a{inset:0!important/*", "a{inset:inherit 1px}", `a{inset:1px !\69mportant}`,
		"a{top:10px;inset:inherit auto}", "a{inset:1px !foo}", "a{inset:calc(1px + 2%) auto}",
	} {
		f.Add([]byte(src))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		bin, _ := ToBinary(src, Options{Lower: lowerTargets(), MinFidelity: Approximate})
		s, err := Load(bin)
		if err != nil {
			t.Fatal(err)
		}
		if s.Source() != string(src) {
			t.Fatal("changed source")
		}
		_ = s.CSS(nil, Caps{})
		_ = s.CSS(nil, Modern, Pretty)
	})
}

func TestLowerSafeWithGaps(t *testing.T) {
	src := []byte("a{inset:1px;color:oklch(var(--l) 0 0);background:oklab(1 0 0)}")
	s, _ := buildLower(t, src, Options{Lower: []Target{Chrome.At("49"), Chrome.At("90")}, MinFidelity: Approximate})
	for _, tc := range []struct {
		caps Caps
		want string
	}{
		{Caps{}, "a{top:1px;right:1px;bottom:1px;left:1px;color:oklch(var(--l) 0 0);background:rgb(255,255,255)}"},
		{CapsFromBits([4]uint64{1 << CapInset}), "a{inset:1px;color:oklch(var(--l) 0 0);background:rgb(255,255,255)}"},
		{CapsFromBits([4]uint64{1 << CapOklch}), "a{top:1px;right:1px;bottom:1px;left:1px;color:oklch(var(--l) 0 0);background:rgb(255,255,255)}"},
		{CapsFromBits([4]uint64{1<<CapInset | 1<<CapOklch}), "a{inset:1px;color:oklch(var(--l) 0 0);background:rgb(255,255,255)}"},
	} {
		if got := string(s.CSS([]byte("prefix:"), tc.caps)); got != "prefix:"+tc.want {
			t.Fatalf("got %s; want %s", got, tc.want)
		}
	}
}

func TestLowerIndependentColors(t *testing.T) {
	src := []byte("a{inset:1px;color:oklch(0 0 0);background:oklab(1 0 0)}")
	old := Caps{}
	inset := CapsFromBits([4]uint64{1 << CapInset})
	lch := CapsFromBits([4]uint64{1 << CapOklch})
	s, report := buildLower(t, src, Options{Lower: []Target{{Caps: old}, {Caps: inset}, {Caps: lch}}, MinFidelity: Approximate})
	if report.Variants != 4 {
		t.Fatalf("three distinct targets plus modern: %d variants", report.Variants)
	}
	want := "a{top:1px;right:1px;bottom:1px;left:1px;color:oklch(0 0 0);background:rgb(255,255,255)}"
	if got := string(s.CSS(nil, lch)); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestLowerDuplicateGaps(t *testing.T) {
	old, middle := Browser(Chrome, "49"), Browser(Chrome, "90")
	_, report := buildLower(t, []byte("a{inset:var(--edges)}"), Options{Lower: []Target{{Caps: old}, {Caps: old}, {Caps: old}}})
	if report.Variants != 2 || len(report.Gaps) != 1 {
		t.Fatalf("duplicate targets repeated gaps: %+v", report)
	}
	_, report = buildLower(t, []byte("a{color:oklch(var(--l) 0 0)}"), Options{Lower: []Target{{Caps: old}, {Caps: old}, {Caps: middle}, {Caps: middle}}})
	if report.Variants != 2 || len(report.Gaps) != 2 || report.Gaps[0].Target != (Target{Caps: old}) || report.Gaps[1].Target != (Target{Caps: middle}) {
		t.Fatalf("gaps must identify each distinct requested target once: %+v", report)
	}
}

var benchmarkKey Key

func TestStyleKeyNoAllocs(t *testing.T) {
	src := readFramework(t, "bootstrap.css")
	for _, targets := range [][]Target{nil, lowerTargets()} {
		s, _ := buildLower(t, src, Options{Lower: targets})
		caps := Browser(Chrome, "49")
		if allocs := testing.AllocsPerRun(10, func() { benchmarkKey = s.Key(caps) }); allocs != 0 {
			t.Fatalf("Key allocations=%g", allocs)
		}
	}
}

func BenchmarkStyleKey(b *testing.B) {
	for _, path := range append(corpusFiles(b), "dense-1MB") {
		b.Run(filepath.Base(path), func(b *testing.B) {
			s, _ := buildLower(b, lowerSource(b, path), Options{Lower: lowerTargets()})
			caps := Browser(Chrome, "49")
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchmarkKey = s.Key(caps)
			}
		})
	}
}

func TestLowerNeutralColorsAtDefaultFidelity(t *testing.T) {
	src := []byte("a{color:oklab(1 0 0);background:oklch(100% 0 0/.01)}")
	s, report := buildLower(t, src, Options{Lower: lowerTargets()})
	want := "a{color:rgb(255,255,255);background:rgba(255,255,255,0.01)}"
	if got := string(s.CSS(nil, Caps{})); got != want || len(report.Gaps) != 0 {
		t.Fatalf("neutral colours need no clipping: %s, %+v", got, report.Gaps)
	}
}
