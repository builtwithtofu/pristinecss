package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type PageAtRule struct {
	Selectors []byte
	Rules     []Node
}

func (r *PageAtRule) Type() NodeType { return NodeAtRule }
func (r *PageAtRule) AtType() AtType { return AtPage }
func (r *PageAtRule) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PageAtRule{Selectors: %q, Rules: [\n", r.Selectors))
	for _, rule := range r.Rules {
		sb.WriteString(indentLines(rule.String(), 2))
		sb.WriteString(",\n")
	}
	sb.WriteString("]}")
	return sb.String()
}

type MarginBox struct {
	Name         []byte
	Declarations []Declaration
}

func (m *MarginBox) Type() NodeType { return NodeAtRule }
func (m *MarginBox) String() string { return descriptorRuleString("MarginBox", m.Name, m.Declarations) }

func visitPageAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*PageAtRule)
	pv.advance()
	r.Selectors = pv.capturePreludeUntil(tokens.LBRACE)
	if !pv.consume(tokens.LBRACE, "Expected '{' after @page") {
		return
	}
	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.AT) {
			box := pv.parseMarginBox()
			if box != nil {
				r.Rules = pv.arena.appendNode(r.Rules, box)
			}
			continue
		}
		if pv.currentTokenIs(tokens.IDENT) {
			decl := pv.arena.newDeclaration()
			decl.Key = pv.currentLiteral()
			visitDeclaration(pv, decl)
			r.Rules = pv.arena.appendNode(r.Rules, decl)
			continue
		}
		if pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance()
			continue
		}
		pv.addError("Expected page declaration or margin box", pv.currentToken)
		pv.skipToNextSemicolonOrBrace()
		pv.ensureProgress(mark, "page rule")
	}
	pv.consume(tokens.RBRACE, "Expected '}' to close @page")
}

func (pv *ParseVisitor) parseMarginBox() *MarginBox {
	pv.advance()
	if !pv.currentTokenIs(tokens.IDENT) {
		pv.addError("Expected margin-box name after @ inside @page", pv.currentToken)
		return nil
	}
	name := pv.currentLiteral()
	if !isMarginBoxName(string(name)) {
		pv.addError("Unknown @page margin box; expected one of top-left-corner, top-left, top-center, top-right, top-right-corner, bottom-left-corner, bottom-left, bottom-center, bottom-right, bottom-right-corner, left-top, left-middle, left-bottom, right-top, right-middle, right-bottom", pv.currentToken)
	}
	pv.advance()
	if !pv.consume(tokens.LBRACE, "Expected '{' after @page margin box") {
		return nil
	}
	box := pv.arena.newMarginBox()
	box.Name = name
	box.Declarations = pv.parseDeclarationListUntilBlockEnd()
	pv.consume(tokens.RBRACE, "Expected '}' to close @page margin box")
	return box
}

func isMarginBoxName(name string) bool {
	switch name {
	case "top-left-corner", "top-left", "top-center", "top-right", "top-right-corner",
		"bottom-left-corner", "bottom-left", "bottom-center", "bottom-right", "bottom-right-corner",
		"left-top", "left-middle", "left-bottom", "right-top", "right-middle", "right-bottom":
		return true
	default:
		return false
	}
}
