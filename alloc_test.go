package pristinecss

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllocBudget(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("test-data", "frameworks"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".css" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src := readFramework(t, e.Name())
			cold := testing.AllocsPerRun(20, func() { _, _ = Parse(src) })
			if cold > 4 {
				t.Fatalf("Parse allocs = %.0f, want <=4", cold)
			}
			var s Sheet
			ParseInto(src, &s)
			warm := testing.AllocsPerRun(20, func() { _ = ParseInto(src, &s) })
			if warm != 0 {
				t.Fatalf("warm ParseInto allocs = %.0f, want 0", warm)
			}
		})
	}
}

func TestParseIntoRelexesRefilledBuffer(t *testing.T) {
	buf := []byte("a{color:red}")
	var s Sheet
	if errs := ParseInto(buf, &s); len(errs) != 0 {
		t.Fatal(errs)
	}
	copy(buf, "@x{y:z}     ")
	if errs := ParseInto(buf, &s); len(errs) != 0 {
		t.Fatal(errs)
	}
	if got := string(s.Emit(nil, EmitOptions{})); got != "@x{y:z}" {
		t.Fatalf("emit after refill = %q", got)
	}
}

func TestFrameworks(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("test-data", "frameworks"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".css" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src := readFramework(t, e.Name())
			s, errs := Parse(src)
			if len(errs) != 0 {
				t.Fatalf("parse errors: %#v", errs[:min(len(errs), 5)])
			}
			verify(t, s)
		})
	}
}

func readFramework(t testing.TB, name string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("test-data", "frameworks", name))
	if err != nil {
		t.Fatal(err)
	}
	return src
}
