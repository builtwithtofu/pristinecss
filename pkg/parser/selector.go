package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

var _ Node = (*Selector)(nil)

type Selector struct {
	Selectors []SelectorValue
	Rules     []Node
}

func (c *Selector) Type() NodeType { return NodeSelector }
func (s *Selector) String() string {
	var sb strings.Builder
	sb.WriteString("Selector{\n")
	sb.WriteString("  Selectors: [\n")
	for _, sel := range s.Selectors {
		sb.WriteString("    " + sel.String() + ",\n")
	}
	sb.WriteString("  ]\n")
	if len(s.Rules) > 0 {
		sb.WriteString("  Rules: [\n")
		for _, rule := range s.Rules {
			sb.WriteString(indentLines(rule.String(), 4) + "\n")
		}
		sb.WriteString("  ]\n")
	} else {
		sb.WriteString("  Rules: []\n")
	}
	sb.WriteString("}")
	return sb.String()
}

type SelectorType int

const (
	Element SelectorType = iota
	Class
	ID
	Attribute
	Pseudo
	Combinator
	Universal
	Nesting
	NamespacePrefix
)

type SelectorValue struct {
	Type  SelectorType
	Value []byte
}

func (sv SelectorValue) String() string {
	return fmt.Sprintf("{Type: %s, Value: %q}", selectorTypeToString(sv.Type), sv.displayValueString())
}

func (sv SelectorValue) displayValueString() string {
	switch {
	case sv.Type == Class && (len(sv.Value) == 0 || sv.Value[0] != '.'):
		return "." + string(sv.Value)
	case sv.Type == ID && (len(sv.Value) == 0 || sv.Value[0] != '#'):
		return "#" + string(sv.Value)
	default:
		return string(sv.Value)
	}
}

const maxRuleBlockDepth = 128

func visitSelector(pv *ParseVisitor, node Node) {
	s := node.(*Selector)
	pv.parseSelector(s)
	if !pv.consume(tokens.LBRACE, "Expected '{' after selector") {
		return
	}
	s.Rules = pv.parseRuleBlock(false)
	pv.consume(tokens.RBRACE, "Expected '}' at the end of declaration block")
}

func (pv *ParseVisitor) parseRuleBlock(allowBareDeclarations bool) []Node {
	pv.blockDepth++
	defer func() { pv.blockDepth-- }()
	if pv.blockDepth > maxRuleBlockDepth {
		pv.addError(fmt.Sprintf("CSS nesting exceeds maximum depth of %d", maxRuleBlockDepth), pv.currentToken)
		pv.skipCurrentBlock()
		return nil
	}

	var rules []Node
	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		switch {
		case pv.currentTokenIs(tokens.COMMENT):
			comment := pv.arena.newComment()
			comment.Text = pv.currentLiteral()
			visitComment(pv, comment)
			rules = pv.arena.appendNode(rules, comment)
		case pv.currentTokenIs(tokens.AT):
			if at := parseAtRule(pv); at != nil {
				rules = pv.arena.appendNode(rules, at)
			}
		case pv.isDeclarationStart():
			declaration := pv.arena.newDeclaration()
			declaration.Key = pv.currentLiteral()
			visitDeclaration(pv, declaration)
			rules = pv.arena.appendNode(rules, declaration)
		case isSelectorStartToken(pv.currentToken.Type):
			selector := pv.arena.newSelector()
			visitSelector(pv, selector)
			rules = pv.arena.appendNode(rules, selector)
		case pv.currentTokenIs(tokens.SEMICOLON):
			pv.advance()
		default:
			pv.addError("Expected declaration, nested rule, at-rule, or comment", pv.currentToken)
			pv.skipToNextSemicolonOrBrace()
		}
		pv.ensureProgress(mark, "rule block")
	}
	return rules
}

func (pv *ParseVisitor) isDeclarationStart() bool {
	if !pv.currentTokenIs(tokens.IDENT) || !pv.nextTokenIs(tokens.COLON) {
		return false
	}
	if isDashedIdent(pv.currentLiteral()) {
		return true
	}
	parenDepth, bracketDepth := 0, 0
	for i := 2; ; i++ {
		tok := pv.peekToken(i)
		switch tok.Type {
		case tokens.LPAREN:
			parenDepth++
		case tokens.RPAREN:
			if parenDepth > 0 {
				parenDepth--
			}
		case tokens.LBRACKET:
			bracketDepth++
		case tokens.RBRACKET:
			if bracketDepth > 0 {
				bracketDepth--
			}
		case tokens.LBRACE:
			if parenDepth == 0 && bracketDepth == 0 {
				return false
			}
		case tokens.SEMICOLON, tokens.RBRACE, tokens.EOF:
			if parenDepth == 0 && bracketDepth == 0 {
				return true
			}
		}
	}
}

func (pv *ParseVisitor) peekToken(offset int) tokens.Token {
	switch offset {
	case 0:
		return *pv.currentToken
	case 1:
		return *pv.nextToken
	default:
		idx := pv.position + offset - 2
		if idx >= 0 && idx < len(pv.tokens) {
			return pv.tokens[idx]
		}
		return tokens.Token{Type: tokens.EOF}
	}
}

func isSelectorStartToken(tokenType tokens.TokenType) bool {
	switch tokenType {
	case tokens.DOT, tokens.HASH, tokens.AMPERSAND, tokens.ASTERISK, tokens.LBRACKET, tokens.COLON, tokens.DBLCOLON, tokens.IDENT, tokens.GREATER, tokens.PLUS, tokens.TILDE, tokens.PIPE:
		return true
	default:
		return false
	}
}

func (pv *ParseVisitor) parseSelector(s *Selector) {
	pv.parseSelectorUntil(s, tokens.LBRACE)
}

func (pv *ParseVisitor) parseSelectorUntil(s *Selector, stops ...tokens.TokenType) {
	var prevLine, prevEndColumn uint32
	hasPrevSimple := false
	for !pv.currentTokenIs(tokens.EOF) && !tokenIn(pv.currentToken.Type, stops...) {
		mark := pv.progressMark()
		before := *pv.currentToken
		var value SelectorValue
		hasValue := false
		combinator := false

		switch pv.currentToken.Type {
		case tokens.COMMENT:
			comment := pv.arena.newComment()
			comment.Text = pv.currentLiteral()
			visitComment(pv, comment)
			s.Rules = pv.arena.appendNode(s.Rules, comment)
			continue
		case tokens.IDENT:
			if pv.nextTokenIs(tokens.PIPE) {
				value, hasValue = pv.parseNamespaceSelector()
			} else {
				value = SelectorValue{Type: Element, Value: pv.currentLiteral()}
				hasValue = true
				pv.advance()
			}
		case tokens.ASTERISK:
			if pv.nextTokenIs(tokens.PIPE) {
				value, hasValue = pv.parseNamespaceSelector()
			} else {
				value = SelectorValue{Type: Universal, Value: pv.currentLiteral()}
				hasValue = true
				pv.advance()
			}
		case tokens.AMPERSAND:
			value = SelectorValue{Type: Nesting, Value: pv.currentLiteral()}
			hasValue = true
			pv.advance()
		case tokens.PIPE:
			if pv.nextTokenIs(tokens.PIPE) {
				start := int(pv.currentToken.Start)
				pv.advance()
				end := int(pv.currentToken.End)
				lit := pv.source[start:end]
				pv.advance()
				value = SelectorValue{Type: Combinator, Value: lit}
				hasValue = true
				combinator = true
			} else {
				value, hasValue = pv.parseNamespaceSelector()
			}
		case tokens.DOT:
			if pv.nextTokenIs(tokens.IDENT) || pv.nextTokenIs(tokens.NUMBER) {
				pv.advance()
				value = SelectorValue{Type: Class, Value: pv.currentLiteral()}
				hasValue = true
				pv.advance()
			} else {
				pv.addError("Expected identifier after '.'", pv.nextToken)
				pv.advance()
				continue
			}
		case tokens.HASH:
			if pv.nextTokenIs(tokens.IDENT) || pv.nextTokenIs(tokens.NUMBER) {
				pv.advance()
				value = SelectorValue{Type: ID, Value: pv.currentLiteral()}
				hasValue = true
				pv.advance()
			} else {
				pv.addError("Expected identifier after '#'", pv.nextToken)
				pv.advance()
				continue
			}
		case tokens.LBRACKET:
			value, hasValue = pv.parseAttributeSelector()
		case tokens.COLON, tokens.DBLCOLON:
			value, hasValue = pv.parsePseudoSelector()
		case tokens.COMMA, tokens.GREATER, tokens.PLUS, tokens.TILDE:
			value = SelectorValue{Type: Combinator, Value: pv.currentLiteral()}
			hasValue = true
			pv.advance()
			combinator = true
		default:
			pv.addError("Unexpected token in selector", pv.currentToken)
			pv.advance()
			continue
		}

		if !hasValue {
			continue
		}
		if !combinator && hasPrevSimple && (before.Line != prevLine || before.Column > prevEndColumn) {
			s.Selectors = pv.arena.appendSelectorValue(s.Selectors, SelectorValue{Type: Combinator, Value: []byte(" ")})
		}
		s.Selectors = pv.arena.appendSelectorValue(s.Selectors, value)
		if combinator {
			hasPrevSimple = false
		} else {
			prevLine = before.Line
			prevEndColumn = before.Column + selectorSourceLength(before, value)
			hasPrevSimple = true
		}
		pv.ensureProgress(mark, "selector")
	}
}

func selectorSourceLength(start tokens.Token, value SelectorValue) uint32 {
	switch value.Type {
	case Class, ID:
		return uint32(len(value.Value) + 1)
	default:
		return uint32(len(value.Value))
	}
}

func (pv *ParseVisitor) parseNamespaceSelector() (SelectorValue, bool) {
	start := int(pv.currentToken.Start)
	end := start
	if pv.currentTokenIs(tokens.IDENT) || pv.currentTokenIs(tokens.ASTERISK) {
		end = int(pv.currentToken.End)
		pv.advance()
	}
	if !pv.currentTokenIs(tokens.PIPE) {
		b := pv.source[start:end]
		return SelectorValue{Type: Element, Value: b}, len(b) > 0
	}
	end = int(pv.currentToken.End)
	pv.advance()
	if pv.currentTokenIs(tokens.IDENT) || pv.currentTokenIs(tokens.ASTERISK) {
		end = int(pv.currentToken.End)
		pv.advance()
	}
	return SelectorValue{Type: NamespacePrefix, Value: pv.source[start:end]}, true
}

func (pv *ParseVisitor) parseAttributeSelector() (SelectorValue, bool) {
	start := int(pv.currentToken.Start)

	pv.advance()
	bracketDepth := 1
	for bracketDepth > 0 && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.LBRACKET) {
			bracketDepth++
		} else if pv.currentTokenIs(tokens.RBRACKET) {
			bracketDepth--
			if bracketDepth == 0 {
				end := int(pv.currentToken.End)
				pv.advance()
				return SelectorValue{Type: Attribute, Value: pv.source[start:end]}, true
			}
		}
		pv.advance()
		pv.ensureProgress(mark, "attribute selector")
	}

	pv.addError("Expected closing bracket for attribute selector", pv.currentToken)
	return SelectorValue{}, false
}

func (pv *ParseVisitor) parsePseudoSelector() (SelectorValue, bool) {
	start := int(pv.currentToken.Start)
	end := int(pv.currentToken.End)
	pv.advance()
	if pv.currentTokenIs(tokens.MINUS) {
		end = int(pv.currentToken.End)
		pv.advance()
	}
	if pv.currentTokenIs(tokens.IDENT) {
		end = int(pv.currentToken.End)
		pv.advance()
		if pv.currentTokenIs(tokens.LPAREN) {
			parenDepth := 0
			for !pv.currentTokenIs(tokens.EOF) {
				mark := pv.progressMark()
				switch pv.currentToken.Type {
				case tokens.LPAREN:
					parenDepth++
				case tokens.RPAREN:
					parenDepth--
				}
				end = int(pv.currentToken.End)
				pv.advance()
				pv.ensureProgress(mark, "pseudo-class contents")
				if parenDepth == 0 {
					break
				}
			}
			if parenDepth != 0 {
				pv.addError("Expected closing parenthesis for pseudo-class", pv.currentToken)
			}
		}
		return SelectorValue{Type: Pseudo, Value: pv.source[start:end]}, true
	}
	pv.addError("Expected identifier after pseudo-selector", pv.currentToken)
	return SelectorValue{}, false
}
