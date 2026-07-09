package pristinecss

import "fmt"

// Kind identifies a CSS syntax construct stored in a flat Sheet node array.
type Kind uint8

const (
	KindStylesheet Kind = iota
	KindStyleRule
	KindBlock
	KindDeclaration
	KindComment

	KindSelElement
	KindSelClass
	KindSelID
	KindSelAttr
	KindSelPseudo
	KindSelCombinator
	KindSelUniversal
	KindSelNesting
	KindSelNamespace

	KindValBasic
	KindValString
	KindValFunction
	KindValRaw

	KindPreludeRaw
	KindMediaQuery
	KindMediaFeature
	KindSupportsCond
	KindContainerQuery
	KindContainerFeature
	KindKeyframeBlock
	KindMarginBox
	KindLayerName
	KindFontFamilyName

	KindAtCharset
	KindAtImport
	KindAtMedia
	KindAtContainer
	KindAtFontFace
	KindAtFontFeatureValues
	KindAtCounterStyle
	KindAtColorProfile
	KindAtKeyframes
	KindAtSupports
	KindAtLayer
	KindAtScope
	KindAtStartingStyle
	KindAtProperty
	KindAtFontPaletteValues
	KindAtNamespace
	KindAtPage
	KindAtPositionTry
	KindAtViewTransition
	KindAtGeneric

	kindCount
)

// Flags stores per-node state and compact payload bits.
type Flags uint8

const (
	// FlagTombstone marks a node and its subtree as logically deleted.
	FlagTombstone Flags = 1 << iota
	// FlagUnwrap makes a container transparent during traversal and emission.
	FlagUnwrap
	// FlagScratch means Lo:Hi indexes replacement bytes in Sheet.scratch.
	FlagScratch
	// FlagImportant marks a declaration as !important.
	FlagImportant
	// FlagCustom marks a custom-property declaration.
	FlagCustom
	// FlagSingleQuote marks a single-quoted string value.
	FlagSingleQuote
)

type emitClass uint8

const (
	classLeaf emitClass = iota
	classDeclaration
	classContainer
	classAtRule
	classBlock
)

type childRole uint8

const (
	roleNone childRole = iota
	roleSelector
	rolePrelude
	roleBlock
)

type kindInfo struct {
	name  string
	class emitClass
	role  childRole
}

var kinds = [kindCount]kindInfo{
	KindStylesheet:          {"stylesheet", classContainer, roleBlock},
	KindStyleRule:           {"style-rule", classContainer, roleSelector},
	KindBlock:               {"block", classBlock, roleBlock},
	KindDeclaration:         {"declaration", classDeclaration, roleNone},
	KindComment:             {"comment", classLeaf, roleNone},
	KindSelElement:          {"selector-element", classLeaf, roleNone},
	KindSelClass:            {"selector-class", classLeaf, roleNone},
	KindSelID:               {"selector-id", classLeaf, roleNone},
	KindSelAttr:             {"selector-attr", classLeaf, roleNone},
	KindSelPseudo:           {"selector-pseudo", classContainer, roleNone},
	KindSelCombinator:       {"selector-combinator", classLeaf, roleNone},
	KindSelUniversal:        {"selector-universal", classLeaf, roleNone},
	KindSelNesting:          {"selector-nesting", classLeaf, roleNone},
	KindSelNamespace:        {"selector-namespace", classLeaf, roleNone},
	KindValBasic:            {"value-basic", classLeaf, roleNone},
	KindValString:           {"value-string", classLeaf, roleNone},
	KindValFunction:         {"value-function", classContainer, roleNone},
	KindValRaw:              {"value-raw", classLeaf, roleNone},
	KindPreludeRaw:          {"prelude-raw", classLeaf, roleNone},
	KindMediaQuery:          {"media-query", classContainer, rolePrelude},
	KindMediaFeature:        {"media-feature", classLeaf, roleNone},
	KindSupportsCond:        {"supports-condition", classContainer, rolePrelude},
	KindContainerQuery:      {"container-query", classContainer, rolePrelude},
	KindContainerFeature:    {"container-feature", classLeaf, roleNone},
	KindKeyframeBlock:       {"keyframe-block", classContainer, roleBlock},
	KindMarginBox:           {"margin-box", classContainer, roleBlock},
	KindLayerName:           {"layer-name", classLeaf, roleNone},
	KindFontFamilyName:      {"font-family-name", classLeaf, roleNone},
	KindAtCharset:           {"@charset", classAtRule, rolePrelude},
	KindAtImport:            {"@import", classAtRule, rolePrelude},
	KindAtMedia:             {"@media", classAtRule, rolePrelude},
	KindAtContainer:         {"@container", classAtRule, rolePrelude},
	KindAtFontFace:          {"@font-face", classAtRule, rolePrelude},
	KindAtFontFeatureValues: {"@font-feature-values", classAtRule, rolePrelude},
	KindAtCounterStyle:      {"@counter-style", classAtRule, rolePrelude},
	KindAtColorProfile:      {"@color-profile", classAtRule, rolePrelude},
	KindAtKeyframes:         {"@keyframes", classAtRule, rolePrelude},
	KindAtSupports:          {"@supports", classAtRule, rolePrelude},
	KindAtLayer:             {"@layer", classAtRule, rolePrelude},
	KindAtScope:             {"@scope", classAtRule, rolePrelude},
	KindAtStartingStyle:     {"@starting-style", classAtRule, rolePrelude},
	KindAtProperty:          {"@property", classAtRule, rolePrelude},
	KindAtFontPaletteValues: {"@font-palette-values", classAtRule, rolePrelude},
	KindAtNamespace:         {"@namespace", classAtRule, rolePrelude},
	KindAtPage:              {"@page", classAtRule, rolePrelude},
	KindAtPositionTry:       {"@position-try", classAtRule, rolePrelude},
	KindAtViewTransition:    {"@view-transition", classAtRule, rolePrelude},
	KindAtGeneric:           {"@generic", classAtRule, rolePrelude},
}

func init() {
	for k := Kind(0); k < kindCount; k++ {
		if kinds[k].name == "" {
			panic(fmt.Sprintf("Kind %d has no kindInfo row", k))
		}
	}
}

func (k Kind) String() string {
	if k < kindCount && kinds[k].name != "" {
		return kinds[k].name
	}
	return fmt.Sprintf("Kind(%d)", k)
}

func (k Kind) isAtRule() bool { return k >= KindAtCharset && k <= KindAtGeneric }
func (k Kind) isOpaque() bool { return k == KindValRaw || kinds[k].class == classLeaf }

func atKind(name []byte) Kind {
	switch {
	case asciiEqual(name, "charset"):
		return KindAtCharset
	case asciiEqual(name, "import"):
		return KindAtImport
	case asciiEqual(name, "media"):
		return KindAtMedia
	case asciiEqual(name, "container"):
		return KindAtContainer
	case asciiEqual(name, "font-face"):
		return KindAtFontFace
	case asciiEqual(name, "font-feature-values"):
		return KindAtFontFeatureValues
	case asciiEqual(name, "counter-style"):
		return KindAtCounterStyle
	case asciiEqual(name, "color-profile"):
		return KindAtColorProfile
	case asciiEqual(name, "keyframes"), asciiEqual(name, "-webkit-keyframes"):
		return KindAtKeyframes
	case asciiEqual(name, "supports"):
		return KindAtSupports
	case asciiEqual(name, "layer"):
		return KindAtLayer
	case asciiEqual(name, "scope"):
		return KindAtScope
	case asciiEqual(name, "starting-style"):
		return KindAtStartingStyle
	case asciiEqual(name, "property"):
		return KindAtProperty
	case asciiEqual(name, "font-palette-values"):
		return KindAtFontPaletteValues
	case asciiEqual(name, "namespace"):
		return KindAtNamespace
	case asciiEqual(name, "page"):
		return KindAtPage
	case asciiEqual(name, "position-try"):
		return KindAtPositionTry
	case asciiEqual(name, "view-transition"):
		return KindAtViewTransition
	default:
		return KindAtGeneric
	}
}

func asciiEqual(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != s[i] {
			return false
		}
	}
	return true
}
