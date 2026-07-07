package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ScopeAtRule struct {
	Start []SelectorValue
	End   []SelectorValue
	Rules []Node
}

func (r *ScopeAtRule) Type() NodeType { return NodeAtRule }
func (r *ScopeAtRule) AtType() AtType { return AtScope }
func (r *ScopeAtRule) String() string {
	return fmt.Sprintf("ScopeAtRule{Start: %v, End: %v, Rules: %d}", r.Start, r.End, len(r.Rules))
}

func visitScopeAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*ScopeAtRule)
	pv.advance()
	if pv.currentTokenIs(tokens.LPAREN) {
		pv.advance()
		s := pv.arena.newSelector()
		pv.parseSelectorUntil(s, tokens.RPAREN)
		r.Start = s.Selectors
		pv.consume(tokens.RPAREN, "Expected ')' after @scope start")
	}
	if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "to" {
		pv.advance()
		if pv.currentTokenIs(tokens.LPAREN) {
			pv.advance()
			s := pv.arena.newSelector()
			pv.parseSelectorUntil(s, tokens.RPAREN)
			r.End = s.Selectors
			pv.consume(tokens.RPAREN, "Expected ')' after @scope end")
		}
	}
	if !pv.consume(tokens.LBRACE, "Expected '{' after @scope") {
		return
	}
	r.Rules = pv.parseRuleBlock(true)
	pv.consume(tokens.RBRACE, "Expected '}' to close @scope rule")
}

var _ = strings.Builder{}
