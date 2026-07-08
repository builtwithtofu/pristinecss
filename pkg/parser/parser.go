package parser

import (
	"fmt"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ParseError struct {
	Message string
	Line    uint32
	Column  uint32
	Token   tokens.Token
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s (token: %s)\n", e.Line, e.Column, e.Message, e.Token.Type)
}

func Parse(source []byte, toks []tokens.Token) (*Stylesheet, []ParseError) {
	arena := NewArena(len(toks))
	return ParseInto(source, toks, arena)
}

func ParseInto(source []byte, toks []tokens.Token, arena *Arena) (*Stylesheet, []ParseError) {
	if arena == nil {
		arena = NewArena(len(toks))
	} else {
		arena.Reset(len(toks))
	}
	stylesheet := &Stylesheet{}
	visitor := NewParseVisitor(source, toks, arena)
	visitStylesheet(visitor, stylesheet)
	return stylesheet, visitor.errors
}

type ParseVisitor struct {
	source       []byte
	tokens       []tokens.Token
	position     int
	currentToken *tokens.Token
	nextToken    *tokens.Token
	eofToken     tokens.Token
	arena        *Arena
	errors       []ParseError
	blockDepth   int
}

func NewParseVisitor(source []byte, toks []tokens.Token, arena *Arena) *ParseVisitor {
	pv := &ParseVisitor{
		source:   source,
		tokens:   toks,
		position: 0,
		arena:    arena,
		errors:   make([]ParseError, 0),
		eofToken: tokens.Token{Type: tokens.EOF},
	}
	pv.advance() // Load the first token
	pv.advance() // Load the second token (now in nextToken)
	return pv
}

func (pv *ParseVisitor) literal(tok *tokens.Token) []byte {
	if tok == nil {
		return nil
	}
	return tok.Literal(pv.source)
}

func (pv *ParseVisitor) currentLiteral() []byte {
	return pv.literal(pv.currentToken)
}

func (pv *ParseVisitor) nextLiteral() []byte {
	return pv.literal(pv.nextToken)
}

func (pv *ParseVisitor) tokenEndColumn(tok tokens.Token) int {
	return int(tok.Column) + int(tok.End-tok.Start)
}

func (pv *ParseVisitor) sourceSpan(start, end int) []byte {
	if start < 0 || end < start || start > len(pv.source) {
		return nil
	}
	if end > len(pv.source) {
		end = len(pv.source)
	}
	return pv.source[start:end]
}

func (pv *ParseVisitor) advance() {
	pv.currentToken = pv.nextToken
	if pv.position < len(pv.tokens) {
		pv.nextToken = &pv.tokens[pv.position]
		pv.position++
	} else {
		pv.nextToken = &pv.eofToken
	}
}

func (pv *ParseVisitor) currentTokenIs(tokenType tokens.TokenType) bool {
	return pv.currentToken != nil && pv.currentToken.Type == tokenType
}

func (pv *ParseVisitor) nextTokenIs(tokenType tokens.TokenType) bool {
	return pv.nextToken != nil && pv.nextToken.Type == tokenType
}

func (pv *ParseVisitor) consume(tokenType tokens.TokenType, errorMessage string) bool {
	if pv.currentTokenIs(tokenType) {
		pv.advance()
		return true
	}
	pv.addError(errorMessage, pv.currentToken)
	return false
}

func (pv *ParseVisitor) skipRuleBoundaryTokens() {
	for pv.currentTokenIs(tokens.CDO) || pv.currentTokenIs(tokens.CDC) || pv.currentTokenIs(tokens.SEMICOLON) {
		pv.advance()
	}
}

func (pv *ParseVisitor) addError(message string, token *tokens.Token) {
	if token == nil {
		token = &pv.eofToken
	}
	pv.errors = append(pv.errors, ParseError{
		Message: message,
		Line:    token.Line,
		Column:  token.Column,
		Token:   *token,
	})
}

func (pv *ParseVisitor) progressMark() int {
	return pv.position
}

func (pv *ParseVisitor) ensureProgress(mark int, context string) bool {
	if pv.position != mark || pv.currentTokenIs(tokens.EOF) {
		return true
	}
	pv.addError(fmt.Sprintf("Parser made no progress while parsing %s", context), pv.currentToken)
	pv.advance()
	return false
}

func (pv *ParseVisitor) skipToNextRule() {
	for !pv.currentTokenIs(tokens.EOF) && !pv.currentTokenIs(tokens.RBRACE) {
		pv.advance()
	}
	if pv.currentTokenIs(tokens.RBRACE) {
		pv.advance() // Consume the '}'
	}
}

func (pv *ParseVisitor) skipToNextSemicolonOrBrace() {
	for !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		pv.advance()
	}
	if pv.currentTokenIs(tokens.SEMICOLON) {
		pv.advance() // Consume the semicolon
	}
}

func (pv *ParseVisitor) skipCurrentBlock() {
	depth := 0
	for !pv.currentTokenIs(tokens.EOF) {
		switch pv.currentToken.Type {
		case tokens.LBRACE:
			depth++
		case tokens.RBRACE:
			if depth == 0 {
				return
			}
			depth--
		}
		pv.advance()
	}
}
