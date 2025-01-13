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
	exposeHeaders []string
	methods       []string
	headers       []string
	orgins        map[string]bool

	worigins []string

	allHeaders bool
	allOrigins bool

	credentials bool
}

// Init creates a cors record
func (c CorsOptions) init() cors {
	co := cors{
		exposeHeaders: golly.Convert(c.ExposeHeaders, http.CanonicalHeaderKey),
		credentials:   c.AllowCredentials,
		orgins:        make(map[string]bool),
	}

	if c.AllowAllHeaders {
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
		co.orgins = make(map[string]bool, len(c.AllowedOrigins)) // Pre-allocate map size
		for _, origin := range c.AllowedOrigins {
			if golly.IsWildcardString(origin) {
				co.worigins = append(co.worigins, origin)
			} else {
				co.orgins[strings.ToLower(origin)] = true
			}
		}
	}

	return co
}

// Cors builds a golly middleware providing cors ooptions
func Cors(co CorsOptions) func(next golly.HandlerFunc) golly.HandlerFunc {
	crs := co.init()

	return func(next golly.HandlerFunc) golly.HandlerFunc {
		return func(wctx *golly.WebContext) {
			r := wctx.Request()

			wctx.Logger().Tracef("Starting cors check for %s", r.Method)

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

	headers := wctx.Response().Header()

	headers.Add(Vary, originHeader)
	headers.Add(Vary, acRequestMethodHeader)
	headers.Add(Vary, acRequestHeadersHeader)

	origin := r.Header.Get(originHeader)
	if origin == "" {
		wctx.Logger().Tracef("empty origin in cors check")
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

	rHeaders := parseHeaders(r.Header.Get(acRequestHeadersHeader))
	if !c.areHeadersAllowed(rHeaders) {
		wctx.Logger().Tracef("preflight: headers '%v' not allowed", rHeaders)
		return
	}

	if c.allOrigins {
		origin = star
	}
	headers.Set(acAllowOriginHeader, origin)
	headers.Set(acAllowMethodHeader, method)

	if len(rHeaders) > 0 {
		headers.Set(acAllowHeadersHeader, strings.Join(rHeaders, ","))
	}

	if len(c.exposeHeaders) > 0 {
		headers.Set(acExposeHeadersHeader, strings.Join(c.exposeHeaders, ","))
	}

	if c.credentials {
		headers.Set(acAllowCredentialsHeader, "true")
	}

	wctx.Response().WriteHeader(http.StatusOK)
}

func (c cors) request(wctx *golly.WebContext) {
	r := wctx.Request()
	headers := wctx.Response().Header()

	origin := r.Header.Get(originHeader)
	if origin == "" {
		wctx.Logger().Tracef("request: empty origin in cors check")
		return
	}

	if !c.isOriginAllowed(origin) {
		wctx.Logger().Tracef("request: origin %s not allowed", origin)
		return
	}

	if !c.isMethodAllowed(r.Method) {
		wctx.Logger().Tracef("request: method %s not allowed for cors", r.Method)
		return
	}

	if c.allOrigins {
		origin = star
	}

	headers.Set(acAllowOriginHeader, origin)

	if len(c.exposeHeaders) > 0 {
		headers.Set(acExposeHeadersHeader, strings.Join(c.exposeHeaders, ","))
	}

	if c.credentials {
		headers.Set(acAllowCredentialsHeader, trueStr)
	}
}

func (c *cors) isOriginAllowed(origin string) bool {
	if c.allOrigins {
		return true
	}

	origin = strings.ToLower(origin)

	if _, exists := c.orgins[origin]; exists {
		return true
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

	method = strings.ToUpper(method)

	if method == http.MethodOptions {
		return true
	}

	return golly.Contains(c.methods, method)
}

func (c *cors) areHeadersAllowed(headers []string) bool {
	if c.allHeaders || len(headers) == 0 {
		return true
	}

	for pos := range headers {
		if golly.Contains(c.headers, http.CanonicalHeaderKey(headers[pos])) {
			return true
		}
	}

	return false
}

func parseHeaders(list string) []string {
	if list == "" {
		return nil
	}

	count := strings.Count(list, ",") + 1
	if count == 0 {
		return []string{http.CanonicalHeaderKey(list)}
	}

	headers := make([]string, count)

	i := 0
	start, end := 0, 0

	for i < count {
		next := strings.Index(list, ",")
		if next == -1 {
			break
		}

		start = 0
		for start < next && (list[start] == ' ' || list[start] == '\t') {
			start++
		}

		end = next - 1
		for end > start && (list[end] == ' ' || list[end] == '\t') {
			end--
		}

		if start <= end {
			headers[i] = list[start : end+1]
			i++
		}
		list = list[next+1:]
	}

	// Process the final segment
	start, end = 0, len(list)-1
	for start < len(list) && (list[start] == ' ' || list[start] == '\t') {
		start++
	}

	for end > start && (list[end] == ' ' || list[end] == '\t') {
		end--
	}

	if start <= end {
		headers[i] = list[start : end+1]
		i++
	}

	// wish i could capitlize inplace
	return golly.Convert(headers[:i], http.CanonicalHeaderKey)
}
