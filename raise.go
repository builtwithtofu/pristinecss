package pristinecss

import (
	"cmp"
	"slices"
	"strings"
)

// A later unprefixed declaration must override the prefixed declaration's
// importance. Property-specific browser value validation remains the caller's concern.
func raiseSheet(s *Sheet, o Options) []Change {
	fidelity := o.MinFidelity
	if fidelity == 0 {
		fidelity = Close
	}
	properties := make(map[string]Cap)
	for cap, r := range rewrites {
		if r.raise != nil && o.Raise.Has(cap) && r.fidelity <= fidelity {
			for _, name := range r.properties {
				properties[name] = cap
			}
		}
	}
	var changes []Change
	type replacement struct{ last, important int }
	names := make(map[string]replacement)
	src := string(s.src)
	keyframeEnd := 0
	for id, block := range s.nodes {
		if block.Kind == KindKeyframeBlock {
			keyframeEnd = id + int(block.Sub)
		}
		if block.Kind != KindBlock {
			continue
		}
		clear(names)
		end := id + int(block.Sub)
		for i := id + 1; i < end; i += int(s.nodes[i].Sub) {
			n := s.nodes[i]
			if n.Kind == KindDeclaration {
				if id < keyframeEnd && n.Flags&FlagImportant != 0 {
					continue
				}
				name := raiseName(src[n.Lo:n.Lo+uint32(n.Aux)], properties)
				if _, ok := properties[name]; !ok {
					continue
				}
				valid, hasValue := true, false
				for _, child := range s.nodes[i+1 : i+int(n.Sub)] {
					hasValue = hasValue || child.Kind != KindComment
				}
				for _, err := range s.errs {
					if err.Offset >= n.Lo && err.Offset < n.Hi {
						valid = false
						break
					}
				}
				if valid && hasValue {
					r := names[name]
					r.last = i
					if n.Flags&FlagImportant != 0 {
						r.important = i
					}
					names[name] = r
				}
			}
		}
		for i := id + 1; i < end; i += int(s.nodes[i].Sub) {
			n := s.nodes[i]
			if n.Kind != KindDeclaration {
				continue
			}
			name := raiseName(src[n.Lo:n.Lo+uint32(n.Aux)], properties)
			for _, prefix := range vendorPrefixes {
				if rest, ok := strings.CutPrefix(name, prefix); ok {
					r := names[rest]
					if r.last <= i || (n.Flags&FlagImportant != 0 && r.important <= i) {
						continue
					}
					cap := properties[rest]
					change := rewrites[cap].raise(s, NodeID(i), cap)
					change.What = "dropped " + name + ": a later " + rest + " overrides it, and the raise floor supports it"
					changes = append(changes, change)
					break
				}
			}
		}
	}
	slices.SortFunc(changes, func(a, b Change) int { return cmp.Compare(a.Lo, b.Lo) })
	return changes
}

var vendorPrefixes = [...]string{"-webkit-", "-moz-", "-ms-", "-o-"}

type prefixSupport struct {
	browser  BrowserID
	from, to uint64 // to is exclusive; zero is unbounded
}

func targetReadPrefix(name string, cap Cap, target Target) bool {
	var version uint64
	if target.Browser != "" {
		var ok bool
		version, ok = browserVersion(target.Browser, target.Version)
		if !ok {
			return false
		}
	}
	for property, spans := range prefixedProperties {
		if !cssNameEqual(name, property) {
			continue
		}
		for _, span := range spans {
			if target.Browser != "" {
				if span.browser == target.Browser && version >= span.from && (span.to == 0 || version < span.to) {
					return true
				}
				continue
			}
			// A capability-only target may be any browser. Look for a part of
			// the prefix's interval not covered by unprefixed feature support.
			end := span.to
			if end == 0 {
				end = ^uint64(0)
			}
			pos := span.from
			for _, native := range capBrowsers[span.browser] {
				if native.cap != cap {
					continue
				}
				if native.from > pos {
					break
				}
				if native.to == 0 {
					pos = end
					break
				}
				pos = max(pos, native.to)
				if pos >= end {
					break
				}
			}
			if pos < end {
				return true
			}
		}
		return false
	}
	return false
}

func raiseName(raw string, properties map[string]Cap) string {
	if !strings.Contains(raw, `\`) {
		return strings.Map(func(c rune) rune {
			if c >= 'A' && c <= 'Z' {
				return c + ('a' - 'A')
			}
			return c
		}, raw)
	}
	for name := range properties {
		if cssNameEqual(raw, name) {
			return name
		}
		for _, prefix := range vendorPrefixes {
			if cssNameEqual(raw, prefix+name) {
				return prefix + name
			}
		}
	}
	return raw
}

func dropPrefix(s *Sheet, id NodeID, cap Cap) Change {
	n := s.nodes[id]
	s.Delete(id)
	return Change{
		Lo: n.Lo, Hi: n.Hi, Cap: cap,
	}
}
