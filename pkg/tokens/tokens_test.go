package tokens

import "testing"

func TestTokenTypeString(t *testing.T) {
	tests := map[TokenType]string{
		ILLEGAL:     "ILLEGAL",
		EOF:         "EOF",
		COMMENT:     "COMMENT",
		IDENT:       "IDENT",
		NUMBER:      "NUMBER",
		STRING:      "STRING",
		COLOR:       "COLOR",
		URI:         "URI",
		STARTS_WITH: "^=",
		DBLCOLON:    "::",
		AT:          "AT",
		COLON:       ":",
		SEMICOLON:   ";",
		COMMA:       ",",
		DOT:         ".",
		HASH:        "#",
		ASTERISK:    "*",
		PLUS:        "+",
		MINUS:       "-",
		DIVIDE:      "/",
		GREATER:     ">",
		LESS:        "<",
		TILDE:       "~",
		EQUALS:      "=",
		PIPE:        "|",
		CARET:       "^",
		PERCENTAGE:  "%",
		DOLLAR:      "$",
		AMPERSAND:   "&",
		EXCLAMATION: "!",
		LPAREN:      "(",
		RPAREN:      ")",
		LBRACKET:    "[",
		RBRACKET:    "]",
		LBRACE:      "{",
		RBRACE:      "}",
	}
	for tokenType, want := range tests {
		if got := tokenType.String(); got != want {
			t.Fatalf("TokenType(%d).String() = %q, want %q", tokenType, got, want)
		}
	}
}

func TestTokenLiteralReadsSourceSpan(t *testing.T) {
	source := []byte(".button { color: red }")
	tok := Token{Type: IDENT, Start: 1, End: 7, Line: 1, Column: 2}
	if got := string(tok.Literal(source)); got != "button" {
		t.Fatalf("Literal() = %q, want button", got)
	}

	source[1] = 'B'
	if got := string(tok.Literal(source)); got != "Button" {
		t.Fatalf("Literal() after source mutation = %q, want Button", got)
	}
}
