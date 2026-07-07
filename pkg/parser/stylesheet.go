package parser

import (
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

var _ Node = (*Stylesheet)(nil)

type Stylesheet struct {
	Rules []Node
}

func NewStylesheet() *Stylesheet {
	return &Stylesheet{}
}

func (s *Stylesheet) Type() NodeType { return NodeStylesheet }
func (s *Stylesheet) String() string {
	var sb strings.Builder
	sb.WriteString("Stylesheet{\n")
	if s.Rules != nil {
		for _, rule := range s.Rules {
			if rule != nil {
				sb.WriteString(indentLines(rule.String(), 2))
				sb.WriteString(",\n")
			}
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func visitStylesheet(pv *ParseVisitor, node Node) {
	s := node.(*Stylesheet)
	for !pv.currentTokenIs(tokens.EOF) {
		var childNode Node
		switch pv.currentToken.Type {
		case tokens.COMMENT:
			comment := pv.arena.newComment()
			comment.Text = pv.currentLiteral()
			childNode = comment
			visitComment(pv, childNode)
		case tokens.AT:
			childNode = parseAtRule(pv)
		case tokens.DOT, tokens.HASH, tokens.COLON, tokens.DBLCOLON, tokens.IDENT, tokens.LBRACKET, tokens.ASTERISK, tokens.AMPERSAND, tokens.PIPE:
			childNode = pv.arena.newSelector()
			visitSelector(pv, childNode)
		default:
			pv.addError("Unexpected token at stylesheet level", pv.currentToken)
			pv.advance()
			continue
		}

		if childNode != nil {
			s.Rules = pv.arena.appendNode(s.Rules, childNode)
		}
	}
}
