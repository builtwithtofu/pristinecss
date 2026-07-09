package pristinecss

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConformanceConstructs(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want []Kind
	}{
		{"declaration values", "a{color:red!important; margin:calc(1px + 2px)}", []Kind{KindStyleRule, KindDeclaration, KindValBasic, KindDeclaration, KindValFunction}},
		{"custom property raw", "a{--x: { token soup; } ; color:blue}", []Kind{KindDeclaration, KindValRaw}},
		{"pseudo rule", "a:has(.x)>b{color:red}", []Kind{KindStyleRule, KindSelPseudo, KindSelClass, KindSelCombinator, KindBlock, KindDeclaration}},
		{"media", "@media (min-width:400px) and (hover:hover){a{color:red}}", []Kind{KindAtMedia, KindMediaQuery, KindMediaFeature, KindStyleRule}},
		{"container", "@container card (inline-size > 30em){a{color:red}}", []Kind{KindAtContainer, KindContainerQuery, KindContainerFeature, KindStyleRule}},
		{"generic parsed block", "@web screen { a{ color:red } @tui { b{display:none} } }", []Kind{KindAtGeneric, KindPreludeRaw, KindBlock, KindStyleRule, KindAtGeneric}},
		{"keyframes", "@keyframes fade{0%{opacity:0}to{opacity:1}}", []Kind{KindAtKeyframes, KindPreludeRaw, KindBlock, KindKeyframeBlock}},
		{"page margin box", "@page{@top-left{content:\"x\"}}", []Kind{KindAtPage, KindMarginBox, KindDeclaration}},
		{"selector full spans", ".a#b svg|a::before{color:red}", []Kind{KindSelClass, KindSelID, KindSelNamespace, KindSelPseudo}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := Parse([]byte(tt.css))
			if len(errs) != 0 {
				t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
			}
			verify(t, s)
			for _, k := range tt.want {
				if !hasKind(s, k) {
					t.Fatalf("missing %s in\n%s", k, dumpString(s))
				}
			}
		})
	}
}

func hasKind(s *Sheet, k Kind) bool {
	for _, n := range s.nodes {
		if n.Kind == k {
			return true
		}
	}
	return false
}

func TestSelectorSpansAndReplacement(t *testing.T) {
	s, errs := Parse([]byte(".old{color:red}"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	var class NodeID = NoNode
	for c := range s.All(KindSelClass) {
		class = c.ID()
	}
	if class == NoNode {
		t.Fatal("missing class selector")
	}
	if got := string(s.nodeBytes(s.nodes[class])); got != ".old" {
		t.Fatalf("class span = %q", got)
	}
	s.ReplaceRaw(class, []byte(".new"))
	if got := string(s.Emit(nil, EmitOptions{})); got != ".new{color:red}" {
		t.Fatalf("selector replacement emit = %q", got)
	}
}

func TestSelectorAndCustomPropertyTokenPreservation(t *testing.T) {
	tests := []struct{ name, css, want string }{
		{"nth formula", "li:nth-child(2n+1){color:red}", "li:nth-child(2n+1){color:red}"},
		{"hex-looking id", "#abc{color:red}", "#abc{color:red}"},
		{"empty namespace", "|title{color:red}", "|title{color:red}"},
		{"column combinator", "span||em{color:red}", "span||em{color:red}"},
		{"named namespace", "svg|a{color:red}", "svg|a{color:red}"},
		{"custom property delimiters", "a{--x: func(;); color:red}", "a{--x:func(;);color:red}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := Parse([]byte(tt.css))
			if len(errs) != 0 {
				t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
			}
			verify(t, s)
			if got := string(s.Emit(nil, EmitOptions{})); got != tt.want {
				t.Fatalf("emit = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeclarationValueBoundaries(t *testing.T) {
	tests := []struct{ name, css, want string }{
		{"important inside function", "a{color:var(--fallback,!important)}", "a{color:var(--fallback,!important)}"},
		{"semicolon inside function", "a{x:func(;);color:red}", "a{x:func(;);color:red}"},
		{"trailing important", "a{color:red!important}", "a{color:red!important}"},
		{"commented important", "a{color:red!/*x*/important}", "a{color:red!important}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := Parse([]byte(tt.css))
			if len(errs) != 0 {
				t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
			}
			verify(t, s)
			if got := string(s.Emit(nil, EmitOptions{})); got != tt.want {
				t.Fatalf("emit = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAtRulePreludeTokenPreservation(t *testing.T) {
	tests := []string{
		"@supports (display:grid) and (color:red){a{color:red}}",
		"@supports not (display:grid){a{display:block}}",
		"@supports selector(:has(>img)){a{display:block}}",
		"@media screen and (min-width:400px){a{color:red}}",
		"@container card (inline-size > 30em){a{color:red}}",
	}
	for _, css := range tests {
		t.Run(css, func(t *testing.T) {
			s, errs := Parse([]byte(css))
			if len(errs) != 0 {
				t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
			}
			verify(t, s)
			if got := string(s.Emit(nil, EmitOptions{})); got != css {
				t.Fatalf("emit = %q, want %q", got, css)
			}
		})
	}
}

func TestMediaFeatureChildren(t *testing.T) {
	s, errs := Parse([]byte("@media (min-width:400px) and (hover:hover){a{color:red}}"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	features := 0
	for c := range s.All(KindMediaFeature) {
		features++
		if !bytes.HasPrefix(c.Text(), []byte("(")) {
			t.Fatalf("media feature span = %q", c.Text())
		}
	}
	if features != 2 {
		t.Fatalf("media features = %d, want 2\n%s", features, dumpString(s))
	}
}

func TestKindReachability(t *testing.T) {
	src := []byte(`/*!bang*/ svg|a#id.class[attr^="x"]:has(.x) > *, &::before { color: rgb(1,2,3); content: 'x'; --raw: { token soup; }; }
@charset "UTF-8"; @import url(x); @media (min-width:1px),(hover:hover){a{color:red}} @container card (inline-size > 30em){a{color:red}}
@font-face{font-family:x} @font-feature-values x{} @counter-style x{} @color-profile x{} @keyframes x{from{opacity:0}}
@supports (display:grid){a{display:grid}} @layer a,b; @scope (.x){a{color:red}} @starting-style{a{opacity:0}} @property --x{syntax:"*"}
@font-palette-values x{} @namespace svg url(x); @page{@top-left{content:"x"}} @position-try --x{} @view-transition{navigation:auto} @web{a{color:red}}`)
	s, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
	}
	for k := Kind(1); k < kindCount; k++ {
		if !hasKind(s, k) {
			t.Fatalf("kind %s was not produced\n%s", k, dumpString(s))
		}
	}
}

func TestParserBehaviorPortedCases(t *testing.T) {
	cases := []string{
		"div{color:blue}", ".highlight{background-color:yellow}", "#main{font-size:16px}", "[type='text']{border:1px solid gray}",
		"div.container{max-width:1200px}", "h1,h2,h3{font-family:sans-serif}", "article p{line-height:1.5}", "ul>li{list-style-type:square}",
		".form-select:not([multiple]):not([size]){padding-right:1.2rem}", "a:hover{color:red}", "p::first-line{font-weight:bold}", "a:has(.x)>b{color:red}",
		"div{color:red!important}", "div{margin:10px 20px 30px 40px}", ".colors{color:#ff0000;background:#00ff00;border-color:#0000ff}",
		"div{background-image:url('image.jpg')}", ".icon{background-image:url(\"test.svg\") /*rtl:url(\"test-rtl.svg\")*/}",
		"@font-face{font-family:\"Open Sans\";src:url(\"/fonts/OpenSans.woff2\") format(\"woff2\")}", "@charset \"UTF-8\";", "@import url(theme.css) layer(base);",
		"@keyframes slide-in{from{transform:translateX(-100%)}to{transform:translateX(0)}}", "@keyframes multi{0%,100%{opacity:0}50%{opacity:1}}",
		"@media (400px <= width <= 700px){body{font-size:14px}}", "@supports (display:grid) and (not (display:flex)){.grid{display:grid}}", "@container card (inline-size > 30em){.card{display:block}}",
		".parent{&:hover{color:red}}", "@page{@top-left{content:\"x\"}}", "@web{a{color:red}}",
	}
	for _, css := range cases {
		t.Run(css, func(t *testing.T) {
			s, errs := Parse([]byte(css))
			if len(errs) != 0 {
				t.Fatalf("errors: %#v\n%s", errs, dumpString(s))
			}
			verify(t, s)
			assertStructuralRoundTrip(t, s)
		})
	}
}

func TestConformanceFixtures(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("test-data", "conformance"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".css" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("test-data", "conformance", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			s, errs := Parse(src)
			if len(errs) != 0 {
				t.Fatalf("errors: %#v", errs[:min(len(errs), 5)])
			}
			verify(t, s)
			assertStructuralRoundTrip(t, s)
		})
	}
}

func assertStructuralRoundTrip(t testing.TB, s *Sheet) {
	t.Helper()
	out := s.Emit(make([]byte, 0, len(s.src)), EmitOptions{})
	reparsed, errs := Parse(out)
	if len(errs) != 0 {
		t.Fatalf("reparse errors: %#v\n%s", errs[:min(len(errs), 5)], out[:min(len(out), 200)])
	}
	if a, b := canonicalSheet(s), canonicalSheet(reparsed); a != b {
		t.Fatalf("structural round-trip mismatch at %s\nemitted: %s", firstDiff(a, b), out[:min(len(out), 200)])
	}
}

func canonicalSheet(s *Sheet) string {
	var b strings.Builder
	var rec func(NodeID)
	rec = func(id NodeID) {
		n := s.nodes[id]
		if n.Kind == KindComment && !bytes.HasPrefix(s.nodeBytes(n), []byte("/*!")) {
			return
		}
		flags := n.Flags & (FlagImportant | FlagCustom | FlagSingleQuote)
		text := canonicalText(s, id)
		b.WriteString(n.Kind.String())
		b.WriteByte('|')
		b.WriteString(flags.String())
		b.WriteByte('|')
		b.WriteString(text)
		b.WriteByte('[')
		if !n.Kind.isOpaque() {
			end := id + NodeID(n.Sub)
			for c := id + 1; c < end; c += NodeID(s.nodes[c].Sub) {
				rec(c)
			}
		}
		b.WriteByte(']')
	}
	rec(0)
	return b.String()
}

func canonicalText(s *Sheet, id NodeID) string {
	n := s.nodes[id]
	if n.Kind.isAtRule() {
		if n.Kind == KindAtGeneric {
			return string(Cursor{s, id}.Name())
		}
		return ""
	}
	switch n.Kind {
	case KindStylesheet, KindStyleRule, KindBlock, KindKeyframeBlock, KindMediaQuery, KindSupportsCond, KindContainerQuery:
		return ""
	case KindMarginBox:
		if int(n.Lo+1+uint32(n.Aux)) <= len(s.src) {
			return string(s.src[n.Lo+1 : n.Lo+1+uint32(n.Aux)])
		}
		return ""
	case KindDeclaration:
		return string(Cursor{s, id}.PropertyName())
	case KindValFunction:
		return string(Cursor{s, id}.FunctionName())
	case KindSelPseudo:
		return string(Cursor{s, id}.Name())
	}
	text := string(s.nodeBytes(n))
	if n.Kind == KindSelCombinator && strings.TrimSpace(text) == "" {
		return " "
	}
	return strings.Join(strings.Fields(text), " ")
}

func (f Flags) String() string { return string(rune('0' + f)) }

func firstDiff(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	sa := max(0, i-120)
	ea := min(len(a), i+240)
	eb := min(len(b), i+240)
	return "offset " + string(rune('0'+min(i, 9))) + "\noriginal: " + a[sa:ea] + "\nreparsed: " + b[sa:eb]
}
