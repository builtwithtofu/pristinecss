package parser

import "testing"

func TestColorProfileAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @color-profile rule with dashed-ident",
			input: `@color-profile --my-profile {
				src: url("https://example.com/my-profile.icc");
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&ColorProfileAtRule{
						Name: []byte("--my-profile"),
						Declarations: []Declaration{
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{SingleQuote: false, Value: []byte("https://example.com/my-profile.icc")},
									},
								},
							}},
						},
					},
				},
			},
		},
		{
			name: "@color-profile rule with device-cmyk",
			input: `@color-profile device-cmyk {
				src: url("default-cmyk.icc");
			}`,
			expected: &Stylesheet{
				Rules: []Node{
					&ColorProfileAtRule{
						IsDeviceCMYK: true,
						Declarations: []Declaration{
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{SingleQuote: false, Value: []byte("default-cmyk.icc")},
									},
								},
							}},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}
