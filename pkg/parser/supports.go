package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type SupportsConditionKind uint8

const (
	ConditionDeclaration SupportsConditionKind = iota
	ConditionNot
	ConditionAnd
	ConditionOr
	ConditionSelector
	ConditionFontTech
	ConditionFontFormat
)

type SupportsCondition struct {
	Kind     SupportsConditionKind
	Children []SupportsCondition
	Raw      []byte
}

func (sc SupportsCondition) String() string {
	return supportConditionToString(&sc, 0)
}

type SupportsAtRule struct {
	Condition SupportsCondition
	Rules     []Node
}

func (r *SupportsAtRule) Type() NodeType { return NodeAtRule }
func (r *SupportsAtRule) AtType() AtType { return AtSupports }
func (r *SupportsAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("SupportsAtRule{\n")
	sb.WriteString("  Condition: ")
	sb.WriteString(indentLines(r.Condition.String(), 2))
	sb.WriteString(",\n")
	sb.WriteString("  Rules: [\n")
	for _, rule := range r.Rules {
		sb.WriteString(indentLines(rule.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func visitSupportsAtRule(pv *ParseVisitor, node AtRule) {
	r := node.(*SupportsAtRule)
	pv.advance()
	r.Condition = pv.parseSupportsConditionUntil(tokens.LBRACE)
	if !pv.consume(tokens.LBRACE, "Expected '{' after @supports condition") {
		return
	}
	r.Rules = pv.parseRuleBlock(true)
	pv.consume(tokens.RBRACE, "Expected '}' to close @supports rule")
}

func (pv *ParseVisitor) parseSupportsConditionUntil(stop tokens.TokenType) SupportsCondition {
	cond := pv.parseSupportsConditionExpr(stop, tokens.EOF)
	return cond
}

func (pv *ParseVisitor) parseSupportsConditionExpr(stops ...tokens.TokenType) SupportsCondition {
	left := pv.parseSupportsConditionTerm(stops...)
	var op string
	var children []SupportsCondition
	for !pv.currentTokenIs(tokens.EOF) && !tokenIn(pv.currentToken.Type, stops...) {
		mark := pv.progressMark()
		if !pv.currentTokenIs(tokens.IDENT) {
			break
		}
		word := string(pv.currentLiteral())
		if word != "and" && word != "or" {
			break
		}
		if op != "" && op != word {
			pv.addError("@supports condition mixes and/or without parentheses", pv.currentToken)
		}
		op = word
		pv.advance()
		if children == nil {
			children = make([]SupportsCondition, 1, 4)
			children[0] = left
		}
		children = append(children, pv.parseSupportsConditionTerm(stops...))
		pv.ensureProgress(mark, "supports condition")
	}
	if children == nil {
		return left
	}
	if op == "or" {
		return SupportsCondition{Kind: ConditionOr, Children: children}
	}
	return SupportsCondition{Kind: ConditionAnd, Children: children}
}

func (pv *ParseVisitor) parseSupportsConditionTerm(stops ...tokens.TokenType) SupportsCondition {
	if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "not" {
		pv.advance()
		child := pv.parseSupportsConditionTerm(stops...)
		return SupportsCondition{Kind: ConditionNot, Children: []SupportsCondition{child}}
	}
	if pv.currentTokenIs(tokens.LPAREN) {
		pv.advance()
		cond := pv.parseSupportsConditionExpr(tokens.RPAREN)
		pv.consume(tokens.RPAREN, "Expected ')' to close supports condition")
		return cond
	}
	if pv.currentTokenIs(tokens.IDENT) && pv.nextTokenIs(tokens.LPAREN) {
		name := string(pv.currentLiteral())
		raw := pv.captureFunctionRaw()
		switch name {
		case "selector":
			return SupportsCondition{Kind: ConditionSelector, Raw: raw}
		case "font-tech":
			return SupportsCondition{Kind: ConditionFontTech, Raw: raw}
		case "font-format":
			return SupportsCondition{Kind: ConditionFontFormat, Raw: raw}
		default:
			return SupportsCondition{Kind: ConditionDeclaration, Raw: raw}
		}
	}
	return pv.parseSupportsDeclarationRaw(stops...)
}

func (pv *ParseVisitor) parseSupportsDeclarationRaw(stops ...tokens.TokenType) SupportsCondition {
	depth := 0
	start, end := -1, -1
	for !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if depth == 0 && tokenIn(pv.currentToken.Type, stops...) {
			break
		}
		if start < 0 {
			start = int(pv.currentToken.Start)
		}
		end = int(pv.currentToken.End)
		switch pv.currentToken.Type {
		case tokens.LPAREN:
			depth++
		case tokens.RPAREN:
			if depth == 0 {
				return SupportsCondition{Kind: ConditionDeclaration, Raw: pv.sourceSpan(start, end)}
			}
			depth--
		}
		pv.advance()
		pv.ensureProgress(mark, "supports declaration")
	}
	return SupportsCondition{Kind: ConditionDeclaration, Raw: pv.sourceSpan(start, end)}
}

func (pv *ParseVisitor) captureFunctionRaw() []byte {
	start := int(pv.currentToken.Start)
	end := int(pv.currentToken.End)
	pv.advance()
	depth := 0
	for !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		end = int(pv.currentToken.End)
		if pv.currentTokenIs(tokens.LPAREN) {
			depth++
		} else if pv.currentTokenIs(tokens.RPAREN) {
			depth--
			pv.advance()
			if depth == 0 {
				break
			}
			continue
		}
		pv.advance()
		pv.ensureProgress(mark, "supports function")
	}
	return pv.sourceSpan(start, end)
}

func tokenIn(tt tokens.TokenType, stops ...tokens.TokenType) bool {
	for _, stop := range stops {
		if tt == stop {
			return true
		}
	}
	return false
}

func supportConditionToString(condition *SupportsCondition, indentLevel int) string {
	if condition == nil {
		return "<nil>"
	}
	indent := strings.Repeat("  ", indentLevel)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%sSupportsCondition{Kind: %d, Raw: %q", indent, condition.Kind, condition.Raw))
	if len(condition.Children) > 0 {
		sb.WriteString(", Children: [\n")
		for i := range condition.Children {
			sb.WriteString(supportConditionToString(&condition.Children[i], indentLevel+1))
			sb.WriteString(",\n")
		}
		sb.WriteString(indent)
		sb.WriteString("]")
	}
	sb.WriteString("}")
	return sb.String()
}
