package pristinecss

import (
	"sync"
	"unsafe"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

// NodeID is a dense index into a Sheet's node array. Node 0 is the stylesheet root.
type NodeID uint32

// NoNode is an invalid node sentinel used by sidecars and navigation helpers.
const NoNode NodeID = ^NodeID(0)

// Node is one 16-byte pointer-free pre-order AST record.
type Node struct {
	Kind  Kind
	Flags Flags
	Aux   uint16
	Lo    uint32
	Hi    uint32
	Sub   uint32
}

var _ [16]byte = [unsafe.Sizeof(Node{})]byte{}

// Sheet owns a flat pre-order CSS tree. A Sheet with no mutator ever called is safe for
// unlimited concurrent readers. Mutators require exclusive access; even a one-flag write
// is a data race.
//
// The source buffer is borrowed: the caller must not mutate or free it while the Sheet
// (or any extracted child) lives. Owned extraction and source-inclusive serialization
// are the escape hatches.
type Sheet struct {
	src      []byte
	nodes    []Node
	scratch  []byte
	lines    []uint32
	lineOnce sync.Once
	errs     []ParseError
	toks     []lexer.Token
	tokSrc   []byte
	mutated  bool
}

// ParseError reports a recoverable CSS parse error at a byte offset.
type ParseError struct {
	Message string
	Offset  uint32
	src     []byte
}

// Line returns the 1-based source line for the error offset.
func (e ParseError) Line() int { line, _ := lineCol(e.src, nil, e.Offset); return line }

// Column returns the 1-based source column for the error offset.
func (e ParseError) Column() int { _, col := lineCol(e.src, nil, e.Offset); return col }

// Source returns the borrowed source buffer for this sheet.
func (s *Sheet) Source() []byte {
	if s == nil {
		return nil
	}
	return s.src
}

// Nodes returns the sheet's node array. Callers must treat it as read-only.
func (s *Sheet) Nodes() []Node {
	if s == nil {
		return nil
	}
	return s.nodes
}

// Errors returns the parse errors retained on this sheet.
func (s *Sheet) Errors() []ParseError {
	if s == nil {
		return nil
	}
	return s.errs
}
