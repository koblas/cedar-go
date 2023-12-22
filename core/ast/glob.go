package ast

import (
	"strings"
)

// The character which is treated like a glob
const GLOB = "*"

// Glob will test a string pattern, potentially containing globs, against a
// subject string. The result is a simple true/false, determining whether or
// not the glob pattern matched the subject text.
func Glob(pattern, subj string) bool {
	// Empty pattern can only match empty subject
	if pattern == "" {
		return subj == pattern
	}

	// If the pattern _is_ a glob, it matches everything
	if pattern == GLOB {
		return true
	}

	parts := []string{}

	inEscape := false
	accum := ""
	for _, ch := range pattern {
		if inEscape {
			inEscape = false
			accum += string(ch)
		} else if ch == '*' {
			parts = append(parts, accum)
			accum = ""
		} else if ch == '\\' {
			inEscape = true
		} else {
			accum += string(ch)
		}
	}
	if inEscape {
		accum += "\\"
	}
	parts = append(parts, accum)

	if len(parts) == 1 {
		// No globs in pattern, so test for equality
		return subj == parts[0]
	}

	end := len(parts) - 1
	leadingGlob := parts[0] == ""
	trailingGlob := parts[end] == ""

	// Go over the leading parts and ensure they match.
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			// Check the first section. Requires special handling.
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			// Check that the middle parts match.
			if idx < 0 {
				return false
			}
		}

		// Trim evaluated text from subj as we loop over the pattern.
		subj = subj[idx+len(parts[i]):]
	}

	// Reached the last section. Requires special handling.
	return trailingGlob || strings.HasSuffix(subj, parts[end])
}
