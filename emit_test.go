package pristinecss

import "testing"

func TestEmitMinifiedPrettyAndVerbatim(t *testing.T) {
	tests := []struct {
		name, in string
		style    Style
		want     string
	}{
		{"minified comments", "/*! keep */\na { color : red ; margin: 0 ; } /* drop */", Minified, "/*! keep */a{color:red;margin:0}"},
		{"pretty pseudo", "a:hover{color:red}", Pretty, "a:hover {\n  color: red\n}\n"},
		{"pretty data uri", "b{background:url(data:image/svg+xml,x y)}", Pretty, "b {\n  background: url(data:image/svg+xml,x y)\n}\n"},
		{"pretty nested", "@media (min-width:400px){a:hover{color:red} b{background:url(data:image/svg+xml,x y)}}", Pretty, "@media (min-width:400px) {\n  a:hover {\n    color: red\n  }\n  b {\n    background: url(data:image/svg+xml,x y)\n  }\n}\n"},
		{"custom tab style", "a{color:red}", Style{Indent: []byte("\t"), Newline: []byte("\n"), AfterColon: []byte(" "), BeforeBrace: []byte(" "), AfterBrace: []byte("\n")}, "a {\n\tcolor: red\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := Parse([]byte(tt.in))
			if len(errs) != 0 {
				t.Fatal(errs)
			}
			if got := string(s.Emit(nil, EmitOptions{Style: tt.style})); got != tt.want {
				t.Fatalf("emit = %q, want %q", got, tt.want)
			}
		})
	}
	src := []byte("/*! keep */\na { color : red ; margin: 0 ; } /* drop */")
	s, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if got := string(s.Emit(nil, EmitOptions{Verbatim: true})); got != string(src) {
		t.Fatalf("verbatim = %q", got)
	}
}

func TestEmitFilterAndOnRule(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}@media print{b{display:none}}c{color:blue}"))
	filtered := string(s.Emit(nil, EmitOptions{Filter: func(c Cursor) bool { return c.Kind() != KindAtMedia }}))
	for c := range s.All(KindAtMedia) {
		s.nodes[c.ID()].Flags |= FlagTombstone
	}
	deleted := string(s.Emit(nil, EmitOptions{}))
	if filtered != deleted {
		t.Fatalf("filter %q != delete %q", filtered, deleted)
	}
	type span struct{ lo, hi int }
	var spans []span
	out := s.Emit(nil, EmitOptions{OnRule: func(id NodeID, lo, hi int) { spans = append(spans, span{lo, hi}) }})
	if len(spans) != 2 {
		t.Fatalf("spans = %#v", spans)
	}
	for _, sp := range spans {
		css := string(out[sp.lo:sp.hi])
		one, errs := Parse([]byte(css))
		if len(errs) != 0 || countKind(one, KindStyleRule)+countKind(one, KindAtMedia) != 1 {
			t.Fatalf("bad OnRule span %q errs=%v", css, errs)
		}
	}
}

func TestEmitWarmBufferNoAllocs(t *testing.T) {
	s, _ := Parse(readFramework(t, "bootstrap.css"))
	dst := make([]byte, 0, len(s.src))
	allocs := testing.AllocsPerRun(50, func() { dst = s.Emit(dst[:0], EmitOptions{}) })
	if allocs != 0 {
		t.Fatalf("Emit allocs = %.0f, want 0", allocs)
	}
}

func TestEmitMutatedPreludeChild(t *testing.T) {
	tests := []struct {
		name, css string
		feature   Kind
		want      string
	}{
		{"media", "@media screen and (min-width:400px){a{color:red}}", KindMediaFeature, "@media screen and (max-width:600px){a{color:red}}"},
		{"container", "@container card (inline-size > 30em){a{color:red}}", KindContainerFeature, "@container card (max-width:600px){a{color:red}}"},
		{"supports", "@supports (display:grid){a{color:red}}", KindMediaFeature, "@supports (max-width:600px){a{color:red}}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := Parse([]byte(tt.css))
			var feature NodeID = NoNode
			for c := range s.All(tt.feature) {
				feature = c.ID()
			}
			if feature == NoNode {
				t.Fatal("missing prelude feature")
			}
			s.ReplaceRaw(feature, []byte("(max-width:600px)"))
			if got := string(s.Emit(nil, EmitOptions{})); got != tt.want {
				t.Fatalf("emit = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerbatimEmitHonorsMutations(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}"))
	var value NodeID
	for c := range s.All(KindValBasic) {
		value = c.ID()
	}
	s.ReplaceRaw(value, []byte("blue"))
	if got := string(s.Emit(nil, EmitOptions{Verbatim: true})); got != "a{color:blue}" {
		t.Fatalf("verbatim mutated emit = %q", got)
	}
}

func TestReplacementSpacingUsesSourceSpans(t *testing.T) {
	s, _ := Parse([]byte("a{border:1px solid black}"))
	var black NodeID
	for c := range s.All(KindValBasic) {
		if string(c.Text()) == "black" {
			black = c.ID()
		}
	}
	s.ReplaceRaw(black, []byte("red"))
	if got := string(s.Emit(nil, EmitOptions{})); got != "a{border:1px solid red}" {
		t.Fatalf("emit = %q", got)
	}
}

func countKind(s *Sheet, k Kind) (n int) {
	for _, node := range s.nodes {
		if node.Kind == k {
			n++
		}
	}
	return n
}
