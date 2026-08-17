package helper

import "testing"

type slugifyTestCase struct {
	name     string
	input    string
	expected string
}

func TestSlugify(t *testing.T) {
	tests := []slugifyTestCase{
		{name: "words", input: "My Room", expected: "my-room"},
		{name: "punctuation", input: "Rock & Roll!", expected: "rock-roll"},
		{name: "repeated separators", input: "-- late   night --", expected: "late-night"},
		{name: "non ASCII", input: "Café Oslo", expected: "caf-oslo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Slugify(tt.input)
			if actual != tt.expected {
				t.Errorf("error slugifying %q: got %q, expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}
