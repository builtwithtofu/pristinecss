package parser

import (
	"strings"
	"testing"

	"github.com/builtwithtofu/pristinecss/pkg/lexer"
)

func TestBasicSelectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Simple element selector",
			input: "div { color: blue; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{Key: []byte("color"), Value: []Value{&BasicValue{Value: []byte("blue")}}},
						},
					},
				},
			},
		},
		{
			name:  "Class selector",
			input: ".highlight { background-color: yellow; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Class, Value: []byte(".highlight")}},
						Rules: []Node{
							&Declaration{Key: []byte("background-color"), Value: []Value{&BasicValue{Value: []byte("yellow")}}},
						},
					},
				},
			},
		},
		{
			name:  "ID selector",
			input: "#main { font-size: 16px; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: ID, Value: []byte("#main")}},
						Rules: []Node{
							&Declaration{Key: []byte("font-size"), Value: []Value{&BasicValue{Value: []byte("16px")}}},
						},
					},
				},
			},
		},
		{
			name:  "Attribute selector",
			input: "[type='text'] { border: 1px solid gray; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Attribute, Value: []byte("[type='text']")}},
						Rules: []Node{
							&Declaration{Key: []byte("border"), Value: []Value{
								&BasicValue{Value: []byte("1px")},
								&BasicValue{Value: []byte("solid")},
								&BasicValue{Value: []byte("gray")},
							}},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}

func TestComplexSelectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Compound selector",
			input: "div.container { max-width: 1200px; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("div")},
							{Type: Class, Value: []byte(".container")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("max-width"), Value: []Value{&BasicValue{Value: []byte("1200px")}}},
						},
					},
				},
			},
		},
		{
			name:  "Multiple selectors",
			input: "h1, h2, h3 { font-family: sans-serif; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("h1")},
							{Type: Combinator, Value: []byte(",")},
							{Type: Element, Value: []byte("h2")},
							{Type: Combinator, Value: []byte(",")},
							{Type: Element, Value: []byte("h3")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("font-family"), Value: []Value{&BasicValue{Value: []byte("sans-serif")}}},
						},
					},
				},
			},
		},
		{
			name:  "Descendant combinator",
			input: "article p { line-height: 1.5; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("article")},
							{Type: Combinator, Value: []byte(" ")},
							{Type: Element, Value: []byte("p")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("line-height"), Value: []Value{&BasicValue{Value: []byte("1.5")}}},
						},
					},
				},
			},
		},
		{
			name:  "Child combinator",
			input: "ul > li { list-style-type: square; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("ul")},
							{Type: Combinator, Value: []byte(">")},
							{Type: Element, Value: []byte("li")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("list-style-type"), Value: []Value{&BasicValue{Value: []byte("square")}}},
						},
					},
				},
			},
		},
		{
			name: "Complex selector with pseudo-classes",
			input: `.form-select:not([multiple]):not([size]) {
				padding-right: 1.2rem;
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Class, Value: []byte(".form-select")},
							{Type: Pseudo, Value: []byte(":not([multiple])")},
							{Type: Pseudo, Value: []byte(":not([size])")},
						},
						Rules: []Node{
							&Declaration{
								Key: []byte("padding-right"),
								Value: []Value{
									&BasicValue{Value: []byte("1.2rem")},
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

func TestPseudoSelectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Pseudo-class selector",
			input: "a:hover { color: red; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("a")},
							{Type: Pseudo, Value: []byte(":hover")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("color"), Value: []Value{&BasicValue{Value: []byte("red")}}},
						},
					},
				},
			},
		},
		{
			name:  "Pseudo-element selector",
			input: "p::first-line { font-weight: bold; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("p")},
							{Type: Pseudo, Value: []byte("::first-line")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("font-weight"), Value: []Value{&BasicValue{Value: []byte("bold")}}},
						},
					},
				},
			},
		},
		{
			name:  "Pseudo-class and pseudo-element selector",
			input: "a:hover::before { content: '→'; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{
							{Type: Element, Value: []byte("a")},
							{Type: Pseudo, Value: []byte(":hover")},
							{Type: Pseudo, Value: []byte("::before")},
						},
						Rules: []Node{
							&Declaration{Key: []byte("content"), Value: []Value{&StringValue{SingleQuote: true, Value: []byte("→")}}},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}

func runTests(t *testing.T, tests []struct {
	name     string
	input    string
	expected *Stylesheet
}) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.input)
			tokens := lexer.Lex(source)
			result, errors := Parse(source, tokens)
			if len(errors) > 0 {
				t.Errorf("Unexpected errors: %v", errors)
			}
			diffStylesheet(t, tt.expected, result)
		})
	}
}

// Custom diff function
func diffStylesheet(t *testing.T, expected, actual *Stylesheet) {
	t.Helper()

	expectedStr := expected.String()
	actualStr := actual.String()

	if expectedStr == actualStr {
		return
	}

	expectedLines := strings.Split(expectedStr, "\n")
	actualLines := strings.Split(actualStr, "\n")

	diff := []string{}

	i, j := 0, 0
	for i < len(expectedLines) && j < len(actualLines) {
		if expectedLines[i] == actualLines[j] {
			diff = append(diff, "  "+expectedLines[i])
			i++
			j++
		} else {
			// Find the next matching line
			nextMatch := findNextMatch(expectedLines[i:], actualLines[j:])

			// Add removed lines
			for k := 0; k < nextMatch.expectedOffset; k++ {
				diff = append(diff, "-"+expectedLines[i+k])
			}

			// Add added lines
			for k := 0; k < nextMatch.actualOffset; k++ {
				diff = append(diff, "+"+actualLines[j+k])
			}

			i += nextMatch.expectedOffset
			j += nextMatch.actualOffset
		}
	}

	// Handle remaining lines
	for ; i < len(expectedLines); i++ {
		diff = append(diff, "-"+expectedLines[i])
	}
	for ; j < len(actualLines); j++ {
		diff = append(diff, "+"+actualLines[j])
	}

	// Adjust indentation for diff lines
	for i, line := range diff {
		indent := countLeadingSpaces(line[1:]) // Ignore the first character (-, +, or space)
		if line[0] == '-' || line[0] == '+' {
			diff[i] = line[:1] + strings.Repeat(" ", indent) + line[1+indent:]
		}
	}

	t.Errorf("CSS mismatch (-want +got):\n%s", strings.Join(diff, "\n"))
}

func countLeadingSpaces(s string) int {
	return len(s) - len(strings.TrimLeft(s, " "))
}

type matchOffset struct {
	expectedOffset int
	actualOffset   int
}

func findNextMatch(expected, actual []string) matchOffset {
	for i := 0; i < len(expected); i++ {
		for j := 0; j < len(actual); j++ {
			if expected[i] == actual[j] {
				return matchOffset{i, j}
			}
		}
	}
	return matchOffset{len(expected), len(actual)}
}
