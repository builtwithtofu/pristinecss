package parser

import (
	"fmt"
	"strings"
)

func selectorTypeToString(st SelectorType) string {
	switch st {
	case Element:
		return "Element"
	case Class:
		return "Class"
	case ID:
		return "ID"
	case Attribute:
		return "Attribute"
	case Pseudo:
		return "Pseudo"
	case Combinator:
		return "Combinator"
	case Universal:
		return "Universal"
	case Nesting:
		return "Nesting"
	case NamespacePrefix:
		return "NamespacePrefix"
	default:
		return fmt.Sprintf("Unknown(%d)", st)
	}
}

func indentLines(s string, spaces int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			lines[i] = strings.Repeat(" ", spaces) + line
		}
	}
	return strings.Join(lines, "\n")
}

func isUnit(literal []byte) bool {
	switch string(literal) {
	case "cm", "mm", "in", "px", "pt", "pc", "Q",
		"em", "ex", "ch", "cap", "ic", "lh", "rem", "rex", "rch", "rcap", "ric", "rlh",
		"vw", "vh", "vi", "vb", "vmin", "vmax",
		"svw", "svh", "svi", "svb", "svmin", "svmax",
		"lvw", "lvh", "lvi", "lvb", "lvmin", "lvmax",
		"dvw", "dvh", "dvi", "dvb", "dvmin", "dvmax",
		"cqw", "cqh", "cqi", "cqb", "cqmin", "cqmax",
		"%",
		"deg", "grad", "rad", "turn",
		"s", "ms",
		"Hz", "kHz",
		"dpi", "dpcm", "dppx", "x",
		"fr":
		return true
	default:
		return false
	}
}

func isDashedIdent(literal []byte) bool {
	return len(literal) >= 2 && literal[0] == '-' && literal[1] == '-'
}
