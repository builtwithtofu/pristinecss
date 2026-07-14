package pristinecss

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParseFrameworks(b *testing.B) {
	entries, err := os.ReadDir(filepath.Join("test-data", "frameworks"))
	if err != nil { b.Fatal(err) }
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".css" { continue }
		src := readFramework(b, e.Name())
		b.Run(e.Name()[:len(e.Name())-4], func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ { _, _ = Parse(src) }
		})
	}
}

func BenchmarkParseIntoFrameworks(b *testing.B) {
	src := readFramework(b, "bootstrap.css")
	var s Sheet
	ParseInto(src, &s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ { _ = ParseInto(src, &s) }
}

func BenchmarkParseErrorPositions(b *testing.B) {
	src := append(bytes.Repeat([]byte("x{}\n"), 10_000), '}')
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(src)
	}
}

func BenchmarkParseErrorAccessors(b *testing.B) {
	src := append(bytes.Repeat([]byte("x{}\n"), 10_000), '}')
	_, errs := Parse(src)
	if len(errs) != 1 {
		b.Fatalf("errors = %d, want 1", len(errs))
	}
	var line, column int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		line, column = errs[0].Line(), errs[0].Column()
	}
	benchmarkParseErrorPosition = line + column
}

var benchmarkParseErrorPosition int
