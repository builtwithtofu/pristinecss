package parser

import "testing"

func TestCommentsInSelectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Single line comment in selector",
			input: `[type="search"] {
                /* 1 */
                outline-offset: -2px; /* 2 */
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Attribute, Value: []byte(`[type="search"]`)},
						},
						Rules: []Node{
							&Comment{Text: []byte("/* 1 */")},
							&Declaration{
								Key:   []byte("outline-offset"),
								Value: []Value{&BasicValue{Value: []byte("-2px")}},
							},
							&Comment{Text: []byte("/* 2 */")},
						},
					},
				},
			},
		},
		{
			name: "Multi-line comment before selector",
			input: `/**
                * 1. Correct the odd appearance in Chrome and Safari.
                * 2. Correct the outline style in Safari.
                */
                [type="search"] {
                    outline-offset: -2px;
                }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Comment{Text: []byte(`/**
                * 1. Correct the odd appearance in Chrome and Safari.
                * 2. Correct the outline style in Safari.
                */`)},
					&Selector{
						Selectors: []SelectorValue{
							{Type: Attribute, Value: []byte(`[type="search"]`)},
						},
						Rules: []Node{
							&Declaration{
								Key:   []byte("outline-offset"),
								Value: []Value{&BasicValue{Value: []byte("-2px")}},
							},
						},
					},
				},
			},
		},
		{
			name: "Mixed comments in selector block",
			input: `/* Header styles */
                .header {
                    color: blue; /* Brand color */
                    /* Navigation spacing */
                    margin: 20px;
                    padding: 10px; /* Standard padding */
                }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Comment{Text: []byte("/* Header styles */")},
					&Selector{
						Selectors: []SelectorValue{
							{Type: Class, Value: []byte(".header")},
						},
						Rules: []Node{
							&Declaration{
								Key:   []byte("color"),
								Value: []Value{&BasicValue{Value: []byte("blue")}},
							},
							&Comment{Text: []byte("/* Brand color */")},
							&Comment{Text: []byte("/* Navigation spacing */")},
							&Declaration{
								Key:   []byte("margin"),
								Value: []Value{&BasicValue{Value: []byte("20px")}},
							},
							&Declaration{
								Key:   []byte("padding"),
								Value: []Value{&BasicValue{Value: []byte("10px")}},
							},
							&Comment{Text: []byte("/* Standard padding */")},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}
