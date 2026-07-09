package pristinecss

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParserRecovery(t *testing.T) {
	tests := []struct {
		name        string
		css         string
		badNeedle   string
		wantMessage string
		wantRule    string
	}{
		{
			name:        "unclosed block at EOF",
			css:         "broken{color:red; after{color:green}",
			badNeedle:   "broken{",
			wantMessage: "expected '}'",
			wantRule:    "after",
		},
		{
			name:        "stray close brace between valid rules",
			css:         "before{color:red}} after{color:green}",
			badNeedle:   "}}",
			wantMessage: "expected rule",
			wantRule:    "after",
		},
		{
			name:        "malformed at-rule prelude",
			css:         "@media (min-width:{bad:css} after{color:green}",
			badNeedle:   "{bad",
			wantMessage: "expected closing",
			wantRule:    "after",
		},
		{
			name:        "garbage bytes mid-sheet",
			css:         "before{color:red} ??? after{color:green}",
			badNeedle:   "???",
			wantMessage: "expected rule",
			wantRule:    "after",
		},
		{
			name:        "block depth cap",
			css:         deepCSS(130),
			badNeedle:   "x128{",
			wantMessage: "maximum block depth exceeded",
			wantRule:    "tail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := parseWithin(t, []byte(tt.css), 250*time.Millisecond)
			if len(errs) == 0 {
				t.Fatalf("Parse returned no errors for malformed CSS\n%s", dumpString(s))
			}
			badStart := strings.Index(tt.css, tt.badNeedle)
			if badStart < 0 {
				t.Fatalf("test bug: badNeedle %q not in input", tt.badNeedle)
			}
			badEnd := uint32(len(tt.css))
			if tt.name != "unclosed block at EOF" {
				badEnd = uint32(badStart + len(tt.badNeedle))
			}
			if !hasRecoveryError(errs, tt.wantMessage, uint32(badStart), badEnd) {
				t.Fatalf("errors did not contain %q in [%d:%d]: %#v", tt.wantMessage, badStart, badEnd, errs)
			}
			verify(t, s)
			if !hasStyleRuleSelector(s, tt.wantRule) {
				t.Fatalf("valid rule %q was not present after recovery\n%s", tt.wantRule, dumpString(s))
			}
			_ = s.Emit(nil, EmitOptions{})
		})
	}
}

func TestMalformedComponentRecovery(t *testing.T) {
	tests := []struct{ name, css, wantMessage string }{
		{"functional pseudo", "a:not(.x{color:red}b{color:blue}", "expected ')'"},
		{"attribute selector", "a[href=x{color:red}b{color:blue}", "expected ']'"},
		{"value function", "a{x:func(1}b{color:blue}", "expected ')'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := parseWithin(t, []byte(tt.css), 250*time.Millisecond)
			if !hasRecoveryError(errs, tt.wantMessage, 0, uint32(len(tt.css))) {
				t.Fatalf("errors did not contain %q: %#v", tt.wantMessage, errs)
			}
			verify(t, s)
			if !hasStyleRuleSelector(s, "b") {
				t.Fatalf("valid trailing rule missing after recovery\n%s", dumpString(s))
			}
			_ = s.Emit(nil, EmitOptions{})
		})
	}
}

func parseWithin(t testing.TB, src []byte, d time.Duration) (*Sheet, []ParseError) {
	t.Helper()
	type result struct {
		s    *Sheet
		errs []ParseError
	}
	ch := make(chan result, 1)
	go func() { s, errs := Parse(src); ch <- result{s, errs} }()
	select {
	case r := <-ch:
		return r.s, r.errs
	case <-time.After(d):
		t.Fatalf("Parse did not return within %s", d)
		return nil, nil
	}
}

func hasRecoveryError(errs []ParseError, msg string, lo, hi uint32) bool {
	for _, err := range errs {
		if strings.Contains(err.Message, msg) && err.Offset >= lo && err.Offset <= hi {
			return true
		}
	}
	return false
}

func hasStyleRuleSelector(s *Sheet, selector string) bool {
	for c := range s.All(KindStyleRule) {
		text := strings.TrimSpace(string(c.Text()))
		if strings.HasPrefix(text, selector+"{") || strings.HasPrefix(text, selector+" {") {
			return true
		}
	}
	return false
}

func deepCSS(depth int) string {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&b, "x%d{", i)
	}
	b.WriteString("after{color:green}")
	for i := 0; i < depth; i++ {
		b.WriteByte('}')
	}
	b.WriteString(" tail{color:blue}")
	return b.String()
}
