package utils

import "strings"

type Wildcard struct {
	Prefix string
	Suffix string
}

type Wildcards []Wildcard

func (wcs Wildcards) Find(pred func(Wildcard) bool) *Wildcard {
	for _, wc := range wcs {
		if pred(wc) {
			return &wc
		}
	}
	return nil
}

func NewWildcard(s string) *Wildcard {
	if i := strings.IndexByte(s, '*'); i >= 0 {
		return &Wildcard{s[0:i], s[i+1:]}
	}
	return nil
}

func (w Wildcard) Match(s string) bool {
	return len(s) >= len(w.Prefix+w.Suffix) && strings.HasPrefix(s, w.Prefix) && strings.HasSuffix(s, w.Suffix)
}
