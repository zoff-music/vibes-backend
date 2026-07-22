package helper

import "testing"

type slugifyTestCase struct {
	name     string
	input    string
	expected string
}

func TestSlugify(t *testing.T) {
	testCases := []slugifyTestCase{
		{name: "words", input: "My Room", expected: "my-room"},
		{name: "punctuation", input: "Rock & Roll!", expected: "rock-roll"},
		{name: "repeated separators", input: "-- late   night --", expected: "late-night"},
		{name: "non ASCII", input: "Café Oslo", expected: "caf-oslo"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := Slugify(testCase.input)
			if actual != testCase.expected {
				t.Errorf("error slugifying %q: got %q, expected %q", testCase.input, actual, testCase.expected)
			}
		})
	}
}
