package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type LayerAtRule struct {
	Names [][]byte
	Rules []Node
}

func (r *LayerAtRule) Type() NodeType { return NodeAtRule }
func (r *LayerAtRule) AtType() AtType { return AtLayer }
func (r *LayerAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("LayerAtRule{\n")
	sb.WriteString("  Names: [")
	for i, name := range r.Names {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q", name))
	}
	sb.WriteString("],\n")
	if r.Rules != nil {
		sb.WriteString("  Rules: [\n")
		for _, rule := range r.Rules {
			sb.WriteString(indentLines(rule.String(), 4))
			sb.WriteString(",\n")
		}
		sb.WriteString("  ]\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func visitLayerAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*LayerAtRule)
	pv.advance()
	for !pv.currentTokenIs(tokens.LBRACE) && !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.COMMA) {
			pv.advance()
			pv.ensureProgress(mark, "layer prelude")
			continue
		}
		nameStart, nameEnd := -1, -1
		for pv.currentTokenIs(tokens.IDENT) || pv.currentTokenIs(tokens.DOT) {
			if nameStart >= 0 && int(pv.currentToken.Start) != nameEnd {
				break // whitespace ends a layer name; dotted segments are adjacent
			}
			if nameStart < 0 {
				nameStart = int(pv.currentToken.Start)
			}
			nameEnd = int(pv.currentToken.End)
			pv.advance()
		}
		if nameStart >= 0 {
			r.Names = pv.arena.appendByteSlice(r.Names, pv.source[nameStart:nameEnd])
			pv.ensureProgress(mark, "layer prelude")
			continue
		}
		pv.advance()
		pv.ensureProgress(mark, "layer prelude")
	}
	if pv.currentTokenIs(tokens.SEMICOLON) {
		pv.advance()
		return
	}
	if !pv.consume(tokens.LBRACE, "Expected '{' or ';' after @layer") {
		return
	}
	r.Rules = pv.parseRuleBlock(true)
	pv.consume(tokens.RBRACE, "Expected '}' to close @layer rule")
}
