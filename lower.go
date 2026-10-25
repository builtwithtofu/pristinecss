package pristinecss

import (
	"strings"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

// Fidelity orders rewrites from exact preservation to visible approximation.
// MinFidelity permits this level and all more faithful levels; zero defaults to Close.
type Fidelity uint8

const (
	Identical Fidelity = iota + 1
	Close
	Approximate
)

// Target is a browser to support. It keeps its identity, so build-time checks
// can consult per-browser data (such as which prefixed properties it reads)
// that doesn't belong in Caps.
type Target struct {
	Browser BrowserID // empty when the target is only a capability set
	Version string
	Caps    Caps
}

// At describes this browser version as a build-time target.
// It accepts the same version forms as Browser and retains the original version.
func (b BrowserID) At(version string) Target {
	return Target{Browser: b, Version: version, Caps: Browser(b, version)}
}

// Options controls build-time rewrites. Zero keeps the CSS as written.
type Options struct {
	Raise       Caps     // Reserved for build-time raising; not implemented yet.
	Lower       []Target // Targets to support; empty disables lowering.
	MinFidelity Fidelity
}

// Change describes a rewrite in original source byte coordinates.
type Change struct {
	Lo, Hi uint32
	Cap    Cap
	What   string
}

// Gap describes a feature left unchanged because it cannot be lowered.
// Lo and Hi are byte offsets into Source(), with Hi exclusive.
type Gap struct {
	Lo, Hi uint32
	Cap    Cap
	What   string
	Target Target // The requested target for which this gap was found.
}

// Mapping records a CSS Color 4 gamut mapping in original source coordinates.
// Gamut is "srgb" or "display-p3". Mappings are Close by definition.
type Mapping struct {
	Lo, Hi uint32
	Cap    Cap
	Target Target
	Gamut  string
}

// Report describes a build, including all unsupported lowering sites.
type Report struct {
	Errors   []ParseError
	Changes  []Change
	Mappings []Mapping // Lowered colours mapped into the target gamut.
	Gaps     []Gap
	Variants int // Complete outputs, each stored in minified and pretty form.
	Bytes    int // Total binary size, including its checksum.
}

type rewrite struct {
	fidelity Fidelity
	raise    func(*Sheet, Caps) []Change // T4 supplies raising.
	lower    func(*lowering, styleSpan, string)
}

var rewrites = map[Cap]rewrite{
	CapInset: {fidelity: Identical, lower: (*lowering).inset},
}

type lowering struct {
	cap         Cap
	fidelity    Fidelity
	allowed     bool
	fidelityGap bool
	target      Caps
	used        bool
	edits       []lowerEdit
	text        []byte
	raw         []byte
	tokens      []lexer.Token
	gaps        []Gap
	mappings    []Mapping
}

type lowerEdit struct {
	lo, hi     uint32
	start, end int
}

func scanLower(src string, spans []styleSpan, cap Cap, fidelity Fidelity, target Caps) *lowering {
	if fidelity == 0 {
		fidelity = Close
	}
	r := rewrites[cap]
	w := &lowering{cap: cap, fidelity: fidelity, allowed: r.fidelity <= fidelity, target: target}
	for _, e := range spans {
		if e.kind == KindDeclaration {
			r.lower(w, e, src)
		}
	}
	return w
}

func (w *lowering) cannot(lo, hi uint32, reason string) {
	w.gaps = append(w.gaps, Gap{Lo: lo, Hi: hi, Cap: w.cap, What: reason})
}

func (w *lowering) permit(lo, hi uint32) bool {
	w.used = true
	if !w.allowed {
		w.fidelityGap = true
		w.cannot(lo, hi, "cannot lower at requested fidelity")
	}
	return w.allowed
}

func (w *lowering) lex(value string) {
	w.raw = append(w.raw[:0], value...)
	w.tokens = lexer.LexInto(w.raw, w.tokens)
}

func (w *lowering) inset(e styleSpan, src string) {
	if !cssNameEqual(src[e.lo:e.nameEnd], "inset") || !w.permit(e.lo, e.valueEnd) {
		return
	}
	// The parser's value span can exclude priority and trailing comments.
	hi := e.valueEnd
	for hi < uint32(len(src)) && src[hi] != ';' && src[hi] != '}' {
		if strings.HasPrefix(src[hi:], "/*") {
			end := strings.Index(src[hi+2:], "*/")
			if end < 0 {
				hi = uint32(len(src))
				break
			}
			hi += uint32(end + 4)
		} else {
			hi++
		}
	}
	w.lex(src[e.valueLo:hi])
	tokens := w.tokens[:0]
	for _, t := range w.tokens {
		if t.Type != lexer.Comment && t.Type != lexer.EOF {
			tokens = append(tokens, t)
		}
	}
	important := e.flags&FlagImportant != 0
	if important {
		tokens = tokens[:len(tokens)-2]
	}
	var spans [4][2]int
	count, depth := 0, 0
	for i, t := range tokens {
		if depth == 0 && (count == 0 || t.Flags&lexer.HasSpaceBefore != 0) {
			if count == 4 {
				w.cannot(e.lo, hi, "cannot lower inset: expected 1 to 4 values")
				return
			}
			spans[count][0] = i
			count++
		}
		spans[count-1][1] = i + 1
		switch t.Type {
		case lexer.LParen:
			depth++
		case lexer.RParen:
			depth--
			if depth < 0 {
				w.cannot(e.lo, hi, "cannot lower inset: unbalanced value")
				return
			}
		}
	}
	if count == 0 || depth != 0 {
		w.cannot(e.lo, hi, "cannot lower inset: empty or unbalanced value")
		return
	}
	for _, span := range spans[:count] {
		component := tokens[span[0]:span[1]]
		name := string(w.raw[component[0].Start:component[0].End])
		valid := len(component) == 1 && component[0].Type == lexer.Ident && cssNameEqual(name, "auto")
		for _, keyword := range [...]string{"inherit", "initial", "unset", "revert", "revert-layer"} {
			if count == 1 && len(component) == 1 && component[0].Type == lexer.Ident && cssNameEqual(name, keyword) {
				valid = true
			}
		}
		if !valid && component[0].Type != lexer.LParen {
			p := insetMath{raw: w.raw, tokens: component}
			kind, value := p.atom()
			valid = p.pos == len(component) && (kind == insetLength || kind == insetNumber && value == 0 && component[0].Type != lexer.Ident)
		}
		if !valid {
			w.cannot(e.lo, hi, "cannot lower inset: invalid or unsupported value")
			return
		}
	}
	indices := [4]int{}
	switch count {
	case 2:
		indices = [4]int{0, 1, 0, 1}
	case 3:
		indices = [4]int{0, 1, 2, 1}
	case 4:
		indices = [4]int{0, 1, 2, 3}
	}
	start := len(w.text)
	for side, name := range [...]string{"top", "right", "bottom", "left"} {
		w.text = append(w.text, name...)
		w.text = append(w.text, ':')
		span := spans[indices[side]]
		w.text = append(w.text, w.raw[tokens[span[0]].Start:tokens[span[1]-1].End]...)
		if important {
			w.text = append(w.text, "!important"...)
		}
		w.text = append(w.text, ';')
	}
	if hi < uint32(len(src)) && src[hi] == ';' {
		hi++
	}
	w.edits = append(w.edits, lowerEdit{e.lo, hi, start, len(w.text)})
}

func (w *lowering) emit(src string) []byte {
	var dst []byte
	var pos uint32
	for _, e := range w.edits {
		dst = append(dst, src[pos:e.lo]...)
		dst = append(dst, w.text[e.start:e.end]...)
		pos = e.hi
	}
	return append(dst, src[pos:]...)
}
