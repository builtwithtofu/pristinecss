package lexer

import "bytes"

const (
	endOfFile   = 0
	semicolon   = ';'
	comma       = ','
	lparen      = '('
	rparen      = ')'
	lbrace      = '{'
	rbrace      = '}'
	lbracket    = '['
	rbracket    = ']'
	equals      = '='
	plus        = '+'
	greater     = '>'
	less        = '<'
	tilde       = '~'
	pipe        = '|'
	caret       = '^'
	ampersand   = '&'
	percentage  = '%'
	dollar      = '$'
	exclamation = '!'
	at          = '@'
	asterisk    = '*'
	colon       = ':'
	dot         = '.'
	hash        = '#'
	dash        = '-'
	backslash   = '\\'
	slash       = '/'
	doubleQuote = '"'
	singleQuote = '\''
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

func estimateTokenCount(inputSize int) int { return inputSize/3 + 16 }

type scanner struct {
	input        []byte
	position     int
	readPosition int
	ch           byte
	spaceBefore  bool
}

// Lex tokenizes CSS into offset-only tokens. The returned slice owns one backing array.
func Lex(input []byte) []Token { return LexInto(input, nil) }

// LexInto tokenizes CSS into dst after reslicing dst to zero for caller buffer reuse.
func LexInto(input []byte, dst []Token) []Token {
	l := scanner{input: input}
	l.readChar()
	if cap(dst) < estimateTokenCount(len(input)) {
		dst = make([]Token, 0, estimateTokenCount(len(input)))
	} else {
		dst = dst[:0]
	}
	for {
		tok := l.nextToken()
		dst = append(dst, tok)
		if tok.Type == EOF {
			return dst
		}
	}
}

// Splits reports whether input lexes into separate tokens at offset at: no
// token crosses it, and no name ends there with "(" after it, which would make
// a function. It does not allocate.
func Splits(input []byte, at int) bool {
	l := scanner{input: input}
	l.readChar()
	ident := false
	for {
		tok := l.nextToken()
		start, end := int(tok.Start), int(tok.End)
		if tok.Type == EOF || start > at {
			return true
		}
		if start < at && end > at {
			return false
		}
		if start == at {
			return !(ident && tok.Type == LParen)
		}
		ident = end == at && tok.Type == Ident
	}
}

func (l *scanner) nextToken() Token {
	l.skipWhitespaceOnly()
	start := l.position
	tok := Token{}
	if l.spaceBefore {
		tok.Flags |= HasSpaceBefore
		l.spaceBefore = false
	}

	if l.ch == endOfFile {
		tok.Type = EOF
		tok.Start = uint32(start)
		tok.End = uint32(start)
		return tok
	}

	switch l.ch {
	case semicolon:
		tok.Type = Semicolon
	case comma:
		tok.Type = Comma
	case ampersand:
		tok.Type = Ampersand
	case lparen:
		tok.Type = LParen
	case rparen:
		tok.Type = RParen
	case lbrace:
		tok.Type = LBrace
	case rbrace:
		tok.Type = RBrace
	case lbracket:
		tok.Type = LBracket
	case rbracket:
		tok.Type = RBracket
	case equals:
		tok.Type = Equals
	case plus:
		tok.Type = Plus
	case greater:
		tok.Type = Greater
	case less:
		if l.peekChar() == '!' && l.peekNextChar() == '-' && l.peekThirdChar() == '-' {
			l.readChar()
			l.readChar()
			l.readChar()
			tok.Type = CDO
		} else {
			tok.Type = Less
		}
	case tilde:
		tok.Type = Tilde
	case pipe:
		tok.Type = Pipe
	case caret:
		if l.peekChar() == equals {
			l.readChar()
			tok.Type = StartsWith
		} else {
			tok.Type = Caret
		}
	case percentage:
		tok.Type = Percentage
	case dollar:
		tok.Type = Dollar
	case exclamation:
		tok.Type = Exclamation
	case at:
		tok.Type = At
	case asterisk:
		tok.Type = Asterisk
	case colon:
		if l.peekChar() == colon {
			l.readChar()
			tok.Type = DblColon
		} else {
			tok.Type = Colon
		}
	case dot:
		if isDigit[l.peekChar()] {
			tok.Type = Number
			l.readNumber()
		} else {
			tok.Type = Dot
		}
	case hash:
		tok.Type = l.readHashOrColor()
	case dash:
		tok.Type = l.handleDash()
	case backslash:
		tok.Type = Ident
		l.readIdentifier()
	case slash:
		if l.peekChar() == '*' {
			l.readChar()
			l.readComment()
			tok.Type = Comment
		} else {
			tok.Type = Divide
		}
	case doubleQuote, singleQuote:
		tok.Type = String
		l.readString()
	case 'u', 'U':
		// After a digit, "url" is a dimension's unit and "(" opens a block.
		if l.matchKeyword("url(") && !l.quotedURL() && (l.position == 0 || !isDigit[l.input[l.position-1]]) {
			tok.Type = URI
			l.readURL()
		} else {
			tok.Type = Ident
			l.readIdentifier()
		}
	default:
		if isLetter[l.ch] {
			tok.Type = Ident
			l.readIdentifier()
		} else if isDigit[l.ch] {
			tok.Type = Number
			l.readNumber()
		} else {
			tok.Type = Illegal
		}
	}

	if l.ch != endOfFile {
		l.readChar()
	}
	tok.Start = uint32(start)
	tok.End = uint32(l.position)
	if tok.Type == Comment {
		l.spaceBefore = true
	}
	return tok
}

func (l *scanner) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = endOfFile
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}
func (l *scanner) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}
func (l *scanner) peekNextChar() byte {
	if l.readPosition+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+1]
}
func (l *scanner) peekThirdChar() byte {
	if l.readPosition+2 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+2]
}

func (l *scanner) handleDash() Type {
	if l.peekChar() == '-' {
		if l.peekNextChar() == '>' {
			l.readChar()
			l.readChar()
			return CDC
		}
		l.readChar()
		return l.readCustomProperty()
	} else if isDigit[l.peekChar()] {
		l.readNumber()
		return Number
	} else if isWhitespace[l.peekChar()] {
		return Minus
	} else if isIdentStart[l.peekChar()] || l.peekChar() == backslash {
		l.readChar()
		l.readIdentifier()
		return Ident
	}
	return Minus
}

func (l *scanner) readString() {
	delimiter := l.ch
	for l.peekChar() != delimiter && l.peekChar() != 0 && !isNewline(l.peekChar()) {
		l.readChar()
		if l.ch == backslash && isNewline(l.peekChar()) {
			l.readChar()
			if l.ch == '\r' && l.peekChar() == '\n' {
				l.readChar()
			}
		} else if l.ch == backslash && l.peekChar() != 0 {
			l.readEscapedChar()
		}
	}
	if l.peekChar() == delimiter {
		l.readChar()
	}
}
func (l *scanner) readNumber() {
	for isDigit[l.peekChar()] {
		l.readChar()
	}
	if l.peekChar() == '.' && isDigit[l.peekNextChar()] {
		l.readChar()
		for isDigit[l.peekChar()] {
			l.readChar()
		}
	}
}
func (l *scanner) readIdentifier() {
	if l.ch == backslash {
		l.readEscapedChar()
	}
	for isIdentPart[l.peekChar()] || l.peekChar() == '-' || l.peekChar() == backslash {
		if l.peekChar() == backslash {
			l.readChar()
			l.readEscapedChar()
		} else {
			l.readChar()
		}
	}
}
func (l *scanner) readEscapedChar() {
	if isHexDigit[l.peekChar()] {
		for n := 0; isHexDigit[l.peekChar()] && n < 6; n++ {
			l.readChar()
		}
		if isWhitespace[l.peekChar()] {
			l.readChar()
			if l.ch == '\r' && l.peekChar() == '\n' {
				l.readChar()
			}
		}
	} else if !isNewline(l.peekChar()) {
		l.readChar()
	}
}
func (l *scanner) readHashOrColor() Type {
	colorLength, start := 0, l.position
	for isHexDigit[l.peekChar()] && colorLength < 6 {
		l.readChar()
		colorLength++
	}
	if (colorLength == 3 || colorLength == 6) && (!isIdentPart[l.peekChar()] || l.peekChar() == 0) {
		return Color
	}
	l.position = start
	l.readPosition = start + 1
	l.ch = '#'
	return Hash
}
func (l *scanner) readCustomProperty() Type {
	for isIdentPart[l.peekChar()] || l.peekChar() == '-' {
		l.readChar()
	}
	return Ident
}
func (l *scanner) readComment() {
	for {
		l.readChar()
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			break
		}
	}
}
func isNewline(c byte) bool { return c == '\n' || c == '\r' || c == '\f' }

func (l *scanner) quotedURL() bool {
	i := l.position + len("url(")
	for i < len(l.input) && isWhitespace[l.input[i]] {
		i++
	}
	return i < len(l.input) && (l.input[i] == '"' || l.input[i] == '\'')
}
func (l *scanner) readURL() {
	for i := 0; i < len("url("); i++ {
		l.readChar()
	}
	// Both URL tokens and bad-URL remnants end at the first unescaped ')'.
	for l.ch != endOfFile && l.ch != ')' {
		if l.ch == backslash && l.peekChar() != endOfFile {
			l.readEscapedChar()
		}
		l.readChar()
	}
}
func (l *scanner) matchKeyword(keyword string) bool {
	return len(l.input[l.position:]) >= len(keyword) && bytes.EqualFold(l.input[l.position:l.position+len(keyword)], []byte(keyword))
}
func (l *scanner) skipWhitespaceOnly() {
	for isWhitespace[l.ch] {
		l.spaceBefore = true
		l.readChar()
	}
}
func (l *scanner) skipWhitespaceAndComments() {
	for {
		for isWhitespace[l.ch] {
			l.spaceBefore = true
			l.readChar()
		}
		if l.ch == slash && l.peekChar() == '*' {
			l.spaceBefore = true
			l.readChar()
			l.readComment()
			if l.ch != endOfFile {
				l.readChar()
			}
			continue
		}
		return
	}
}
