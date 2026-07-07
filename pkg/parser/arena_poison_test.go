//go:build arenadebug

package parser

import (
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestArenaResetPoisonsStaleAST(t *testing.T) {
	source := []byte(`.x { color: red; }`)
	toks := lexer.Lex(source)
	arena := NewArena(len(toks))
	stale, errs := ParseInto(source, toks, arena)
	if len(errs) != 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	newSource := []byte(`.y { color: blue; }`)
	newToks := lexer.Lex(newSource)
	_, errs = ParseInto(newSource, newToks, arena)
	if len(errs) != 0 {
		t.Fatalf("reuse parse errors: %v", errs)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("stale AST use did not panic after Arena.Reset")
		}
		if !strings.Contains(r.(string), "stale AST") {
			t.Fatalf("panic = %v, want stale AST poison", r)
		}
	}()
	_ = stale.String()
}
