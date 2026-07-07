package parser

import (
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestImportAtRule(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ImportAtRule)
	}{
		{
			name:  "simple url",
			input: `@import url("styles.css");`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if rule.URL == nil {
					t.Fatalf("URL not parsed")
				}
			},
		},
		{
			name:  "media query",
			input: `@import url("mobile.css") screen and (max-width: 600px);`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if got := len(rule.Media.Queries); got != 1 {
					t.Fatalf("media queries = %d", got)
				}
			},
		},
		{
			name:  "layer",
			input: `@import url("theme.css") layer(theme);`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if rule.Layer == nil {
					t.Fatalf("layer not parsed")
				}
			},
		},
		{
			name:  "supports declaration",
			input: `@import url("modern.css") supports(display: flex);`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if rule.Supports == nil || rule.Supports.Kind != ConditionDeclaration {
					t.Fatalf("supports = %#v", rule.Supports)
				}
			},
		},
		{
			name:  "supports not",
			input: `@import url("fallback.css") supports(not (display: flex));`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if rule.Supports == nil || rule.Supports.Kind != ConditionNot {
					t.Fatalf("supports = %#v", rule.Supports)
				}
			},
		},
		{
			name:  "supports function",
			input: `@import url("feature.css") supports(selector(:has(> img)));`,
			check: func(t *testing.T, rule *ImportAtRule) {
				if rule.Supports == nil || rule.Supports.Kind != ConditionSelector {
					t.Fatalf("supports = %#v", rule.Supports)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.input)
			toks := lexer.Lex(source)
			ss, errs := Parse(source, toks)
			if len(errs) > 0 {
				t.Fatalf("unexpected parse errors: %v", errs)
			}
			if len(ss.Rules) != 1 {
				t.Fatalf("rules = %d", len(ss.Rules))
			}
			rule, ok := ss.Rules[0].(*ImportAtRule)
			if !ok {
				t.Fatalf("rule type %T", ss.Rules[0])
			}
			tt.check(t, rule)
		})
	}
}
