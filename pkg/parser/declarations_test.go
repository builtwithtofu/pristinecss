package parser

import (
	"testing"
)

func TestBasicDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Simple color value",
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
			name:  "Important declaration",
			input: "div { color: red !important; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key:       []byte("color"),
								Value:     []Value{&BasicValue{Value: []byte("red")}},
								Important: true,
							},
						},
					},
				},
			},
		},
		{
			name:  "Multiple space-separated values",
			input: "div { margin: 10px 20px 30px 40px; }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("margin"),
								Value: []Value{
									&BasicValue{Value: []byte("10px")},
									&BasicValue{Value: []byte("20px")},
									&BasicValue{Value: []byte("30px")},
									&BasicValue{Value: []byte("40px")},
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

func TestColorDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "RGB function",
			input: "div { color: rgb(255, 0, 0); }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("color"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("rgb"),
										Arguments: []Value{
											&BasicValue{Value: []byte("255")},
											&BasicValue{Value: []byte("0")},
											&BasicValue{Value: []byte("0")},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Hex colors",
			input: `.colors {
                color: #ff0000;
                background: #00ff00;
                border-color: #0000ff;
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Class, Value: []byte(".colors")}},
						Rules: []Node{
							&Declaration{Key: []byte("color"), Value: []Value{&BasicValue{Value: []byte("#ff0000")}}},
							&Declaration{Key: []byte("background"), Value: []Value{&BasicValue{Value: []byte("#00ff00")}}},
							&Declaration{Key: []byte("border-color"), Value: []Value{&BasicValue{Value: []byte("#0000ff")}}},
						},
					},
				},
			},
		},
	}
	runTests(t, tests)
}

func TestURLDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Simple URL",
			input: `div { background-image: url('image.jpg'); }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("background-image"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("url"),
										Arguments: []Value{
											&StringValue{Value: []byte("image.jpg"), SingleQuote: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "URL with RTL comment",
			input: `.icon {
                background-image: url("test.svg") /*rtl:url("test-rtl.svg")*/;
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Class, Value: []byte(".icon")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("background-image"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("url"),
										Arguments: []Value{
											&StringValue{Value: []byte("test.svg")},
										},
									},
									&Comment{Text: []byte("/*rtl:url(\"test-rtl.svg\")*/")},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Data URI with RTL comment",
			input: `.carousel-prev {
                background-image: url("data:image/svg+xml,%3csvg viewBox='0 0 16 16'%3e%3cpath d='M11.354 1.646'/%3e%3c/svg%3e") /*rtl:url("data:image/svg+xml,%3csvg viewBox='0 0 16 16'%3e%3cpath d='M4.646 1.646'/%3e%3c/svg%3e")*/;
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Class, Value: []byte(".carousel-prev")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("background-image"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("url"),
										Arguments: []Value{
											&StringValue{Value: []byte("data:image/svg+xml,%3csvg viewBox='0 0 16 16'%3e%3cpath d='M11.354 1.646'/%3e%3c/svg%3e")},
										},
									},
									&Comment{Text: []byte("/*rtl:url(\"data:image/svg+xml,%3csvg viewBox='0 0 16 16'%3e%3cpath d='M4.646 1.646'/%3e%3c/svg%3e\")*/")},
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

func TestFunctionDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Calc function",
			input: "div { width: calc(100% - 20px); }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("width"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("calc"),
										Arguments: []Value{
											&BasicValue{Value: []byte("100%")},
											&BasicValue{Value: []byte("-")},
											&BasicValue{Value: []byte("20px")},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "Gradient with nested functions",
			input: "div { background: linear-gradient(to right, rgb(255,0,0), rgba(0,0,255,0.5)); }",
			expected: &Stylesheet{
				Rules: []Node{
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("div")}},
						Rules: []Node{
							&Declaration{
								Key: []byte("background"),
								Value: []Value{
									&FunctionValue{
										Name: []byte("linear-gradient"),
										Arguments: []Value{
											&BasicValue{Value: []byte("to")},
											&BasicValue{Value: []byte("right")},
											&FunctionValue{
												Name: []byte("rgb"),
												Arguments: []Value{
													&BasicValue{Value: []byte("255")},
													&BasicValue{Value: []byte("0")},
													&BasicValue{Value: []byte("0")},
												},
											},
											&FunctionValue{
												Name: []byte("rgba"),
												Arguments: []Value{
													&BasicValue{Value: []byte("0")},
													&BasicValue{Value: []byte("0")},
													&BasicValue{Value: []byte("255")},
													&BasicValue{Value: []byte("0.5")},
												},
											},
										},
									},
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
