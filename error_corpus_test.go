package pristinecss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

type errorInput struct {
	set, name string
	src       []byte
	wantRule  string
}

// Error offsets may point at EOF, so the valid range is [0, len(src)].
type errorChecks struct {
	errors, panic, bounds, order, stable, survival bool
	details                                        []string
}

func inspectErrorInput(in errorInput) (out errorChecks) {
	defer func() {
		if v := recover(); v != nil {
			out.panic = true
			out.details = append(out.details, fmt.Sprintf("panic: %v", v))
		}
	}()
	checkOffsets := func(src []byte, errs []ParseError, phase string) {
		for i, err := range errs {
			if uint64(err.Offset) > uint64(len(src)) {
				out.bounds = true
				out.details = append(out.details, fmt.Sprintf("%s offset %d outside [0,%d]", phase, err.Offset, len(src)))
			}
			if i > 0 && err.Offset < errs[i-1].Offset {
				out.order = true
				out.details = append(out.details, fmt.Sprintf("%s offsets %d then %d", phase, errs[i-1].Offset, err.Offset))
			}
		}
	}
	s, errs := Parse(in.src)
	out.errors = len(errs) > 0
	checkOffsets(in.src, errs, "parse")
	if in.wantRule != "" && !hasStyleRuleSelector(s, in.wantRule) {
		out.survival = true
		out.details = append(out.details, "lost valid trailing rule "+in.wantRule)
	}
	for name, style := range map[string]EmitStyle{"minified": Minified, "pretty": Pretty} {
		o := EmitOptions{Style: style}
		first := s.Emit(nil, o)
		reparsed, reerrs := Parse(first)
		checkOffsets(first, reerrs, name+" reparse")
		if len(reerrs) > len(errs) {
			out.details = append(out.details, fmt.Sprintf("%s errors increased: %d -> %d; output=%q", name, len(errs), len(reerrs), first))
		}
		if in.wantRule != "" && !hasStyleRuleSelector(reparsed, in.wantRule) {
			out.survival = true
			out.details = append(out.details, name+" lost emitted trailing rule "+in.wantRule)
		}
		second := reparsed.Emit(nil, o)
		if !bytes.Equal(first, second) {
			out.stable = true
			i := 0
			for i < min(len(first), len(second)) && first[i] == second[i] {
				i++
			}
			out.details = append(out.details, fmt.Sprintf("%s unstable emit at byte %d: first=%q second=%q", name, i, first[max(0, i-30):min(len(first), i+80)], second[max(0, i-30):min(len(second), i+80)]))
		}
	}
	return out
}

func errorInputs(t testing.TB) []errorInput {
	t.Helper()
	var inputs []errorInput
	for _, set := range []string{"frameworks", "corpus"} {
		paths, err := filepath.Glob("test-data/" + set + "/*.css")
		if err != nil || len(paths) == 0 {
			t.Fatalf("load %s: paths=%d err=%v", set, len(paths), err)
		}
		for _, path := range paths {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			inputs = append(inputs, damageSheet(set, filepath.Base(path), src)...)
		}
	}
	for _, set := range []string{"css-parsing-tests", "wpt"} {
		paths, err := filepath.Glob("test-data/errors/" + set + "/*.json")
		if err != nil || len(paths) == 0 {
			t.Fatalf("load %s: paths=%d err=%v", set, len(paths), err)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var cases []string
			if err := json.Unmarshal(data, &cases); err != nil {
				t.Fatal(err)
			}
			for i, css := range cases {
				if strings.HasSuffix(path, "declaration_list.json") {
					css = "a{" + css + "}"
				} else if set == "wpt" && !strings.HasSuffix(path, "trailing-braces.json") {
					// Inline-style inputs end at EOF: do not append a closing brace.
					css = "a{" + css
				}
				inputs = append(inputs, errorInput{set: set, name: fmt.Sprintf("%s/%02d", filepath.Base(path), i), src: []byte(css)})
			}
		}
	}
	inputs = append(inputs, recoverySpotChecks()...)
	return append(inputs, errorRegressionInputs()...)
}

func damageSheet(set, name string, src []byte) []errorInput {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	rng := rand.New(rand.NewSource(int64(h.Sum64()) ^ 0x5eedc55))
	var blocks, quotes, comments, urls, boundaries, declarations []int
	depth := 0
	tokens := lexer.Lex(src)
	for i, tok := range tokens {
		switch tok.Type {
		case lexer.LBrace:
			blocks = append(blocks, int(tok.End))
			depth++
		case lexer.RBrace:
			depth--
			if depth == 0 {
				boundaries = append(boundaries, int(tok.End))
			}
		case lexer.String:
			quotes = append(quotes, int(tok.Start)+1)
		case lexer.Comment:
			comments = append(comments, int(tok.Start)+2)
		case lexer.URI:
			if i := bytes.IndexByte(tok.Literal(src), '('); i >= 0 {
				urls = append(urls, int(tok.Start)+i+1)
			}
		case lexer.Colon:
			if depth > 0 && i >= 2 && tokens[i-1].Type == lexer.Ident &&
				(tokens[i-2].Type == lexer.LBrace || tokens[i-2].Type == lexer.Semicolon) {
				declarations = append(declarations, int(tok.End))
			}
		}
	}
	var out []errorInput
	add := func(kind string, data []byte) {
		out = append(out, errorInput{set: "damaged-" + set, name: name + "/" + kind, src: data})
	}
	for _, cut := range []struct {
		name     string
		points   []int
		fallback string
	}{
		{"open-block", blocks, "\n.broken{"},
		{"open-string", quotes, "\n.broken{content:\""},
		{"open-comment", comments, "\n/*"},
		{"open-url", urls, "\n.broken{background:url("},
	} {
		if len(cut.points) > 0 {
			add(cut.name, bytes.Clone(src[:cut.points[rng.Intn(len(cut.points))]]))
		} else {
			// Sheets without this construct get an open synthetic tail.
			add(cut.name+"-tail", append(bytes.Clone(src), cut.fallback...))
		}
	}
	boundary := 0
	if len(boundaries) > 0 {
		boundary = boundaries[rng.Intn(len(boundaries))]
	}
	inject := func(at int, junk string) []byte {
		data := append(bytes.Clone(src[:at]), junk...)
		return append(data, src[at:]...)
	}
	add("stray-brace", inject(boundary, " } "))
	add("stray-semicolon", inject(boundary, " ; "))
	add("junk-between-rules", inject(boundary, " \x01 ???; "))
	if len(declarations) > 0 {
		add("junk-in-declaration", inject(declarations[rng.Intn(len(declarations))], " \x01 ] ) ??? "))
	} else {
		add("junk-in-declaration-tail", append(bytes.Clone(src), "\n.broken{color: \x01 ] ) ???;}"...))
	}
	return out
}

func recoverySpotChecks() []errorInput {
	// CSS Syntax: invalid declarations end at semicolons; bad strings end at
	// newlines; balanced unknown at-rules cannot consume following rules.
	return []errorInput{
		{set: "handwritten", name: "invalid-declaration", src: []byte("a{color red; width:1px}after{color:green}"), wantRule: "after"},
		{set: "handwritten", name: "bad-string-newline", src: []byte("a{content:\"bad\n;color:red}after{color:green}"), wantRule: "after"},
		{set: "handwritten", name: "unknown-at-rule", src: []byte("@unknown broken [x]; after{color:green}"), wantRule: "after"},
		{set: "handwritten", name: "balanced-invalid-value", src: []byte("a{color: red {}; width:1px}after{color:green}"), wantRule: "after"},
		{set: "handwritten", name: "stray-close", src: []byte("a{} } after{color:green}"), wantRule: "after"},
	}
}

func TestErrorCorpusDeterministic(t *testing.T) {
	first, second := errorInputs(t), errorInputs(t)
	if len(first) != len(second) {
		t.Fatalf("input count changed: %d then %d", len(first), len(second))
	}
	for i, in := range first {
		if in.set != second[i].set || in.name != second[i].name || !bytes.Equal(in.src, second[i].src) {
			t.Errorf("input %d (%s/%s) changed between generations", i, in.set, in.name)
		}
	}
}

func TestErrorCorpus(t *testing.T) {
	type counts struct{ inputs, errors, panics, bounds, order, stable, survival int }
	sets := make(map[string]*counts)
	var order []string
	for _, in := range errorInputs(t) {
		c := sets[in.set]
		if c == nil {
			c = new(counts)
			sets[in.set] = c
			order = append(order, in.set)
		}
		c.inputs++
		t.Run(in.set+"/"+in.name, func(t *testing.T) {
			got := inspectErrorInput(in)
			for _, pair := range []struct {
				hit bool
				n   *int
			}{
				{got.errors, &c.errors}, {got.panic, &c.panics}, {got.bounds, &c.bounds},
				{got.order, &c.order}, {got.stable, &c.stable}, {got.survival, &c.survival},
			} {
				if pair.hit {
					*pair.n++
				}
			}
			if len(got.details) > 0 {
				t.Error(strings.Join(got.details, "; "))
			}
		})
	}
	t.Log("set | inputs | with errors | panic | bounds | order | unstable | lost rule")
	for _, set := range order {
		c := sets[set]
		t.Logf("%s | %d | %d | %d | %d | %d | %d | %d", set, c.inputs, c.errors, c.panics, c.bounds, c.order, c.stable, c.survival)
	}
}

func FuzzParse(f *testing.F) {
	inputs := errorInputs(f)
	for _, in := range inputs {
		if len(in.src) <= 4096 {
			f.Add(in.src)
		}
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		in := errorInput{src: src}
		for _, spot := range recoverySpotChecks() {
			if bytes.Equal(src, spot.src) {
				in.wantRule = spot.wantRule
			}
		}
		if got := inspectErrorInput(in); len(got.details) > 0 {
			t.Fatal(strings.Join(got.details, "; "))
		}
	})
}
