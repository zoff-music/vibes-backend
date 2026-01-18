package helper

import (
	"regexp"
	"strings"
)

// Slugify converts a string into a URL-safe slug.
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	// Replace spaces with dashes
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric characters (except dashes)
	reg := regexp.MustCompile("[^a-z0-9-]+")
	s = reg.ReplaceAllString(s, "")
	// Remove multiple dashes
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")
	// Trim dashes
	s = strings.Trim(s, "-")
	return s
}
