package parser

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/pristinecss/pkg/tokens"
)

type FontFeatureValuesAtRule struct {
	FontFamilies [][]byte
	Blocks       []FontFeatureValuesBlock
}

type FontFeatureValuesBlock struct {
	Name         []byte
	Declarations []Declaration
}

func (r *FontFeatureValuesAtRule) Type() NodeType { return NodeAtRule }
func (r *FontFeatureValuesAtRule) AtType() AtType { return AtFontFeatureValues }
func (r *FontFeatureValuesAtRule) String() string {
	var sb strings.Builder
	sb.WriteString("FontFeatureValuesAtRule{\n")
	sb.WriteString("  FontFamilies: [")
	for i, family := range r.FontFamilies {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q", family))
	}
	sb.WriteString("],\n")
	sb.WriteString("  Blocks: [\n")
	for _, block := range r.Blocks {
		sb.WriteString(indentLines(block.String(), 4))
		sb.WriteString(",\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}")
	return sb.String()
}

func (b FontFeatureValuesBlock) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FontFeatureValuesBlock{Name: %q, Declarations: [\n", b.Name))
	for _, decl := range b.Declarations {
		sb.WriteString(indentLines(decl.String(), 2))
		sb.WriteString(",\n")
	}
	sb.WriteString("]}")
	return sb.String()
}

func visitFontFeatureValuesAtRule(pv *ParseVisitor, node AtRule) {
	ffv := node.(*FontFeatureValuesAtRule)
	pv.advance() // Consume 'font-feature-values'

	// Parse font families
	for {
		mark := pv.progressMark()
		if pv.currentTokenIs(tokens.STRING) {
			lit := pv.currentLiteral()
			ffv.FontFamilies = append(ffv.FontFamilies, lit[1:len(lit)-1]) // Remove quotes
			pv.advance()
		} else if pv.currentTokenIs(tokens.IDENT) {
			var familyName strings.Builder
			for pv.currentTokenIs(tokens.IDENT) {
				if familyName.Len() > 0 {
					familyName.WriteByte(' ')
				}
				familyName.Write(pv.currentLiteral())
				pv.advance()
			}
			ffv.FontFamilies = append(ffv.FontFamilies, []byte(familyName.String()))
		} else {
			break
		}
		pv.ensureProgress(mark, "font-feature-values family list")

		if pv.currentTokenIs(tokens.COMMA) {
			pv.advance()
		} else {
			break
		}
	}

	if !pv.consume(tokens.LBRACE, "Expected '{' after @font-feature-values font families") {
		return
	}

	// Parse feature value blocks
	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		mark := pv.progressMark()
		if !pv.currentTokenIs(tokens.AT) {
			pv.addError("Expected feature value block starting with '@'", pv.currentToken)
			pv.skipToNextSemicolonOrBrace()
			continue
		}
		pv.advance() // Consume '@'

		if !pv.currentTokenIs(tokens.IDENT) {
			pv.addError("Expected feature value block name", pv.currentToken)
			pv.skipToNextSemicolonOrBrace()
			continue
		}

		block := FontFeatureValuesBlock{
			Name: pv.currentLiteral(),
		}
		pv.advance()

		if !pv.consume(tokens.LBRACE, "Expected '{' after feature value block name") {
			continue
		}

		for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
			if !pv.currentTokenIs(tokens.IDENT) {
				pv.addError("Expected property name in feature value block", pv.currentToken)
				pv.skipToNextSemicolonOrBrace()
				continue
			}
			declaration := pv.arena.newDeclaration()
			declaration.Key = pv.currentLiteral()
			visitDeclaration(pv, declaration)
			block.Declarations = append(block.Declarations, *declaration)

			if pv.currentTokenIs(tokens.SEMICOLON) {
				pv.advance() // Consume ';'
			}
		}

		pv.consume(tokens.RBRACE, "Expected '}' to close feature value block")
		ffv.Blocks = append(ffv.Blocks, block)
		pv.ensureProgress(mark, "font-feature-values block")
	}

	pv.consume(tokens.RBRACE, "Expected '}' to close @font-feature-values rule")
}
