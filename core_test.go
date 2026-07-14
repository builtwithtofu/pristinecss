package pristinecss

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func verify(t testing.TB, s *Sheet) {
	t.Helper()
	if s == nil { t.Fatal("nil sheet") }
	if len(s.nodes) == 0 { t.Fatal("empty node array") }
	var rec func(NodeID) uint32
	rec = func(id NodeID) uint32 {
		n := s.nodes[id]
		if n.Sub == 0 { t.Fatalf("node %d has Sub=0", id) }
		if int(id)+int(n.Sub) > len(s.nodes) { t.Fatalf("node %d subtree escapes node array", id) }
		if n.Hi < n.Lo { t.Fatalf("node %d has inverted span", id) }
		if n.Flags&FlagScratch == 0 && int(n.Hi) > len(s.src) { t.Fatalf("node %d span escapes src", id) }
		if n.Flags&FlagScratch != 0 && int(n.Hi) > len(s.scratch) { t.Fatalf("node %d span escapes scratch", id) }
		if !n.Kind.isOpaque() {
			end := id + NodeID(n.Sub)
			for c := id + 1; c < end; c += NodeID(s.nodes[c].Sub) {
				child := s.nodes[c]
				if n.Flags&FlagScratch == 0 && child.Flags&FlagScratch == 0 && child.Lo < n.Lo || (n.Flags&FlagScratch == 0 && child.Flags&FlagScratch == 0 && child.Hi > n.Hi) { t.Fatalf("child %d span [%d:%d] outside parent %d [%d:%d]", c, child.Lo, child.Hi, id, n.Lo, n.Hi) }
				rec(c)
			}
		}
		return n.Sub
	}
	rec(0)
}

func TestNodeSizeAndNoPointers(t *testing.T) {
	if got := unsafe.Sizeof(Node{}); got != 16 { t.Fatalf("Node size = %d, want 16", got) }
	typ := reflect.TypeOf(Node{})
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Type.Kind() == reflect.Pointer || typ.Field(i).Type.Kind() == reflect.Slice || typ.Field(i).Type.Kind() == reflect.Map || typ.Field(i).Type.Kind() == reflect.Interface || typ.Field(i).Type.Kind() == reflect.String {
			t.Fatalf("Node field %s contains/scans a pointer", typ.Field(i).Name)
		}
	}
}

func TestKindsTableComplete(t *testing.T) {
	for k := Kind(0); k < kindCount; k++ { if kinds[k].name == "" { t.Fatalf("missing kind row %d", k) } }
}

func TestDumpHandBuiltSheet(t *testing.T) {
	s := &Sheet{src: []byte("a{color:red}"), nodes: []Node{{Kind:KindStylesheet,Lo:0,Hi:12,Sub:5},{Kind:KindStyleRule,Lo:0,Hi:12,Sub:4},{Kind:KindSelElement,Lo:0,Hi:1,Sub:1},{Kind:KindBlock,Lo:1,Hi:12,Sub:2},{Kind:KindDeclaration,Aux:5,Lo:2,Hi:11,Sub:1}}}
	verify(t, s)
	var b bytes.Buffer
	s.Dump(&b)
	want := "0 stylesheet [0:12] sub=5 \"a{color:red}\"\n  1 style-rule [0:12] sub=4 \"a{color:red}\"\n    2 selector-element [0:1] sub=1 \"a\"\n    3 block [1:12] sub=2 \"{color:red}\"\n      4 declaration [2:11] sub=1 \"color:red\"\n"
	if got := b.String(); got != want { t.Fatalf("Dump mismatch\nwant:\n%s\ngot:\n%s", want, got) }
}

func TestParseErrorPositionsAreOwned(t *testing.T) {
	src := []byte("a{}\n  }\n    }")
	_, errs := Parse(src)
	if len(errs) != 2 {
		t.Fatalf("errors = %d, want 2", len(errs))
	}
	want := [][2]int{{2, 3}, {3, 5}}
	for i, err := range errs {
		if err.Line() != want[i][0] || err.Column() != want[i][1] {
			t.Fatalf("error %d line/col = %d/%d, want %d/%d", i, err.Line(), err.Column(), want[i][0], want[i][1])
		}
	}

	for i := range src {
		src[i] = 'x'
	}
	if errs[0].Line() != 2 || errs[0].Column() != 3 {
		t.Fatalf("position changed after source reuse: %d/%d", errs[0].Line(), errs[0].Column())
	}

	var line, column int
	allocs := testing.AllocsPerRun(100, func() {
		line, column = errs[0].Line(), errs[0].Column()
	})
	if allocs != 0 {
		t.Fatalf("Line/Column allocs = %.0f, want 0", allocs)
	}
	if line != 2 || column != 3 {
		t.Fatalf("line/col after allocation check = %d/%d", line, column)
	}

	var zero ParseError
	if zero.Line() != 1 || zero.Column() != 1 {
		t.Fatalf("zero ParseError line/col = %d/%d, want 1/1", zero.Line(), zero.Column())
	}
}

func dumpString(s *Sheet) string { var b strings.Builder; s.Dump(&b); return b.String() }
