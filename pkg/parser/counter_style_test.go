package parser

import "testing"

func TestCounterStyleAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @counter-style rule",
			input: `@counter-style thumbs {
				system: cyclic;
				symbols: "\1F44D";
				suffix: " ";
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&CounterStyleAtRule{
						Name: []byte("thumbs"),
						Declarations: []Declaration{
							{Key: []byte("system"), Value: []Value{&BasicValue{Value: []byte("cyclic")}}},
							{Key: []byte("symbols"), Value: []Value{&StringValue{SingleQuote: false, Value: []byte("\\1F44D")}}},
							{Key: []byte("suffix"), Value: []Value{&StringValue{SingleQuote: false, Value: []byte(" ")}}},
						},
					},
				},
			},
		},
		{
			name: "@counter-style rule with complex symbols",
			input: `@counter-style dice {
				system: cyclic;
				symbols: ⚀ ⚁ ⚂ ⚃ ⚄ ⚅;
				suffix: " ";
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&CounterStyleAtRule{
						Name: []byte("dice"),
						Declarations: []Declaration{
							{Key: []byte("system"), Value: []Value{&BasicValue{Value: []byte("cyclic")}}},
							{Key: []byte("symbols"), Value: []Value{
								&BasicValue{Value: []byte("⚀")},
								&BasicValue{Value: []byte("⚁")},
								&BasicValue{Value: []byte("⚂")},
								&BasicValue{Value: []byte("⚃")},
								&BasicValue{Value: []byte("⚄")},
								&BasicValue{Value: []byte("⚅")},
							}},
							{Key: []byte("suffix"), Value: []Value{&StringValue{SingleQuote: false, Value: []byte(" ")}}},
						},
					},
				},
			},
		},
		{
			name: "@counter-style rule with additive system",
			input: `@counter-style roman {
				system: additive;
				range: 1 3999;
				additive-symbols: 1000 M, 900 CM, 500 D, 400 CD, 100 C, 90 XC, 50 L, 40 XL, 10 X, 9 IX, 5 V, 4 IV, 1 I;
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&CounterStyleAtRule{
						Name: []byte("roman"),
						Declarations: []Declaration{
							{Key: []byte("system"), Value: []Value{&BasicValue{Value: []byte("additive")}}},
							{Key: []byte("range"), Value: []Value{
								&BasicValue{Value: []byte("1")},
								&BasicValue{Value: []byte("3999")},
							}},
							{Key: []byte("additive-symbols"), Value: []Value{
								&BasicValue{Value: []byte("1000")},
								&BasicValue{Value: []byte("M")},
								&BasicValue{Value: []byte("900")},
								&BasicValue{Value: []byte("CM")},
								&BasicValue{Value: []byte("500")},
								&BasicValue{Value: []byte("D")},
								&BasicValue{Value: []byte("400")},
								&BasicValue{Value: []byte("CD")},
								&BasicValue{Value: []byte("100")},
								&BasicValue{Value: []byte("C")},
								&BasicValue{Value: []byte("90")},
								&BasicValue{Value: []byte("XC")},
								&BasicValue{Value: []byte("50")},
								&BasicValue{Value: []byte("L")},
								&BasicValue{Value: []byte("40")},
								&BasicValue{Value: []byte("XL")},
								&BasicValue{Value: []byte("10")},
								&BasicValue{Value: []byte("X")},
								&BasicValue{Value: []byte("9")},
								&BasicValue{Value: []byte("IX")},
								&BasicValue{Value: []byte("5")},
								&BasicValue{Value: []byte("V")},
								&BasicValue{Value: []byte("4")},
								&BasicValue{Value: []byte("IV")},
								&BasicValue{Value: []byte("1")},
								&BasicValue{Value: []byte("I")},
							}},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}
