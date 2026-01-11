package golly

const (
	star     = '*'
	question = '?'
)

// WildcardMatch searches for standard wildcards and matches a string to them
// generally used as a * but can support ?'s too
// example localhost:* or http://*.domain.com
// mi*ch
// matches:
// mitch
// mittttch
func WildcardMatch(pattern, str string) bool {
	patEnd, strEnd := len(pattern)-1, len(str)-1
	p, s := 0, 0
	matchIndex := -1
	wildcardIndex := -1

	// Exact match early exit
	if pattern == str {
		return true
	}

	if patEnd == 0 && pattern[0] == star {
		return true
	}

	// Empty pattern only matches empty string
	if patEnd < 0 {
		return false
	}

	for s <= strEnd {
		switch {
		case p <= patEnd && pattern[p] == star:
			wildcardIndex = p
			matchIndex = s
			p++
		case p <= patEnd && (pattern[p] == question || pattern[p] == str[s]):
			p++
			s++
		case wildcardIndex != -1:
			p = wildcardIndex + 1
			matchIndex++
			s = matchIndex
		default:
			return false
		}
	}

	// If remaining pattern is '*', match
	for p <= patEnd && pattern[p] == star {
		p++
	}

	return p > patEnd
}

func IsWildcardString(str string) bool {
	for pos := range str {
		if str[pos] == star || str[pos] == question {
			return true
		}
	}
	return false
}
