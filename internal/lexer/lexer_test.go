package lexer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"
)

func TestTokenSize(t *testing.T) {
	if got := unsafe.Sizeof(Token{}); got != 12 {
		t.Fatalf("Token size = %d, want 12 (<=16, no strings or line/column)", got)
	}
}

func TestLexTokenStreamsAndSpans(t *testing.T) {
	tests := []struct {
		in   string
		want []struct {
			typ Type
			lit string
		}
	}{
		{"<!-- a{} -->", []struct {
			typ Type
			lit string
		}{{CDO, "<!--"}, {Ident, "a"}, {LBrace, "{"}, {RBrace, "}"}, {CDC, "-->"}, {EOF, ""}}},
		{".a#b[href^=\"x\"]::before{color:#fff;margin:-.5px 10px;}", []struct {
			typ Type
			lit string
		}{{Dot, "."}, {Ident, "a"}, {Hash, "#"}, {Ident, "b"}, {LBracket, "["}, {Ident, "href"}, {StartsWith, "^="}, {String, "\"x\""}, {RBracket, "]"}, {DblColon, "::"}, {Ident, "before"}, {LBrace, "{"}, {Ident, "color"}, {Colon, ":"}, {Color, "#fff"}, {Semicolon, ";"}, {Ident, "margin"}, {Colon, ":"}, {Minus, "-"}, {Number, ".5"}, {Ident, "px"}, {Number, "10"}, {Ident, "px"}, {Semicolon, ";"}, {RBrace, "}"}, {EOF, ""}}},
		{"@media screen and (width>=10px){a{background:url(data:image/svg+xml,%3Csvg%3E)}}", []struct {
			typ Type
			lit string
		}{{At, "@"}, {Ident, "media"}, {Ident, "screen"}, {Ident, "and"}, {LParen, "("}, {Ident, "width"}, {Greater, ">"}, {Equals, "="}, {Number, "10"}, {Ident, "px"}, {RParen, ")"}, {LBrace, "{"}, {Ident, "a"}, {LBrace, "{"}, {Ident, "background"}, {Colon, ":"}, {URI, "url(data:image/svg+xml,%3Csvg%3E)"}, {RBrace, "}"}, {RBrace, "}"}, {EOF, ""}}},
	}
	for _, tt := range tests {
		toks := Lex([]byte(tt.in))
		if len(toks) != len(tt.want) {
			t.Fatalf("%q len=%d want %d: %#v", tt.in, len(toks), len(tt.want), toks)
		}
		for i, tok := range toks {
			if tok.Type != tt.want[i].typ || string(tok.Literal([]byte(tt.in))) != tt.want[i].lit {
				t.Fatalf("%q token %d = %s %q, want %s %q", tt.in, i, tok.Type, tok.Literal([]byte(tt.in)), tt.want[i].typ, tt.want[i].lit)
			}
		}
	}
}

func TestHasSpaceBefore(t *testing.T) {
	src := []byte(".a.b .c/*x*/.d 10px solid 10px) --custom: a b")
	toks := Lex(src)
	for _, check := range []struct {
		lit    string
		spaced bool
	}{
		{".", false}, {"b", false}, {".", true}, {".", true}, {"10", true}, {"px", false}, {"solid", true}, {")", false}, {"--custom", true}, {"a", true}, {"b", true},
	} {
		found := false
		for _, tok := range toks {
			if string(tok.Literal(src)) == check.lit && ((tok.Flags&HasSpaceBefore) != 0) == check.spaced {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no token %q with HasSpaceBefore=%v in %#v", check.lit, check.spaced, toks)
		}
	}
}

func TestLexBootstrapOneAllocation(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "test-data", "frameworks", "bootstrap.css"))
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(100, func() { _ = Lex(src) })
	if allocs != 1 {
		t.Fatalf("Lex bootstrap allocs = %.0f, want 1", allocs)
	}
}

func TestLexNestedUnquotedURLReturns(t *testing.T) {
	src := []byte(`a{background:url(foo(bar).png);mask:url("foo)bar.svg");x:url(data:image/svg+xml,x y)}`)
	done := make(chan []Token, 1)
	go func() { done <- Lex(src) }()
	select {
	case toks := <-done:
		var urls []string
		for _, tok := range toks {
			if tok.Type == URI {
				urls = append(urls, string(tok.Literal(src)))
			}
		}
		foundString := false
		for i, tok := range toks {
			if tok.Type == String && string(tok.Literal(src)) == `"foo)bar.svg"` {
				foundString = i >= 2 && toks[i-1].Type == LParen && toks[i-2].Type == Ident && string(toks[i-2].Literal(src)) == "url" && toks[i+1].Type == RParen
			}
		}
		if !foundString {
			t.Fatal("quoted URL must be a function with an ordinary string")
		}
		want := []string{`url(foo(bar)`, `url(data:image/svg+xml,x y)`}
		if len(urls) != len(want) {
			t.Fatalf("URL tokens = %q, want %q", urls, want)
		}
		for i := range want {
			if urls[i] != want[i] {
				t.Fatalf("URL token %d = %q, want %q", i, urls[i], want[i])
			}
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Lex did not return for a nested unquoted url()")
	}
}

// css-syntax-3: CR, FF and CRLF are newlines; a newline ends a string and
// one whitespace (CRLF as one) after a hex escape belongs to the escape.
func TestLexNewlinesAndEscapeWhitespace(t *testing.T) {
	tests := []struct{ in, first string }{
		{"\"a\rb\"", "\"a"},
		{"\"a\fb\"", "\"a"},
		{"\"a\r\nb\"", "\"a"},
		{"\"a\\\r\nb\"", "\"a\\\r\nb\""},
		{"a\\9\fb", "a\\9\fb"},
		{"a\\9\r\nb", "a\\9\r\nb"},
		{"a\\9\tb", "a\\9\tb"},
		{"a\\9\n\nb", "a\\9\n"},
		{"\"a\\61\r\nb\"", "\"a\\61\r\nb\""},
		{"\"a\\61\fb\"", "\"a\\61\fb\""},
	}
	for _, tt := range tests {
		src := []byte(tt.in)
		if got := string(Lex(src)[0].Literal(src)); got != tt.first {
			t.Errorf("Lex(%q)[0] = %q, want %q", tt.in, got, tt.first)
		}
	}
}

// css-syntax-3: "0url(" is a dimension with unit "url" followed by "(", not a URL.
func TestLexURLAfterNumberIsUnit(t *testing.T) {
	for _, src := range []string{"0url(x)", ".5url(x)"} {
		for _, tok := range Lex([]byte(src)) {
			if tok.Type == URI {
				t.Errorf("Lex(%q) produced URL token %q", src, tok.Literal([]byte(src)))
			}
		}
	}
}
