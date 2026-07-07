//go:build arenadebug

package parser

var arenaPoisonBytes = []byte("<arena reset: stale AST node>")

type arenaPoisonNode struct{}

func (arenaPoisonNode) Type() NodeType { panic("stale AST node used after Arena.Reset") }
func (arenaPoisonNode) String() string { panic("stale AST node used after Arena.Reset") }

type arenaPoisonValue struct{}

func (arenaPoisonValue) Type() NodeType       { panic("stale AST value used after Arena.Reset") }
func (arenaPoisonValue) ValueType() ValueType { panic("stale AST value used after Arena.Reset") }
func (arenaPoisonValue) String() string       { panic("stale AST value used after Arena.Reset") }

func (a *Arena) poison() {
	poisonSlab(&a.declarations, func(d *Declaration) {
		d.Key = arenaPoisonBytes
		d.Value = []Value{arenaPoisonValue{}}
	})
	poisonSlab(&a.selectors, func(s *Selector) {
		s.Selectors = []SelectorValue{{Value: arenaPoisonBytes}}
		s.Rules = []Node{arenaPoisonNode{}}
	})
	poisonSlab(&a.comments, func(c *Comment) { c.Text = arenaPoisonBytes })
	poisonSlab(&a.basicValues, func(v *BasicValue) { v.Value = arenaPoisonBytes })
	poisonSlab(&a.stringValues, func(v *StringValue) { v.Value = arenaPoisonBytes })
	poisonSlab(&a.functionValues, func(v *FunctionValue) {
		v.Name = arenaPoisonBytes
		v.Arguments = []Value{arenaPoisonValue{}}
	})
	poisonSlab(&a.keyframeStops, func(s *KeyframeStop) { s.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.mediaRules, func(r *MediaAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.containerRules, func(r *ContainerAtRule) { r.Declarations = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.supportsRules, func(r *SupportsAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.layerRules, func(r *LayerAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.scopeRules, func(r *ScopeAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.startingRules, func(r *StartingStyleAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.pageRules, func(r *PageAtRule) { r.Rules = []Node{arenaPoisonNode{}} })
	poisonSlab(&a.genericRules, func(r *GenericAtRule) { r.Block = []Node{arenaPoisonNode{}} })
	poisonPool(&a.nodeItems, func(n *Node) { *n = arenaPoisonNode{} })
	poisonPool(&a.valueItems, func(v *Value) { *v = arenaPoisonValue{} })
	poisonPool(&a.selectorItems, func(sv *SelectorValue) { sv.Value = arenaPoisonBytes })
	poisonPool(&a.byteSliceItems, func(b *[]byte) { *b = arenaPoisonBytes })
	poisonPool(&a.mediaExprItems, func(e *MediaQueryExpression) {
		e.MediaType = arenaPoisonBytes
		e.Features = nil
	})
	poisonPool(&a.mediaFeatItems, func(f *MediaFeature) {
		f.Name = arenaPoisonBytes
		f.Value = arenaPoisonBytes
	})
}

func poisonPool[T any](pool *arenaPool[T], poison func(*T)) {
	for chunkIndex, chunk := range pool.chunks {
		limit := len(chunk)
		if chunkIndex == pool.chunk {
			limit = pool.used
		} else if chunkIndex > pool.chunk {
			limit = 0
		}
		for i := 0; i < limit; i++ {
			poison(&chunk[i])
		}
	}
	pool.chunks = nil
	pool.chunk = 0
	pool.used = 0
}

func poisonSlab[T any](slab *arenaSlab[T], poison func(*T)) {
	for chunkIndex, chunk := range slab.chunks {
		limit := len(chunk)
		switch {
		case chunkIndex < slab.chunk:
			limit = len(chunk)
		case chunkIndex == slab.chunk:
			limit = slab.index
		default:
			limit = 0
		}
		for i := 0; i < limit; i++ {
			poison(&chunk[i])
		}
	}
	slab.chunks = nil
	slab.chunk = 0
	slab.index = 0
}
