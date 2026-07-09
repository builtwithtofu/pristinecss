package pristinecss

import (
	"bytes"
	"encoding/binary"
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
