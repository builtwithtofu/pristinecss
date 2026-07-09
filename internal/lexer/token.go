package lexer

// Type identifies one CSS token class.
type Type uint8

const (
	Illegal Type = iota
	EOF

	Comment
	Ident
	Number
	String
	Color
	URI

	CDO
	CDC
	StartsWith
	DblColon

	At
	Colon
	Semicolon
	Comma
	Dot
	Hash
	Asterisk
	Plus
	Minus
	Divide
	Greater
	Less
	Tilde
	Equals
	Pipe
	Caret
	Percentage
	Dollar
	Ampersand
	Exclamation

	LParen
	RParen
	LBracket
	RBracket
	LBrace
	RBrace
)

const (
	// HasSpaceBefore reports whether whitespace or a comment was skipped before this token.
	HasSpaceBefore uint8 = 1 << iota
)

// Token is an offset-only CSS token. It is pointer-free and currently packs to 12 bytes.
type Token struct {
	Type       Type
	Flags      uint8
	Start, End uint32
}

// Literal returns the token's source bytes.
func (t Token) Literal(src []byte) []byte {
	if int(t.Start) < 0 || t.End < t.Start || int(t.Start) > len(src) {
		return nil
	}
	end := int(t.End)
	if end > len(src) {
		end = len(src)
	}
	return src[t.Start:uint32(end)]
}

func (t Type) String() string {
	switch t {
	case Illegal:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case Comment:
		return "COMMENT"
	case Ident:
		return "IDENT"
	case Number:
		return "NUMBER"
	case String:
		return "STRING"
	case Color:
		return "COLOR"
	case URI:
		return "URI"
	case CDO:
		return "<!--"
	case CDC:
		return "-->"
	case StartsWith:
		return "^="
	case DblColon:
		return "::"
	case At:
		return "AT"
	case Colon:
		return ":"
	case Semicolon:
		return ";"
	case Comma:
		return ","
	case Dot:
		return "."
	case Hash:
		return "#"
	case Asterisk:
		return "*"
	case Plus:
		return "+"
	case Minus:
		return "-"
	case Divide:
		return "/"
	case Greater:
		return ">"
	case Less:
		return "<"
	case Tilde:
		return "~"
	case Equals:
		return "="
	case Pipe:
		return "|"
	case Caret:
		return "^"
	case Percentage:
		return "%"
	case Dollar:
		return "$"
	case Ampersand:
		return "&"
	case Exclamation:
		return "!"
	case LParen:
		return "("
	case RParen:
		return ")"
	case LBracket:
		return "["
	case RBracket:
		return "]"
	case LBrace:
		return "{"
	case RBrace:
		return "}"
	default:
		return "Type(?)"
	}
}
