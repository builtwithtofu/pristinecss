package pristinecss

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

func TestMarshalLoad(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}@media screen{b{display:block}}"))
	want := string(s.Emit(nil, EmitOptions{}))
	loaded, err := Load(s.Marshal(true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(loaded.Emit(nil, EmitOptions{})); got != want {
		t.Fatalf("embedded load emit = %q", got)
	}
	blob := s.Marshal(false)
	loaded, err = Load(blob, s.src)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(loaded.Emit(nil, EmitOptions{})); got != want {
		t.Fatalf("external load emit = %q", got)
	}
	if _, err := Load(blob, []byte("bad")); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("mismatched src error = %v", err)
	}
	var value NodeID
	for c := range s.All(KindValBasic) {
		value = c.ID()
		break
	}
	s.ReplaceRaw(value, []byte("blue"))
	loaded, err = Load(s.Marshal(true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(loaded.Emit(nil, EmitOptions{})); got != "a{color:blue}@media screen{b{display:block}}" {
		t.Fatalf("mutated load emit = %q", got)
	}
}

func TestLoadVersionError(t *testing.T) {
	s, _ := Parse([]byte("a{}"))
	blob := s.Marshal(true)
	binary.LittleEndian.PutUint16(blob[4:], frameVersion+1)
	_, err := Load(blob, nil)
	if err == nil || !strings.Contains(err.Error(), "regenerate the blob") {
		t.Fatalf("version error = %v", err)
	}
}

func TestCompileLoadFilterIR(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}b{color:blue}"))
	var masks = []IRMasks{{Required: [4]uint64{1}}, {}}
	blob, err := s.CompileIR(masks)
	if err != nil {
		t.Fatal(err)
	}
	ir, err := LoadIR(blob)
	if err != nil {
		t.Fatal(err)
	}
	full := FilterIR(ir, [4]uint64{1}, nil)
	if !bytes.Equal(full, s.Emit(nil, EmitOptions{})) {
		t.Fatalf("full IR = %q", full)
	}
	zero := string(FilterIR(ir, [4]uint64{}, nil))
	if zero != "b{color:blue}" {
		t.Fatalf("zero mask IR = %q", zero)
	}
}

func TestLoadRejectsMalformedASTNodes(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}"))
	base := s.Marshal(true)
	tests := []struct {
		name   string
		mutate func([]byte)
	}{
		{"kind", func(blob []byte) { blob[frameHeaderLen+16] = byte(kindCount) }},
		{"zero subtree", func(blob []byte) { binary.LittleEndian.PutUint32(blob[frameHeaderLen+16+12:], 0) }},
		{"escaping span", func(blob []byte) { binary.LittleEndian.PutUint32(blob[frameHeaderLen+16+8:], uint32(len(s.src)+1)) }},
		{"escaping subtree", func(blob []byte) { binary.LittleEndian.PutUint32(blob[frameHeaderLen+16+12:], uint32(len(s.nodes))) }},
		{"unknown flags", func(blob []byte) { blob[frameHeaderLen+16+1] = 0x80 }},
		{"opaque children", func(blob []byte) {
			blob[frameHeaderLen+16] = byte(KindValRaw)
			binary.LittleEndian.PutUint32(blob[frameHeaderLen+16+12:], 2)
		}},
		{"raw statement children", func(blob []byte) {
			blob[frameHeaderLen+16] = byte(KindRawStatement)
			binary.LittleEndian.PutUint32(blob[frameHeaderLen+16+12:], 2)
		}},
		{"invalid aux", func(blob []byte) {
			blob[frameHeaderLen+16] = byte(KindSelPseudo)
			binary.LittleEndian.PutUint16(blob[frameHeaderLen+16+2:], 0xffff)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := append([]byte(nil), base...)
			tt.mutate(blob)
			if _, err := Load(blob, nil); err == nil {
				t.Fatal("Load accepted malformed AST nodes")
			}
		})
	}
}

func TestLoadIRRejectsInvalidRuleSpans(t *testing.T) {
	s, _ := Parse([]byte("a{color:red}"))
	blob, err := s.CompileIR(nil)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		lo, hi uint32
	}{{"inverted", 4, 2}, {"out of range", 0, uint32(len(s.src) + 100)}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := append([]byte(nil), blob...)
			binary.LittleEndian.PutUint32(bad[frameHeaderLen+64:], tt.lo)
			binary.LittleEndian.PutUint32(bad[frameHeaderLen+68:], tt.hi)
			if _, err := LoadIR(bad); err == nil {
				t.Fatal("LoadIR accepted an invalid rule span")
			}
		})
	}
}

func TestRawStatementFrameRoundTrip(t *testing.T) {
	const src = "*zoom:1;.x{*display:inline;--y:foo;zoom:1}"
	s, errs := Parse([]byte(src))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	for _, includeSrc := range []bool{false, true} {
		blob := s.Marshal(includeSrc)
		if got := binary.LittleEndian.Uint16(blob[4:]); got != 1 {
			t.Fatalf("frame version = %d, want 1", got)
		}
		loaded, err := Load(blob, []byte(src))
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for c := range loaded.All(KindRawStatement) {
			if got := c.Kind().String(); got != "raw-statement" {
				t.Fatalf("kind string = %q", got)
			}
			count++
		}
		if count != 2 {
			t.Fatalf("raw statements = %d, want 2", count)
		}
		if got := string(loaded.Emit(nil, EmitOptions{})); got != src {
			t.Fatalf("loaded emit = %q, want %q", got, src)
		}
	}
}

func TestLoadVersionOneFrame(t *testing.T) {
	const frame = "5043535301000000080000000000000012000000250d5cb86ae70772" +
		"0000000000000000120000000800000001000000000000001200000007000000" +
		"0600000000000000020000000100000002000000020000001200000005000000" +
		"03100300030000000b0000000200000011000000070000000a00000001000000" +
		"030004000b00000011000000020000000e000000100000001100000001000000" +
		"2e787b2d2d793a666f6f3b7a6f6f6d3a317d"
	blob, err := hex.DecodeString(frame)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(blob, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(loaded.Emit(nil, EmitOptions{})); got != ".x{--y:foo;zoom:1}" {
		t.Fatalf("version 1 emit = %q", got)
	}
	for c := range loaded.All(KindValRaw) {
		if got := string(loaded.Extract(c.ID()).Emit(nil, EmitOptions{})); got != "foo" {
			t.Fatalf("version 1 extracted value = %q", got)
		}
	}
}
