package pristinecss

import (
	"fmt"
	"io"
	"strings"
)

// Dump writes an indented, kind-named view of the sheet for tests and debugging.
func (s *Sheet) Dump(w io.Writer) {
	if s == nil { return }
	var rec func(NodeID, int)
	rec = func(id NodeID, depth int) {
		if int(id) >= len(s.nodes) { return }
		n := s.nodes[id]
		text := s.nodeText(n)
		text = strings.ReplaceAll(text, "\n", "\\n")
		if len(text) > 48 { text = text[:48] + "…" }
		fmt.Fprintf(w, "%s%d %s [%d:%d] sub=%d %q\n", strings.Repeat("  ", depth), id, n.Kind, n.Lo, n.Hi, n.Sub, text)
		if n.Kind.isOpaque() { return }
		end := id + NodeID(n.Sub)
		for c := id + 1; c < end && int(c) < len(s.nodes); c += NodeID(s.nodes[c].Sub) { rec(c, depth+1) }
	}
	rec(0, 0)
}

func (s *Sheet) nodeText(n Node) string {
	buf := s.src
	if n.Flags&FlagScratch != 0 { buf = s.scratch }
	if int(n.Lo) > len(buf) || n.Hi < n.Lo { return "" }
	hi := int(n.Hi); if hi > len(buf) { hi = len(buf) }
	return string(buf[n.Lo:uint32(hi)])
}
