package service

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizeGameTitle canonicalizes a game title for matching: NFKC
// normalization (full-width to half-width), lowercasing, punctuation removal,
// and whitespace collapsing. Version-significant words are preserved, so
// e.g. "Summer Pockets" and "Summer Pockets REFLECTION BLUE" stay distinct.
func NormalizeGameTitle(value string) string {
	normalized := strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			return char
		}
		return ' '
	}, strings.ToLower(norm.NFKC.String(strings.TrimSpace(value))))

	return strings.Join(strings.Fields(normalized), " ")
}

func casefoldValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
