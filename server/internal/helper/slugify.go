package helper

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Slugify derives an ASCII room ID while keeping the display name separate.
// Non-ASCII names use delimited Unicode code points so the mapping is lossless.
func Slugify(s string) string {
	s = strings.TrimSpace(s)
	for _, character := range s {
		if character >= utf8.RuneSelf {
			var encoded strings.Builder
			for _, codePoint := range s {
				if encoded.Len() > 0 {
					encoded.WriteByte('-')
				}
				encoded.WriteByte('u')
				encoded.WriteString(strconv.FormatInt(int64(codePoint), 16))
			}
			return encoded.String()
		}
	}

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

	result := strings.TrimSuffix(slug.String(), "-")
	return result
}
