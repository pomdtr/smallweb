package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

func IsGlob(input string) bool {
	return strings.Contains(input, "*")
}

func ExtractGlobPattern(input, pattern string) (string, error) {
	if strings.Count(pattern, "*") != 1 {
		return "", fmt.Errorf("pattern must contain exactly one wildcard (*)")
	}

	g, err := glob.Compile(pattern)
	if err != nil {
		return "", err
	}

	if !g.Match(input) {
		return "", fmt.Errorf("input does not match pattern")
	}

	regexPattern := globToRegex(pattern)
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(input)
	if len(matches) != 2 {
		return "", fmt.Errorf("unexpected number of matches")
	}

	return matches[1], nil
}

func globToRegex(pattern string) string {
	parts := strings.Split(pattern, ".")
	for i, part := range parts {
		if strings.Contains(part, "*") {
			// Allow valid hostname characters: letters, digits, hyphens, and underscores
			// The first and last character of a label should be a letter, digit, or underscore
			parts[i] = strings.Replace(part, "*", "([a-zA-Z0-9_](?:[a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])?)", 1)
		} else {
			parts[i] = regexp.QuoteMeta(part)
		}
	}
	return "^" + strings.Join(parts, "\\.") + "$"
}
