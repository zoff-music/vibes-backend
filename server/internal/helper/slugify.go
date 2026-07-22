package helper

import (
	"strings"
)

// Slugify converts a string into a URL-safe slug.
func Slugify(s string) string {
	var slug strings.Builder
	previousWasDash := false

	for _, character := range strings.ToLower(s) {
		isLetter := character >= 'a' && character <= 'z'
		isDigit := character >= '0' && character <= '9'
		if isLetter || isDigit {
			slug.WriteRune(character)
			previousWasDash = false
			continue
		}

		if (character == ' ' || character == '-') && slug.Len() > 0 && !previousWasDash {
			slug.WriteByte('-')
			previousWasDash = true
		}
	}

	return strings.TrimSuffix(slug.String(), "-")
}
