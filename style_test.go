package pristinecss

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestStyleCorpus(t *testing.T) {
	for _, path := range corpusFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			checkStyle(t, readCSS(t, path))
		})
	}
}

func checkStyle(t *testing.T, src []byte) {
	t.Helper()
	s, errs := Parse(src)
	wantSource := string(src)
	wantMin := s.Emit(nil, EmitOptions{Style: Minified})
	wantPretty := s.Emit(nil, EmitOptions{Style: Pretty})
	blob, diags := ToBinary(src)
	if !reflect.DeepEqual(diags, errs) {
		t.Fatalf("diagnostics = %v, want %v", diags, errs)
	}
	opened := Open(string(blob))
	loaded, err := Load(blob)
	if err != nil {
		t.Fatal(err)
	}
	clear(src)
	clear(blob)
	for _, style := range []*Style{opened, loaded} {
		if style.Source() != wantSource {
			t.Fatal("Source differs from input (or borrowed mutable bytes)")
		}
		for _, tt := range []struct {
			format EmitStyle
			want   []byte
		}{{Minified, wantMin}, {Pretty, wantPretty}} {
			dst := make([]byte, 7, 7+len(tt.want))
			copy(dst, "prefix:")
			got := style.CSS(dst, Modern, tt.format)
			if !bytes.Equal(got[:7], []byte("prefix:")) || !bytes.Equal(got[7:], tt.want) {
				t.Fatalf("CSS differs from Sheet.Emit: got %d bytes, want %d", len(got)-7, len(tt.want))
			}
			if allocs := testing.AllocsPerRun(10, func() { dst = style.CSS(dst[:7], Modern, tt.format) }); allocs != 0 {
				t.Fatalf("CSS allocations = %v, want 0", allocs)
			}
		}
		if !bytes.Equal(style.CSS(nil, Modern), wantMin) {
			t.Fatal("default CSS is not minified")
		}
	}
}

func TestStyleEdges(t *testing.T) {
	for _, src := range []string{
		"", " \r\n\t", "/* comment */", "/*! license */a {color:red}",
		"*zoom:1;.x{*display:inline;--y:foo;zoom:1}",
		"@media screen{a:hover, b{color:red!important;--x: {a:b};}}",
		"@keyframes a{from{opacity:0}to{opacity:1}}",
		"a{c\\:olor:red;color:;--x: \u00a0;}",
		"a{color: red;", "a{color:red; broken; x:func(}",
		"@unknown foo; @unknown bar{x:y}", "a{content:'x'}\x00\xff",
		":is(/*x*/b){}", ":is(/**/ .a){}", "a{--x:   ;}",
	} {
		t.Run(fmt.Sprintf("%q", src), func(t *testing.T) { checkStyle(t, []byte(src)) })
	}
}

func TestStyleDeclarationIndex(t *testing.T) {
	check := func(t *testing.T, src []byte) {
		t.Helper()
		s, _ := Parse(src)
		blob, _ := ToBinary(src)
		if _, err := Load(blob); err != nil {
			t.Fatal(err)
		}
		indexEnd := styleHeaderLen + int(binary.LittleEndian.Uint32(blob[12:]))
		index := blob[styleHeaderLen:indexEnd]
		source := blob[indexEnd : indexEnd+len(src)]
		type declaration struct{ name, value []byte }
		var declarations []declaration
		var lo uint64
		for event := uint32(0); event < binary.LittleEndian.Uint32(blob[8:]); event++ {
			if len(index) < 2 {
				t.Fatal("truncated event")
			}
			kind := Kind(index[0])
			index = index[2:]
			var fields [5]uint64
			for j := range fields {
				value, n := binary.Uvarint(index)
				if n <= 0 {
					t.Fatal("invalid index varint")
				}
				fields[j], index = value, index[n:]
			}
			lo += fields[1]
			nameEnd := lo + fields[2]
			valueLo := nameEnd + fields[3]
			if kind == KindDeclaration {
				declarations = append(declarations, declaration{source[lo:nameEnd], source[valueLo : valueLo+fields[4]]})
			}
		}
		if len(index) != 0 {
			t.Fatal("trailing index bytes")
		}
		j := 0
		for c := range s.All(KindDeclaration) {
			if j >= len(declarations) {
				t.Fatal("missing declaration in index")
			}
			got := declarations[j]
			if want := c.PropertyName(); !bytes.Equal(got.name, want) {
				t.Errorf("declaration %d name = %q, want %q", j, got.name, want)
			}
			var want []byte
			n := s.nodes[c.ID()]
			if n.Sub > 1 {
				children := s.nodes[int(c.ID())+1 : int(c.ID())+int(n.Sub)]
				end := children[0].Hi
				for _, child := range children {
					end = max(end, child.Hi)
				}
				want = src[children[0].Lo:end]
			}
			if !bytes.Equal(got.value, want) {
				t.Errorf("declaration %d (%s) value = %q, want %q", j, got.name, got.value, want)
			}
			j++
		}
		if j != len(declarations) {
			t.Fatalf("index has %d declarations, parser has %d", len(declarations), j)
		}
	}
	for _, path := range corpusFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) { check(t, readCSS(t, path)) })
	}
	for _, src := range []string{
		"a{color:red!important}",
		"a{color:red ! important}",
		"a{color:;}",
		"a{--custom: {color:red};}",
		"a{color:/* before */red/* middle */ blue/* after */;}",
		"a{width:calc(1px + var(--offset));}",
	} {
		t.Run(src, func(t *testing.T) { check(t, []byte(src)) })
	}
}

func TestStyleCorruption(t *testing.T) {
	blob, _ := ToBinary([]byte("/*! hi */ @media screen{a{color:red!important;--x:10px}}"))
	for i := range blob {
		for bit := range 8 {
			blob[i] ^= 1 << bit
			if _, err := Load(blob); err == nil {
				t.Fatalf("accepted flipped bit %d at byte %d", bit, i)
			}
			blob[i] ^= 1 << bit
		}
	}
	for n := range len(blob) {
		if _, err := Load(blob[:n]); err == nil {
			t.Fatalf("accepted truncation at %d", n)
		}
	}
	if _, err := Load(append(blob, 0)); err == nil {
		t.Fatal("accepted appended bytes")
	}
}

// These fixtures have valid checksums: validation must check structure as well.
func TestLoadStructure(t *testing.T) {
	validEvent := []byte{byte(KindStyleRule), 0, 1, 0, 1, 0, 0}
	for _, tt := range []struct {
		name  string
		count uint32
		index []byte
	}{
		{"kind", 1, []byte{255, 0, 1, 0, 0, 0, 0}},
		{"flags", 1, []byte{byte(KindStyleRule), 128, 1, 0, 0, 0, 0}},
		{"empty subtree", 1, []byte{byte(KindStyleRule), 0, 0, 0, 0, 0, 0}},
		{"escaping subtree", 1, []byte{byte(KindStyleRule), 0, 2, 0, 0, 0, 0}},
		{"escaping parent", 3, []byte{byte(KindStyleRule), 0, 2, 0, 0, 0, 0, byte(KindStyleRule), 0, 2, 0, 0, 0, 0, byte(KindStyleRule), 0, 1, 0, 0, 0, 0}},
		{"declaration children", 2, []byte{byte(KindDeclaration), 0, 2, 0, 0, 0, 0, byte(KindDeclaration), 0, 1, 0, 0, 0, 0}},
		{"span", 1, []byte{byte(KindStyleRule), 0, 1, 0, 2, 0, 0}},
		{"delta overflow", 1, append([]byte{byte(KindStyleRule), 0, 1}, binary.AppendUvarint(nil, ^uint64(0))...)},
		{"index truncated", 1, validEvent[:6]},
		{"index varint", 1, []byte{byte(KindStyleRule), 0, 128}},
		{"index trailing", 0, validEvent},
		{"count", ^uint32(0), nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			blob := make([]byte, styleHeaderLen)
			copy(blob, "PCS1")
			for i, n := range []uint32{1, tt.count, uint32(len(tt.index)), 0, 0} {
				binary.LittleEndian.PutUint32(blob[4+4*i:], n)
			}
			blob = append(blob, tt.index...)
			blob = append(blob, 'a')
			blob = binary.LittleEndian.AppendUint32(blob, checksum(blob))
			if _, err := Load(blob); err == nil {
				t.Fatal("accepted malformed structure with valid checksum")
			}
		})
	}
	base, _ := ToBinary([]byte("a{}"))
	for _, off := range []int{4, 8, 12, 16, 20} {
		b := bytes.Clone(base)
		binary.LittleEndian.PutUint32(b[off:], ^uint32(0))
		binary.LittleEndian.PutUint32(b[len(b)-4:], checksum(b[:len(b)-4]))
		if _, err := Load(b); err == nil {
			t.Fatalf("accepted invalid header at %d", off)
		}
	}
}

func TestStyleConcurrent(t *testing.T) {
	blob, _ := ToBinary([]byte("@media screen{a{color:red}}"))
	loaded, err := Load(blob)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Style{Open(string(blob)), loaded} {
		wantMin, wantPretty := s.CSS(nil, Modern), s.CSS(nil, Modern, Pretty)
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					if !bytes.Equal(s.CSS(nil, Modern), wantMin) || !bytes.Equal(s.CSS(nil, Modern, Pretty), wantPretty) || s.Source() != "@media screen{a{color:red}}" {
						t.Error("concurrent output differs")
					}
				}
			}()
		}
		wg.Wait()
	}
}

func FuzzLoad(f *testing.F) {
	for _, path := range []string{"test-data/frameworks/milligram.css", "test-data/corpus/pure-0.6.0.min.css"} {
		blob, _ := ToBinary(readCSS(f, path))
		f.Add(blob)
	}
	for _, src := range []string{"", "a{color:red}", "@media screen{a{--x:1}}"} {
		blob, _ := ToBinary([]byte(src))
		f.Add(blob)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		generated, _ := ToBinary(b)
		if _, err := Load(generated); err != nil {
			t.Fatalf("Load rejected ToBinary output for %q: %v", b, err)
		}
		check := func(b []byte) {
			if s, err := Load(b); err == nil {
				_ = s.Source()
				_ = s.CSS(nil, Modern)
				_ = s.CSS(nil, Modern, Pretty)
			}
		}
		check(b)
		// Also reach structural validation rather than stopping at the checksum.
		if len(b) >= styleHeaderLen+4 {
			b = bytes.Clone(b)
			binary.LittleEndian.PutUint32(b[len(b)-4:], checksum(b[:len(b)-4]))
			check(b)
		}
	})
}

func TestStyleStringConstant(t *testing.T) {
	const blob = "PCS1\x10\x00\x00\x00\x02\x00\x00\x00\x0e\x00\x00\x00\f\x00\x00\x00\x13\x00\x00\x00\x01\x00\x02\x00\x01\x00\x00\x03\x00\x01\x04\x05\x02\x03a { color: red }a{color:red}a {\n  color: red\n}\n9\x17\v\xfe"
	loaded, err := Load([]byte(blob))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Style{Open(blob), loaded} {
		if s.Source() != "a { color: red }" || string(s.CSS(nil, Modern)) != "a{color:red}" || string(s.CSS(nil, Modern, Pretty)) != "a {\n  color: red\n}\n" {
			t.Fatal("constant output differs")
		}
	}
}
