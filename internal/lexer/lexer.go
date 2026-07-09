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
		if l.matchKeyword("url(") {
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
	for l.peekChar() != delimiter && l.peekChar() != 0 && l.peekChar() != '\n' {
		if l.peekChar() == backslash {
			l.readChar()
			if l.peekChar() == delimiter {
				l.readChar()
			}
		}
		l.readChar()
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
		if l.peekChar() == ' ' {
			l.readChar()
		}
	} else if l.peekChar() != '\n' {
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
func (l *scanner) readURL() {
	for i := 0; i < len("url("); i++ {
		l.readChar()
	}
	depth := 1
	var quote byte
	for l.ch != endOfFile {
		if quote != 0 {
			if l.ch == backslash {
				l.readChar()
				if l.ch != endOfFile {
					l.readChar()
				}
				continue
			}
			if l.ch == quote {
				quote = 0
			}
			l.readChar()
			continue
		}
		switch l.ch {
		case '"', '\'':
			quote = l.ch
			l.readChar()
		case backslash:
			l.readChar()
			if l.ch != endOfFile {
				l.readChar()
			}
		case '(':
			depth++
			l.readChar()
		case ')':
			depth--
			if depth == 0 {
				return
			}
			l.readChar()
		default:
			l.readChar()
		}
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
