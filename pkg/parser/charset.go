package parser

import (
	"fmt"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type CharsetAtRule struct {
	Charset Value
}

func (r *CharsetAtRule) Type() NodeType { return NodeAtRule }
func (r *CharsetAtRule) AtType() AtType { return AtCharset }
func (r *CharsetAtRule) String() string {
	return fmt.Sprintf("CharsetAtRule{Charset: %q}", r.Charset)
}

func visitCharsetAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*CharsetAtRule)
	pv.advance() // Consume 'charset'

	if pv.currentTokenIs(tokens.STRING) {
		r.Charset = pv.parseValue()
	} else {
		pv.addError("Expected string after @charset", pv.currentToken)
		return
	}

	pv.consume(tokens.SEMICOLON, "Expected ';' after @charset rule")
}
