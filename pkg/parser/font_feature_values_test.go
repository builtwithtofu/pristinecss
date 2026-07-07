package parser

import "testing"

func TestFontFeatureValuesAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @font-feature-values rule",
			input: `@font-feature-values "Some Font", Other Font {
				@stylistic {
					fancy-style: 12;
					flowering: 14;
				}
				@swash {
					ornate: 1;
				}
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&FontFeatureValuesAtRule{
						FontFamilies: [][]byte{[]byte("Some Font"), []byte("Other Font")},
						Blocks: []FontFeatureValuesBlock{
							{
								Name: []byte("stylistic"),
								Declarations: []Declaration{
									{Key: []byte("fancy-style"), Value: []Value{&BasicValue{Value: []byte("12")}}},
									{Key: []byte("flowering"), Value: []Value{&BasicValue{Value: []byte("14")}}},
								},
							},
							{
								Name: []byte("swash"),
								Declarations: []Declaration{
									{Key: []byte("ornate"), Value: []Value{&BasicValue{Value: []byte("1")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@font-feature-values rule with multiple blocks",
			input: `@font-feature-values Font One {
				@styleset {
					double-bs: 1;
					sharp-terminals: 2;
				}
				@annotation {
					circled-digits: 3;
				}
				@ornaments {
					fleurons: 4;
				}
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&FontFeatureValuesAtRule{
						FontFamilies: [][]byte{[]byte("Font One")},
						Blocks: []FontFeatureValuesBlock{
							{
								Name: []byte("styleset"),
								Declarations: []Declaration{
									{Key: []byte("double-bs"), Value: []Value{&BasicValue{Value: []byte("1")}}},
									{Key: []byte("sharp-terminals"), Value: []Value{&BasicValue{Value: []byte("2")}}},
								},
							},
							{
								Name: []byte("annotation"),
								Declarations: []Declaration{
									{Key: []byte("circled-digits"), Value: []Value{&BasicValue{Value: []byte("3")}}},
								},
							},
							{
								Name: []byte("ornaments"),
								Declarations: []Declaration{
									{Key: []byte("fleurons"), Value: []Value{&BasicValue{Value: []byte("4")}}},
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
