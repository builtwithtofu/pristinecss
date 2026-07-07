package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type CounterStyleAtRule struct {
	Name         []byte
	Declarations []Declaration
}

func (r *CounterStyleAtRule) Type() NodeType { return NodeAtRule }
func (r *CounterStyleAtRule) AtType() AtType { return AtCounterStyle }
func (r *CounterStyleAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("CounterStyleAtRule{\n")
	sb.WriteString(fmt.Sprintf("  Name: %q,\n", r.Name))
	sb.WriteString("  Declarations: [\n")
	for _, decl := range r.Declarations {
		sb.WriteString(indentLines(decl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func visitCounterStyleAtRule(pv *ParseVisitor, node AtRule) {
	cs := node.(*CounterStyleAtRule)
	pv.advance() // Consume 'counter-style'

	if !pv.currentTokenIs(tokens.IDENT) {
		pv.addError("Expected identifier after @counter-style", pv.currentToken)
		return
	}

	cs.Name = pv.currentLiteral()
	pv.advance()

	if !pv.consume(tokens.LBRACE, "Expected '{' after @counter-style name") {
		return
	}

	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		if !pv.currentTokenIs(tokens.IDENT) {
			pv.addError("Expected property name", pv.currentToken)
			pv.skipToNextSemicolonOrBrace()
			continue
		}
		declaration := pv.arena.newDeclaration()
		declaration.Key = pv.currentLiteral()
		visitDeclaration(pv, declaration)
		cs.Declarations = append(cs.Declarations, *declaration)

		if pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance() // Consume ';'
		}
	}

	pv.consume(tokens.RBRACE, "Expected '}' to close @counter-style rule")
}
