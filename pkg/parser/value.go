package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ValueType int

const (
	Basic ValueType = iota
	String
	Function
	ValueComment
)

type Value interface {
	Node
	ValueType() ValueType
}

var _ Value = (*BasicValue)(nil)
var _ Value = (*StringValue)(nil)
var _ Value = (*FunctionValue)(nil)

type BasicValue struct {
	Value []byte
}

func (bv *BasicValue) ValueType() ValueType { return Basic }
func (bv *BasicValue) Type() NodeType       { return NodeValue }
func (bv *BasicValue) String() string {
	return fmt.Sprintf("BasicValue{Value: %q}", string(bv.Value))
}

type StringValue struct {
	SingleQuote bool
	Value       []byte
}

func (bv *StringValue) ValueType() ValueType { return String }
func (bv *StringValue) Type() NodeType       { return NodeValue }
func (bv *StringValue) String() string {
	return fmt.Sprintf("StringValue{SingleQuote: %v, Value: %q}", bv.SingleQuote, string(bv.Value))
}

type FunctionValue struct {
	Name      []byte
	Arguments []Value
}

func (fv *FunctionValue) ValueType() ValueType { return Function }
func (fv *FunctionValue) Type() NodeType       { return NodeValue }
func (fv *FunctionValue) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FunctionValue{Name: %q, Arguments: [", string(fv.Name)))
	for i, arg := range fv.Arguments {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.String())
	}
	sb.WriteString("]}")
	return sb.String()
}

func (pv *ParseVisitor) parseValue() Value {
	switch pv.currentToken.Type {
	case tokens.NUMBER:
		return pv.parseNumberValue()
	case tokens.IDENT:
		if pv.isUnicodeRangeStart(*pv.currentToken, *pv.nextToken) {
			return pv.parseUnicodeRangeValue()
		}
		if pv.nextTokenIs(tokens.LPAREN) {
			fv := pv.arena.newFunctionValue()
			visitFunctionValue(pv, fv)
			return fv
		}
		bv := pv.arena.newBasicValue()
		bv.Value = pv.currentLiteral()
		pv.advance()
		return bv
	case tokens.URI:
		value := pv.parseURLValue()
		pv.advance()
		return value
	case tokens.STRING:
		sv := pv.arena.newStringValue()
		visitStringValue(pv, sv)
		return sv
	default:
		bv := pv.arena.newBasicValue()
		bv.Value = pv.currentLiteral()
		pv.advance()
		return bv
	}
}

func visitStringValue(pv *ParseVisitor, node Node) {
	sv := node.(*StringValue)
	str := pv.currentLiteral()
	if len(str) >= 2 {
		sv.SingleQuote = str[0] == '\''
		sv.Value = str[1 : len(str)-1]
	}
	pv.advance()
}

func visitFunctionValue(pv *ParseVisitor, node Node) {
	fv := node.(*FunctionValue)
	fv.Name = pv.currentLiteral()
	pv.advance()
	pv.advance()
	for !pv.currentTokenIs(tokens.RPAREN) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.COMMA) {
			pv.advance()
			pv.ensureProgress(mark, "function arguments")
			continue
		}
		fv.Arguments = pv.arena.appendValue(fv.Arguments, pv.parseValue())
		pv.ensureProgress(mark, "function arguments")
	}
	pv.consume(tokens.RPAREN, "Expected ')' to close function")
}

func (pv *ParseVisitor) parseNumberValue() Value {
	numberToken := *pv.currentToken
	number := pv.currentLiteral()
	pv.advance()
	if (pv.currentTokenIs(tokens.PERCENTAGE) || isUnit(pv.currentLiteral())) && adjacentToken(numberToken, *pv.currentToken) {
		end := int(pv.currentToken.End)
		start := int(numberToken.Start)
		if start <= end && end <= len(pv.source) {
			number = pv.source[start:end]
		}
		pv.advance()
	}
	bv := pv.arena.newBasicValue()
	bv.Value = number
	return bv
}

func (pv *ParseVisitor) isUnicodeRangeStart(curr, next tokens.Token) bool {
	lit := curr.Literal(pv.source)
	if len(lit) != 1 || (lit[0] != 'U' && lit[0] != 'u') {
		return false
	}
	return next.Type == tokens.PLUS
}

func (pv *ParseVisitor) parseUnicodeRangeValue() Value {
	value := make([]byte, 0, 16)
	start := *pv.currentToken
	value = append(value, pv.currentLiteral()...)
	pv.advance()
	for !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.COMMA) && !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if len(value) > 0 && !adjacentToken(start, *pv.currentToken) && !pv.currentTokenIs(tokens.MINUS) {
			break
		}
		value = append(value, pv.currentLiteral()...)
		start = *pv.currentToken
		pv.advance()
		pv.ensureProgress(mark, "unicode-range value")
	}
	bv := pv.arena.newBasicValue()
	bv.Value = value
	return bv
}

func adjacentToken(prev, curr tokens.Token) bool {
	return prev.Line == curr.Line && curr.Column <= prev.Column+(prev.End-prev.Start)
}

func (pv *ParseVisitor) parseURLValue() Value {
	urlContent, singleQuote, quoteless := extractURLContent(pv.currentLiteral())
	var arg Value
	if quoteless {
		bv := pv.arena.newBasicValue()
		bv.Value = urlContent
		arg = bv
	} else {
		sv := pv.arena.newStringValue()
		sv.SingleQuote = singleQuote
		sv.Value = urlContent
		arg = sv
	}
	fv := pv.arena.newFunctionValue()
	fv.Name = []byte("url")
	fv.Arguments = pv.arena.appendValue(fv.Arguments, arg)
	return fv
}

func extractURLContent(uri []byte) ([]byte, bool, bool) {
	content := uri[4 : len(uri)-1]
	singleQuote := false
	quotless := false
	if len(content) >= 2 {
		if content[0] == '\'' && content[len(content)-1] == '\'' {
			singleQuote = true
			content = content[1 : len(content)-1]
		} else if content[0] == '"' && content[len(content)-1] == '"' {
			content = content[1 : len(content)-1]
		} else {
			quotless = true
		}
	}
	return content, singleQuote, quotless
}
