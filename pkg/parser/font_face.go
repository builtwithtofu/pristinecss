package parser

import (
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type FontFaceAtRule struct {
	Declarations []Declaration
}

func (r *FontFaceAtRule) Type() NodeType { return NodeAtRule }
func (r *FontFaceAtRule) AtType() AtType { return AtFontFace }
func (r *FontFaceAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("FontFaceAtRule{\n")
	sb.WriteString("  Declarations: [\n")
	for _, decl := range r.Declarations {
		sb.WriteString(indentLines(decl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func visitFontFaceAtRule(pv *ParseVisitor, node AtRule) {
	ff := node.(*FontFaceAtRule)
	pv.advance() // Consume 'font-face'

	if !pv.consume(tokens.LBRACE, "Expected '{' after @font-face") {
		return
	}

	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		declaration := pv.arena.newDeclaration()
		declaration.Key = pv.currentLiteral()
		visitDeclaration(pv, declaration)
		ff.Declarations = append(ff.Declarations, *declaration)

		if pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance()
		}
	}

	pv.consume(tokens.RBRACE, "Expected '}' to close @font-face rule")
}
