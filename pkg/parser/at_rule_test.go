package parser

import "testing"

func TestFontFaceAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @font-face rule",
			input: `@font-face {
                font-family: "Open Sans";
                src: url("/fonts/OpenSans-Regular-webfont.woff2") format("woff2");
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&FontFaceAtRule{
						Declarations: []Declaration{
							{Key: []byte("font-family"), Value: []Value{
								&StringValue{Value: []byte("Open Sans")},
							}},
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{Value: []byte("/fonts/OpenSans-Regular-webfont.woff2")},
									},
								},
								&FunctionValue{
									Name: []byte("format"),
									Arguments: []Value{
										&StringValue{Value: []byte("woff2")},
									},
								},
							}},
						},
					},
				},
			},
		},
		{
			name: "@font-face with multiple src",
			input: `@font-face {
                font-family: "Bitstream Vera Serif Bold";
                src: url("https://mdn.mozillademos.org/files/2468/VeraSeBd.ttf");
                src: local("Bitstream Vera Serif Bold"),
                     local("BitstreamVeraSerif-Bold"),
                     url("VeraSeBd.ttf") format("truetype");
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&FontFaceAtRule{
						Declarations: []Declaration{
							{Key: []byte("font-family"), Value: []Value{
								&StringValue{Value: []byte("Bitstream Vera Serif Bold")},
							}},
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{Value: []byte("https://mdn.mozillademos.org/files/2468/VeraSeBd.ttf")},
									},
								},
							}},
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("local"),
									Arguments: []Value{
										&StringValue{Value: []byte("Bitstream Vera Serif Bold")},
									},
								},
								&FunctionValue{
									Name: []byte("local"),
									Arguments: []Value{
										&StringValue{Value: []byte("BitstreamVeraSerif-Bold")},
									},
								},
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{Value: []byte("VeraSeBd.ttf")},
									},
								},
								&FunctionValue{
									Name: []byte("format"),
									Arguments: []Value{
										&StringValue{Value: []byte("truetype")},
									},
								},
							}},
						},
					},
				},
			},
		},
		{
			name: "@font-face with additional properties",
			input: `@font-face {
                font-family: "Roboto";
                src: url("Roboto-Regular.woff2") format("woff2");
                font-weight: 400;
                font-style: normal;
                font-display: swap;
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&FontFaceAtRule{
						Declarations: []Declaration{
							{Key: []byte("font-family"), Value: []Value{
								&StringValue{Value: []byte("Roboto")},
							}},
							{Key: []byte("src"), Value: []Value{
								&FunctionValue{
									Name: []byte("url"),
									Arguments: []Value{
										&StringValue{Value: []byte("Roboto-Regular.woff2")},
									},
								},
								&FunctionValue{
									Name: []byte("format"),
									Arguments: []Value{
										&StringValue{Value: []byte("woff2")},
									},
								},
							}},
							{Key: []byte("font-weight"), Value: []Value{
								&BasicValue{Value: []byte("400")},
							}},
							{Key: []byte("font-style"), Value: []Value{
								&BasicValue{Value: []byte("normal")},
							}},
							{Key: []byte("font-display"), Value: []Value{
								&BasicValue{Value: []byte("swap")},
							}},
						},
					},
				},
			},
		},
	}
	runTests(t, tests)
}

func TestCharsetAtRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name:  "Simple @charset rule",
			input: `@charset "UTF-8";`,
			expected: &Stylesheet{
				Rules: []Node{
					&CharsetAtRule{
						Charset: &StringValue{Value: []byte("UTF-8")},
					},
				},
			},
		},
		{
			name: "@charset rule at the beginning of a stylesheet",
			input: `@charset "UTF-8";
                    body { font-family: Arial, sans-serif; }`,
			expected: &Stylesheet{
				Rules: []Node{
					&CharsetAtRule{
						Charset: &StringValue{Value: []byte("UTF-8")},
					},
					&Selector{
						Selectors: []SelectorValue{{Type: Element, Value: []byte("body")}},
						Rules: []Node{
							&Declaration{Key: []byte("font-family"), Value: []Value{
								&BasicValue{Value: []byte("Arial")},
								&BasicValue{Value: []byte("sans-serif")},
							}},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}
