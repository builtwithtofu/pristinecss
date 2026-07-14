package pristinecss

import "sort"

// Position returns a node's 1-based line and column without mutating the sheet.
func (s *Sheet) Position(id NodeID) (line, col int) {
	if s == nil || int(id) >= len(s.nodes) {
		return 0, 0
	}
	s.lineOnce.Do(func() { s.lines = buildLines(s.src, s.lines) })
	return lineCol(s.src, s.lines, s.nodes[id].Lo)
}

func buildLines(src []byte, dst []uint32) []uint32 {
	dst = dst[:0]
	for i, b := range src {
		if b == '\n' {
			dst = append(dst, uint32(i))
		}
	}
	return dst
}

func resolveParseErrorPositions(src []byte, errs []ParseError) {
	var cursor, line, lineStart uint32
	for i := range errs {
		off := errs[i].Offset
		if int(off) > len(src) {
			off = uint32(len(src))
		}
		if off < cursor {
			cursor, line, lineStart = 0, 0, 0
		}
		for cursor < off {
			if src[cursor] == '\n' {
				line++
				lineStart = cursor + 1
			}
			cursor++
		}
		errs[i].line = line
		errs[i].column = off - lineStart
	}
}

func lineCol(src []byte, lines []uint32, off uint32) (int, int) {
	if int(off) > len(src) {
		off = uint32(len(src))
	}
	i := sort.Search(len(lines), func(i int) bool { return lines[i] >= off })
	lineStart := uint32(0)
	if i > 0 {
		lineStart = lines[i-1] + 1
	}
	return i + 1, int(off-lineStart) + 1
}
