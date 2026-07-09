package pristinecss

import (
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
