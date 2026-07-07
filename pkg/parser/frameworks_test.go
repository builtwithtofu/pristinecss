package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestCanParseFrameworksWithoutError(t *testing.T) {
	frameworkDir := filepath.Join("..", "..", "test-data", "frameworks")
	entries, err := os.ReadDir(frameworkDir)
	if err != nil {
		t.Fatalf("Could not read framework fixture directory %s: %v", frameworkDir, err)
	}

	cssFiles := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		cssFiles++
		path := filepath.Join(frameworkDir, entry.Name())
		t.Run(strings.TrimSuffix(entry.Name(), ".css"), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Could not read the file %s: %v", path, err)
			}

			tokens := lexer.Lex(content)
			_, errors := Parse(content, tokens)

			if len(errors) > 0 {
				t.Errorf("Parsing %s produced %d errors:", entry.Name(), len(errors))
				for _, err := range errors {
					t.Errorf("  - %v", err)
				}
			}
		})
	}
	if cssFiles == 0 {
		t.Fatalf("No CSS framework fixtures found in %s", frameworkDir)
	}
}
