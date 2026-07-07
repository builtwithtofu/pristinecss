package parser

import "github.com/builtwithtofu/pristinecss/pkg/tokens"

func (pv *ParseVisitor) parseDeclarationListUntilBlockEnd() []Declaration {
	decls := make([]Declaration, 0)
	for !pv.currentTokenIs(tokens.RBRACE) && !pv.currentTokenIs(tokens.EOF) {
		if pv.currentTokenIs(tokens.COMMENT) || pv.currentTokenIs(tokens.SEMICOLON) {
			pv.advance()
			continue
		}
		if !pv.currentTokenIs(tokens.IDENT) {
			pv.addError("Expected descriptor name", pv.currentToken)
			pv.skipToNextSemicolonOrBrace()
			continue
		}
		decl := pv.arena.newDeclaration()
		decl.Key = pv.currentLiteral()
		visitDeclaration(pv, decl)
		decls = append(decls, *decl)
	}
	return decls
}

func (pv *ParseVisitor) capturePreludeUntil(tt tokens.TokenType) []byte {
	start, end := -1, -1
	for !pv.currentTokenIs(tt) && !pv.currentTokenIs(tokens.EOF) {
		if start < 0 {
			start = int(pv.currentToken.Start)
		}
		end = int(pv.currentToken.End)
		pv.advance()
	}
	return pv.sourceSpan(start, end)
}
