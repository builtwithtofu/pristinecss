package parser

import "github.com/builtwithtofu/pristinecss/pkg/tokens"

type ViewTransitionAtRule struct{ Declarations []Declaration }

func (r *ViewTransitionAtRule) Type() NodeType { return NodeAtRule }
func (r *ViewTransitionAtRule) AtType() AtType { return AtViewTransition }
func (r *ViewTransitionAtRule) String() string {
	return descriptorRuleString("ViewTransitionAtRule", nil, r.Declarations)
}
func visitViewTransitionAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*ViewTransitionAtRule)
	pv.advance()
	if !pv.consume(tokens.LBRACE, "Expected '{' after @view-transition") {
		return
	}
	r.Declarations = pv.parseDeclarationListUntilBlockEnd()
	pv.consume(tokens.RBRACE, "Expected '}' to close @view-transition")
}
