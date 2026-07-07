package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type ContainerAtRule struct {
	Name         []byte
	Query        ContainerQuery
	Declarations []Node
}

type ContainerQuery struct {
	Conditions []ContainerCondition
}

type ContainerCondition struct {
	Features []ContainerFeature
}

type ContainerFeature struct {
	Name  []byte
	Value []byte
}

func (r *ContainerAtRule) Type() NodeType { return NodeAtRule }
func (r *ContainerAtRule) AtType() AtType { return AtContainer }
func (r *ContainerAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("ContainerAtRule{\n")
	if r.Name != nil {
		sb.WriteString(fmt.Sprintf("  Name: %q,\n", r.Name))
	}
	sb.WriteString("  Query: ")
	sb.WriteString(r.Query.String())
	sb.WriteString(",\n")
	sb.WriteString("  Declarations: [\n")
	for _, decl := range r.Declarations {
		sb.WriteString(indentLines(decl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func (cq ContainerQuery) String() string {
	var sb strings.Builder
	sb.WriteString("ContainerQuery{\n")
	sb.WriteString("    Conditions: [\n")
	for _, cond := range cq.Conditions {
		sb.WriteString(indentLines(cond.String(), 6))
		sb.WriteString(",\n")
	}
	sb.WriteString("    ]\n")
	sb.WriteString("  }")
	return sb.String()
}

func (cc ContainerCondition) String() string {
	var sb strings.Builder
	sb.WriteString("ContainerCondition{\n")
	sb.WriteString("      Features: [\n")
	for _, feat := range cc.Features {
		sb.WriteString(indentLines(feat.String(), 8))
		sb.WriteString(",\n")
	}
	sb.WriteString("      ]\n")
	sb.WriteString("    }")
	return sb.String()
}

func (cf ContainerFeature) String() string {
	return fmt.Sprintf("ContainerFeature{Name: %q, Value: %q}", cf.Name, cf.Value)
}

func visitContainerAtRule(pv *ParseVisitor, node AtRule) {
	c := node.(*ContainerAtRule)
	pv.advance() // Consume 'container'

	// Parse optional container name
	if pv.currentTokenIs(tokens.IDENT) {
		c.Name = pv.currentLiteral()
		pv.advance()
	}

	// Parse container query
	c.Query = *parseContainerQuery(pv)

	if !pv.consume(tokens.LBRACE, "Expected '{' after @container") {
		return
	}

	c.Declarations = pv.parseRuleBlock(true)

	pv.consume(tokens.RBRACE, "Expected '}' to close @container rule")
}

func parseContainerQuery(pv *ParseVisitor) *ContainerQuery {
	query := &ContainerQuery{}

	for !pv.currentTokenIs(tokens.LBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		condition := parseContainerCondition(pv)
		if condition != nil {
			query.Conditions = append(query.Conditions, *condition)
		}

		if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "and" {
			pv.advance() // Consume 'and'
		} else {
			break
		}
		pv.ensureProgress(mark, "container query")
	}

	return query
}

func parseContainerCondition(pv *ParseVisitor) *ContainerCondition {
	condition := &ContainerCondition{}

	if !pv.consume(tokens.LPAREN, "Expected '(' for container condition") {
		return nil
	}

	for !pv.currentTokenIs(tokens.RPAREN) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		feature := parseContainerFeature(pv)
		if feature != nil {
			condition.Features = append(condition.Features, *feature)
		}

		if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "and" {
			pv.advance() // Consume 'and'
		}
		pv.ensureProgress(mark, "container condition")
	}

	if !pv.consume(tokens.RPAREN, "Expected ')' to close container condition") {
		return nil
	}

	return condition
}

func parseContainerFeature(pv *ParseVisitor) *ContainerFeature {
	feature := &ContainerFeature{}

	if !pv.currentTokenIs(tokens.IDENT) {
		feature.Name = pv.captureContainerFeatureValue()
		return feature
	}

	feature.Name = pv.currentLiteral()
	pv.advance()

	if pv.currentTokenIs(tokens.COLON) {
		pv.advance()
		feature.Value = pv.captureContainerFeatureValue()
	} else if !pv.currentTokenIs(tokens.RPAREN) {
		feature.Name = append(feature.Name, pv.captureContainerFeatureValue()...)
	}

	return feature
}

func (pv *ParseVisitor) captureContainerFeatureValue() []byte {
	start, end := -1, -1
	for !pv.currentTokenIs(tokens.RPAREN) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.IDENT) && string(pv.currentLiteral()) == "and" {
			break
		}
		if start < 0 {
			start = int(pv.currentToken.Start)
		}
		end = int(pv.currentToken.End)
		pv.advance()
		pv.ensureProgress(mark, "container feature value")
	}
	return pv.sourceSpan(start, end)
}
