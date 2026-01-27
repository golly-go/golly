package golly

import (
	"regexp"
	"strings"
)

type RouteToken struct {
	value     string
	matcher   string
	isDynamic bool
}

func (rs *RouteToken) Value() string   { return rs.value }
func (rs *RouteToken) Matcher() string { return rs.matcher }
func (rs *RouteToken) IsDynamic() bool { return rs.isDynamic }
func (rs *RouteToken) Equal(rs2 *RouteToken) bool {
	return rs2.value == rs.value
}

// Match takes a string and matches it against hte current RouteToken
// if its not dynamic then EqualFold
// This pointer might be overly optimized i will try with it and without it to see
func (rs *RouteToken) Match(str *string) bool {
	if !rs.isDynamic {
		return rs.value == *str
	}

	if rs.matcher == "" {
		return true
	}

	matched, _ := regexp.MatchString(rs.matcher, *str)
	return matched
}

// tokenize takes a string path and turns it into RouteTokens.
// Optimized for hotpath: no []byte conversion, no per-token heap objects.
// All strings are slices of the original path (zero alloc).
func tokenize(path string) []RouteToken {
	if path == "" {
		return nil
	}

	segmentCount := strings.Count(path, "/") + 1
	tokens := make([]RouteToken, 0, segmentCount)

	n := len(path)
	pos := 0

	// Leading root if path starts with "/"
	if path[0] == '/' {
		tokens = append(tokens, RouteToken{value: "/"})
		pos = 1
	}

	for pos < n {
		start := pos

		// Dynamic segment: {var} or {var:matcher}
		if path[pos] == '{' {
			pos++
			for pos < n && path[pos] != '}' {
				pos++
			}
			if pos < n && path[pos] == '}' {
				// scan for colon inside braces
				colon := -1
				for i := start + 1; i < pos; i++ {
					if path[i] == ':' {
						colon = i
						break
					}
				}

				if colon >= 0 {
					tokens = append(tokens, RouteToken{
						value:     path[start+1 : colon],
						isDynamic: true,
						matcher:   path[colon+1 : pos],
					})
				} else {
					tokens = append(tokens, RouteToken{
						value:     path[start+1 : pos],
						isDynamic: true,
					})
				}

				pos++ // past '}'
				if pos < n && path[pos] == '/' {
					pos++
				}
				continue
			}
			// malformed '{' falls through and treated as static below
			pos = start
		}

		// Static segment
		for pos < n && path[pos] != '/' && path[pos] != '{' {
			pos++
		}
		if start != pos {
			tokens = append(tokens, RouteToken{value: path[start:pos]})
		}

		if pos < n && path[pos] == '/' {
			pos++
		}
	}

	return tokens
}

func makePathCount(path string) int {
	if path == "" {
		return 0
	}
	if path == "/" {
		return 1
	}
	return strings.Count(path, "/") + 1
}

// pathSegments takes a string path and turns it into segments.
// Used for walking/matching/vars. Optimized for runtime hotpath:
// - no []byte conversion
// - no []byte->string copies
// - segments are string views into `path` (zero-alloc)
// Only allocation is the segments slice backing array (once).
func pathSegments(stack []string, path string) {
	if path == "" || cap(stack) == 0 {
		return
	}

	if path == "/" {
		stack[0] = "/"

		return
	}

	n := len(path)
	i := 0
	cnt := 0

	// Leading root segment if starts with '/'
	if path[0] == '/' {
		stack[cnt] = "/"

		i = 1
		cnt++
	}

	for i < n {
		// Skip any extra slashes (prevents empty segments)
		for i < n && path[i] == '/' {
			i++
		}
		if i >= n {
			break
		}

		start := i
		for i < n && path[i] != '/' {
			i++
		}

		// Zero-alloc substring view
		stack[cnt] = path[start:i]
		cnt++
	}

}
