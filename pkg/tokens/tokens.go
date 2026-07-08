package tokens

type TokenType uint8

const (
	ILLEGAL TokenType = iota
	EOF

	// Identifiers + literals
	COMMENT
	IDENT
	NUMBER
	STRING
	COLOR
	URI

	// Constructs
	CDO
	CDC
	STARTS_WITH
	DBLCOLON

	// Symbols and operators
	AT
	COLON
	SEMICOLON
	COMMA
	DOT
	HASH
	ASTERISK
	PLUS
	MINUS
	DIVIDE
	GREATER
	LESS
	TILDE
	EQUALS
	PIPE
	CARET
	PERCENTAGE
	DOLLAR
	AMPERSAND
	EXCLAMATION

	// Brackets
	LPAREN
	RPAREN
	LBRACKET
	RBRACKET
	LBRACE
	RBRACE
)

func (tt TokenType) String() string {
	switch tt {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case COMMENT:
		return "COMMENT"
	case IDENT:
		return "IDENT"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case COLOR:
		return "COLOR"
	case URI:
		return "URI"
	case CDO:
		return "<!--"
	case CDC:
		return "-->"
	case STARTS_WITH:
		return "^="
	case DBLCOLON:
		return "::"
	case AT:
		return "AT"
	case COLON:
		return ":"
	case SEMICOLON:
		return ";"
	case COMMA:
		return ","
	case DOT:
		return "."
	case HASH:
		return "#"
	case ASTERISK:
		return "*"
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case DIVIDE:
		return "/"
	case GREATER:
		return ">"
	case LESS:
		return "<"
	case TILDE:
		return "~"
	case EQUALS:
		return "="
	case PIPE:
		return "|"
	case CARET:
		return "^"
	case PERCENTAGE:
		return "%"
	case DOLLAR:
		return "$"
	case AMPERSAND:
		return "&"
	case EXCLAMATION:
		return "!"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACKET:
		return "["
	case RBRACKET:
		return "]"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	default:
		return "TokenType(?)"
	}
}

type Token struct {
	Type   TokenType
	Start  uint32
	End    uint32
	Line   uint32
	Column uint32
}

func (t Token) Literal(source []byte) []byte {
	start := int(t.Start)
	end := int(t.End)
	if start < 0 || end < start || start > len(source) {
		return nil
	}
	if end > len(source) {
		end = len(source)
	}
	return source[start:end]
}

// Token needs to implement the Erasable interface
func (t *Token) Erase() {
	t.Type = ILLEGAL
	t.Start = 0
	t.End = 0
	t.Line = 0
	t.Column = 0
}

func NewToken() *Token {
	return &Token{}
}

func (t *Token) Reset() {
	t.Type = ILLEGAL
	t.Start = 0
	t.End = 0
	t.Line = 0
	t.Column = 0
}
