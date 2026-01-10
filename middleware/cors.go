package middleware

import (
	"net/http"
	"strings"

	"github.com/golly-go/golly"
)

const (
	star = "*"

	// Allow Headers
	acAllowOriginHeader      = "Access-Control-Allow-Origin"
	acAllowMethodHeader      = "Access-Control-Allow-Methods"
	acAllowHeadersHeader     = "Access-Control-Allow-Headers"
	acAllowCredentialsHeader = "Access-Control-Allow-Credentials"

	acRequestMethodHeader  = "Access-Control-Request-Method"
	acRequestHeadersHeader = "Access-Control-Request-Headers"
	acExposeHeadersHeader  = "Access-Control-Expose-Headers"

	Vary    = "Vary"
	trueStr = "true"

	originHeader = "Origin"
)

var (
	defaultHeaders = []string{"Origin", "Accept", "Content-Type"}
	defaultMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead}
)

// CorsOptions defines the cors options
type CorsOptions struct {
	AllowAllHeaders bool
	AllowAllOrigins bool

	AllowedHeaders []string
	AllowedMethods []string
	AllowedOrigins []string

	ExposeHeaders    []string
	AllowCredentials bool
}

type cors struct {
	exposeHeadersStr string
	methodsStr       string

	exposeHeaders []string
	methods       []string
	headers       []string

	allowedOrigins []string // Linear search with EqualFold (faster for typical small sets, zero alloc)
	worigins       []string

	allHeaders bool
	allOrigins bool

	credentials bool
	credStr     string
}

// Init creates a cors record
func (c CorsOptions) init() cors {
	co := cors{
		exposeHeaders: golly.Convert(c.ExposeHeaders, http.CanonicalHeaderKey),
		credentials:   c.AllowCredentials,
		credStr:       trueStr,
	}

	if c.AllowAllHeaders {
		co.allHeaders = true
		co.headers = defaultHeaders
	} else {
		co.headers = golly.Convert(append(c.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
	}

	if len(c.AllowedMethods) == 0 {
		co.methods = defaultMethods
	} else {
		co.methods = golly.Convert(c.AllowedMethods, strings.ToUpper)
	}

	if c.AllowAllOrigins {
		co.allOrigins = true
	} else {
		// Separate wildcard and exact matches
		for _, origin := range c.AllowedOrigins {
			if golly.IsWildcardString(origin) {
				co.worigins = append(co.worigins, origin)
			} else {
				co.allowedOrigins = append(co.allowedOrigins, origin)
			}
		}
	}

	// Pre-compute joined strings for headers
	if len(co.exposeHeaders) > 0 {
		co.exposeHeadersStr = strings.Join(co.exposeHeaders, ",")
	}
	if len(co.methods) > 0 {
		co.methodsStr = strings.Join(co.methods, ",")
	}

	return co
}

// Cors builds a golly middleware providing cors ooptions
func Cors(co CorsOptions) func(next golly.HandlerFunc) golly.HandlerFunc {
	crs := co.init()

	return func(next golly.HandlerFunc) golly.HandlerFunc {
		return func(wctx *golly.WebContext) {
			r := wctx.Request()

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				crs.preflight(wctx)
			} else {
				crs.request(wctx)
				next(wctx)
			}
		}
	}
}

func (c cors) preflight(wctx *golly.WebContext) {
	r := wctx.Request()

	// Check if the request method is OPTIONS
	if r.Method != http.MethodOptions {
		return
	}

	headers := wctx.ResponseHeaders()

	headers.Add(Vary, originHeader)
	headers.Add(Vary, acRequestMethodHeader)
	headers.Add(Vary, acRequestHeadersHeader)

	origin := r.Header.Get(originHeader)
	if origin == "" {
		return
	}

	if !c.isOriginAllowed(origin) {
		wctx.Logger().Tracef("preflight: origin %s not allowed", origin)
		return
	}

	method := r.Header.Get(acRequestMethodHeader)
	if !c.isMethodAllowed(method) {
		wctx.Logger().Tracef("preflight: method %s not allowed for cors", method)
		return
	}

	reqHeadersStr := r.Header.Get(acRequestHeadersHeader)
	if !c.validateHeaders(reqHeadersStr) {
		wctx.Logger().Tracef("preflight: headers '%s' not allowed", reqHeadersStr)
		return
	}

	if c.allOrigins {
		origin = star
	}
	headers.Set(acAllowOriginHeader, origin)
	headers.Set(acAllowMethodHeader, c.methodsStr) // Use pre-computed string

	if reqHeadersStr != "" {
		headers.Set(acAllowHeadersHeader, reqHeadersStr) // Echo back input string
	}

	if c.exposeHeadersStr != "" {
		headers.Set(acExposeHeadersHeader, c.exposeHeadersStr)
	}

	if c.credentials {
		headers.Set(acAllowCredentialsHeader, c.credStr)
	}

	wctx.Response().WriteHeader(http.StatusOK)
}

func (c cors) request(wctx *golly.WebContext) {
	r := wctx.Request()
	headers := wctx.ResponseHeaders()

	origin := r.Header.Get(originHeader)
	if origin == "" {
		return
	}

	if !c.isOriginAllowed(origin) {
		wctx.Logger().Tracef("request: origin %s not allowed", origin)
		return
	}

	// Method check (optional in actual request but good for strictness, but usually OPTIONS handles it)
	// Spec says Origin check is sufficient.
	// But staying consistent with previous implementation.
	if !c.isMethodAllowed(r.Method) {
		wctx.Logger().Tracef("request: method %s not allowed for cors", r.Method)
		return
	}

	if c.allOrigins {
		origin = star
	}

	headers.Set(acAllowOriginHeader, origin)

	if c.exposeHeadersStr != "" {
		headers.Set(acExposeHeadersHeader, c.exposeHeadersStr)
	}

	if c.credentials {
		headers.Set(acAllowCredentialsHeader, c.credStr)
	}
}

func (c *cors) isOriginAllowed(origin string) bool {
	if c.allOrigins {
		return true
	}

	// Linear scan with EqualFold (Zero Alloc)
	for _, o := range c.allowedOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}

	// Check wildcard origins in-place
	for i := range c.worigins {
		if golly.WildcardMatch(c.worigins[i], origin) {
			return true
		}
	}

	return false
}

func (c *cors) isMethodAllowed(method string) bool {
	if len(c.methods) == 0 {
		return false
	}

	if method == http.MethodOptions {
		return true
	}

	// Usually method is short list, linear scan is fast
	// Upper case check
	for _, m := range c.methods {
		if m == method || (len(m) == len(method) && strings.ToUpper(method) == m) {
			return true
		}
	}

	return false
}

// validateHeaders checks if all headers in the comma-separated list are allowed.
// Zero-allocation implementation.
func (c *cors) validateHeaders(list string) bool {
	if c.allHeaders || list == "" {
		return true
	}

	// Iterate over comma-separated headers
	// Logic similar to parseHeaders but validation only

	start := 0
	n := len(list)

	for start < n {
		// Skip leading whitespace
		for start < n && (list[start] == ' ' || list[start] == '\t') {
			start++
		}

		if start >= n {
			break
		}

		// Find end of token
		end := start
		for end < n && list[end] != ',' {
			end++
		}

		// Token is list[start:end]
		// Trim trailing whitespace
		tokenEnd := end
		for tokenEnd > start && (list[tokenEnd-1] == ' ' || list[tokenEnd-1] == '\t') {
			tokenEnd--
		}

		if tokenEnd > start {
			// Check if token allowed
			// We avoid allocating string for token if possible?
			// CanonicalHeaderKey requires string.
			// But allowed headers are usually "Content-Type" etc.
			// Case insensitive check against c.headers?
			// c.headers are canonical.
			// Input might be "content-type".
			// We need to match.
			// If we iterate c.headers and EqualFold?

			header := list[start:tokenEnd] // Zero-copy slicing? No, in Go string slicing is zero-copy (shares backing array).

			allowed := false
			for _, h := range c.headers {
				if strings.EqualFold(h, header) {
					allowed = true
					break
				}
			}
			if !allowed {
				return false
			}
		}

		start = end + 1
	}

	return true
}

func (c *cors) areHeadersAllowed(headers []string) bool {
	// Keep for backward compatibility/testing if needed, or update tests
	// Implementation using slice
	if c.allHeaders || len(headers) == 0 {
		return true
	}

	for _, h := range headers {
		allowed := false
		for _, allowedH := range c.headers {
			if strings.EqualFold(allowedH, h) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return false
}

// Deprecated: Internal use replaced by validateHeaders
func parseHeaders(list string) []string {
	// Keep for tests?
	if list == "" {
		return nil
	}
	// ... (Implementation wrapped to reuse logic or just keep old one for tests?)
	// Actually we can keep old implementation for tests checking correctness of parsing,
	// but production code uses validateHeaders.
	// Reverting to optimized parsing logic if we really need []string return?
	// Benchmarks use this.

	// Optimized version using strings.FieldsFunc logic manually
	var headers []string

	start := 0
	n := len(list)

	for start < n {
		for start < n && (list[start] == ' ' || list[start] == '\t') {
			start++
		}
		if start >= n {
			break
		}

		end := start
		for end < n && list[end] != ',' {
			end++
		}

		tokenEnd := end
		for tokenEnd > start && (list[tokenEnd-1] == ' ' || list[tokenEnd-1] == '\t') {
			tokenEnd--
		}

		if tokenEnd > start {
			headers = append(headers, http.CanonicalHeaderKey(list[start:tokenEnd]))
		}

		start = end + 1
	}
	return headers
}
