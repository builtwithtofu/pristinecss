package pristinecss

import (
	"math"
	"strconv"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

const (
	insetInvalid byte = iota
	insetNumber
	insetLength
)

// Validate the literal calc() arithmetic supported by older targets. Other
// functions and unresolved substitutions stay intact as reported gaps.
type insetMath struct {
	raw    []byte
	tokens []lexer.Token
	pos    int
}

func (p *insetMath) expr(minPrecedence int) (byte, float64) {
	kind, value := p.atom()
	if kind == insetInvalid {
		return insetInvalid, 0
	}
	for p.pos < len(p.tokens) {
		op := p.tokens[p.pos]
		precedence := 0
		switch op.Type {
		case lexer.Plus, lexer.Minus:
			precedence = 1
			if !isCSSSpace(p.raw[op.Start-1]) || !isCSSSpace(p.raw[op.End]) {
				return insetInvalid, 0
			}
		case lexer.Asterisk, lexer.Divide:
			precedence = 2
		}
		if precedence < minPrecedence {
			break
		}
		p.pos++
		right, n := p.expr(precedence + 1)
		if right == insetInvalid {
			return insetInvalid, 0
		}
		switch op.Type {
		case lexer.Plus, lexer.Minus:
			if kind != right {
				return insetInvalid, 0
			}
			if op.Type == lexer.Plus {
				value += n
			} else {
				value -= n
			}
		case lexer.Asterisk:
			if kind == insetLength && right == insetLength {
				return insetInvalid, 0
			}
			kind = max(kind, right)
			value *= n
		case lexer.Divide:
			if right != insetNumber || n == 0 {
				return insetInvalid, 0
			}
			value /= n
		}
		if math.IsInf(value, 0) {
			return insetInvalid, 0
		}
	}
	return kind, value
}

func (p *insetMath) atom() (byte, float64) {
	if p.pos == len(p.tokens) {
		return insetInvalid, 0
	}
	t := p.tokens[p.pos]
	if t.Type == lexer.Ident && cssNameEqual(string(p.raw[t.Start:t.End]), "calc") && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.LParen && t.End == p.tokens[p.pos+1].Start {
		p.pos++
		t = p.tokens[p.pos]
	}
	if t.Type == lexer.LParen {
		p.pos++
		kind, value := p.expr(1)
		if kind == insetInvalid || p.pos == len(p.tokens) || p.tokens[p.pos].Type != lexer.RParen {
			return insetInvalid, 0
		}
		p.pos++
		return kind, value
	}
	if t.Type != lexer.Number && t.Type != lexer.Plus && t.Type != lexer.Minus {
		return insetInvalid, 0
	}
	end := t.End
	p.pos++
	for p.pos < len(p.tokens) && p.tokens[p.pos].Start == end {
		next := p.tokens[p.pos]
		switch next.Type {
		case lexer.Number, lexer.Ident, lexer.Plus, lexer.Minus, lexer.Dot, lexer.Percentage:
			end = next.End
			p.pos++
		default:
			goto numeric
		}
	}
numeric:
	s := string(p.raw[t.Start:end])
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == start {
			return insetInvalid, 0
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		start := j
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j != start {
			i = j
		}
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return insetInvalid, 0
	}
	unit := s[i:]
	if unit == "" {
		return insetNumber, n
	}
	if unit == "%" {
		return insetLength, 0
	}
	for _, length := range [...]string{
		"px", "cm", "mm", "q", "in", "pc", "pt", "em", "ex", "cap", "ch", "ic", "rem", "rex", "rcap", "rch", "ric", "lh", "rlh",
		"vw", "vh", "vi", "vb", "vmin", "vmax", "svw", "svh", "svi", "svb", "svmin", "svmax", "lvw", "lvh", "lvi", "lvb", "lvmin", "lvmax", "dvw", "dvh", "dvi", "dvb", "dvmin", "dvmax",
		"cqw", "cqh", "cqi", "cqb", "cqmin", "cqmax",
	} {
		if cssNameEqual(unit, length) {
			return insetLength, 0
		}
	}
	return insetInvalid, 0
}
