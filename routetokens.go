package golly

import (
	"regexp"
	"strings"
	"unsafe"
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

// stringToBytes converts a string to byte slice without allocation
// using unsafe pointer conversion
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// tokenize takes a string path and turns them into RouteTokens
// Optimized to reduce allocations by working with byte arrays
func tokenize(path string) []*RouteToken {
	if path == "" {
		return nil
	}

	// Use unsafe conversion to avoid copying the string to bytes
	pathBytes := stringToBytes(path)
	segmentCount := strings.Count(path, "/") + 1
	tokens := make([]*RouteToken, segmentCount)
	cnt := 0

	pos := 0
	end := len(pathBytes)

	// Add leading root if path starts with "/"
	if pathBytes[0] == '/' {
		tokens[cnt] = &RouteToken{value: "/"}
		cnt++
		pos++ // Skip the initial slash
	}

	for pos < end {
		start := pos

		// Handle variable segments {var:[0-9]+}
		if pathBytes[pos] == '{' {
			pos++
			for pos < end && pathBytes[pos] != '}' {
				pos++
			}
			if pos < end && pathBytes[pos] == '}' {
				colon := -1
				for i := start + 1; i < pos; i++ {
					if pathBytes[i] == ':' {
						colon = i
						break
					}
				}

				if colon >= 0 {
					// Convert bytes to string only when creating the token
					tokens[cnt] = &RouteToken{
						value:     string(pathBytes[start+1 : colon]),
						isDynamic: true,
						matcher:   string(pathBytes[colon+1 : pos]),
					}
				} else {
					tokens[cnt] = &RouteToken{
						value:     string(pathBytes[start+1 : pos]),
						isDynamic: true,
					}
				}

				cnt++

				pos++ // Move past '}'
				continue
			}
		}

		// Handle static path segments
		for pos < end && pathBytes[pos] != '/' && pathBytes[pos] != '{' {
			pos++
		}

		if start != pos {
			// Convert bytes to string only when creating the token
			tokens[cnt] = &RouteToken{value: string(pathBytes[start:pos])}
			cnt++
		}

		// Skip trailing slash
		if pos < end && pathBytes[pos] == '/' {
			pos++
		}
	}

	return tokens[:cnt]
}

// pathSegments takes a string path and turns them into segments
// this is used for walking the path, matching routes and creating variables
// Optimized to reduce allocations by working with byte arrays
func pathSegments(path string) []string {

	if path == "" {
		return []string{}
	}

	if path == "/" {
		return []string{path}
	}

	// Use unsafe conversion to avoid copying the string to bytes
	pathBytes := stringToBytes(path)
	tokenCount := strings.Count(path, "/")
	segments := make([]string, tokenCount+1) // add + 1 to handle /

	start, cnt := 0, 0
	if pathBytes[cnt] == '/' {
		segments[cnt] = "/"
		start = 1
		cnt++
	}

	tokenStart := start
	for i := start; i < len(pathBytes); i++ {
		if pathBytes[i] == '/' {
			if tokenStart != i {
				// Convert bytes to string only when storing the segment
				segments[cnt] = string(pathBytes[tokenStart:i])
				cnt++
			}
			tokenStart = i + 1
		}
	}

	// Capture the last segment if the path doesn't end with '/'
	if tokenStart < len(pathBytes) {
		segments[cnt] = string(pathBytes[tokenStart:])
	}

	return segments
}
