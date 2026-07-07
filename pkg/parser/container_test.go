package parser

import "testing"

func TestContainerAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @container rule",
			input: `@container (min-width: 700px) {
				.card {
					font-size: 1.5em;
				}
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&ContainerAtRule{
						Query: ContainerQuery{
							Conditions: []ContainerCondition{
								{
									Features: []ContainerFeature{
										{Name: []byte("min-width"), Value: []byte("700px")},
									},
								},
							},
						},
						Declarations: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Class, Value: []byte(".card")}},
								Rules: []Node{
									&Declaration{Key: []byte("font-size"), Value: []Value{&BasicValue{Value: []byte("1.5em")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@container rule with name",
			input: `@container sidebar (min-width: 400px) {
				.sidebar {
					flex: 1 1 auto;
				}
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&ContainerAtRule{
						Name: []byte("sidebar"),
						Query: ContainerQuery{
							Conditions: []ContainerCondition{
								{
									Features: []ContainerFeature{
										{Name: []byte("min-width"), Value: []byte("400px")},
									},
								},
							},
						},
						Declarations: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Class, Value: []byte(".sidebar")}},
								Rules: []Node{
									&Declaration{Key: []byte("flex"), Value: []Value{
										&BasicValue{Value: []byte("1")},
										&BasicValue{Value: []byte("1")},
										&BasicValue{Value: []byte("auto")},
									}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@container rule with complex condition",
			input: `@container (min-width: 700px) and (max-width: 1000px) {
				.container {
					display: flex;
					flex-wrap: wrap;
				}
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&ContainerAtRule{
						Query: ContainerQuery{
							Conditions: []ContainerCondition{
								{
									Features: []ContainerFeature{
										{Name: []byte("min-width"), Value: []byte("700px")},
									},
								},
								{
									Features: []ContainerFeature{
										{Name: []byte("max-width"), Value: []byte("1000px")},
									},
								},
							},
						},
						Declarations: []Node{
							&Selector{
								Selectors: []SelectorValue{{Type: Class, Value: []byte(".container")}},
								Rules: []Node{
									&Declaration{Key: []byte("display"), Value: []Value{&BasicValue{Value: []byte("flex")}}},
									&Declaration{Key: []byte("flex-wrap"), Value: []Value{&BasicValue{Value: []byte("wrap")}}},
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

func TestContainerRangeSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "single comparison", input: `@container (inline-size > 30rem) { .x { color: red; } }`},
		{name: "double comparison", input: `@container (400px <= width <= 700px) { .x { color: red; } }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseNoErrors(t, tt.input)
		})
	}
}
