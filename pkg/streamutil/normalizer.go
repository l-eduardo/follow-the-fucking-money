package streamutil

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NormalizeText converts to uppercase, removes diacritics (accents) and collapses whitespace.
func NormalizeText(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = strings.ToUpper(result)
	return strings.Join(strings.Fields(result), " ")
}

// NormalizePersonName removes accents, standardizes to uppercase, and strips common prepositions.
func NormalizePersonName(name string) string {
	clean := NormalizeText(name)
	words := strings.Fields(clean)
	filtered := make([]string, 0, len(words))

	prepositions := map[string]bool{
		"DE": true, "DA": true, "DO": true, "DAS": true, "DOS": true, "E": true,
	}

	for _, w := range words {
		if !prepositions[w] {
			filtered = append(filtered, w)
		}
	}
	return strings.Join(filtered, " ")
}
