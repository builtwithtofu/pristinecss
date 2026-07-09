package pristinecss

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripFrameworks(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("test-data", "frameworks")); if err != nil { t.Fatal(err) }
	for _, e := range entries { if e.IsDir() || filepath.Ext(e.Name()) != ".css" { continue }; t.Run(e.Name(), func(t *testing.T){ src:=readFramework(t,e.Name()); s,errs:=Parse(src); if len(errs)!=0{t.Fatal(errs)}; assertStructuralRoundTrip(t, s) }) }
}

func BenchmarkEmit(b *testing.B) {
	s, _ := Parse(readFramework(b, "bootstrap.css"))
	dst := make([]byte, 0, len(s.src))
	b.ReportAllocs(); b.ResetTimer()
	for i:=0;i<b.N;i++ { dst = s.Emit(dst[:0], EmitOptions{}) }
}
