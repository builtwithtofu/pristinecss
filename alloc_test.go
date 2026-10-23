package pristinecss

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllocBudget(t *testing.T) {
	for _, path := range corpusFiles(t) {
		t.Run(path, func(t *testing.T) {
			src := readCSS(t, path)
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
	for _, path := range corpusFiles(t) {
		t.Run(path, func(t *testing.T) {
			src := readCSS(t, path)
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
	return readCSS(t, filepath.Join("test-data", "frameworks", name))
}

func corpusFiles(t testing.TB) []string {
	t.Helper()
	var paths []string
	for _, dir := range []string{"frameworks", "corpus"} {
		matches, err := filepath.Glob(filepath.Join("test-data", dir, "*.css"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) == 0 {
			t.Fatalf("no CSS fixtures in %s", dir)
		}
		paths = append(paths, matches...)
	}
	return paths
}

func readCSS(t testing.TB, path string) []byte {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return src
}
