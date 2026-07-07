package parser

import "testing"

func TestKeyframes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Stylesheet
	}{
		{
			name: "Basic @keyframes rule",
			input: `@keyframes slide-in {
                from { transform: translateX(-100%); }
                to { transform: translateX(0); }
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&KeyframesAtRule{
						Name: []byte("slide-in"),
						Stops: []KeyframeStop{
							{
								Stops: []Value{&BasicValue{Value: []byte("from")}},
								Rules: []Node{
									&Declaration{Key: []byte("transform"), Value: []Value{
										&FunctionValue{
											Name: []byte("translateX"),
											Arguments: []Value{
												&BasicValue{Value: []byte("-100%")},
											},
										},
									}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("to")}},
								Rules: []Node{
									&Declaration{Key: []byte("transform"), Value: []Value{
										&FunctionValue{
											Name: []byte("translateX"),
											Arguments: []Value{
												&BasicValue{Value: []byte("0")},
											},
										},
									}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@keyframes rule with percentages",
			input: `@keyframes color-change {
                0% { background-color: red; }
                50% { background-color: green; }
                100% { background-color: blue; }
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&KeyframesAtRule{
						Name: []byte("color-change"),
						Stops: []KeyframeStop{
							{
								Stops: []Value{&BasicValue{Value: []byte("0%")}},
								Rules: []Node{
									&Declaration{Key: []byte("background-color"), Value: []Value{&BasicValue{Value: []byte("red")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("50%")}},
								Rules: []Node{
									&Declaration{Key: []byte("background-color"), Value: []Value{&BasicValue{Value: []byte("green")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("100%")}},
								Rules: []Node{
									&Declaration{Key: []byte("background-color"), Value: []Value{&BasicValue{Value: []byte("blue")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@keyframes with multiple selectors per stop",
			input: `@keyframes multi-step {
                0%, 100% { opacity: 0; }
                25%, 75% { opacity: 0.5; }
                50% { opacity: 1; }
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&KeyframesAtRule{
						Name: []byte("multi-step"),
						Stops: []KeyframeStop{
							{
								Stops: []Value{&BasicValue{Value: []byte("0%")}, &BasicValue{Value: []byte("100%")}},
								Rules: []Node{
									&Declaration{Key: []byte("opacity"), Value: []Value{&BasicValue{Value: []byte("0")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("25%")}, &BasicValue{Value: []byte("75%")}},
								Rules: []Node{
									&Declaration{Key: []byte("opacity"), Value: []Value{&BasicValue{Value: []byte("0.5")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("50%")}},
								Rules: []Node{
									&Declaration{Key: []byte("opacity"), Value: []Value{&BasicValue{Value: []byte("1")}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@keyframes with vendor prefix",
			input: `@-webkit-keyframes bounce {
                0%, 20%, 50%, 80%, 100% { transform: translateY(0); }
                40% { transform: translateY(-30px); }
                60% { transform: translateY(-15px); }
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&KeyframesAtRule{
						WebKitPrefix: true,
						Name:         []byte("bounce"),
						Stops: []KeyframeStop{
							{
								Stops: []Value{
									&BasicValue{Value: []byte("0%")},
									&BasicValue{Value: []byte("20%")},
									&BasicValue{Value: []byte("50%")},
									&BasicValue{Value: []byte("80%")},
									&BasicValue{Value: []byte("100%")},
								},
								Rules: []Node{
									&Declaration{Key: []byte("transform"), Value: []Value{
										&FunctionValue{
											Name: []byte("translateY"),
											Arguments: []Value{
												&BasicValue{Value: []byte("0")},
											},
										},
									}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("40%")}},
								Rules: []Node{
									&Declaration{Key: []byte("transform"), Value: []Value{
										&FunctionValue{
											Name: []byte("translateY"),
											Arguments: []Value{
												&BasicValue{Value: []byte("-30px")},
											},
										},
									}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("60%")}},
								Rules: []Node{
									&Declaration{Key: []byte("transform"), Value: []Value{
										&FunctionValue{
											Name: []byte("translateY"),
											Arguments: []Value{
												&BasicValue{Value: []byte("-15px")},
											},
										},
									}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "@keyframes with multiple properties per stop",
			input: `@keyframes complex-animation {
                from {
                    left: 0;
                    top: 0;
                }
                50% {
                    left: 50%;
                    top: 100px;
                    background-color: blue;
                }
                to {
                    left: 100%;
                    top: 0;
                }
            }`,
			expected: &Stylesheet{
				Rules: []Node{
					&KeyframesAtRule{
						Name: []byte("complex-animation"),
						Stops: []KeyframeStop{
							{
								Stops: []Value{&BasicValue{Value: []byte("from")}},
								Rules: []Node{
									&Declaration{Key: []byte("left"), Value: []Value{&BasicValue{Value: []byte("0")}}},
									&Declaration{Key: []byte("top"), Value: []Value{&BasicValue{Value: []byte("0")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("50%")}},
								Rules: []Node{
									&Declaration{Key: []byte("left"), Value: []Value{&BasicValue{Value: []byte("50%")}}},
									&Declaration{Key: []byte("top"), Value: []Value{&BasicValue{Value: []byte("100px")}}},
									&Declaration{Key: []byte("background-color"), Value: []Value{&BasicValue{Value: []byte("blue")}}},
								},
							},
							{
								Stops: []Value{&BasicValue{Value: []byte("to")}},
								Rules: []Node{
									&Declaration{Key: []byte("left"), Value: []Value{&BasicValue{Value: []byte("100%")}}},
									&Declaration{Key: []byte("top"), Value: []Value{&BasicValue{Value: []byte("0")}}},
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
