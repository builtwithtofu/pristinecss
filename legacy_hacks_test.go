package pristinecss

import (
	"reflect"
	"strings"
	"testing"
)

func TestLegacyHacksRemainRaw(t *testing.T) {
	for _, hack := range []string{"*display:inline;", "#zoom:1;", "#zoom: 1;"} {
		for _, suffix := range []string{"", "zoom:1"} {
			src := ".x{" + hack + suffix + "}"
			t.Run(src, func(t *testing.T) {
				s, errs := Parse([]byte(src))
				if len(errs) != 0 {
					t.Fatalf("parse errors: %v", errs)
				}
				verify(t, s)
				var raw []string
				for _, n := range s.nodes {
					if n.Kind == KindRawStatement {
						raw = append(raw, string(s.nodeBytes(n)))
					}
				}
				wantRaw := strings.TrimSuffix(hack, ";")
				if !reflect.DeepEqual(raw, []string{wantRaw}) {
					t.Fatalf("raw nodes = %q, want %q", raw, []string{wantRaw})
				}
				want := src
				if suffix == "" {
					want = ".x{" + wantRaw + "}"
				}
				if got := string(s.Emit(nil, EmitOptions{})); got != want {
					t.Fatalf("emitted = %q, want %q", got, want)
				}
				assertStructuralRoundTrip(t, s)
			})
		}
	}
}

func TestValidBlockContentsRemainStructured(t *testing.T) {
	for _, tc := range []struct {
		src      string
		kinds    []Kind
		property string
	}{
		{".a{_height:32px}", []Kind{KindDeclaration}, "_height"},
		{".a{*{color:red}}", []Kind{KindStyleRule}, "color"},
		{".a{:hover{color:red}}", []Kind{KindStyleRule}, "color"},
		{".a{& .b{color:red}}", []Kind{KindStyleRule}, "color"},
		{".a{*{color:red}height:32px}", []Kind{KindStyleRule, KindDeclaration}, "height"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			s, errs := Parse([]byte(tc.src))
			if len(errs) != 0 {
				t.Fatalf("parse errors: %v", errs)
			}
			verify(t, s)
			block, ok := s.blockChild(1)
			if !ok {
				t.Fatal("missing style block")
			}
			var kinds []Kind
			for id, end := block+1, block+NodeID(s.nodes[block].Sub); id < end; id += NodeID(s.nodes[id].Sub) {
				kinds = append(kinds, s.nodes[id].Kind)
			}
			if !reflect.DeepEqual(kinds, tc.kinds) {
				t.Fatalf("block children = %v, want %v", kinds, tc.kinds)
			}
			var property string
			for id, n := range s.nodes {
				if n.Kind == KindValRaw || n.Kind == KindRawStatement {
					t.Fatal("valid CSS parsed as raw")
				}
				if n.Kind == KindDeclaration {
					property = string((Cursor{s, NodeID(id)}).PropertyName())
				}
			}
			if property != tc.property {
				t.Fatalf("last declaration property = %q, want %q", property, tc.property)
			}
			if got := string(s.Emit(nil, EmitOptions{})); got != tc.src {
				t.Fatalf("emitted = %q, want %q", got, tc.src)
			}
			assertStructuralRoundTrip(t, s)
		})
	}
}

func TestLegacyHackSeparators(t *testing.T) {
	for _, tc := range []struct {
		src    string
		unwrap Kind
		want   string
	}{
		{"*display:inline}.a{color:red}", 0, "*display:inline;.a{color:red}"},
		{"@media print{*display:inline}.a{color:red}", KindAtMedia, "*display:inline;.a{color:red}"},
		{".a{.b{*display:inline}color:red}", KindStyleRule, ".a{*display:inline;color:red}"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			s, _ := Parse([]byte(tc.src))
			if tc.unwrap != 0 {
				for c := range s.All(tc.unwrap) {
					if tc.unwrap != KindStyleRule || string(s.nodeBytes(s.nodes[c.ID()])) == ".b{*display:inline}" {
						s.Unwrap(c.ID())
					}
				}
			}
			if got := string(s.Emit(nil, EmitOptions{})); got != tc.want {
				t.Fatalf("emitted = %q, want %q", got, tc.want)
			}
			s.Compact()
			assertStructuralRoundTrip(t, s)
		})
	}
}

func TestLegacyHacksPretty(t *testing.T) {
	s, _ := Parse([]byte(".x{*display:inline;zoom:1}"))
	want := ".x {\n  *display:inline;\n  zoom: 1\n}\n"
	if got := string(s.Emit(nil, EmitOptions{Style: Pretty})); got != want {
		t.Fatalf("pretty = %q, want %q", got, want)
	}
}

func TestDescendantAfterFunctionalPseudoComment(t *testing.T) {
	s, errs := Parse([]byte(":not(.a /*x*/) .c{color:red}"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	assertStructuralRoundTrip(t, s)
	if got := string(s.Emit(nil, EmitOptions{})); got != ":not(.a) .c{color:red}" {
		t.Fatalf("emitted = %q", got)
	}
}

func TestTopLevelLegacyHacksIR(t *testing.T) {
	for _, tc := range []struct {
		src   string
		spans []string
	}{
		{"*display:inline;", []string{"*display:inline;"}},
		{"*display:inline}.a{color:red}", []string{"*display:inline;", ".a{color:red}"}},
		{".a{color:red}*display:inline;.b{color:blue}", []string{".a{color:red}", "*display:inline;", ".b{color:blue}"}},
	} {
		t.Run(tc.src, func(t *testing.T) {
			s, _ := Parse([]byte(tc.src))
			blob, err := s.CompileIR(nil)
			if err != nil {
				t.Fatal(err)
			}
			ir, err := LoadIR(blob)
			if err != nil {
				t.Fatal(err)
			}
			var spans []string
			for _, r := range ir.Rules {
				spans = append(spans, string(ir.CSS[r.Lo:r.Hi]))
			}
			if !reflect.DeepEqual(spans, tc.spans) {
				t.Fatalf("IR spans = %q, want %q", spans, tc.spans)
			}
			want := string(s.Emit(nil, EmitOptions{}))
			if got := string(FilterIR(ir, [4]uint64{}, nil)); got != want {
				t.Fatalf("IR = %q, want %q", got, want)
			}
		})
	}
}

func TestReplaceRawStatementSeparators(t *testing.T) {
	for _, tc := range []struct {
		name, src, replacement, minified, pretty string
	}{
		{"block", ".x{*display:inline;zoom:1}", "*display:block", ".x{*display:block;zoom:1}", ".x {\n  *display:block;\n  zoom: 1\n}\n"},
		{"top level", "*zoom:1;.a{b:c}", "*zoom:2", "*zoom:2;.a{b:c}", "*zoom:2;\n.a {\n  b: c\n}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, errs := Parse([]byte(tc.src))
			if len(errs) != 0 {
				t.Fatal(errs)
			}
			var raw NodeID = NoNode
			for c := range s.All(KindRawStatement) {
				raw = c.ID()
				break
			}
			if raw == NoNode {
				t.Fatal("missing raw statement")
			}
			s.ReplaceRaw(raw, []byte(tc.replacement))
			for _, format := range []struct {
				name  string
				style Style
				want  string
			}{{"minified", Minified, tc.minified}, {"pretty", Pretty, tc.pretty}} {
				t.Run(format.name, func(t *testing.T) {
					if got := string(s.Emit(nil, EmitOptions{Style: format.style})); got != format.want {
						t.Fatalf("emitted = %q, want %q", got, format.want)
					}
				})
			}
		})
	}
}

func TestExtractRawCustomPropertyValue(t *testing.T) {
	for _, replace := range []bool{false, true} {
		s, errs := Parse([]byte(".x{--y:foo;zoom:1}"))
		if len(errs) != 0 {
			t.Fatal(errs)
		}
		var value NodeID = NoNode
		for c := range s.All(KindValRaw) {
			value = c.ID()
			break
		}
		if value == NoNode {
			t.Fatal("missing raw value")
		}
		want := "foo"
		if replace {
			want = "bar"
			s.ReplaceRaw(value, []byte(want))
		}
		extracted := s.Extract(value)
		for _, style := range []Style{Minified, Pretty} {
			if got := string(extracted.Emit(nil, EmitOptions{Style: style})); got != want {
				t.Errorf("replace=%v: extracted value = %q, want %q", replace, got, want)
			}
		}
	}
}
