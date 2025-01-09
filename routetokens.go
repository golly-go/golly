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

// tokenize takes a string path and turns them into RouteTokens
func tokenize(path string) []*RouteToken {
	if path == "" {
		return nil
	}

	segmentCount := strings.Count(path, "/") + 1
	tokens := make([]*RouteToken, segmentCount)
	cnt := 0

	pos := 0
	end := len(path)

	// Add leading root if path starts with "/"
	if path[0] == '/' {
		tokens[cnt] = &RouteToken{value: "/"}
		cnt++
		pos++ // Skip the initial slash
	}

	for pos < end {
		start := pos

		// Handle variable segments {var:[0-9]+}
		if path[pos] == '{' {
			pos++
			for pos < end && path[pos] != '}' {
				pos++
			}
			if pos < end && path[pos] == '}' {
				colon := -1
				for i := start + 1; i < pos; i++ {
					if path[i] == ':' {
						colon = i
						break
					}
				}

				if colon >= 0 {
					tokens[cnt] = &RouteToken{value: path[start+1 : colon], isDynamic: true, matcher: path[colon+1 : pos]}
				} else {
					tokens[cnt] = &RouteToken{value: path[start+1 : pos], isDynamic: true}
				}

				cnt++

				pos++ // Move past '}'
				continue
			}
		}

		// Handle static path segments
		for pos < end && path[pos] != '/' && path[pos] != '{' {
			pos++
		}

		if start != pos {
			tokens[cnt] = &RouteToken{value: path[start:pos]}
			cnt++
		}

		// Skip trailing slash
		if pos < end && path[pos] == '/' {
			pos++
		}
	}

	return tokens[:cnt]
}

// pathSegments takes a string path and turns them into segments
// this is used for walking the path, matching routes and creating variables
func pathSegments(path string) []string {

	if path == "" {
		return []string{}
	}

	if path == "/" {
		return []string{path}
	}

	// if path == "" || path == "/" {
	// 	return []string{"/"}
	// }

	tokenCount := strings.Count(path, "/")
	segments := make([]string, tokenCount+1) // add + 1 to handle /

	cnt := 0

	start := 0

	if path[0] == '/' {
		segments[cnt] = "/"
		start = 1
		cnt++
	}

	tokenStart := start

	for i := start; i < len(path); i++ {
		if path[i] == '/' {
			if tokenStart != i {
				segments[cnt] = path[tokenStart:i]
				cnt++
			}
			tokenStart = i + 1
		}
	}

	// Capture the last segment if the path doesn't end with '/'
	if tokenStart < len(path) {
		segments[cnt] = path[tokenStart:]
	}

	return segments
}
