package utils

import "unicode"

// Capitalize returns the string with its first character capitalized.
func Capitalize(str string) string {
	r := []rune(str)
	if len(r) < 1 {
		return ""
	}

	r[0] = unicode.ToUpper(r[0])

	return string(r)
}

// UniqueStrings returns a new slice containing only the unique strings from the input slice.
func UniqueStrings(slice []string) []string {
	uniques := make(map[string]bool, len(slice))
	for _, s := range slice {
		uniques[s] = true
	}

	result := make([]string, 0, len(uniques))
	for s := range uniques {
		result = append(result, s)
	}

	return result
}
