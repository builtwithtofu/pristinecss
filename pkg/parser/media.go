package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

var _ Node = (*MediaAtRule)(nil)

type MediaAtRule struct {
	Name  []byte
	Query MediaQuery
	Rules []Node
}

func (m *MediaAtRule) Type() NodeType { return NodeAtRule }
func (m *MediaAtRule) AtType() AtType { return AtMedia }
func (m *MediaAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("MediaAtRule{\n")
	sb.WriteString(fmt.Sprintf("  Name: %q,\n", m.Name))
	sb.WriteString("  Query: ")
	sb.WriteString(indentLines(m.Query.String(), 2))
	sb.WriteString(",\n")
	if len(m.Rules) > 0 {
		sb.WriteString("  Rules: [\n")
		for _, rule := range m.Rules {
			sb.WriteString(indentLines(rule.String(), 4))
			sb.WriteString(",\n")
		}
		sb.WriteString("  ]\n")
	}
	sb.WriteString("}")
	return sb.String()
}

type MediaQuery struct {
	Queries []MediaQueryExpression
}

func (mq MediaQuery) String() string {
	var sb strings.Builder
	sb.WriteString("MediaQuery{\n")
	sb.WriteString("  Queries: [\n")
	for _, query := range mq.Queries {
		sb.WriteString(indentLines(query.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

type MediaQueryExpression struct {
	MediaType []byte
	Not       bool
	Only      bool
	Features  []MediaFeature
}

func (mqe MediaQueryExpression) String() string {
	var sb strings.Builder
	sb.WriteString("MediaQueryExpression{\n")
	sb.WriteString(fmt.Sprintf("  MediaType: %q,\n", mqe.MediaType))
	sb.WriteString(fmt.Sprintf("  Not: %v,\n", mqe.Not))
	sb.WriteString(fmt.Sprintf("  Only: %v,\n", mqe.Only))
	sb.WriteString("  Features: [\n")
	for _, feature := range mqe.Features {
		sb.WriteString(indentLines(feature.String(), 4) + ",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

type MediaFeature struct {
	Name  []byte
	Value []byte
}

func (mf MediaFeature) String() string {
	var valueStr string
	if mf.Value != nil {
		valueStr = fmt.Sprintf("%q", mf.Value)
	} else {
		valueStr = "nil"
	}
	return fmt.Sprintf("MediaFeature{Name: %q, Value: %s}", mf.Name, valueStr)
}

func visitMediaAtRule(pv *ParseVisitor, node AtRule) {
	m := node.(*MediaAtRule)
	pv.advance() // Consume 'media'
	m.Query = pv.parseMediaQuery()

	if !pv.consume(tokens.LBRACE, "Expected '{' after media query") {
		return
	}

	m.Rules = pv.parseRuleBlock(true)

	pv.consume(tokens.RBRACE, "Expected '}' to close media block")
}

func (pv *ParseVisitor) parseMediaQuery() MediaQuery {
	var mediaQuery MediaQuery

	for !pv.currentTokenIs(tokens.LBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		expr := pv.parseMediaQueryExpression()
		mediaQuery.Queries = pv.arena.appendMediaQueryExpression(mediaQuery.Queries, expr)

		if pv.currentTokenIs(tokens.COMMA) {
			pv.advance() // Consume comma
		} else {
			break
		}
		pv.ensureProgress(mark, "media query")
	}

	return mediaQuery
}

func (pv *ParseVisitor) parseMediaQueryExpression() MediaQueryExpression {
	expr := MediaQueryExpression{}

	if pv.currentTokenIs(tokens.IDENT) {
		switch string(pv.currentLiteral()) {
		case "not":
			expr.Not = true
			pv.advance()
		case "only":
			expr.Only = true
			pv.advance()
		}
	}

	if pv.currentTokenIs(tokens.IDENT) {
		expr.MediaType = pv.currentLiteral()
		pv.advance()
	}

	for pv.currentTokenIs(tokens.LPAREN) || (pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "and") {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "and" {
			pv.advance() // Consume 'and'
		}
		feature, ok := pv.parseMediaFeature()
		if ok {
			expr.Features = pv.arena.appendMediaFeature(expr.Features, feature)
		}
		pv.ensureProgress(mark, "media query expression")
	}

	return expr
}

func (pv *ParseVisitor) parseMediaFeature() (MediaFeature, bool) {
	var feature MediaFeature
	if !pv.consume(tokens.LPAREN, "Expected '(' for media feature") {
		return feature, false
	}

	if !pv.currentTokenIs(tokens.IDENT) {
		feature.Name = pv.captureMediaFeatureValue()
		pv.consume(tokens.RPAREN, "Expected ')' to close media feature")
		return feature, true
	}

	nameStart := int(pv.currentToken.Start)
	feature.Name = pv.currentLiteral()
	pv.advance()

	if pv.currentTokenIs(tokens.COLON) {
		pv.advance() // Consume ':'
		feature.Value = pv.captureMediaFeatureValue()
	} else if !pv.currentTokenIs(tokens.RPAREN) {
		// Range syntax (`width > 400px`): the whole comparison is one contiguous
		// source span starting at the feature name — no copy needed.
		if _, end := pv.captureMediaFeatureSpan(); end > nameStart {
			feature.Name = pv.sourceSpan(nameStart, end)
		}
	}

	if !pv.consume(tokens.RPAREN, "Expected ')' to close media feature") {
		return feature, false
	}

	return feature, true
}

func (pv *ParseVisitor) captureMediaFeatureValue() []byte {
	start, end := pv.captureMediaFeatureSpan()
	return pv.sourceSpan(start, end)
}

func (pv *ParseVisitor) captureMediaFeatureSpan() (int, int) {
	parenDepth := 0
	start, end := -1, -1
	for !pv.currentTokenIs(tokens.RPAREN) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if start < 0 {
			start = int(pv.currentToken.Start)
		}
		end = int(pv.currentToken.End)
		if pv.currentTokenIs(tokens.LPAREN) {
			parenDepth++
		} else if pv.currentTokenIs(tokens.RPAREN) {
			if parenDepth == 0 {
				break
			}
			parenDepth--
		}
		pv.advance()
		pv.ensureProgress(mark, "media feature value")
	}
	return start, end
}
