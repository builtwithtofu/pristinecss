package pristinecss

import "testing"

func errorRegressionInputs() []errorInput {
	var inputs []errorInput
	for _, tt := range []struct{ name, css string }{
		{"open-string", `a{x:"`},
		{"open-url", `a{x:url(`},
		{"malformed-at-name", `@ x;`},
		{"dangling-at-block", `@a{x`},
		{"missing-colon", `a{x y;z:0}`},
		{"function-trailing-comment", `a{x:f() /*x*/;}`},
		{"function-trailing-space-control", `a{x:var(--x, "/") }`},
		{"star-property", `a{*x:y;z:0;w:0}`},
		{"hash-property", `a{#x:y;z:0;w:0}`},
		{"bad-string-newline", "a{x:\"a\n;y:0}b{}"},
		{"double-colon", `a{x::y}`},
		{"double-colon-eof", `a{x::y`},
		{"unclosed-components", `@a{[(0`},
		{"raw-open-selector-string", `input[type='`},
		{"invalid-at-rule-selector", `a{@ media screen{div{}}}`},
		{"quoted-url-newline-eof", "{url(\"\n0"},
		{"selector-class-comment", `.a/*x*/b{color:red}`},
		{"selector-id-comment", `#a/*x*/b{color:red}`},
		{"selector-pseudo-comment", `:hover/**/a{color:red}`},
		{"selector-number-comment", `a/**/1{color:red}`},
		{"open-bang-comment", `a{x:y/*!abc`},
		{"open-comment-in-block", `a{/*abc`},
		{"string-cr", "a{x:\"a\rb}c{color:red}"},
		{"string-ff", "a{x:\"a\fb}c{color:red}"},
		{"string-crlf", "a{x:\"a\r\nb}c{color:red}"},
		{"url-string-newline", "a{x:url(\"a\nb)}c{color:red}"},
		{"trailing-escape-value", "a{x:abc\\"},
		{"trailing-escape-rule", "{\\"},
		{"trailing-escape-url", "a{x:url(abc\\"},
		{"custom-open-brace", `a{--x:{abc`},
		{"custom-open-brace-number", `000000{--:0{`},
		{"spaced-colons", `a: ::`},
		{"comment-before-comma", `a /**/,b{}`},
		{"newline-before-statement-end", "A:(\"\n"},
		{"pretty-selector-comma", `a, b{}`},
		{"overlapping-comment-close", `0/*/0`},
		{"nested-bare-parenthesis", `A:u((`},
		{"selector-bare-parenthesis", `({`},
		{"nested-attribute-bracket", `[[]{`},
		{"selector-hex-escape", `\0{`},
		{"bad-url-quote", "url(0\"\n"},
		{"at-name-trailing-escape", "{@\\"},
		{"at-name-hex-escape", `@\0!`},
		{"mismatched-function-blocks", `A:A(([)`},
		{"mismatched-declaration-blocks", `A:(;{(}`},
		{"escaped-semicolon-before-comment", `\;;/*`},
		{"comment-after-at-delimiter", `@/**/A!{`},
		{"escaped-at-name-semicolon", `@\;;0`},
		{"at-name-escape-formfeed", "@\\0\f0!"},
		{"string-escaped-crlf-eof", "{\"\\\r\n"},
		{"bad-string-escaped-cr", "\"\\\r\r"},
		{"at-name-escaped-space", `@\  0!`},
		{"at-name-hex-escape-space", `@\0  0!`},
		{"bad-string-semicolons", "\";;\n"},
		{"hex-escape-cr-in-block", "{\\0\r"},
	} {
		inputs = append(inputs, errorInput{set: "regressions", name: tt.name, src: []byte(tt.css)})
	}
	return inputs
}

func TestBrokenCSSRegressions(t *testing.T) {
	for _, in := range errorRegressionInputs() {
		t.Run(in.name, func(t *testing.T) {
			s, errs := Parse(in.src)
			first := s.Emit(nil, EmitOptions{})
			r, reerrs := Parse(first)
			t.Logf("input=%q errors=%v; first=%q reparse_errors=%v; second=%q", in.src, errs, first, reerrs, r.Emit(nil, EmitOptions{}))
			if got := inspectErrorInput(in); len(got.details) > 0 {
				t.Error(got.details)
			}
		})
	}
}
