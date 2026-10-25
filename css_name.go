package pristinecss

// cssNameEqual compares a CSS identifier with a lowercase ASCII keyword,
// decoding escapes and folding only ASCII letters, without allocating.
func cssNameEqual(raw, name string) bool {
	p := 0
	for _, want := range name {
		if p == len(raw) {
			return false
		}
		c := rune(raw[p])
		p++
		if c == '\\' {
			start := p
			c = 0
			for p < len(raw) && p-start < 6 {
				digit := raw[p]
				var n rune
				switch {
				case digit >= '0' && digit <= '9':
					n = rune(digit - '0')
				case digit >= 'a' && digit <= 'f':
					n = rune(digit-'a') + 10
				case digit >= 'A' && digit <= 'F':
					n = rune(digit-'A') + 10
				default:
					goto escaped
				}
				c = c*16 + n
				p++
			}
		escaped:
			if p == start {
				if p == len(raw) {
					return false
				}
				c = rune(raw[p])
				p++
			} else if p < len(raw) && isCSSSpace(raw[p]) {
				if raw[p] == '\r' && p+1 < len(raw) && raw[p+1] == '\n' {
					p++
				}
				p++
			}
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != want {
			return false
		}
	}
	return p == len(raw)
}
