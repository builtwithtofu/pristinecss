package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestFrameworkParseAllocationBudgets(t *testing.T) {
	// Ceilings are ~1.5x the values measured after the arena container-pool
	// fixes (2026-07-07, count=6: bootstrap 431, daisyui 2693, shoelace 24).
	budgets := map[string]float64{
		"bootstrap.css":      650,
		"bulma.css":          1700,
		"daisyui.css":        4100,
		"foundation.css":     1300,
		"materialize.css":    1650,
		"milligram.css":      120,
		"open-props.css":     210,
		"pico.css":           450,
		"primeflex.css":      550,
		"shoelace-dark.css":  40,
		"shoelace-light.css": 40,
		"spectre.css":        600,
		"water.css":          220,
	}

	frameworkDir := filepath.Join("..", "..", "test-data", "frameworks")
	entries, err := os.ReadDir(frameworkDir)
	if err != nil {
		t.Fatalf("read framework fixture directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		budget, ok := budgets[entry.Name()]
		if !ok {
			t.Fatalf("missing allocation budget for %s", entry.Name())
		}
		path := filepath.Join(frameworkDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		toks := lexer.Lex(content)

		t.Run(entry.Name(), func(t *testing.T) {
			allocs := testing.AllocsPerRun(1, func() {
				_, errs := Parse(content, toks)
				if len(errs) > 0 {
					t.Fatalf("parse errors: %v", errs)
				}
			})
			if allocs > budget {
				t.Fatalf("allocs/run = %.0f, budget %.0f", allocs, budget)
			}
		})
	}
}

func TestArenaOverflowParses(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString(".x { color: red; margin: 1px; }\n")
	}
	css := []byte(b.String())
	toks := lexer.Lex(css)
	arena := NewArena(1)
	_, errs := ParseInto(css, toks, arena)
	if len(errs) > 0 {
		limit := len(errs)
		if limit > 5 {
			limit = 5
		}
		t.Fatalf("overflow parse errors: %v", errs[:limit])
	}
}

func TestParseIntoReuseMatchesParse(t *testing.T) {
	css := `.x { color: red; @media (width >= 1px) { color: blue; .y { margin: 1px; } } }`
	source := []byte(css)
	toks := lexer.Lex(source)
	fresh, freshErrs := Parse(source, toks)
	arena := NewArena(len(toks))
	reused, reusedErrs := ParseInto(source, toks, arena)
	if len(freshErrs) != 0 || len(reusedErrs) != 0 {
		t.Fatalf("fresh errors %v, reused errors %v", freshErrs, reusedErrs)
	}
	if fresh.String() != reused.String() {
		t.Fatalf("ParseInto AST differs from Parse\nfresh: %s\nreused: %s", fresh.String(), reused.String())
	}
}
