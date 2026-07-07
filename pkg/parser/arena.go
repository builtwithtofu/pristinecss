package parser

import "unsafe"

// Arena owns AST node storage for one parse. Nodes handed out by an Arena must not
// outlive the Arena when callers use ParseInto and later call Reset. Parse creates an
// internal Arena for callers that do not need reuse.
//
// Two storage kinds live here:
//   - arenaSlab[T]: fixed-size element structs (Declaration, Selector, ...) handed out
//     one at a time. Chunks materialize lazily on first use and follow-up chunks grow
//     linearly (initialSize/2), so an unused or over-estimated slab costs nothing and a
//     misestimate wastes at most half a chunk, not a doubling.
//   - arenaPool[T]: backing storage for the growing container slices ([]Node, []Value,
//     []SelectorValue). Slices are carved zero-length and grow in place while they are
//     the pool's tail carve (the common case: parsers finish a child's slice before the
//     parent appends again); a non-tail regrow copies to a fresh carve and abandons the
//     old segment, bounded by the linear chunking.
//
// Element sizing ratios were measured on test-data/frameworks (2026-07-07, post-M7):
// declarations max 0.112/tok, selectors max 0.063/tok, values max 0.220/tok. Container
// demand medians (post-M9 utilization probe): nodes 0.29/tok, values 0.33/tok,
// selector-values 0.22/tok. Divisors sit just above the measured maxima for slabs and
// at the medians for pools — linear chunk growth makes undersizing cheap, so headroom
// beyond the measurement is deliberate waste and was removed.
type Arena struct {
	declarations      arenaSlab[Declaration]
	selectors         arenaSlab[Selector]
	comments          arenaSlab[Comment]
	basicValues       arenaSlab[BasicValue]
	stringValues      arenaSlab[StringValue]
	functionValues    arenaSlab[FunctionValue]
	charsetRules      arenaSlab[CharsetAtRule]
	importRules       arenaSlab[ImportAtRule]
	mediaRules        arenaSlab[MediaAtRule]
	containerRules    arenaSlab[ContainerAtRule]
	fontFaceRules     arenaSlab[FontFaceAtRule]
	fontFeatureRules  arenaSlab[FontFeatureValuesAtRule]
	counterRules      arenaSlab[CounterStyleAtRule]
	colorProfileRules arenaSlab[ColorProfileAtRule]
	keyframesRules    arenaSlab[KeyframesAtRule]
	supportsRules     arenaSlab[SupportsAtRule]
	layerRules        arenaSlab[LayerAtRule]
	scopeRules        arenaSlab[ScopeAtRule]
	startingRules     arenaSlab[StartingStyleAtRule]
	propertyRules     arenaSlab[PropertyAtRule]
	paletteRules      arenaSlab[FontPaletteValuesAtRule]
	namespaceRules    arenaSlab[NamespaceAtRule]
	pageRules         arenaSlab[PageAtRule]
	marginBoxes       arenaSlab[MarginBox]
	positionTryRules  arenaSlab[PositionTryAtRule]
	viewTransRules    arenaSlab[ViewTransitionAtRule]
	genericRules      arenaSlab[GenericAtRule]
	keyframeStops     arenaSlab[KeyframeStop]
	nodeItems         arenaPool[Node]
	valueItems        arenaPool[Value]
	selectorItems     arenaPool[SelectorValue]
	byteSliceItems    arenaPool[[]byte]
	mediaExprItems    arenaPool[MediaQueryExpression]
	mediaFeatItems    arenaPool[MediaFeature]
}

const (
	declarationPerTokenDivisor = 8  // measured max 0.112/tok; sized to 0.125/tok
	selectorPerTokenDivisor    = 14 // measured max 0.063/tok; sized to 0.071/tok
	valuePerTokenDivisor       = 4  // measured max 0.220/tok; sized to 0.250/tok
	commentPerTokenDivisor     = 64
	atRulePerTokenDivisor      = 64
	nodeItemsPerTokenDivisor   = 3 // measured median 0.29/tok
	valueItemsPerTokenDivisor  = 3 // measured median 0.33/tok
	selItemsPerTokenDivisor    = 4 // measured median 0.22/tok
)

func NewArena(tokenCount int) *Arena {
	a := &Arena{}
	a.init(tokenCount)
	return a
}

func (a *Arena) init(tokenCount int) {
	if tokenCount < 1 {
		tokenCount = 1
	}
	a.declarations.init(tokenCount/declarationPerTokenDivisor + 8)
	a.selectors.init(tokenCount/selectorPerTokenDivisor + 8)
	a.comments.init(tokenCount/commentPerTokenDivisor + 4)
	a.basicValues.init(tokenCount/valuePerTokenDivisor + 8)
	a.stringValues.init(tokenCount/64 + 4)
	a.functionValues.init(tokenCount/24 + 4)
	atRules := tokenCount/atRulePerTokenDivisor + 4
	a.charsetRules.init(2)
	a.importRules.init(4)
	a.mediaRules.init(atRules)
	a.containerRules.init(atRules)
	a.fontFaceRules.init(4)
	a.fontFeatureRules.init(2)
	a.counterRules.init(4)
	a.colorProfileRules.init(2)
	a.keyframesRules.init(4)
	a.supportsRules.init(atRules)
	a.layerRules.init(atRules)
	a.scopeRules.init(atRules)
	a.startingRules.init(atRules)
	a.propertyRules.init(4)
	a.paletteRules.init(2)
	a.namespaceRules.init(2)
	a.pageRules.init(2)
	a.marginBoxes.init(8)
	a.positionTryRules.init(4)
	a.viewTransRules.init(2)
	a.genericRules.init(4)
	a.keyframeStops.init(tokenCount/128 + 4)
	a.nodeItems.init(tokenCount/nodeItemsPerTokenDivisor + 16)
	a.valueItems.init(tokenCount/valueItemsPerTokenDivisor + 16)
	a.selectorItems.init(tokenCount/selItemsPerTokenDivisor + 16)
	a.byteSliceItems.init(tokenCount/atRulePerTokenDivisor + 8)
	a.mediaExprItems.init(tokenCount/atRulePerTokenDivisor + 8)
	a.mediaFeatItems.init(tokenCount/atRulePerTokenDivisor + 8)
}

func (a *Arena) Reset(tokenCount int) {
	if a == nil {
		return
	}
	a.poison()
	a.init(tokenCount)
}

// ArenaStats reports container-pool demand and behaviour for sizing work. Allocated is
// the chunk capacity materialized so far, Used the cursor total across chunks,
// TailGrows in-place extensions, Fallbacks oversized carves that escaped to the heap.
type ArenaStats struct {
	NodeItemsAllocated, NodeItemsUsed, NodeItemsTailGrows, NodeItemsFallbacks            int
	ValueItemsAllocated, ValueItemsUsed, ValueItemsTailGrows, ValueItemsFallbacks        int
	SelectorItemsAllocated, SelectorItemsUsed, SelectorItemsTailGrows, SelectorFallbacks int
}

func (a *Arena) Stats() ArenaStats {
	if a == nil {
		return ArenaStats{}
	}
	return ArenaStats{
		NodeItemsAllocated: a.nodeItems.allocated(), NodeItemsUsed: a.nodeItems.usedTotal(),
		NodeItemsTailGrows: a.nodeItems.tailGrows, NodeItemsFallbacks: a.nodeItems.fallbacks,
		ValueItemsAllocated: a.valueItems.allocated(), ValueItemsUsed: a.valueItems.usedTotal(),
		ValueItemsTailGrows: a.valueItems.tailGrows, ValueItemsFallbacks: a.valueItems.fallbacks,
		SelectorItemsAllocated: a.selectorItems.allocated(), SelectorItemsUsed: a.selectorItems.usedTotal(),
		SelectorItemsTailGrows: a.selectorItems.tailGrows, SelectorFallbacks: a.selectorItems.fallbacks,
	}
}

// arenaSlab hands out one zeroed *T at a time. Chunks materialize lazily; follow-up
// chunks are initialSize/2 so a size misestimate wastes at most half a chunk. Chunks
// retained across init (Reset reuse) hold stale values and are zeroed on handout;
// freshly made chunks are already zero and skip that store.
type arenaSlab[T any] struct {
	chunks      [][]T
	chunk       int
	index       int
	initialSize int
	dirtyChunks int
}

func (s *arenaSlab[T]) init(size int) {
	if size < 1 {
		size = 1
	}
	s.initialSize = size
	s.dirtyChunks = 0
	if len(s.chunks) > 0 && cap(s.chunks[0]) >= size {
		s.chunks = s.chunks[:1]
		s.chunks[0] = s.chunks[0][:cap(s.chunks[0])]
		s.dirtyChunks = 1
	} else {
		s.chunks = nil
	}
	s.chunk = 0
	s.index = 0
}

func (s *arenaSlab[T]) new() *T {
	if len(s.chunks) == 0 {
		s.chunks = append(s.chunks, make([]T, s.initialSize))
	}
	if s.index >= len(s.chunks[s.chunk]) {
		s.chunks = append(s.chunks, make([]T, s.growSize()))
		s.chunk++
		s.index = 0
	}
	ptr := &s.chunks[s.chunk][s.index]
	if s.chunk < s.dirtyChunks {
		var zero T
		*ptr = zero
	}
	s.index++
	return ptr
}

func (s *arenaSlab[T]) growSize() int {
	next := s.initialSize / 2
	if next < 8 {
		next = 8
	}
	return next
}

// arenaPool backs the growing container slices. Carves are zero-length with capacity;
// elements past len are never read before being written by append, so reused chunk
// memory needs no zeroing here (Reset's poison pass handles the debug build).
type arenaPool[T any] struct {
	chunks      [][]T
	chunk       int
	used        int
	initialSize int
	tailGrows   int
	fallbacks   int
}

func (p *arenaPool[T]) init(size int) {
	if size < 8 {
		size = 8
	}
	p.initialSize = size
	if len(p.chunks) > 0 && cap(p.chunks[0]) >= size {
		p.chunks = p.chunks[:1]
		p.chunks[0] = p.chunks[0][:cap(p.chunks[0])]
	} else {
		p.chunks = nil
	}
	p.chunk = 0
	p.used = 0
	p.tailGrows = 0
	p.fallbacks = 0
}

func (p *arenaPool[T]) growSize() int {
	next := p.initialSize / 2
	if next < 8 {
		next = 8
	}
	return next
}

func (p *arenaPool[T]) allocated() int {
	total := 0
	for _, c := range p.chunks {
		total += len(c)
	}
	return total
}

func (p *arenaPool[T]) usedTotal() int {
	total := p.used
	for i := 0; i < p.chunk; i++ {
		total += len(p.chunks[i])
	}
	return total
}

// carve returns a zero-length slice with the given capacity backed by the pool, or a
// plain heap slice when the request exceeds a growth chunk (tracked as a fallback).
func (p *arenaPool[T]) carve(capacity int) []T {
	if capacity < 1 {
		capacity = 1
	}
	if len(p.chunks) == 0 {
		p.chunks = append(p.chunks, make([]T, p.initialSize))
	}
	chunk := p.chunks[p.chunk]
	if p.used+capacity > len(chunk) {
		grow := p.growSize()
		if capacity > grow {
			p.fallbacks++
			return make([]T, 0, capacity)
		}
		p.chunks = append(p.chunks, make([]T, grow))
		p.chunk++
		p.used = 0
		chunk = p.chunks[p.chunk]
	}
	start := p.used
	p.used += capacity
	return chunk[start:start:p.used]
}

// poolAppend grows slice by one element. A slice that is the pool's tail carve extends
// in place by bumping the cursor — no copy, no allocation. Anything else regrows into a
// fresh carve (abandoning its old segment) or, when it never came from the pool and has
// spare capacity, appends normally.
func poolAppend[T any](p *arenaPool[T], slice []T, value T, initialCap int) []T {
	if len(slice) < cap(slice) {
		return append(slice, value)
	}
	if cap(slice) == 0 {
		return append(p.carve(initialCap), value)
	}
	n := len(slice)
	if len(p.chunks) > 0 {
		chunk := p.chunks[p.chunk]
		start := p.used - n
		if start >= 0 && unsafe.SliceData(slice) == unsafe.SliceData(chunk[start:p.used]) {
			extra := n
			if p.used+extra <= len(chunk) {
				p.used += extra
				p.tailGrows++
				return append(chunk[start:p.used-extra:p.used], value)
			}
		}
	}
	next := p.carve(n * 2)
	next = append(next, slice...)
	return append(next, value)
}

func (a *Arena) appendNode(slice []Node, value Node) []Node {
	return poolAppend(&a.nodeItems, slice, value, 4)
}

func (a *Arena) appendValue(slice []Value, value Value) []Value {
	return poolAppend(&a.valueItems, slice, value, 2)
}

func (a *Arena) appendSelectorValue(slice []SelectorValue, value SelectorValue) []SelectorValue {
	return poolAppend(&a.selectorItems, slice, value, 4)
}

func (a *Arena) appendByteSlice(slice [][]byte, value []byte) [][]byte {
	return poolAppend(&a.byteSliceItems, slice, value, 2)
}

func (a *Arena) appendMediaQueryExpression(slice []MediaQueryExpression, value MediaQueryExpression) []MediaQueryExpression {
	return poolAppend(&a.mediaExprItems, slice, value, 1)
}

func (a *Arena) appendMediaFeature(slice []MediaFeature, value MediaFeature) []MediaFeature {
	return poolAppend(&a.mediaFeatItems, slice, value, 2)
}

func arenaNew[T any](a *Arena, slab *arenaSlab[T]) *T {
	return slab.new()
}

func (a *Arena) newDeclaration() *Declaration         { return arenaNew(a, &a.declarations) }
func (a *Arena) newSelector() *Selector               { return arenaNew(a, &a.selectors) }
func (a *Arena) newComment() *Comment                 { return arenaNew(a, &a.comments) }
func (a *Arena) newBasicValue() *BasicValue           { return arenaNew(a, &a.basicValues) }
func (a *Arena) newStringValue() *StringValue         { return arenaNew(a, &a.stringValues) }
func (a *Arena) newFunctionValue() *FunctionValue     { return arenaNew(a, &a.functionValues) }
func (a *Arena) newCharsetAtRule() *CharsetAtRule     { return arenaNew(a, &a.charsetRules) }
func (a *Arena) newImportAtRule() *ImportAtRule       { return arenaNew(a, &a.importRules) }
func (a *Arena) newMediaAtRule() *MediaAtRule         { return arenaNew(a, &a.mediaRules) }
func (a *Arena) newContainerAtRule() *ContainerAtRule { return arenaNew(a, &a.containerRules) }
func (a *Arena) newFontFaceAtRule() *FontFaceAtRule   { return arenaNew(a, &a.fontFaceRules) }
func (a *Arena) newFontFeatureValuesAtRule() *FontFeatureValuesAtRule {
	return arenaNew(a, &a.fontFeatureRules)
}
func (a *Arena) newCounterStyleAtRule() *CounterStyleAtRule { return arenaNew(a, &a.counterRules) }
func (a *Arena) newColorProfileAtRule() *ColorProfileAtRule {
	return arenaNew(a, &a.colorProfileRules)
}
func (a *Arena) newKeyframesAtRule() *KeyframesAtRule         { return arenaNew(a, &a.keyframesRules) }
func (a *Arena) newSupportsAtRule() *SupportsAtRule           { return arenaNew(a, &a.supportsRules) }
func (a *Arena) newLayerAtRule() *LayerAtRule                 { return arenaNew(a, &a.layerRules) }
func (a *Arena) newScopeAtRule() *ScopeAtRule                 { return arenaNew(a, &a.scopeRules) }
func (a *Arena) newStartingStyleAtRule() *StartingStyleAtRule { return arenaNew(a, &a.startingRules) }
func (a *Arena) newPropertyAtRule() *PropertyAtRule           { return arenaNew(a, &a.propertyRules) }
func (a *Arena) newFontPaletteValuesAtRule() *FontPaletteValuesAtRule {
	return arenaNew(a, &a.paletteRules)
}
func (a *Arena) newNamespaceAtRule() *NamespaceAtRule     { return arenaNew(a, &a.namespaceRules) }
func (a *Arena) newPageAtRule() *PageAtRule               { return arenaNew(a, &a.pageRules) }
func (a *Arena) newMarginBox() *MarginBox                 { return arenaNew(a, &a.marginBoxes) }
func (a *Arena) newPositionTryAtRule() *PositionTryAtRule { return arenaNew(a, &a.positionTryRules) }
func (a *Arena) newViewTransitionAtRule() *ViewTransitionAtRule {
	return arenaNew(a, &a.viewTransRules)
}
func (a *Arena) newGenericAtRule() *GenericAtRule { return arenaNew(a, &a.genericRules) }
func (a *Arena) newKeyframeStop() *KeyframeStop   { return arenaNew(a, &a.keyframeStops) }
