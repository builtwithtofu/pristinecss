package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss"
)

var cliFixture = []byte("a{\n  -webkit-transform:rotate(0);transform:rotate(0);\n  inset:1px 2px;\n  color:oklch(.9 .35 0);\n  background:oklch(var(--c))\n}\nb{inset:var(--x)}")

func TestCommandsMatchLibrary(t *testing.T) {
	fixturePath := filepath.Join(t.TempDir(), "app.css")
	if err := os.WriteFile(fixturePath, cliFixture, 0600); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{fixturePath, "../../test-data/corpus/bootstrap-3.4.1.min.css"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"raise", "lower"} {
			t.Run(filepath.Base(file)+"/"+command, func(t *testing.T) {
				options := pristinecss.Options{Raise: pristinecss.Baseline(2024)}
				caps := pristinecss.Modern
				args := []string{command, file, "--to", "baseline-2024"}
				if command == "lower" {
					target := pristinecss.Chrome.At("49")
					options = pristinecss.Options{Lower: []pristinecss.Target{target}}
					caps = target.Caps
					args = []string{command, file, "--target", "chrome 49"}
				}
				bin, _ := pristinecss.ToBinary(src, options)
				var out, diagnostics bytes.Buffer
				if code := run(args, strings.NewReader(""), &out, &diagnostics); code != 0 {
					t.Fatalf("exit %d: %s", code, &diagnostics)
				}
				if want := append(pristinecss.Open(string(bin)).CSS(nil, caps), '\n'); !bytes.Equal(out.Bytes(), want) {
					t.Fatal("stdout differs from library CSS")
				}
				if file == fixturePath {
					want := file + ":2:3: dropped -webkit-transform: a later transform overrides it, and the raise floor supports it\n"
					if command == "lower" {
						want = file + ":5:14: chrome 49: cannot lower colour: needs literal numeric components (no none/calc/var or relative syntax)\n" +
							file + ":7:3: chrome 49: cannot lower inset: invalid or unsupported value\n" +
							file + ":4:9: chrome 49: colour mapped into the srgb gamut\n"
					}
					if diagnostics.String() != want {
						t.Fatalf("stderr = %q, want %q", diagnostics.String(), want)
					}
				}
			})
		}
	}
}

func TestStdinAndVersionForms(t *testing.T) {
	src := []byte(`a{inset:1px;color:oklch(.9 .35 0)}`)
	for _, version := range []struct {
		id pristinecss.BrowserID
		v  string
	}{
		{pristinecss.Chrome, "49"},
		{pristinecss.IOSSafari, "15.2-15.3"},
		{pristinecss.Safari, "TP"},
		{pristinecss.Firefox, "115.0esr"},
	} {
		t.Run(string(version.id)+"/"+version.v, func(t *testing.T) {
			target := version.id.At(version.v)
			bin, _ := pristinecss.ToBinary(src, pristinecss.Options{Lower: []pristinecss.Target{target}})
			var out, diagnostics bytes.Buffer
			if code := run([]string{"lower", "--target", string(version.id) + " " + version.v, "-"}, bytes.NewReader(src), &out, &diagnostics); code != 0 {
				t.Fatalf("exit %d: %s", code, &diagnostics)
			}
			if !bytes.Equal(out.Bytes(), append(pristinecss.Open(string(bin)).CSS(nil, target.Caps), '\n')) {
				t.Fatal("stdout differs from library CSS")
			}
		})
	}
}

func TestParseErrorsRecover(t *testing.T) {
	src := []byte("a{color:red}\n  }")
	for _, args := range [][]string{{"raise", "-", "--to", "baseline-2024"}, {"lower", "-", "--target", "chrome 49"}} {
		var out, diagnostics bytes.Buffer
		if code := run(args, bytes.NewReader(src), &out, &diagnostics); code != 0 {
			t.Fatalf("exit %d: %s", code, &diagnostics)
		}
		if !strings.Contains(diagnostics.String(), "parse error:") || !strings.Contains(out.String(), "color:red") {
			t.Fatalf("expected recovered CSS and parse diagnostic; stdout=%q stderr=%q", &out, &diagnostics)
		}
	}
}

func TestUsageAndReadErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.css")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "usage:"},
		{[]string{"unknown"}, "usage:"},
		{[]string{"raise", "-"}, "--to is required"},
		{[]string{"raise", "-", "--to", "2024"}, "invalid --to"},
		{[]string{"raise", "-", "--to", "baseline-20x4"}, "invalid --to"},
		{[]string{"raise", "-", "--to", "baseline-2010"}, "unknown browser, or no recorded support for that version"},
		{[]string{"lower", "-"}, "invalid --target"},
		{[]string{"lower", "-", "--target", "chrome"}, "invalid --target"},
		{[]string{"lower", "-", "--target", "unknown 49"}, "unknown browser, or no recorded support for that version; valid IDs: chrome and_chr edge firefox and_ff safari ios_saf ie samsung opera op_mob"},
		{[]string{"lower", "-", "--target", "chrome banana"}, "unknown browser, or no recorded support for that version"},
		{[]string{"lower", "-", "--bogus"}, "flag provided but not defined"},
		{[]string{"raise", "--to", "baseline-2024"}, "usage:"},
		{[]string{"raise", "a.css", "--to", "baseline-2024", "b.css"}, "usage:"},
		{[]string{"raise", missing, "--to", "baseline-2024"}, missing},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			wantCode := 2
			if tc.want == missing {
				wantCode = 1
			}
			if code := run(tc.args, strings.NewReader(""), &out, &diagnostics); code != wantCode {
				t.Fatalf("exit %d, want %d", code, wantCode)
			}
			if out.Len() != 0 || !strings.Contains(diagnostics.String(), tc.want) {
				t.Fatalf("stdout=%q stderr=%q; want diagnostic %q", &out, &diagnostics, tc.want)
			}
		})
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{
		{"-h"}, {"--help"}, {"raise", "-h"}, {"raise", "--help"},
		{"lower", "-h"}, {"lower", "--help"}, {"raise", "-", "-h"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			if code := run(args, strings.NewReader(""), &out, &diagnostics); code != 0 {
				t.Fatalf("exit %d: %s", code, &diagnostics)
			}
			if out.Len() != 0 || !strings.Contains(diagnostics.String(), "usage:") {
				t.Fatalf("stdout=%q stderr=%q", &out, &diagnostics)
			}
		})
	}
}

func TestPosition(t *testing.T) {
	src := []byte("é\r\n  a{}\n")
	starts := lineStarts(src)
	for _, tc := range []struct {
		offset    uint32
		line, col int
	}{
		{0, 1, 1}, {2, 1, 3}, {3, 1, 4}, {4, 2, 1},
		{6, 2, 3}, {9, 2, 6}, {10, 3, 1},
	} {
		line, col := position(starts, tc.offset)
		if line != tc.line || col != tc.col {
			t.Fatalf("position(%d) = %d:%d, want %d:%d", tc.offset, line, col, tc.line, tc.col)
		}
	}
}

func TestPositionOnFixture(t *testing.T) {
	starts := lineStarts(cliFixture)
	last := uint32(len(cliFixture) - 1)
	for _, tc := range []struct {
		name      string
		offset    uint32
		line, col int
	}{
		{"first byte", 0, 1, 1},
		{"after newline", 3, 2, 1},
		{"-webkit-transform", 5, 2, 3},
		{"last byte", last, 7, 17},
	} {
		t.Run(tc.name, func(t *testing.T) {
			line, col := position(starts, tc.offset)
			if line != tc.line || col != tc.col {
				t.Fatalf("position(%d) = %d:%d, want %d:%d", tc.offset, line, col, tc.line, tc.col)
			}
		})
	}
}

func BenchmarkPositionLargeSheet(b *testing.B) {
	src := bytes.Repeat([]byte("a{inset:1px}\n"), (4<<20)/len("a{inset:1px}\n"))
	var offsets []uint32
	for i := 0; i < len(src); i += 50 {
		offsets = append(offsets, uint32(i))
	}
	b.ResetTimer()
	for range b.N {
		starts := lineStarts(src)
		for _, offset := range offsets {
			position(starts, offset)
		}
	}
}
