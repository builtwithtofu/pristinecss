package pristinecss

import "testing"

func TestRecoverySelectorTokenBoundaries(t *testing.T) {
	for _, selector := range []string{".a", "#a", ":hover", "a"} {
		suffix := "b"
		if selector == "a" {
			suffix = "1"
		}
		src := selector + "/*x*/" + suffix + "{color:red}"
		s, _ := Parse([]byte(src))
		want := selector + "/**/" + suffix + "{color:red}"
		if got := string(s.Emit(nil, EmitOptions{})); got != want {
			t.Errorf("%q: got %q want %q", src, got, want)
		}
	}
}

func TestRecoveryTrailingRule(t *testing.T) {
	for _, newline := range []string{"\n", "\r", "\f", "\r\n"} {
		for _, prefix := range []string{"a{x:\"a", "a{x:url(\"a"} {
			in := errorInput{src: []byte(prefix + newline + "b)}c{color:red}"), wantRule: "c"}
			if got := inspectErrorInput(in); len(got.details) > 0 {
				t.Error(got.details)
			}
		}
	}
}

func TestRecoveryEmptyReplacement(t *testing.T) {
	s, _ := Parse([]byte(".x{color:red;zoom:1}"))
	for id, n := range s.nodes {
		if n.Kind == KindDeclaration {
			s.ReplaceRaw(NodeID(id), nil)
			break
		}
	}
	if got := string(s.Emit(nil, EmitOptions{})); got != ".x{zoom:1}" {
		t.Fatal(got)
	}
}

func TestRecoveryPrettySpacing(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"a, b{}", "a, b {\n}\n"},
		{"a{x:f(1, 2)}", "a {\n  x: f(1, 2)\n}\n"},
		// The ';' stays so the newline after it cannot join the \9 escape.
		{"a{color:red\\9}", "a {\n  color: red\\9;\n}\n"},
	} {
		s, _ := Parse([]byte(tt.in))
		if got := string(s.Emit(nil, EmitOptions{Style: Pretty})); got != tt.want {
			t.Errorf("%q: got %q want %q", tt.in, got, tt.want)
		}
	}
}

// css-syntax-3: a name followed by whitespace and "(" is not a function, so
// the whitespace must survive emission.
func TestRecoverySpaceBeforeParen(t *testing.T) {
	for _, src := range []string{"a{b:foo (1)}", "a{b:url (x)}", "a:not (b){}"} {
		s, _ := Parse([]byte(src))
		if got := string(s.Emit(nil, EmitOptions{})); got != src {
			t.Errorf("%q: got %q", src, got)
		}
	}
}

// A dropped comment must not glue a name to the "(" after it: that would make
// a function the source did not have.
func TestRecoveryCommentBeforeParen(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{":not/**/( .a){}", ":not/**/( .a){}"},
		{":is(.good,:not/**/(.bad)){color:red}", ":is(.good,:not/**/(.bad)){color:red}"},
		{"a/**/(b){}", "a/**/(b){}"},
	} {
		s, _ := Parse([]byte(tt.in))
		if got := string(s.Emit(nil, EmitOptions{})); got != tt.want {
			t.Errorf("%q: got %q want %q", tt.in, got, tt.want)
		}
		if got := inspectErrorInput(errorInput{src: []byte(tt.in)}); len(got.details) > 0 {
			t.Errorf("%q: %v", tt.in, got.details)
		}
	}
}

// A CRLF that ends a hex escape belongs to the escape, so the whitespace after
// it still separates tokens.
func TestRecoveryCRLFEndsEscape(t *testing.T) {
	for _, src := range []string{"a{x:\\61\r\n ( )}", "\\61\r\n .b{}", "\\61\r .b{}", "\\61\f .b{}"} {
		s, _ := Parse([]byte(src))
		if got := string(s.Emit(nil, EmitOptions{})); got != src {
			t.Errorf("%q: got %q", src, got)
		}
		if got := inspectErrorInput(errorInput{src: []byte(src)}); len(got.details) > 0 {
			t.Errorf("%q: %v", src, got.details)
		}
	}
}

// "url(" inside a longer name is a function name, not the start of a URL.
func TestRecoveryURLInsideEscapedName(t *testing.T) {
	for _, src := range []string{`a{--x:(\@url(x ")"))}`, `a{--x:(\\url(x ")"))}`, `a{--x:(\61 url(x ")"))}`, "a{--x:(\\61\r\nurl(x \")\"))}", "a{b:\"\\61\r\nurl(\"}"} {
		s, _ := Parse([]byte(src))
		if got := string(s.Emit(nil, EmitOptions{})); got != src {
			t.Errorf("%q: got %q", src, got)
		}
		if got := inspectErrorInput(errorInput{src: []byte(src)}); len(got.details) > 0 {
			t.Errorf("%q: %v", src, got.details)
		}
	}
}

// Comments are not tokens: ":", a comment and a name are still a pseudo-class,
// and "." with a comment before the name is still a class.
func TestRecoveryCommentInsideSimpleSelector(t *testing.T) {
	for _, tt := range []struct {
		src  string
		kind Kind
	}{{":/**/not(.a){}", KindSelPseudo}, {":/**/A( 0){}", KindSelPseudo}, {"./**/a{}", KindSelClass}} {
		s, _ := Parse([]byte(tt.src))
		if got := s.Nodes()[2]; got.Kind != tt.kind || got.Hi != uint32(len(tt.src)-2) {
			t.Errorf("%q: first selector node = %v [%d:%d]", tt.src, got.Kind, got.Lo, got.Hi)
		}
		if got := inspectErrorInput(errorInput{src: []byte(tt.src)}); len(got.details) > 0 {
			t.Errorf("%q: %v", tt.src, got.details)
		}
	}
}

// A comment before the first component of a selector argument is not a
// descendant combinator, and every node must stay inside its parent.
func TestRecoveryLeadingCommentInSelectorArgument(t *testing.T) {
	for _, src := range []string{"a {color:red}:is(/*x*/b){}", ":is(/**/ .a){}"} {
		s, _ := Parse([]byte(src))
		if _, err := Load(s.Marshal(true), nil); err != nil {
			t.Errorf("%q: Load(Marshal) = %v", src, err)
		}
		if got := inspectErrorInput(errorInput{src: []byte(src)}); len(got.details) > 0 {
			t.Errorf("%q: %v", src, got.details)
		}
	}
}
