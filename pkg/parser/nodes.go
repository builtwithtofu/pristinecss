package parser

import "fmt"

type NodeType uint8

const (
	NodeStylesheet NodeType = iota
	NodeSelector
	NodeDeclaration
	NodeValue
	NodeAtRule
	NodeComment
	NodeKeyframeStop
)

func (nt NodeType) String() string {
	switch nt {
	case NodeStylesheet:
		return "stylesheet"
	case NodeSelector:
		return "selector"
	case NodeDeclaration:
		return "declaration"
	case NodeValue:
		return "value"
	case NodeAtRule:
		return "at-rule"
	case NodeComment:
		return "comment"
	case NodeKeyframeStop:
		return "keyframe-stop"
	default:
		return fmt.Sprintf("NodeType(%d)", nt)
	}
}

type Node interface {
	Type() NodeType
	String() string
}

type AtType uint8

const (
	AtCharset AtType = iota
	AtImport
	AtMedia
	AtContainer
	AtFontFace
	AtFontFeatureValues
	AtCounterStyle
	AtColorProfile
	AtKeyframes
	AtSupports
	AtLayer
	AtScope
	AtStartingStyle
	AtProperty
	AtFontPaletteValues
	AtNamespace
	AtPage
	AtPositionTry
	AtViewTransition
	AtGeneric
)

func (at AtType) String() string {
	switch at {
	case AtCharset:
		return "charset"
	case AtImport:
		return "import"
	case AtMedia:
		return "media"
	case AtContainer:
		return "container"
	case AtFontFace:
		return "font-face"
	case AtFontFeatureValues:
		return "font-feature-values"
	case AtCounterStyle:
		return "counter-style"
	case AtColorProfile:
		return "color-profile"
	case AtKeyframes:
		return "keyframes"
	case AtSupports:
		return "supports"
	case AtLayer:
		return "layer"
	case AtScope:
		return "scope"
	case AtStartingStyle:
		return "starting-style"
	case AtProperty:
		return "property"
	case AtFontPaletteValues:
		return "font-palette-values"
	case AtNamespace:
		return "namespace"
	case AtPage:
		return "page"
	case AtPositionTry:
		return "position-try"
	case AtViewTransition:
		return "view-transition"
	case AtGeneric:
		return "generic"
	default:
		return fmt.Sprintf("AtType(%d)", at)
	}
}

type AtRule interface {
	Node
	AtType() AtType
}
