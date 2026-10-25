package pristinecss

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRaiseKnown(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		drops    int
	}{
		{"a{-webkit-transform:none;transform:none}", "a{transform:none}", 1},
		{"a{-moz-transform:none;-ms-transform:none;-o-transform:none;transform:none}", "a{transform:none}", 3},
		{"a{transform:none;-webkit-transform:rotate(1deg)!important}", "a{transform:none;-webkit-transform:rotate(1deg)!important}", 0},
		{"a{-WEBKIT-TRANSFORM:none;TRANSFORM:none}", "a{TRANSFORM:none}", 1},
		{`a{-webkit-tr\61 nsform:none;tr\61 nsform:none}`, `a{tr\61 nsform:none}`, 1},
		{`a{\2d webkit-transform:none;transform:none}`, "a{transform:none}", 1},
		{"a{-webkit-transform:none}b{transform:none}", "a{-webkit-transform:none}b{transform:none}", 0},
		{"a{-webkit-transform:none;b{transform:none}}", "a{-webkit-transform:none;b{transform:none}}", 0},
		{"a{transform:none;b{-webkit-transform:none}}", "a{transform:none;b{-webkit-transform:none}}", 0},
		{"@media screen{a{-webkit-transform:none;transform:none}}", "@media screen{a{transform:none}}", 1},
		{"@keyframes x{from{-webkit-transform:none;transform:none}}", "@keyframes x{from{transform:none}}", 1},
		{`a{-webkit-transform:rotate(45deg)!\69mportant;transform:none}`, `a{-webkit-transform:rotate(45deg)!important;transform:none}`, 0},
		{"a{-webkit-mask:url(mask.svg);masK:none}", "a{-webkit-mask:url(mask.svg);masK:none}", 0},
		{"a{-webkit-mask:url(mask.svg);masK:none}", "a{masK:none}", 1},
		{"@keyframes k{from{-webkit-transform:rotate(45deg);transform:none!important}to{transform:none}}", "@keyframes k{from{-webkit-transform:rotate(45deg);transform:none!important}to{transform:none}}", 0},
		{`@keyframes k{from{-webkit-transform:rotate(45deg);transform:none!\69mportant}}`, "@keyframes k{from{-webkit-transform:rotate(45deg);transform:none!important}}", 0},
		{"a{-webkit-unknown:1;unknown:1;--webkit-transform:1;--transform:1}", "a{-webkit-unknown:1;unknown:1;--webkit-transform:1;--transform:1}", 0},
		{"a{-webkit-user-select:none;user-select:none}", "a{-webkit-user-select:none;user-select:none}", 0},
		{"a{-webkit-transition:none;transition:none;-webkit-animation:none;animation:none}", "a{transition:none;animation:none}", 2},
		{"a{-webkit-inset:1px;inset:1px}", "a{-webkit-inset:1px;inset:1px}", 0},
	} {
		t.Run(tc.in, func(t *testing.T) {
			s, report := buildLower(t, []byte(tc.in), Options{Raise: Baseline(2024)})
			if got := string(s.CSS(nil, Modern)); got != minCSS(tc.want) || len(report.Changes) != tc.drops {
				t.Fatalf("got %s, drops=%d; want %s, drops=%d", got, len(report.Changes), minCSS(tc.want), tc.drops)
			}
			for _, c := range report.Changes {
				if c.Lo >= c.Hi || c.Hi > uint32(len(tc.in)) || c.What == "" || !Baseline(2024).Has(c.Cap) {
					t.Fatalf("invalid change: %+v", c)
				}
			}
		})
	}
}

func TestRaiseCascade(t *testing.T) {
	for _, src := range []string{
		"#a{transform:none;-webkit-transform:rotate(45deg)}",
		"#a{-webkit-transform:rotate(45deg)!important;transform:none}",
		"#a{transition:none;-webkit-transition:opacity 2s}",
		"#a{-webkit-transition:opacity 2s!important;transition:none}",
		".yui3-skin-sam .yui3-console-collapsed .yui3-console-hd{border-radius:10px;-moz-border-radius:10px;-webkit-border-radius:0}",
	} {
		t.Run(src, func(t *testing.T) {
			s, report := buildLower(t, []byte(src), Options{Raise: Baseline(2024)})
			if got := string(s.CSS(nil, Modern)); got != minCSS(src) || len(report.Changes) != 0 {
				t.Fatalf("cascade changed: %s, changes=%+v", got, report.Changes)
			}
			t.Logf("browser-case: %q => %q", src, s.CSS(nil, Modern))
		})
	}
	for _, decls := range []string{
		"-webkit-transform:rotate(45deg);transform:none!important",
		"-webkit-transform:rotate(45deg)!important;transform:none!important",
		"-webkit-transform:rotate(45deg)!important;transform:none!important;transform:rotate(90deg)",
	} {
		t.Run(decls, func(t *testing.T) {
			s, report := buildLower(t, []byte("a{"+decls+"}"), Options{Raise: Baseline(2024)})
			if strings.Contains(string(s.CSS(nil, Modern)), "-webkit-") || len(report.Changes) != 1 {
				t.Fatalf("overridden prefix not dropped: %+v", report)
			}
		})
	}
}

func TestRaiseInvalidValues(t *testing.T) {
	for _, value := range []string{"", " ", "/**/", "!important", "rotate(1deg"} {
		t.Run(value, func(t *testing.T) {
			src := []byte("a{-webkit-transform:rotate(45deg);transform:" + value + "}")
			bin, report := ToBinary(src, Options{Raise: Baseline(2024)})
			if value == "rotate(1deg" && len(report.Errors) == 0 {
				t.Fatal("expected parser diagnostic")
			}
			if len(report.Changes) != 0 || !strings.Contains(string(Open(string(bin)).CSS(nil, Modern)), "-webkit-transform") {
				t.Fatalf("prefix dropped for invalid/empty replacement: %+v", report)
			}
		})
	}
}

func TestRaiseFloor(t *testing.T) {
	src := []byte("a{-webkit-transform:none;transform:none;-moz-box-sizing:border-box;box-sizing:border-box}")
	for _, floor := range []Caps{{}, Baseline(2014), CapsFromBits([4]uint64{0, 1 << (CapTransform % 64)})} {
		// A single capability must not enable unrelated rewrites.
		s, r := buildLower(t, src, Options{Raise: floor, MinFidelity: Identical})
		wantDrops := 0
		for _, cap := range []Cap{CapTransform, CapBoxSizing} {
			if floor.Has(cap) {
				wantDrops++
			}
		}
		if len(r.Changes) != wantDrops {
			t.Fatalf("floor %x: %+v", floor.Bits(), r.Changes)
		}
		if wantDrops == 0 && string(s.CSS(nil, Modern)) != minCSS(string(src)) {
			t.Fatal("raised without support")
		}
	}
}

func TestRaiseLowerInteraction(t *testing.T) {
	src := []byte("a{-webkit-transform:none;transform:none;inset:1px;color:oklch(var(--l) .1 30);background:oklab(.5 1 1)}")
	old, older := Safari.At("8"), Chrome.At("20")
	s, r := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{old, older, old, {Caps: Modern}}})
	if len(r.Changes) != 1 || len(r.Gaps) != 4 || len(r.Mappings) != 2 {
		t.Fatalf("unexpected report: %+v", r)
	}
	for _, target := range []Target{old, older} {
		out := string(s.CSS(nil, target.Caps))
		if strings.Contains(out, "-webkit-") || strings.Contains(out, "inset:") || !strings.Contains(out, "top:1px") {
			t.Fatalf("variant not built from raised CSS: %s", out)
		}
		for _, c := range r.Changes {
			count := 0
			for _, gap := range r.Gaps {
				if gap.Target == target && gap.Cap == c.Cap && gap.Lo == c.Lo && gap.Hi == c.Hi && gap.What != "" {
					count++
				}
			}
			want := 0
			if c.Cap == CapTransform {
				want = 1
			}
			if count != want {
				t.Fatalf("gap count for %+v = %d, want %d", c, count, want)
			}
		}
	}
	for _, gap := range r.Gaps {
		if gap.Target.Caps == Modern {
			t.Fatal("gap for supported floor")
		}
		if gap.Cap == CapOklch && string(src[gap.Lo:gap.Hi]) != "oklch(var(--l) .1 30)" {
			t.Fatalf("rebased gap: %+v", gap)
		}
	}
	for _, m := range r.Mappings {
		if string(src[m.Lo:m.Hi]) != "oklab(.5 1 1)" {
			t.Fatalf("rebased mapping: %+v", m)
		}
	}
}

func TestRaiseLowerGapWithoutVariant(t *testing.T) {
	src := []byte("a{-webkit-transform:none;transform:none}")
	target := Safari.At("8")
	s, report := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{target, target, {Caps: Modern}}})
	if report.Variants != 1 || len(report.Changes) != 1 || len(report.Gaps) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	gap := report.Gaps[0]
	if gap.Target != target || gap.Cap != CapTransform || string(src[gap.Lo:gap.Hi]) != "-webkit-transform:none;" || gap.What == "" {
		t.Fatalf("missing original-position gap: %+v", gap)
	}
	if got := string(s.CSS(nil, Caps{})); got != "a{transform:none}" {
		t.Fatalf("unexpected fallback: %s", got)
	}
}

func TestRaiseCompatibilityAliasGaps(t *testing.T) {
	for _, tc := range []struct {
		id                BrowserID
		version, property string
		gaps              int
	}{
		{"edge", "16", "backdrop-filter", 0},
		{"edge", "17", "backdrop-filter", 1},
		{"edge", "18", "backdrop-filter", 1},
		{"edge", "79", "backdrop-filter", 0},
		{"edge", "17-18", "backdrop-filter", 1},
		{"firefox", "63", "appearance", 0},
		{"firefox", "64", "appearance", 1},
		{"firefox", "64.0esr", "appearance", 1},
		{"firefox", "80", "appearance", 0},
	} {
		t.Run(string(tc.id)+tc.version, func(t *testing.T) {
			value := "auto"
			if tc.property == "backdrop-filter" {
				value = "blur(8px)"
			}
			src := []byte("a{-webkit-" + tc.property + ":" + value + ";" + tc.property + ":" + value + "}")
			target := tc.id.At(tc.version)
			_, r := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{target}})
			if len(r.Changes) != 1 || len(r.Gaps) != tc.gaps {
				t.Fatalf("alias gap count: %+v", r)
			}
			for _, gap := range r.Gaps {
				if gap.Target != target || string(src[gap.Lo:gap.Hi]) != "-webkit-"+tc.property+":"+value+";" {
					t.Fatalf("alias gap position/target: %+v", gap)
				}
			}
		})
	}
}

func TestRaisePrefixGapsKeepTargetIdentity(t *testing.T) {
	src := []byte("a{-webkit-backdrop-filter:blur(8px);backdrop-filter:blur(8px)}")
	before, alias := Edge.At("16"), Edge.At("18")
	// Both targets deliberately use the same features; identity decides prefix support.
	before.Caps, alias.Caps = Caps{}, Caps{}
	_, r := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{before, alias, alias}})
	if r.Variants != 1 || len(r.Changes) != 1 || len(r.Gaps) != 1 || r.Gaps[0].Target != alias {
		t.Fatalf("prefix gaps lost target identity: %+v", r)
	}
}

func TestRaiseLowerPrefixGaps(t *testing.T) {
	for _, tc := range []struct {
		id              BrowserID
		version, prefix string
	}{
		{"chrome", "20", "-webkit-"}, {"and_chr", "20", "-webkit-"},
		{"safari", "8", "-webkit-"}, {"ios_saf", "8", "-webkit-"},
		{"samsung", "4", "-webkit-"}, {"edge", "79", "-webkit-"},
		{"opera", "15", "-webkit-"}, {"op_mob", "14", "-webkit-"},
		{"firefox", "15", "-moz-"}, {"and_ff", "15", "-moz-"},
		{"ie", "9", "-ms-"}, {"edge", "18", "-webkit-"},
		{"opera", "12.1", "-o-"}, {"op_mob", "12.1", "-o-"},
		{"unknown", "1", ""},
	} {
		t.Run(string(tc.id)+tc.version, func(t *testing.T) {
			target := tc.id.At(tc.version)
			// Isolate prefixed-property support from unprefixed property support.
			target.Caps.bits[CapTransform/64] &^= uint64(1) << (CapTransform % 64)
			src := []byte("a{-webkit-transform:none;-moz-transform:none;-ms-transform:none;-o-transform:none;transform:none}")
			_, report := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{target, target}})
			want := 1
			if tc.prefix == "" {
				want = 0
			}
			if len(report.Changes) != 4 || len(report.Gaps) != want {
				t.Fatalf("unexpected report: %+v", report)
			}
			for _, gap := range report.Gaps {
				if string(src[gap.Lo:gap.Hi]) != tc.prefix+"transform:none;" || gap.Target != target {
					t.Fatalf("wrong prefixed-property gap: %+v", gap)
				}
			}
		})
	}
	for _, target := range []Target{Safari.At("8"), Firefox.At("15")} {
		src := []byte(`a{\2d moz-tr\61 nsform:none;transform:none}`)
		_, report := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{target}})
		want := 0
		if target == Firefox.At("15") {
			want = 1
		}
		if len(report.Changes) != 1 || len(report.Gaps) != want {
			t.Fatalf("escaped prefix gap: %+v", report)
		}
	}
}

func TestRaiseCapabilityOnlyPrefixGaps(t *testing.T) {
	for _, tc := range []struct {
		declarations string
		caps         Caps
		gaps         int
	}{
		{"-webkit-backdrop-filter:blur(8px);backdrop-filter:blur(8px)", Caps{}, 1},
		{"-webkit-transform:none;transform:none", Baseline(2014), 1},
		{"-webkit-transform:none;transform:none", Baseline(2024), 0},
		{"-moz-transform:none;transform:none", Caps{}, 1},
		{"-o-transform:none;transform:none", Caps{}, 1},
		// BCD records -ms-transition only where unprefixed transition also works.
		{"-ms-transition:none;transition:none", Caps{}, 0},
	} {
		t.Run(tc.declarations, func(t *testing.T) {
			target := Target{Caps: tc.caps}
			src := []byte("a{" + tc.declarations + "}")
			_, r := buildLower(t, src, Options{Raise: Baseline(2024), Lower: []Target{target, target}})
			if len(r.Changes) != 1 || len(r.Gaps) != tc.gaps {
				t.Fatalf("capability-only gaps: %+v", r)
			}
			for _, gap := range r.Gaps {
				if gap.Target != target || gap.Lo != r.Changes[0].Lo || gap.Hi != r.Changes[0].Hi {
					t.Fatalf("lost target or source position: %+v", gap)
				}
			}
		})
	}
}

func TestRaiseSheets(t *testing.T) {
	for _, path := range corpusFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src := readCSS(t, path)
			plain, _ := buildLower(t, src, Options{})
			s, report := buildLower(t, src, Options{Raise: Baseline(2024)})
			var expected []byte
			var pos uint32
			for _, c := range report.Changes {
				if c.Lo < pos || c.Hi > uint32(len(src)) || c.What == "" || !Baseline(2024).Has(c.Cap) {
					t.Fatalf("invalid change: %+v", c)
				}
				expected = append(expected, src[pos:c.Lo]...)
				pos = c.Hi
			}
			expected = append(expected, src[pos:]...)
			before, _ := Parse(plain.CSS(nil, Modern))
			var after *Sheet
			for _, format := range []EmitStyle{Minified, Pretty} {
				out := s.CSS(nil, Modern, format)
				var errs []ParseError
				after, errs = Parse(out)
				if len(errs) != 0 {
					t.Fatal(errs)
				}
				tree, _ := Parse(expected)
				if !bytes.Equal(out, tree.Emit(nil, EmitOptions{Style: format})) {
					t.Fatal("output does not match reported drops")
				}
			}
			lowered, _ := buildLower(t, src, Options{Raise: Baseline(2024), Lower: lowerTargets()})
			for _, target := range append(lowerTargets(), Target{Caps: Modern}) {
				for _, format := range []EmitStyle{Minified, Pretty} {
					out := lowered.CSS(nil, target.Caps, format)
					if _, errs := Parse(out); len(errs) != 0 {
						t.Fatal(errs)
					}
					if target.Caps == Modern && !bytes.Equal(out, s.CSS(nil, Modern, format)) {
						t.Fatal("lowering changed the raised modern output")
					}
				}
			}
			count := func(tree *Sheet) int {
				n := 0
				for range tree.All(KindDeclaration) {
					n++
				}
				return n
			}
			if count(before)-count(after) != len(report.Changes) {
				t.Fatal("not every dropped declaration was reported")
			}
			t.Logf("dropped=%d minified_bytes_saved=%d (%d -> %d)", len(report.Changes), len(plain.CSS(nil, Modern))-len(s.CSS(nil, Modern)), len(plain.CSS(nil, Modern)), len(s.CSS(nil, Modern)))
		})
	}
}

func BenchmarkToBinaryRaise(b *testing.B) {
	for _, path := range append(corpusFiles(b), "dense-1MB") {
		src := lowerSource(b, path)
		for _, raise := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/raise=%t", filepath.Base(path), raise), func(b *testing.B) {
				var o Options
				if raise {
					o.Raise = Baseline(2024)
				}
				b.ReportAllocs()
				b.SetBytes(int64(len(src)))
				b.ResetTimer()
				for range b.N {
					benchmarkBytes, _ = ToBinary(src, o)
				}
			})
		}
	}
}
