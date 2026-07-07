package parser

import "github.com/builtwithtofu/pristinecss/pkg/tokens"

func parseAtRule(pv *ParseVisitor) Node {
	pv.advance() // consume @
	if !pv.currentTokenIs(tokens.IDENT) {
		pv.addError("Expected identifier after @", pv.currentToken)
		return nil
	}

	switch string(pv.currentLiteral()) {
	case "charset":
		rule := pv.arena.newCharsetAtRule()
		visitCharsetAtRule(pv, rule)
		return rule
	case "import":
		rule := pv.arena.newImportAtRule()
		visitImportAtRule(pv, rule)
		return rule
	case "media":
		rule := pv.arena.newMediaAtRule()
		rule.Name = []byte("media")
		visitMediaAtRule(pv, rule)
		return rule
	case "container":
		rule := pv.arena.newContainerAtRule()
		visitContainerAtRule(pv, rule)
		return rule
	case "font-face":
		rule := pv.arena.newFontFaceAtRule()
		visitFontFaceAtRule(pv, rule)
		return rule
	case "font-feature-values":
		rule := pv.arena.newFontFeatureValuesAtRule()
		visitFontFeatureValuesAtRule(pv, rule)
		return rule
	case "counter-style":
		rule := pv.arena.newCounterStyleAtRule()
		visitCounterStyleAtRule(pv, rule)
		return rule
	case "color-profile":
		rule := pv.arena.newColorProfileAtRule()
		visitColorProfileAtRule(pv, rule)
		return rule
	case "keyframes":
		rule := pv.arena.newKeyframesAtRule()
		visitKeyframesAtRule(pv, rule)
		return rule
	case "-webkit-keyframes":
		rule := pv.arena.newKeyframesAtRule()
		rule.WebKitPrefix = true
		visitKeyframesAtRule(pv, rule)
		return rule
	case "supports":
		rule := pv.arena.newSupportsAtRule()
		visitSupportsAtRule(pv, rule)
		return rule
	case "layer":
		rule := pv.arena.newLayerAtRule()
		visitLayerAtRule(pv, rule)
		return rule
	case "scope":
		rule := pv.arena.newScopeAtRule()
		visitScopeAtRule(pv, rule)
		return rule
	case "starting-style":
		rule := pv.arena.newStartingStyleAtRule()
		visitStartingStyleAtRule(pv, rule)
		return rule
	case "property":
		rule := pv.arena.newPropertyAtRule()
		visitPropertyAtRule(pv, rule)
		return rule
	case "font-palette-values":
		rule := pv.arena.newFontPaletteValuesAtRule()
		visitFontPaletteValuesAtRule(pv, rule)
		return rule
	case "namespace":
		rule := pv.arena.newNamespaceAtRule()
		visitNamespaceAtRule(pv, rule)
		return rule
	case "page":
		rule := pv.arena.newPageAtRule()
		visitPageAtRule(pv, rule)
		return rule
	case "position-try":
		rule := pv.arena.newPositionTryAtRule()
		visitPositionTryAtRule(pv, rule)
		return rule
	case "view-transition":
		rule := pv.arena.newViewTransitionAtRule()
		visitViewTransitionAtRule(pv, rule)
		return rule
	default:
		rule := pv.arena.newGenericAtRule()
		visitGenericAtRule(pv, rule)
		return rule
	}
}
