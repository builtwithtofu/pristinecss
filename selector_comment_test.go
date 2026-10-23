package pristinecss

import "testing"

func TestSelectorCommentBetweenWhitespace(t *testing.T) {
	s, errs := Parse([]byte(".a, /* comment */ .b{color:red}"))
	if len(errs) != 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	assertStructuralRoundTrip(t, s)
	if got := string(s.Emit(nil, EmitOptions{})); got != ".a, .b{color:red}" {
		t.Fatalf("emitted = %q", got)
	}
}
