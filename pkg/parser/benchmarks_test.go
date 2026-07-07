package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

// Benchmark for parsing all checked-in CSS framework fixtures.
func BenchmarkParseFrameworks(b *testing.B) {
	frameworkDir := filepath.Join("..", "..", "test-data", "frameworks")
	entries, err := os.ReadDir(frameworkDir)
	if err != nil {
		b.Fatalf("Could not read framework fixture directory %s: %v", frameworkDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		path := filepath.Join(frameworkDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("Could not read the file %s: %v", path, err)
		}
		tokens := lexer.Lex(content)

		b.Run(strings.TrimSuffix(entry.Name(), ".css"), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				Parse(content, tokens)
			}
		})
	}
}
