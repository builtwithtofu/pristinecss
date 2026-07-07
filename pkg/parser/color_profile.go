package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ColorProfileAtRule struct {
	Name         []byte
	IsDeviceCMYK bool
	Declarations []Declaration
}

func (r *ColorProfileAtRule) Type() NodeType { return NodeAtRule }
func (r *ColorProfileAtRule) AtType() AtType { return AtColorProfile }
func (r *ColorProfileAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("ColorProfileAtRule{\n")
	if r.IsDeviceCMYK {
		sb.WriteString("  Name: device-cmyk,\n")
	} else {
		sb.WriteString(fmt.Sprintf("  Name: %q,\n", r.Name))
	}
	sb.WriteString("  Declarations: [\n")
	for _, decl := range r.Declarations {
		sb.WriteString(indentLines(decl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func visitColorProfileAtRule(pv *ParseVisitor, node AtRule) {
	cp := node.(*ColorProfileAtRule)
	pv.advance() // Consume 'color-profile'

	if pv.currentTokenIs(tokens.IDENT) {
		if string(pv.currentLiteral()) == "device-cmyk" {
			cp.IsDeviceCMYK = true
			pv.advance()
		} else if isDashedIdent(pv.currentLiteral()) {
			cp.Name = pv.currentLiteral()
			pv.advance()
		} else {
			pv.addError("Expected <dashed-ident> or 'device-cmyk' after @color-profile", pv.currentToken)
			return
		}
	} else {
		pv.addError("Expected identifier after @color-profile", pv.currentToken)
		return
	}

	if !pv.consume(tokens.LBRACE, "Expected '{' after @color-profile name") {
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
		cp.Declarations = append(cp.Declarations, *declaration)

		if pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance() // Consume ';'
		}
	}

	pv.consume(tokens.RBRACE, "Expected '}' to close @color-profile rule")
}
