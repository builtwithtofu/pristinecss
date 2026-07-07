package parser

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func BenchmarkEmitSynthetic(b *testing.B) {
	css := sampleCSS(500)
	source := []byte(css)
	tokens := lexer.Lex(source)
	stylesheet, errs := Parse(source, tokens)
	if len(errs) > 0 {
		b.Fatalf("could not parse synthetic benchmark css: %v", errs)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(css)))
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		emitNode(&buf, stylesheet)
	}
}

func BenchmarkEmitAndGzipSynthetic(b *testing.B) {
	css := sampleCSS(500)
	source := []byte(css)
	tokens := lexer.Lex(source)
	stylesheet, errs := Parse(source, tokens)
	if len(errs) > 0 {
		b.Fatalf("could not parse synthetic benchmark css: %v", errs)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(css)))
	var buf bytes.Buffer
	var compressed bytes.Buffer
	zw, err := gzip.NewWriterLevel(&compressed, gzip.BestSpeed)
	if err != nil {
		b.Fatalf("could not create gzip writer: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		emitNode(&buf, stylesheet)
		compressed.Reset()
		_, _ = zw.Write(buf.Bytes())
		_ = zw.Close()
		zw.Reset(&compressed)
	}
}

func sampleCSS(repeat int) string {
	const snippet = `/* demo */
.button, #hero > a:hover {
  color: #ff0000;
  margin: 1rem 2rem;
  background-image: url("image.png");
}

@media screen and (min-width: 600px) {
  .button { padding: 1rem; }
}

@keyframes fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

@container card (min-width: 40ch) {
  .card { padding: 1rem; }
}
`

	var builder strings.Builder
	builder.Grow(len(snippet) * repeat)
	for i := 0; i < repeat; i++ {
		builder.WriteString(snippet)
	}

	return builder.String()
}

func emitNode(buf *bytes.Buffer, node Node) {
	switch n := node.(type) {
	case *Stylesheet:
		for _, rule := range n.Rules {
			emitNode(buf, rule)
		}
	case *Selector:
		emitSelector(buf, n)
	case *Declaration:
		emitDeclaration(buf, n)
	case *Comment:
		buf.WriteString("/*")
		buf.Write(n.Text)
		buf.WriteString("*/")
	case *MediaAtRule:
		buf.WriteString("@media ")
		emitMediaQuery(buf, n.Query)
		buf.WriteByte('{')
		for _, rule := range n.Rules {
			emitNode(buf, rule)
		}
		buf.WriteByte('}')
	case *KeyframesAtRule:
		buf.WriteString("@keyframes ")
		buf.Write(n.Name)
		buf.WriteByte('{')
		for i := range n.Stops {
			emitNode(buf, &n.Stops[i])
		}
		buf.WriteByte('}')
	case *KeyframeStop:
		emitKeyframeStop(buf, n)
	case *ImportAtRule:
		buf.WriteString("@import ")
		emitValue(buf, n.URL)
		buf.WriteByte(';')
	case *ContainerAtRule:
		buf.WriteString("@container ")
		if n.Name != nil {
			buf.Write(n.Name)
			buf.WriteByte(' ')
		}
		emitContainerQuery(buf, n.Query)
		buf.WriteByte('{')
		for _, decl := range n.Declarations {
			emitNode(buf, decl)
		}
		buf.WriteByte('}')
	case *CounterStyleAtRule:
		buf.WriteString("@counter-style ")
		buf.Write(n.Name)
		buf.WriteByte('{')
		for i := range n.Declarations {
			emitNode(buf, &n.Declarations[i])
		}
		buf.WriteByte('}')
	case *ColorProfileAtRule:
		buf.WriteString("@color-profile ")
		buf.Write(n.Name)
		buf.WriteByte('{')
		for i := range n.Declarations {
			emitNode(buf, &n.Declarations[i])
		}
		buf.WriteByte('}')
	case *FontFeatureValuesAtRule:
		buf.WriteString("@font-feature-values ")
		for i, family := range n.FontFamilies {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(family)
		}
		buf.WriteByte('{')
		for i := range n.Blocks {
			emitFontFeatureBlock(buf, &n.Blocks[i])
		}
		buf.WriteByte('}')
	}
}

func emitSelector(buf *bytes.Buffer, selector *Selector) {
	for _, sel := range selector.Selectors {
		buf.Write(sel.Value)
	}
	buf.WriteByte('{')
	for _, rule := range selector.Rules {
		emitNode(buf, rule)
	}
	buf.WriteByte('}')
}

func emitDeclaration(buf *bytes.Buffer, declaration *Declaration) {
	buf.Write(declaration.Key)
	buf.WriteByte(':')
	for i, value := range declaration.Value {
		if i > 0 {
			buf.WriteByte(' ')
		}
		emitValue(buf, value)
	}
	if declaration.Important {
		buf.WriteString("!important")
	}
	buf.WriteByte(';')
}

func emitValue(buf *bytes.Buffer, value Value) {
	switch v := value.(type) {
	case *BasicValue:
		buf.Write(v.Value)
	case *StringValue:
		if v.SingleQuote {
			buf.WriteByte('\'')
			buf.Write(v.Value)
			buf.WriteByte('\'')
			return
		}
		buf.WriteByte('"')
		buf.Write(v.Value)
		buf.WriteByte('"')
	case *FunctionValue:
		buf.Write(v.Name)
		buf.WriteByte('(')
		for i, arg := range v.Arguments {
			if i > 0 {
				buf.WriteByte(',')
			}
			emitValue(buf, arg)
		}
		buf.WriteByte(')')
	case *Comment:
		buf.WriteString("/*")
		buf.Write(v.Text)
		buf.WriteString("*/")
	}
}

func emitMediaQuery(buf *bytes.Buffer, query MediaQuery) {
	for i, expr := range query.Queries {
		if i > 0 {
			buf.WriteByte(',')
		}
		if expr.Not {
			buf.WriteString("not ")
		}
		if expr.Only {
			buf.WriteString("only ")
		}
		wroteTerm := false
		if len(expr.MediaType) > 0 {
			buf.Write(expr.MediaType)
			wroteTerm = true
		}
		for _, feature := range expr.Features {
			if wroteTerm {
				buf.WriteString(" and ")
			}
			buf.WriteByte('(')
			buf.Write(feature.Name)
			if feature.Value != nil {
				buf.WriteByte(':')
				buf.Write(feature.Value)
			}
			buf.WriteByte(')')
			wroteTerm = true
		}
	}
}

func emitContainerQuery(buf *bytes.Buffer, query ContainerQuery) {
	for i, cond := range query.Conditions {
		if i > 0 {
			buf.WriteString(" and ")
		}
		buf.WriteByte('(')
		for j, feature := range cond.Features {
			if j > 0 {
				buf.WriteString(" and ")
			}
			buf.Write(feature.Name)
			if feature.Value != nil {
				buf.WriteByte(':')
				buf.Write(feature.Value)
			}
		}
		buf.WriteByte(')')
	}
}

func emitKeyframeStop(buf *bytes.Buffer, stop *KeyframeStop) {
	for i, value := range stop.Stops {
		if i > 0 {
			buf.WriteByte(',')
		}
		emitValue(buf, value)
	}
	buf.WriteByte('{')
	for _, rule := range stop.Rules {
		emitNode(buf, rule)
	}
	buf.WriteByte('}')
}

func emitFontFeatureBlock(buf *bytes.Buffer, block *FontFeatureValuesBlock) {
	buf.WriteByte('@')
	buf.Write(block.Name)
	buf.WriteByte('{')
	for i := range block.Declarations {
		emitNode(buf, &block.Declarations[i])
	}
	buf.WriteByte('}')
}
