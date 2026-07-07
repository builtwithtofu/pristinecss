package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type PropertyAtRule struct {
	Name         []byte
	Declarations []Declaration
}

func (r *PropertyAtRule) Type() NodeType { return NodeAtRule }
func (r *PropertyAtRule) AtType() AtType { return AtProperty }
func (r *PropertyAtRule) String() string {
	return descriptorRuleString("PropertyAtRule", r.Name, r.Declarations)
}
func visitPropertyAtRule(pv *ParseVisitor, node AtRule) {
	parseDashedDescriptorRule(pv, node.(*PropertyAtRule))
}

type dashedDescriptor interface {
	setName([]byte)
	setDecls([]Declaration)
}

func (r *PropertyAtRule) setName(v []byte)         { r.Name = v }
func (r *PropertyAtRule) setDecls(v []Declaration) { r.Declarations = v }

func parseDashedDescriptorRule(pv *ParseVisitor, r dashedDescriptor) {
	pv.advance()
	if pv.currentTokenIs(tokens.IDENT) {
		if !isDashedIdent(pv.currentLiteral()) {
			pv.addError("Expected dashed identifier starting with '--' after descriptor at-rule", pv.currentToken)
		}
		r.setName(pv.currentLiteral())
		pv.advance()
	} else {
		pv.addError("Expected dashed identifier after descriptor at-rule", pv.currentToken)
	}
	if !pv.consume(tokens.LBRACE, "Expected '{' after descriptor at-rule prelude") {
		return
	}
	r.setDecls(pv.parseDeclarationListUntilBlockEnd())
	pv.consume(tokens.RBRACE, "Expected '}' to close descriptor at-rule")
}

func descriptorRuleString(name string, prelude []byte, declarations []Declaration) string {
	var sb strings.Builder
	sb.WriteString(name + "{\n")
	sb.WriteString(fmt.Sprintf("  Prelude: %q,\n", prelude))
	sb.WriteString("  Declarations: [\n")
	for _, decl := range declarations {
		sb.WriteString(indentLines(decl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n}")
	return sb.String()
}
