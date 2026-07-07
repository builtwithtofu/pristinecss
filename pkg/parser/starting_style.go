package parser

import (
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type StartingStyleAtRule struct{ Rules []Node }

func (r *StartingStyleAtRule) Type() NodeType { return NodeAtRule }
func (r *StartingStyleAtRule) AtType() AtType { return AtStartingStyle }
func (r *StartingStyleAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("StartingStyleAtRule{\n  Rules: [\n")
	for _, rule := range r.Rules {
		sb.WriteString(indentLines(rule.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n}")
	return sb.String()
}

func visitStartingStyleAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*StartingStyleAtRule)
	pv.advance()
	if !pv.consume(tokens.LBRACE, "Expected '{' after @starting-style") {
		return
	}
	r.Rules = pv.parseRuleBlock(true)
	pv.consume(tokens.RBRACE, "Expected '}' to close @starting-style")
}
