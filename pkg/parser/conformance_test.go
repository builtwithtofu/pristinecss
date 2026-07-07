package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestConformance(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "test-data", "conformance")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read conformance fixtures: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		path := filepath.Join(fixtureDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			ss, errs := parseCSSBytes(content)
			if len(errs) > 0 {
				for _, err := range errs {
					t.Errorf("%s: %v (nearest label: %s)", entry.Name(), err, nearestFixtureLabel(content, err.Line))
				}
			}
			if entry.Name() == "future.css" && !containsGenericAtRule(ss.Rules) {
				t.Fatalf("future.css should produce at least one GenericAtRule")
			}
		})
	}
}

func nearestFixtureLabel(content []byte, line uint32) string {
	lines := strings.Split(string(content), "\n")
	label := "<no label>"
	lineIndex := int(line)
	if lineIndex > len(lines) {
		lineIndex = len(lines)
	}
	for i := 0; i < lineIndex; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
			label = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "/*"), "*/"))
		}
	}
	return label
}

func containsGenericAtRule(nodes []Node) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *GenericAtRule:
			return true
		case *Selector:
			if containsGenericAtRule(n.Rules) {
				return true
			}
		case *MediaAtRule:
			if containsGenericAtRule(n.Rules) {
				return true
			}
		case *SupportsAtRule:
			if containsGenericAtRule(n.Rules) {
				return true
			}
		case *LayerAtRule:
			if containsGenericAtRule(n.Rules) {
				return true
			}
		}
	}
	return false
}

func parseCSS(t testing.TB, input string) (*Stylesheet, []ParseError) {
	t.Helper()
	return parseCSSBytes([]byte(input))
}

func parseCSSBytes(content []byte) (*Stylesheet, []ParseError) {
	toks := lexer.Lex(content)
	return Parse(content, toks)
}

func parseNoErrors(t testing.TB, input string) *Stylesheet {
	t.Helper()
	ss, errs := parseCSS(t, input)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	return ss
}

func TestGenericAtRule(t *testing.T) {
	t.Run("unknown statement falls through and following rule parses", func(t *testing.T) {
		ss := parseNoErrors(t, `@unknown thing; .ok { color: green; }`)
		if _, ok := ss.Rules[0].(*GenericAtRule); !ok {
			t.Fatalf("first rule = %T, want *GenericAtRule", ss.Rules[0])
		}
		if _, ok := ss.Rules[1].(*Selector); !ok {
			t.Fatalf("following rule = %T, want *Selector", ss.Rules[1])
		}
	})
	t.Run("unknown block preserves nested known at-rule", func(t *testing.T) {
		ss := parseNoErrors(t, `@future { @media (width >= 1px) { .x { color: red; } } }`)
		generic := ss.Rules[0].(*GenericAtRule)
		if len(generic.Block) != 1 {
			t.Fatalf("generic block len = %d", len(generic.Block))
		}
		if _, ok := generic.Block[0].(*MediaAtRule); !ok {
			t.Fatalf("nested rule = %T, want *MediaAtRule", generic.Block[0])
		}
	})
	t.Run("unclosed unknown block reaches EOF without panic", func(t *testing.T) {
		_, _ = parseCSS(t, `@future { .x { color: red; }`)
	})
}

func TestSelectorCompatibility(t *testing.T) {
	cases := map[string]string{
		"descendant":        `.a .b { color: red; }`,
		"child":             `.a > .b { color: red; }`,
		"next sibling":      `.a + .b { color: red; }`,
		"subsequent":        `.a ~ .b { color: red; }`,
		"column":            `.a || .b { color: red; }`,
		"compound no space": `.a.b { color: red; }`,
		"namespace type":    `svg|rect { fill: red; }`,
		"namespace any":     `*|rect { fill: red; }`,
		"namespace empty":   `|rect { fill: red; }`,
		"universal":         `*.cls { color: red; }`,
		"attributes":        `a[href="x"][rel~="tag"][lang|="en"][href^="https"][href$=".org" i][href*="example" s] { color: red; }`,
		"vendor pseudo":     `button:-moz-focusring { outline: auto; }`,
		"functional":        `li:nth-child(2n+1 of .x):is(h1, h2):has(> img)::part(tab) { color: red; }`,
	}
	for name, css := range cases {
		t.Run(name, func(t *testing.T) { parseNoErrors(t, css) })
	}
}

func TestMDNPseudoSelectorNames(t *testing.T) {
	cases := []string{
		":active", ":any-link", ":autofill", ":blank", ":checked", ":current", ":default", ":defined", ":dir(rtl)", ":disabled", ":empty", ":enabled", ":first", ":first-child", ":first-of-type", ":fullscreen", ":future", ":focus", ":focus-visible", ":focus-within", ":has(> img)", ":host", ":host(.card)", ":host-context(.theme)", ":hover", ":indeterminate", ":in-range", ":invalid", ":is(h1, h2)", ":lang(en)", ":last-child", ":last-of-type", ":left", ":link", ":local-link", ":modal", ":muting", ":not(.disabled)", ":nth-child(2n+1 of .x)", ":nth-col(2n+1)", ":nth-last-child(odd)", ":nth-last-col(even)", ":nth-last-of-type(2)", ":nth-of-type(3n)", ":only-child", ":only-of-type", ":optional", ":out-of-range", ":past", ":paused", ":picture-in-picture", ":placeholder-shown", ":playing", ":popover-open", ":read-only", ":read-write", ":required", ":right", ":root", ":scope", ":seeking", ":state(checked)", ":target", ":target-current", ":target-within", ":user-invalid", ":user-valid", ":valid", ":visited", ":where(.x)",
		"::after", "::backdrop", "::before", "::checkmark", "::cue", "::cue-region", "::details-content", "::file-selector-button", "::first-letter", "::first-line", "::grammar-error", "::highlight(search)", "::marker", "::part(tab)", "::picker(select)", "::placeholder", "::selection", "::slotted(span)", "::spelling-error", "::target-text", "::view-transition", "::view-transition-group(root)", "::view-transition-image-pair(root)", "::view-transition-new(root)", "::view-transition-old(root)",
		":before", ":after", ":first-line", ":first-letter",
	}
	for _, pseudo := range cases {
		t.Run(pseudo, func(t *testing.T) {
			ss := parseNoErrors(t, `.x`+pseudo+` { color: red; }`)
			selector := ss.Rules[0].(*Selector)
			got := selector.Selectors[len(selector.Selectors)-1]
			if got.Type != Pseudo || string(got.Value) != pseudo {
				t.Fatalf("last selector = (%v, %q), want Pseudo %q", got.Type, got.Value, pseudo)
			}
		})
	}
}

func TestNesting(t *testing.T) {
	css := `.card { color: black; & .title { color: blue; } background: white; .bare { color: red; } &:hover { color: green; } .parent & { color: purple; } :is(&) { color: orange; } @media (width >= 40rem) { color: red; .wide { color: blue; } } }`
	ss := parseNoErrors(t, css)
	rules := ss.Rules[0].(*Selector).Rules
	if len(rules) < 7 {
		t.Fatalf("nested rule/declaration order too short: %d", len(rules))
	}
}

func TestNestingDepthCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < maxRuleBlockDepth+2; i++ {
		fmt.Fprintf(&b, `.a%d {`, i)
	}
	for i := 0; i < maxRuleBlockDepth+2; i++ {
		b.WriteByte('}')
	}
	_, errs := parseCSS(t, b.String())
	if len(errs) == 0 {
		t.Fatalf("expected depth error")
	}
}

func TestSupports(t *testing.T) {
	ss := parseNoErrors(t, `@supports not (display: grid) { .x { display: block; } } @supports selector(:has(> img)) { .y { color: red; } }`)
	first := ss.Rules[0].(*SupportsAtRule)
	if first.Condition.Kind != ConditionNot {
		t.Fatalf("condition kind = %v, want ConditionNot", first.Condition.Kind)
	}
	if got := string(first.Condition.Children[0].Raw); got != "display: grid" {
		t.Fatalf("supports raw = %q", got)
	}
}

func TestLayer(t *testing.T) {
	ss := parseNoErrors(t, `@layer reset, theme; @layer components { .x { color: red; } }`)
	if ss.Rules[0].(*LayerAtRule).Rules != nil {
		t.Fatalf("statement @layer should keep nil Rules")
	}
	if len(ss.Rules[1].(*LayerAtRule).Rules) == 0 {
		t.Fatalf("block @layer should parse rules")
	}
}

func TestScope(t *testing.T) {
	parseNoErrors(t, `@scope (.card) to (.footer) { :scope { color: red; } }`)
}
func TestStartingStyle(t *testing.T) { parseNoErrors(t, `@starting-style { .x { opacity: 0; } }`) }
func TestNamespace(t *testing.T) {
	parseNoErrors(t, `@namespace svg url(http://www.w3.org/2000/svg); svg|rect { fill: red; }`)
}
func TestPositionTry(t *testing.T)    { parseNoErrors(t, `@position-try --tip { position-area: top; }`) }
func TestViewTransition(t *testing.T) { parseNoErrors(t, `@view-transition { navigation: auto; }`) }
func TestFontPaletteValues(t *testing.T) {
	parseNoErrors(t, `@font-palette-values --brand { font-family: Demo; base-palette: 1; }`)
}

func TestPage(t *testing.T) {
	parseNoErrors(t, `@page :first { margin: 1cm; @top-left { content: "x"; } @right-bottom { content: counter(page); } }`)
	_, errs := parseCSS(t, `@page { @middle { content: "x"; } }`)
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "top-left-corner") {
		t.Fatalf("expected margin-box teaching error, got %v", errs)
	}
}

func TestProperty(t *testing.T) {
	parseNoErrors(t, `@property --angle { syntax: "<angle>"; inherits: false; initial-value: 0deg; }`)
	_, errs := parseCSS(t, `@property angle { syntax: "<angle>"; inherits: false; initial-value: 0deg; }`)
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "--") {
		t.Fatalf("expected dashed-ident error, got %v", errs)
	}
}

func TestDeclarationValues(t *testing.T) {
	cases := []string{
		`.x { margin: -4px; }`,
		`.x { font: 16px/1.5 sans-serif; }`,
		`.x { width: calc(100% - 2px); }`,
		`.x { grid-template-columns: repeat(auto-fill, minmax(0, 1fr)); }`,
		`@font-face { font-family: Demo; src: url(demo.woff2); unicode-range: U+0-7F, U+4??; }`,
		`<!-- .x { color: red; } -->`,
		`.x { color: var(--x, fallback); }`,
	}
	for _, css := range cases {
		t.Run(css, func(t *testing.T) { parseNoErrors(t, css) })
	}
}

func TestUnitTable(t *testing.T) {
	units := []string{"px", "cm", "mm", "in", "pt", "pc", "Q", "em", "ex", "ch", "cap", "ic", "lh", "rem", "rex", "rch", "rcap", "ric", "rlh", "vw", "vh", "vi", "vb", "vmin", "vmax", "svw", "svh", "svi", "svb", "svmin", "svmax", "lvw", "lvh", "lvi", "lvb", "lvmin", "lvmax", "dvw", "dvh", "dvi", "dvb", "dvmin", "dvmax", "cqw", "cqh", "cqi", "cqb", "cqmin", "cqmax", "fr", "deg", "rad", "grad", "turn", "s", "ms", "Hz", "kHz", "dpi", "dpcm", "dppx", "x"}
	for _, unit := range units {
		t.Run(unit, func(t *testing.T) {
			ss := parseNoErrors(t, `.x { width: 1`+unit+`; }`)
			decl := ss.Rules[0].(*Selector).Rules[0].(*Declaration)
			if got := string(decl.Value[0].(*BasicValue).Value); got != "1"+unit {
				t.Fatalf("value = %q", got)
			}
		})
	}
}

func TestCustomPropertyRawRoundTrip(t *testing.T) {
	ss := parseNoErrors(t, `:root { --shadow: 1px   solid rgba(0, 0, 0, .5); --tokens: { a:   b }; }`)
	rules := ss.Rules[0].(*Selector).Rules
	if got := string(rules[0].(*Declaration).Value[0].(*BasicValue).Value); got != "1px   solid rgba(0, 0, 0, .5)" {
		t.Fatalf("custom property raw = %q", got)
	}
	if got := string(rules[1].(*Declaration).Value[0].(*BasicValue).Value); got != "{ a:   b }" {
		t.Fatalf("balanced custom property raw = %q", got)
	}
}

func TestSourceMutationAfterParseChangesASTViews(t *testing.T) {
	source := []byte(`.x { --token: red; }`)
	toks := lexer.Lex(source)
	ss, errs := Parse(source, toks)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	value := ss.Rules[0].(*Selector).Rules[0].(*Declaration).Value[0].(*BasicValue).Value
	copy(source[14:17], []byte("blu"))
	if got := string(value); got != "blu" {
		t.Fatalf("AST value after source mutation = %q, want blu", got)
	}
}

func TestDimensionSpans(t *testing.T) {
	ss := parseNoErrors(t, `.x { margin: 1.5rem -4px 10%; }`)
	values := ss.Rules[0].(*Selector).Rules[0].(*Declaration).Value
	want := []string{"1.5rem", "-4px", "10%"}
	if len(values) != len(want) {
		t.Fatalf("values = %d, want %d", len(values), len(want))
	}
	for i, want := range want {
		if got := string(values[i].(*BasicValue).Value); got != want {
			t.Fatalf("value %d = %q, want %q", i, got, want)
		}
	}
}
