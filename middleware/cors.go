package middleware

import (
	"net/http"
	"strings"

	"github.com/slimloans/golly"
	"github.com/slimloans/golly/utils"
)

const (
	toLower     = 'a' - 'A'
	wildcardSym = "*"
)

var (
	defaultHeaders = []string{"Origin", "Accept", "Content-Type"}
	defaultMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead}
)

// CorsOptions defines the cors options
type CorsOptions struct {
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
	worigins      utils.Wildcards
	orgins        []string

	allHeaders bool
	allOrigins bool

	credentials bool
}

// Init creates a cors record
func (c CorsOptions) init() cors {
	co := cors{
		exposeHeaders: utils.Convert(c.ExposeHeaders, http.CanonicalHeaderKey),
		credentials:   c.AllowCredentials,
	}

	if len(c.AllowedHeaders) == 0 {
		co.headers = defaultHeaders
	} else {
		co.headers = utils.Convert(append(c.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, header := range co.headers {
			if header == wildcardSym {
				co.allHeaders = true
				break
			}
		}
	}

	if len(c.AllowedMethods) == 0 {
		co.methods = defaultMethods
	} else {
		co.methods = utils.Convert(c.AllowedMethods, strings.ToUpper)
	}

	if len(c.AllowedOrigins) == 0 {
		co.allOrigins = true
	} else {
		// To slow
		// orgins := utils.Convert(c.AllowedOrigins, strings.ToLower)
		for _, origin := range c.AllowedOrigins {
			if origin == wildcardSym {
				co.allOrigins = true
				break
			}

			origin = strings.ToLower(origin)
			if w := utils.NewWildcard(origin); w != nil {
				co.worigins = append(co.worigins, *w)
			} else {
				co.orgins = append(co.orgins, origin)
			}
		}

	}
	return co
}

// Cors builds a golly middleware providing cors ooptions
func Cors(co CorsOptions) func(next golly.HandlerFunc) golly.HandlerFunc {
	crs := co.init()

	return func(next golly.HandlerFunc) golly.HandlerFunc {
		return func(wctx golly.WebContext) {
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

func (c cors) preflight(wctx golly.WebContext) {
	r := wctx.Request()

	headers := wctx.Response().Header()
	origin := r.Header.Get("Origin")

	if r.Method != http.MethodOptions {
		return
	}

	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")

	if origin == "" {
		wctx.Logger().Debug("empty origin in cors check")
		return
	}

	if !c.isOriginAllowed(origin) {
		wctx.Logger().Debugf("preflight: origin not %s allowed", origin)
		return
	}

	method := r.Header.Get("Access-Control-Request-Method")

	if !c.isMethodAllowed(method) {
		wctx.Logger().Debugf("preflight: method %s not allowed for cors", method)
		return
	}

	rHeaders := parseHeaders(r.Header.Get("Access-Control-Request-Headers"))
	if !c.areHeadersAllowed(rHeaders) {
		wctx.Logger().Debugf("preflight: headers '%v' not allowed", rHeaders)
		return
	}

	if c.allOrigins {
		origin = wildcardSym
	}

	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(method))

	if len(rHeaders) > 0 {
		headers.Set("Access-Control-Allow-Headers", strings.Join(rHeaders, ", "))
	}

	if len(c.exposeHeaders) != 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.exposeHeaders, ", "))
	}

	if c.credentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	wctx.RenderStatus(http.StatusOK)
}

// parseHeaders this is probably not the fast impl will refactor later
func parseHeaders(headerList string) []string {
	return strings.Split(strings.ToUpper(headerList), ",")
}

func (c cors) request(wctx golly.WebContext) {
	r := wctx.Request()

	headers := wctx.Response().Header()
	origin := r.Header.Get("Origin")

	if origin == "" {
		wctx.Logger().Debug("request: empty origin in cors check")
		return
	}

	if !c.isOriginAllowed(origin) {
		wctx.Logger().Debugf("request: origin not %s allowed", origin)
		return
	}

	if !c.isMethodAllowed(r.Method) {
		wctx.Logger().Debugf("request: method %s not allowed for cors", r.Method)
		return
	}

	if c.allOrigins {
		origin = wildcardSym
	}

	headers.Set("Access-Control-Allow-Origin", origin)

	if len(c.exposeHeaders) != 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.exposeHeaders, ", "))
	}

	if c.credentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
}

func (c *cors) isOriginAllowed(origin string) bool {
	if c.allOrigins {
		return true
	}

	origin = strings.ToLower(origin)

	if utils.StringSliceContains(c.orgins, origin) {
		return true
	}

	if w := c.worigins.Find(func(w utils.Wildcard) bool { return w.Match(origin) }); w != nil {
		return true
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

	return utils.StringSliceContains(c.methods, method)
}

func (c *cors) areHeadersAllowed(headers []string) bool {
	if c.allHeaders || len(headers) == 0 {
		return true
	}

	for _, header := range headers {
		if utils.StringSliceContains(c.headers, http.CanonicalHeaderKey(header)) {
			return true
		}
	}

	return false
}
