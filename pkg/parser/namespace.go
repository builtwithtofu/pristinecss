package parser

import (
	"fmt"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type NamespaceAtRule struct {
	Prefix []byte
	URI    Value
}

func (r *NamespaceAtRule) Type() NodeType { return NodeAtRule }
func (r *NamespaceAtRule) AtType() AtType { return AtNamespace }
func (r *NamespaceAtRule) String() string {
	return fmt.Sprintf("NamespaceAtRule{Prefix: %q, URI: %v}", r.Prefix, r.URI)
}
func visitNamespaceAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*NamespaceAtRule)
	pv.advance()
	if pv.currentTokenIs(tokens.IDENT) && !pv.nextTokenIs(tokens.LPAREN) {
		r.Prefix = pv.currentLiteral()
		pv.advance()
	}
	if pv.currentTokenIs(tokens.URI) || pv.currentTokenIs(tokens.STRING) {
		r.URI = pv.parseValue()
	} else {
		pv.addError("Expected string or url() in @namespace", pv.currentToken)
	}
	pv.consume(tokens.SEMICOLON, "Expected ';' after @namespace")
}
