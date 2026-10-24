package pristinecss

import "testing"

var errorBenchSheet *Sheet
var errorBenchErrors []ParseError

func BenchmarkParseBroken(b *testing.B) {
	for _, in := range errorInputs(b) {
		b.Run(in.set+"/"+in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.src)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				errorBenchSheet, errorBenchErrors = Parse(in.src)
			}
		})
	}
}
