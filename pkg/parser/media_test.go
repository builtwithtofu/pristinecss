package parser

import "testing"

func TestMediaQueries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Basic media query",
			input: "@media screen { body { font-size: 16px; } }",
			expected: &Stylesheet{
				Rules: []Node{
					&MediaAtRule{
						Name: []byte("media"),
						Query: MediaQuery{
							Queries: []MediaQueryExpression{
								{
									MediaType: []byte("screen"),
								},
							},
						},
						Rules: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Element, Value: []byte("body")}},
								Rules: []Node{
									&Declaration{Key: []byte("font-size"), Value: []Value{&BasicValue{Value: []byte("16px")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "Media query with condition",
			input: "@media (max-width: 600px) { .container { width: 100%; } }",
			expected: &Stylesheet{
				Rules: []Node{
					&MediaAtRule{
						Name: []byte("media"),
						Query: MediaQuery{
							Queries: []MediaQueryExpression{
								{
									Features: []MediaFeature{
										{Name: []byte("max-width"), Value: []byte("600px")},
									},
								},
							},
						},
						Rules: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Class, Value: []byte(".container")}},
								Rules: []Node{
									&Declaration{Key: []byte("width"), Value: []Value{&BasicValue{Value: []byte("100%")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "Complex media query",
			input: "@media screen and (min-width: 768px) and (max-width: 1024px) { .sidebar { display: none; } }",
			expected: &Stylesheet{
				Rules: []Node{
					&MediaAtRule{
						Name: []byte("media"),
						Query: MediaQuery{
							Queries: []MediaQueryExpression{
								{
									MediaType: []byte("screen"),
									Features: []MediaFeature{
										{Name: []byte("min-width"), Value: []byte("768px")},
										{Name: []byte("max-width"), Value: []byte("1024px")},
									},
								},
							},
						},
						Rules: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Class, Value: []byte(".sidebar")}},
								Rules: []Node{
									&Declaration{Key: []byte("display"), Value: []Value{&BasicValue{Value: []byte("none")}}},
								},
							},
						},
					},
				},
			},
		},
	}
	runTests(t, tests)
}

func TestMediaRangeSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "single comparison", input: `@media (inline-size > 30rem) { .x { color: red; } }`},
		{name: "double comparison", input: `@media (400px <= width <= 700px) { .x { color: red; } }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseNoErrors(t, tt.input)
		})
	}
}

func TestMediaFeatureNestedValueParens(t *testing.T) {
	ss := parseNoErrors(t, `@media (foo: bar(1)) { .a { color: red; } }`)
	media := ss.Rules[0].(*MediaAtRule)
	got := string(media.Query.Queries[0].Features[0].Value)
	if got != "bar(1)" {
		t.Fatalf("media feature value = %q", got)
	}
	if len(media.Rules) != 1 {
		t.Fatalf("media rules = %d, want 1", len(media.Rules))
	}
}
