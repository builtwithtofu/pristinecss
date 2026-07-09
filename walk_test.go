package pristinecss

import (
	"sync"
	"testing"
)

func TestWalkDocumentOrderAndSkip(t *testing.T) {
	s, errs := Parse([]byte("a{color:red}@media screen{b{display:block}}"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	verify(t, s)
	var got []Kind
	s.Walk(func(c Cursor) Action {
		got = append(got, c.Kind())
		if c.Kind() == KindAtMedia {
			return SkipChildren
		}
		return Continue
	})
	if !equalKinds(got, []Kind{KindStyleRule, KindSelElement, KindBlock, KindDeclaration, KindValBasic, KindAtMedia}) {
		t.Fatalf("walk order = %v", got)
	}
}

func TestWalkTombstoneAndAll(t *testing.T) {
	s, _ := Parse([]byte("a{}b{}"))
	var first NodeID
	for c := range s.All(KindStyleRule) {
		first = c.ID()
		break
	}
	s.nodes[first].Flags |= FlagTombstone
	count := 0
	for range s.All(KindStyleRule) {
		count++
	}
	if count != 1 {
		t.Fatalf("style rules after tombstone = %d, want 1", count)
	}
}

func TestWalkUnwrap(t *testing.T) {
	s, _ := Parse([]byte("@web{x{}y{}}"))
	for c := range s.All(KindAtGeneric) {
		s.nodes[c.ID()].Flags |= FlagUnwrap
	}
	var rules int
	s.Walk(func(c Cursor) Action {
		if c.Kind() == KindStyleRule {
			rules++
		}
		if c.Kind() == KindAtGeneric {
			t.Fatal("unwrapped node was visible")
		}
		return Continue
	})
	if rules != 2 {
		t.Fatalf("unwrapped rules = %d", rules)
	}
}

func TestPosition(t *testing.T) {
	s, _ := Parse([]byte("a{}\n\n  b{color:red}"))
	var b NodeID
	for c := range s.All(KindStyleRule) {
		b = c.ID()
	}
	line, col := s.Position(b)
	if line != 3 || col != 3 {
		t.Fatalf("position = %d:%d", line, col)
	}
}

func TestConcurrentWalk(t *testing.T) {
	s, _ := Parse(readFramework(t, "bootstrap.css"))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.Walk(func(Cursor) Action { return Continue })
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentPosition(t *testing.T) {
	s, _ := Parse(readFramework(t, "bootstrap.css"))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = s.Position(NodeID(j % len(s.nodes)))
			}
		}()
	}
	wg.Wait()
}

func TestNextSiblingStaysWithinParent(t *testing.T) {
	s, _ := Parse([]byte("a{color:red;margin:0}b{color:blue}"))
	var declarations []Cursor
	for c := range s.All(KindDeclaration) {
		declarations = append(declarations, c)
	}
	sibling, ok := declarations[0].NextSibling()
	if !ok || sibling.ID() != declarations[1].ID() {
		t.Fatalf("first declaration sibling = %d, %v", sibling.ID(), ok)
	}
	if sibling, ok = declarations[1].NextSibling(); ok {
		t.Fatalf("last declaration sibling = %s", sibling.Kind())
	}
}

func equalKinds(a, b []Kind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
