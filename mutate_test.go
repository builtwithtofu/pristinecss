package pristinecss

import "testing"

func TestDeleteUnwrapReplaceCompactAndExtract(t *testing.T) {
	s, _ := Parse([]byte("@web{a{color:oklch(1 2 3)}b{display:block}}@tui{c{display:none}}"))
	var web, tui, fn NodeID
	for c := range s.All(KindAtGeneric) {
		if string(c.Name()) == "web" {
			web = c.ID()
		} else {
			tui = c.ID()
		}
	}
	for c := range s.All(KindValFunction) {
		if string(c.FunctionName()) == "oklch" {
			fn = c.ID()
		}
	}
	s.Unwrap(web)
	s.Delete(tui)
	s.ReplaceRaw(fn, []byte("rgb(1 2 3)"))
	out := string(s.Emit(nil, EmitOptions{}))
	if out != "a{color:rgb(1 2 3)}b{display:block}" {
		t.Fatalf("emit after mutation = %q", out)
	}
	remap := s.Compact()
	verify(t, s)
	if remap[fn] == NoNode {
		t.Fatalf("remap lost replaced node")
	}
	if got := string(s.Emit(nil, EmitOptions{})); got != out {
		t.Fatalf("compact changed emit: %q vs %q", got, out)
	}
	ex := s.ExtractOwned(remap[fn])
	if string(ex.Emit(nil, EmitOptions{})) != "rgb(1 2 3)" {
		t.Fatalf("extract owned = %q", ex.Emit(nil, EmitOptions{}))
	}
}

func TestExtractOwnedSurvivesSourceOverwrite(t *testing.T) {
	src := []byte("a{color:red}")
	s, _ := Parse(src)
	var rule NodeID
	for c := range s.All(KindStyleRule) {
		rule = c.ID()
	}
	ex := s.ExtractOwned(rule)
	for i := range src {
		src[i] = 0
	}
	if got := string(ex.Emit(nil, EmitOptions{})); got != "a{color:red}" {
		t.Fatalf("owned extract after overwrite = %q", got)
	}
}

func TestCompactReclaimsScratch(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}b{color:blue}"))
	var first, second NodeID
	for c := range s.All(KindStyleRule) {
		if first == 0 {
			first = c.ID()
		} else {
			second = c.ID()
		}
	}
	s.ReplaceRaw(first, []byte("x{color:black}"))
	s.ReplaceRaw(first, []byte("x{color:white}"))
	s.ReplaceRaw(second, []byte("y{color:green}"))
	s.Delete(second)
	before := len(s.scratch)
	s.Compact()
	wantScratch := scratchHeaderLen + len("x{color:white}")
	if len(s.scratch) != wantScratch {
		t.Fatalf("scratch size after compact = %d, want %d (before %d)", len(s.scratch), wantScratch, before)
	}
	if got := string(s.Emit(nil, EmitOptions{})); got != "x{color:white}" {
		t.Fatalf("emit after compact = %q", got)
	}
}

func TestReplaceRawTopLevelRuleRemainsInIR(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}"))
	var rule NodeID
	for c := range s.All(KindStyleRule) {
		rule = c.ID()
	}
	s.ReplaceRaw(rule, []byte("x{color:blue}"))
	blob, err := s.CompileIR(nil)
	if err != nil {
		t.Fatal(err)
	}
	ir, err := LoadIR(blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Rules) != 1 {
		t.Fatalf("IR rules = %d, want 1", len(ir.Rules))
	}
	if got := string(FilterIR(ir, [4]uint64{}, nil)); got != "x{color:blue}" {
		t.Fatalf("filtered IR = %q", got)
	}
}

func TestRootMutationsAffectWholeSheet(t *testing.T) {
	t.Run("replace", func(t *testing.T) {
		s, _ := Parse([]byte("a{color:red}"))
		s.ReplaceRaw(0, []byte("b{color:blue}"))
		assertEmit := func(stage string, sheet *Sheet) {
			t.Helper()
			if got := string(sheet.Emit(nil, EmitOptions{})); got != "b{color:blue}" {
				t.Fatalf("%s emit = %q", stage, got)
			}
		}
		assertEmit("replace", s)
		s.Compact()
		assertEmit("compact", s)
		loaded, err := Load(s.Marshal(true), nil)
		if err != nil {
			t.Fatal(err)
		}
		assertEmit("load", loaded)
	})
	t.Run("delete", func(t *testing.T) {
		s, _ := Parse([]byte("a{color:red}"))
		s.Delete(0)
		if got := s.Emit(nil, EmitOptions{}); len(got) != 0 {
			t.Fatalf("deleted root emit = %q", got)
		}
		s.Compact()
		verify(t, s)
		loaded, err := Load(s.Marshal(true), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := loaded.Emit(nil, EmitOptions{}); len(got) != 0 {
			t.Fatalf("loaded deleted root emit = %q", got)
		}
	})
	t.Run("unwrap", func(t *testing.T) {
		s, _ := Parse([]byte("a{color:red}"))
		s.Unwrap(0)
		s.Compact()
		verify(t, s)
		if got := string(s.Emit(nil, EmitOptions{})); got != "a{color:red}" {
			t.Fatalf("unwrapped root emit = %q", got)
		}
	})
}

func TestNestedReplacementSurvivesCompactAndMarshal(t *testing.T) {
	s, _ := Parse([]byte("@media screen and (min-width:1px){a{color:red}}"))
	var feature NodeID = NoNode
	for c := range s.All(KindMediaFeature) {
		feature = c.ID()
	}
	s.ReplaceRaw(feature, []byte("(max-width:2px)"))
	want := "@media screen and (max-width:2px){a{color:red}}"
	if got := string(s.Emit(nil, EmitOptions{})); got != want {
		t.Fatalf("replace emit = %q", got)
	}
	s.Compact()
	if got := string(s.Emit(nil, EmitOptions{})); got != want {
		t.Fatalf("compact emit = %q", got)
	}
	loaded, err := Load(s.Marshal(true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(loaded.Emit(nil, EmitOptions{})); got != want {
		t.Fatalf("load emit = %q", got)
	}
}
