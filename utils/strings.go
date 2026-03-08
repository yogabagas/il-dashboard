package utils

import "strings"

// SplitAndTrim splits the input string by sep, trims whitespace from each
// token and returns only non-empty tokens. Returns nil for empty input.
func SplitAndTrim(s string, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			res = append(res, t)
		}
	}
	return res
}
