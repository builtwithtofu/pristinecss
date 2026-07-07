package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

var _ Node = (*Declaration)(nil)

type Declaration struct {
	Key       []byte
	Value     []Value
	Important bool
	Custom    bool
}

func (d *Declaration) Type() NodeType { return NodeDeclaration }
func (d *Declaration) String() string {
	var sb strings.Builder
	sb.WriteString("Declaration{\n")
	sb.WriteString(fmt.Sprintf("  Key: %q,\n", d.Key))
	sb.WriteString("  Value: [\n")
	for _, vl := range d.Value {
		sb.WriteString(indentLines(vl.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString(fmt.Sprintf("  Important: %v\n", d.Important))
	if d.Custom {
		sb.WriteString("  Custom: true\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func visitDeclaration(pv *ParseVisitor, node Node) {
	d := node.(*Declaration)
	d.Custom = isCustomProperty(d.Key)
	pv.advance()
	if !pv.consume(tokens.COLON, "Expected ':' after property name") {
		pv.skipToNextSemicolonOrBrace()
		return
	}
	if d.Custom {
		value := pv.arena.newBasicValue()
		value.Value = pv.captureCustomPropertyValue()
		d.Value = pv.arena.appendValue(d.Value, value)
		if pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance()
		}
		return
	}
	for !pv.currentTokenIs(tokens.SEMICOLON) && !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		switch pv.currentToken.Type {
		case tokens.COMMENT:
			comment := pv.arena.newComment()
			comment.Text = pv.currentLiteral()
			visitComment(pv, comment)
			d.Value = pv.arena.appendValue(d.Value, comment)
		case tokens.EXCLAMATION:
			if pv.nextTokenIs(tokens.IDENT) && string(pv.nextLiteral()) == "important" {
				d.Important = true
				pv.advance()
				pv.advance()
			} else {
				d.Value = pv.arena.appendValue(d.Value, pv.parseValue())
			}
		case tokens.COMMA:
			pv.advance()
		default:
			d.Value = pv.arena.appendValue(d.Value, pv.parseValue())
		}
		pv.ensureProgress(mark, "declaration value")
	}
	if pv.currentTokenIs(tokens.SEMICOLON) {
		pv.advance()
	}
}

func isCustomProperty(key []byte) bool {
	return isDashedIdent(key)
}

func (pv *ParseVisitor) captureCustomPropertyValue() []byte {
	parenDepth, braceDepth, bracketDepth := 0, 0, 0
	start, end := -1, -1
	for !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && (pv.currentTokenIs(tokens.SEMICOLON) || pv.currentTokenIs(tokens.RBRACE)) {
			break
		}
		if start < 0 {
			start = int(pv.currentToken.Start)
		}
		end = int(pv.currentToken.End)
		switch pv.currentToken.Type {
		case tokens.LPAREN:
			parenDepth++
		case tokens.RPAREN:
			if parenDepth > 0 {
				parenDepth--
			}
		case tokens.LBRACE:
			braceDepth++
		case tokens.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
		case tokens.LBRACKET:
			bracketDepth++
		case tokens.RBRACKET:
			if bracketDepth > 0 {
				bracketDepth--
			}
		}
		pv.advance()
		pv.ensureProgress(mark, "custom property value")
	}
	if start < 0 || end < start || start > len(pv.source) {
		return nil
	}
	if end > len(pv.source) {
		end = len(pv.source)
	}
	return pv.source[start:end]
}
