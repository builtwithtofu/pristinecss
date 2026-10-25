package pristinecss

import "testing"

func TestParseEscapedImportant(t *testing.T) {
	for _, priority := range []string{
		`!\69mportant`, `!\49MPORTANT`, `!\000069 mportant`,
		`!\69\6d\70\6f\72\74\61\6e\74`, `!/**/\69mportant/**/`,
		"!\\69\r\nmportant",
	} {
		t.Run(priority, func(t *testing.T) {
			src := "a{-webkit-transform:rotate(45deg)" + priority + ";transform:none}"
			s, errs := Parse([]byte(src))
			if len(errs) != 0 {
				t.Fatal(errs)
			}
			verify(t, s)
			declarations := 0
			for c := range s.All(KindDeclaration) {
				want := declarations == 0
				if important := c.Flags()&FlagImportant != 0; important != want {
					t.Errorf("%s important=%v, want %v", c.PropertyName(), important, want)
				}
				declarations++
			}
			if declarations != 2 {
				t.Fatalf("declarations=%d, want 2", declarations)
			}
			want := "a{-webkit-transform:rotate(45deg)!important;transform:none}"
			if got := string(s.Emit(nil, EmitOptions{Style: Minified})); got != want {
				t.Errorf("emit=%s, want %s", got, want)
			}
		})
	}
}

func TestParseImportantOnlyAtDeclarationEnd(t *testing.T) {
	for _, value := range []string{
		`rotate(45deg)!\69mportantx`, `rotate(45deg)!\69mportant junk`,
		`var(--fallback,!\69mportant)`, `rotate(45deg)!İmportant`,
	} {
		s, errs := Parse([]byte("a{transform:" + value + "}"))
		if len(errs) != 0 {
			t.Fatal(errs)
		}
		for c := range s.All(KindDeclaration) {
			if c.Flags()&FlagImportant != 0 {
				t.Errorf("incorrect priority for %s", value)
			}
		}
	}
}
