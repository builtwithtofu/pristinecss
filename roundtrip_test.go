package pristinecss

import "testing"

func TestRoundTripFrameworks(t *testing.T) {
	for _, path := range corpusFiles(t) {
		t.Run(path, func(t *testing.T) {
			src := readCSS(t, path)
			s, errs := Parse(src)
			t.Logf("bytes=%d parse_errors=%d", len(src), len(errs))
			if len(errs) != 0 {
				t.Fatalf("parse errors: %#v", errs)
			}
			assertStructuralRoundTrip(t, s)
		})
	}
}

func BenchmarkEmit(b *testing.B) {
	s, _ := Parse(readFramework(b, "bootstrap.css"))
	dst := make([]byte, 0, len(s.src))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = s.Emit(dst[:0], EmitOptions{})
	}
}
