package pristinecss

import (
	"path/filepath"
	"testing"
)

// Fixture I/O and initial parse validation are outside the timed loops.
func benchmarkSheets(b *testing.B, run func(*testing.B, []byte, *Sheet)) {
	b.Helper()
	for _, path := range corpusFiles(b) {
		src := readCSS(b, path)
		s, errs := Parse(src)
		if len(errs) != 0 {
			b.Fatal(errs)
		}
		name := filepath.Base(path)
		b.Run(name[:len(name)-4], func(b *testing.B) { run(b, src, s) })
	}
}

var benchmarkWork int
var benchmarkBytes []byte
var benchmarkSheet *Sheet

// A lint-style consumer reads every visible kind/text and indexes named syntax.
func benchmarkLint(s *Sheet) int {
	properties := make(map[string]int)
	functions := make(map[string]int)
	var kinds [256]int
	n := 0
	s.Walk(func(c Cursor) Action {
		k := c.Kind()
		kinds[k]++
		text := c.Text()
		n += len(text)
		if len(text) > 0 {
			n += int(text[0]) + int(text[len(text)-1])
		}
		switch k {
		case KindDeclaration:
			properties[string(c.PropertyName())]++
		case KindValFunction:
			functions[string(c.FunctionName())]++
		}
		return Continue
	})
	for name, count := range properties {
		n += len(name) * count
	}
	for name, count := range functions {
		n += len(name) * count
	}
	for k, count := range kinds {
		n += k * count
	}
	return n
}

// Limit each edit kind to eight nodes. Separate targets avoid overlapping edits;
// sheets without a wrapper simply have no Unwrap target.
func benchmarkEdit(s *Sheet) {
	replaced, deleted, unwrapped := 0, 0, 0
	s.Walk(func(c Cursor) Action {
		switch c.Kind() {
		case KindValBasic:
			if replaced < 8 {
				s.ReplaceRaw(c.ID(), []byte("0"))
				replaced++
				return SkipChildren
			}
		case KindDeclaration:
			if replaced == 8 && deleted < 8 {
				s.Delete(c.ID())
				deleted++
				return SkipChildren
			}
		case KindAtMedia, KindAtSupports, KindAtLayer, KindAtContainer:
			if unwrapped < 8 {
				s.Unwrap(c.ID())
				unwrapped++
			}
		}
		return Continue
	})
}

func BenchmarkPipeline(b *testing.B) {
	benchmarkSheets(b, func(b *testing.B, src []byte, s *Sheet) {
		b.Run("Parse", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			b.ResetTimer()
			for range b.N {
				benchmarkSheet, _ = Parse(src)
			}
		})
		b.Run("ParseInto", func(b *testing.B) {
			var reuse Sheet
			ParseInto(src, &reuse)
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			b.ResetTimer()
			for range b.N {
				ParseInto(src, &reuse)
			}
		})
		b.Run("Lint", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchmarkWork = benchmarkLint(s)
			}
		})
		// Reparse restores mutations; EndToEnd includes this parse only once.
		b.Run("ParseIntoEditCompact", func(b *testing.B) {
			var reuse Sheet
			ParseInto(src, &reuse)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				ParseInto(src, &reuse)
				benchmarkEdit(&reuse)
				benchmarkWork = len(reuse.Compact())
			}
		})
		for _, style := range []struct {
			name  string
			style Style
		}{{"EmitMinified", Minified}, {"EmitPretty", Pretty}} {
			b.Run(style.name, func(b *testing.B) {
				dst := s.Emit(nil, EmitOptions{Style: style.style})
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					dst = s.Emit(dst[:0], EmitOptions{Style: style.style})
				}
				benchmarkBytes = dst
			})
		}
		for _, owned := range []bool{false, true} {
			name := "Bound"
			if owned {
				name = "Owned"
			}
			b.Run("Marshal"+name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					benchmarkBytes = s.Marshal(owned)
				}
			})
			blob := s.Marshal(owned)
			b.Run("Load"+name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					var err error
					benchmarkSheet, err = Load(blob, src)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
		b.Run("CompileIR", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var err error
				benchmarkBytes, err = s.CompileIR(nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		blob, err := s.CompileIR(nil)
		if err != nil {
			b.Fatal(err)
		}
		ir, err := LoadIR(blob)
		if err != nil {
			b.Fatal(err)
		}
		// Alternate required/fallback bits to select half the rules.
		for i := range ir.Rules {
			if i%2 == 0 {
				ir.Rules[i].Required[0] = 1
			} else {
				ir.Rules[i].Fallback[0] = 1
			}
		}
		b.Run("FilterIR", func(b *testing.B) {
			dst := make([]byte, 0, len(ir.CSS))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				dst = FilterIR(ir, [4]uint64{1}, dst[:0])
			}
			benchmarkBytes = dst
		})
		b.Run("EndToEnd", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			b.ResetTimer()
			for range b.N {
				fresh, errs := Parse(src)
				if len(errs) != 0 {
					b.Fatal(errs)
				}
				benchmarkWork = benchmarkLint(fresh)
				benchmarkEdit(fresh)
				benchmarkWork += len(fresh.Compact())
				benchmarkBytes = fresh.Emit(nil, EmitOptions{Style: Minified})
				benchmarkBytes = fresh.Emit(nil, EmitOptions{Style: Pretty})
				loaded, err := Load(fresh.Marshal(true), nil)
				if err != nil {
					b.Fatal(err)
				}
				frame, err := loaded.CompileIR(nil)
				if err != nil {
					b.Fatal(err)
				}
				compiled, err := LoadIR(frame)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkBytes = FilterIR(compiled, [4]uint64{}, nil)
			}
		})
	})
}
