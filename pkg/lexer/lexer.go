package lexer

import (
	"bytes"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

const (
	EOF         = 0
	SEMICOLON   = ';'
	COMMA       = ','
	LPAREN      = '('
	RPAREN      = ')'
	LBRACE      = '{'
	RBRACE      = '}'
	LBRACKET    = '['
	RBRACKET    = ']'
	EQUALS      = '='
	PLUS        = '+'
	GREATER     = '>'
	LESS        = '<'
	TILDE       = '~'
	PIPE        = '|'
	CARET       = '^'
	AMPERSAND   = '&'
	PERCENTAGE  = '%'
	DOLLAR      = '$'
	EXCLAMATION = '!'
	AT          = '@'
	ASTERISK    = '*'
	COLON       = ':'
	DOT         = '.'
	HASH        = '#'
	DASH        = '-'
	BACKSLASH   = '\\'
	SLASH       = '/'
	DOUBLEQUOTE = '"'
	SINGLEQUOTE = '\''
)

var (
	isWhitespace [256]bool
	isLetter     [256]bool
	isDigit      [256]bool
	isIdentStart [256]bool
	isIdentPart  [256]bool
	isHexDigit   [256]bool
)

func init() {
	for i := 0; i < 256; i++ {
		ch := byte(i)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_' || ch >= 0x80
		isDigit[i] = '0' <= ch && ch <= '9'
		isIdentStart[i] = isLetter[i] || ch == '_' || ch >= 0x80
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		isHexDigit[i] = isDigit[i] || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
	}
}

// estimateTokenCount estimates the number of tokens based on input size
func estimateTokenCount(inputSize int) int {
	return inputSize/3 + 16
}

type lexer struct {
	input        []byte
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func Lex(input []byte) []tokens.Token {
	l := lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()

	return l.tokenize()
}

func (l *lexer) Erase() {
	l.input = l.input[:0]
	l.position = 0
	l.readPosition = 0
	l.ch = 0
	l.line = 1
	l.column = 0
}

func (l *lexer) tokenize() []tokens.Token {
	estimatedTokens := estimateTokenCount(len(l.input))
	result := make([]tokens.Token, 0, estimatedTokens)
	for {
		tok := l.nextToken()
		result = append(result, tok)
		if tok.Type == tokens.EOF {
			break
		}
	}

	return result
}

func (l *lexer) nextToken() tokens.Token {
	l.skipWhitespace()
	for l.skipCDOCDC() {
		l.skipWhitespace()
	}
	tok := tokens.Token{
		Line:   uint32(l.line),
		Column: uint32(l.column),
	}
	start := l.position

	if l.ch == EOF {
		tok.Type = tokens.EOF
		tok.Start = uint32(start)
		tok.End = uint32(start)
		return tok
	}

	switch l.ch {
	case SEMICOLON:
		tok.Type = tokens.SEMICOLON
	case COMMA:
		tok.Type = tokens.COMMA
	case AMPERSAND:
		tok.Type = tokens.AMPERSAND
	case LPAREN:
		tok.Type = tokens.LPAREN
	case RPAREN:
		tok.Type = tokens.RPAREN
	case LBRACE:
		tok.Type = tokens.LBRACE
	case RBRACE:
		tok.Type = tokens.RBRACE
	case LBRACKET:
		tok.Type = tokens.LBRACKET
	case RBRACKET:
		tok.Type = tokens.RBRACKET
	case EQUALS:
		tok.Type = tokens.EQUALS
	case PLUS:
		tok.Type = tokens.PLUS
	case GREATER:
		tok.Type = tokens.GREATER
	case LESS:
		tok.Type = tokens.LESS
	case TILDE:
		tok.Type = tokens.TILDE
	case PIPE:
		tok.Type = tokens.PIPE
	case CARET:
		if l.peekChar() == EQUALS {
			l.readChar()
			tok.Type = tokens.STARTS_WITH
		} else {
			tok.Type = tokens.ILLEGAL
		}
	case PERCENTAGE:
		tok.Type = tokens.PERCENTAGE
	case DOLLAR:
		tok.Type = tokens.DOLLAR
	case EXCLAMATION:
		tok.Type = tokens.EXCLAMATION
	case AT:
		tok.Type = tokens.AT
	case ASTERISK:
		tok.Type = tokens.ASTERISK
	case COLON:
		if l.peekChar() == COLON {
			l.readChar()
			tok.Type = tokens.DBLCOLON
		} else {
			tok.Type = tokens.COLON
		}
	case DOT:
		if isDigit[l.peekChar()] {
			tok.Type = tokens.NUMBER
			l.readNumber()
		} else {
			tok.Type = tokens.DOT
		}
	case HASH:
		tok.Type = l.readHashOrColor()
	case DASH:
		tok.Type = l.handleDash()
	case BACKSLASH:
		tok.Type = tokens.IDENT
		l.readIdentifier()
	case SLASH:
		tok.Type = l.handleSlash()
	case DOUBLEQUOTE, SINGLEQUOTE:
		tok.Type = tokens.STRING
		l.readString()
	case 'u', 'U':
		if l.matchKeyword("url(") {
			tok.Type = tokens.URI
			l.readChar() // consume '('
			if l.matchKeyword("data:") {
				l.readDataURI()
			} else {
				l.readURL()
			}
		} else {
			tok.Type = tokens.IDENT
			l.readIdentifier()
		}
	default:
		if isLetter[l.ch] {
			tok.Type = tokens.IDENT
			l.readIdentifier()
		} else if isDigit[l.ch] {
			tok.Type = tokens.NUMBER
			l.readNumber()
		} else {
			tok.Type = tokens.ILLEGAL
		}
	}

	if l.ch != EOF {
		l.readChar()
	}

	end := l.position
	tok.Start = uint32(start)
	tok.End = uint32(end)

	return tok
}

func (l *lexer) skipCDOCDC() bool {
	if l.ch == '<' && l.peekChar() == '!' && l.peekNextChar() == '-' && l.peekThirdChar() == '-' {
		for i := 0; i < 4; i++ {
			l.readChar()
		}
		return true
	}
	if l.ch == '-' && l.peekChar() == '-' && l.peekNextChar() == '>' {
		for i := 0; i < 3; i++ {
			l.readChar()
		}
		return true
	}
	return false
}

func (l *lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = EOF
	} else {
		l.column++
	}
}

func (l *lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *lexer) peekNextChar() byte {
	if l.readPosition+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+1]
}

func (l *lexer) peekThirdChar() byte {
	if l.readPosition+2 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+2]
}

func (l *lexer) handleSlash() tokens.TokenType {
	if l.peekChar() == '*' {
		l.readChar() // consume '*'
		l.readComment()
		return tokens.COMMENT
	}
	return tokens.DIVIDE
}

func (l *lexer) handleDash() tokens.TokenType {
	if l.peekChar() == '-' {
		l.readChar() // consume second '-'
		return l.readCustomProperty()
	} else if isDigit[l.peekChar()] {
		l.readNumber()
		return tokens.NUMBER
	} else if isWhitespace[l.peekChar()] {
		return tokens.MINUS
	} else if isIdentStart[l.peekChar()] || l.peekChar() == '\\' {
		l.readChar() // consume next char
		l.readIdentifier()
		return tokens.IDENT
	}
	return tokens.MINUS
}

func (l *lexer) readString() {
	delimiter := l.ch
	for l.peekChar() != delimiter && l.peekChar() != 0 && l.peekChar() != '\n' {
		if l.peekChar() == '\\' {
			l.readChar() // consume '\'
			if l.peekChar() == delimiter {
				l.readChar() // consume escaped quote
			}
		}
		l.readChar()
	}
	if l.peekChar() == delimiter {
		l.readChar() // consume closing quote
	}
}

func (l *lexer) readNumber() {
	for isDigit[l.peekChar()] {
		l.readChar()
	}
	if l.peekChar() == '.' && isDigit[l.peekNextChar()] {
		l.readChar() // consume '.'
		for isDigit[l.peekChar()] {
			l.readChar()
		}
	}
}

func (l *lexer) readIdentifier() {
	if l.ch == '\\' {
		l.readEscapedChar()
	}
	for isIdentPart[l.peekChar()] || l.peekChar() == '-' || l.peekChar() == '\\' {
		if l.peekChar() == '\\' {
			l.readChar() // consume '\'
			l.readEscapedChar()
		} else {
			l.readChar()
		}
	}
}

func (l *lexer) readEscapedChar() {
	if isHexDigit[l.peekChar()] {
		hexChars := 0
		for isHexDigit[l.peekChar()] && hexChars < 6 {
			l.readChar()
			hexChars++
		}
		if l.peekChar() == ' ' {
			l.readChar()
		}
	} else if l.peekChar() != '\n' {
		l.readChar()
	}
}

func (l *lexer) readHashOrColor() tokens.TokenType {
	colorLength := 0
	start := l.position

	for isHexDigit[l.peekChar()] && colorLength < 6 {
		l.readChar()
		colorLength++
	}

	if (colorLength == 3 || colorLength == 6) &&
		(!isIdentPart[l.peekChar()] || l.peekChar() == 0) {
		return tokens.COLOR
	}

	// If it's not a valid color, treat it as a HASH
	l.position = start // Reset position to just after the '#'
	l.readPosition = start + 1
	l.ch = '#'
	return tokens.HASH
}

func (l *lexer) readCustomProperty() tokens.TokenType {
	for isIdentPart[l.peekChar()] || l.peekChar() == '-' {
		l.readChar()
	}
	return tokens.IDENT
}

func (l *lexer) readComment() {
	for {
		l.readChar()
		if l.ch == 0 { // EOF
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume '/'
			break
		}
	}
}

func (l *lexer) readURL() {
	// Skip whitespace after 'url('
	l.skipWhitespace()
	// Read "url"
	for i := 0; i < 3; i++ {
		l.readChar()
	}

	// Check if the URL is quoted
	if l.ch == '"' || l.ch == '\'' {
		quote := l.ch
		l.readChar() // consume opening quote
		for l.ch != quote && l.ch != EOF {
			if l.ch == '\\' {
				l.readChar() // consume backslash
				if l.ch != EOF {
					l.readChar() // consume escaped character
				}
			} else {
				l.readChar()
			}
		}
		if l.ch == quote {
			l.readChar() // consume closing quote
		}
	} else {
		// Unquoted URL
		parenCount := 1
		for parenCount > 0 && l.ch != EOF {
			if l.ch == '(' {
				parenCount++
			} else if l.ch == ')' {
				parenCount--
			} else {
				l.readChar()
			}
		}
	}

	// Skip whitespace before closing paren
	l.skipWhitespace()
}

func (l *lexer) readDataURI() {
	// Read "data:"
	for i := 0; i < 5; i++ {
		l.readChar()
	}

	// Read MIME type and encoding
	for l.ch != ',' && l.ch != EOF {
		l.readChar()
	}

	// Read the actual data
	if l.ch == ',' {
		l.readChar() // consume comma
		for l.ch != ')' && l.ch != EOF {
			if l.ch == '%' {
				// Handle percent-encoding
				l.readChar()
				l.readChar()
				l.readChar()
			} else {
				l.readChar()
			}
		}
	}
}

func (l *lexer) matchKeyword(keyword string) bool {
	if len(l.input[l.position:]) < len(keyword) {
		return false
	}
	return bytes.EqualFold(l.input[l.position:l.position+len(keyword)], []byte(keyword))
}

func (l *lexer) skipWhitespace() {
	for isWhitespace[l.ch] {
		l.readChar()
	}
}
