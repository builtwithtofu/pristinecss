package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ImportAtRule struct {
	URL      Value
	Layer    Value
	Media    MediaQuery
	Supports *SupportsCondition
}

func (r *ImportAtRule) Type() NodeType { return NodeAtRule }
func (r *ImportAtRule) AtType() AtType { return AtImport }
func (r *ImportAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("ImportAtRule{\n")
	if r.URL != nil {
		sb.WriteString(fmt.Sprintf("  URL: %s,\n", r.URL.String()))
	}
	if r.Layer != nil {
		sb.WriteString(fmt.Sprintf("  Layer: %s,\n", r.Layer.String()))
	}
	if len(r.Media.Queries) > 0 {
		sb.WriteString(fmt.Sprintf("  Media: %s,\n", r.Media.String()))
	}
	if r.Supports != nil {
		sb.WriteString("  Supports: ")
		sb.WriteString(supportConditionToString(r.Supports, 1) + ",\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func visitImportAtRule(pv *ParseVisitor, node AtRule) {
	i := node.(*ImportAtRule)
	pv.advance()
	if pv.currentTokenIs(tokens.URI) || pv.currentTokenIs(tokens.STRING) {
		i.URL = pv.parseValue()
	} else {
		pv.addError("Expected string or URI after @import", pv.currentToken)
		return
	}

	for !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.EOF) {
		currTok := string(pv.currentLiteral())
		switch {
		case pv.currentTokenIs(tokens.IDENT) && currTok == "supports":
			pv.advance()
			cond := pv.parseSupportsConditionUntil(tokens.SEMICOLON)
			i.Supports = &cond
		case pv.currentTokenIs(tokens.IDENT) && currTok == "layer":
			i.Layer = pv.parseValue()
		case pv.currentTokenIs(tokens.IDENT) || pv.currentTokenIs(tokens.LPAREN):
			i.Media = pv.parseMediaQuery()
		default:
			pv.addError("Unexpected token in @import rule", pv.currentToken)
			return
		}
	}

	pv.consume(tokens.SEMICOLON, "Expected ';' after @import rule")
}
