package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type GenericAtRule struct {
	Name    []byte
	Prelude []byte
	Block   []Node
}

func (g *GenericAtRule) Type() NodeType { return NodeAtRule }
func (g *GenericAtRule) AtType() AtType { return AtGeneric }
func (g *GenericAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("GenericAtRule{\n")
	sb.WriteString(fmt.Sprintf("  Name: %q,\n", g.Name))
	sb.WriteString(fmt.Sprintf("  Prelude: %q,\n", g.Prelude))
	if g.Block != nil {
		sb.WriteString("  Block: [\n")
		for _, rule := range g.Block {
			sb.WriteString(indentLines(rule.String(), 4))
			sb.WriteString(",\n")
		}
		sb.WriteString("  ]\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func visitGenericAtRule(pv *ParseVisitor, node AtRule) {
	g := node.(*GenericAtRule)
	g.Name = pv.currentLiteral()
	pv.advance()

	g.Prelude = make([]byte, 0, 16)
	for !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.LBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		g.Prelude = append(g.Prelude, pv.currentLiteral()...)
		pv.advance()
		pv.ensureProgress(mark, "generic at-rule prelude")
	}
	if pv.currentTokenIs(tokens.SEMICOLON) {
		pv.advance()
		return
	}
	if !pv.currentTokenIs(tokens.LBRACE) {
		return
	}
	pv.advance()
	g.Block = pv.parseRuleBlock(true)
	if pv.currentTokenIs(tokens.RBRACE) {
		pv.advance()
	}
}
