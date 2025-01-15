package golly

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type FormatOption uint

type marshalFunc func(interface{}) ([]byte, error)

type Marshaler struct {
	Handler     marshalFunc
	ContentType string
}

var (
	FormatTypeJSON       FormatOption = 0x0001
	FormatTypeXML        FormatOption = 0x0002
	FormatTypeText       FormatOption = 0x0004
	FormatTypeData       FormatOption = 0x0008
	FormatTypeAttachment FormatOption = 0x0010
	FormatTypeHTML       FormatOption = 0x0020

	ErrorInvalidType = fmt.Errorf("invalid data type provided")

	marshalers = map[FormatOption]Marshaler{
		FormatTypeJSON: {json.Marshal, "application/json"},
		FormatTypeXML:  {xml.Marshal, "application/xml"},
		FormatTypeData: {marshalContent, ""},
		FormatTypeText: {marshalContent, "text/none"},
		FormatTypeHTML: {marshalContent, "text/html; charset=UTF-8"},
	}
)

func RegisterMarshaler(tpe FormatOption, marshal marshalFunc, contentType string) {
	marshalers[tpe] = Marshaler{marshal, contentType}
}

func Render(wctx *WebContext, format FormatOption, res interface{}) {
	// Default format
	if format == 0 {
		format = FormatTypeJSON
	}

	resp := wctx.Response()

	// Handle HEAD requests early
	if wctx.Request().Method == http.MethodHead {
		resp.WriteHeader(http.StatusOK)
		return
	}

	marshal, hasMarshal := marshalers[format]

	// Prepare response body and status
	status := http.StatusOK
	var (
		b  []byte
		ct string
	)

	switch v := res.(type) {
	case []byte:
		b = v
	case string:
		b = unsafeBytes(v) // Avoid allocations
	case error:
		b = unsafeBytes(v.Error()) // Avoid allocations
		status = http.StatusInternalServerError
	default:
		// Marshal the response
		var err error
		if hasMarshal {
			b, err = marshal.Handler(v)
		} else {
			b, err = marshalContent(v)
		}

		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			wctx.Logger().Errorf("Marshaling error: %v", err)
			return
		}
	}

	// Determine content type
	if hasMarshal && marshal.ContentType != "" {
		ct = marshal.ContentType
	} else {
		ct = http.DetectContentType(b)
	}

	// Set headers and write response
	h := resp.Header()
	h.Set("Content-Type", ct)
	resp.WriteHeader(status)

	// Write response body
	if _, err := resp.Write(b); err != nil {
		wctx.Logger().Errorf("Error writing response: %v", err)
	}
}

// MarshalContent marshals data into a []byte, supporting common Go buffer types.
func marshalContent(v interface{}) ([]byte, error) {
	switch r := v.(type) {
	case []byte:
		return r, nil
	case string:
		return unsafeBytes(r), nil
	case *strings.Builder:
		return unsafeBytes(r.String()), nil
	case *bytes.Buffer:
		return r.Bytes(), nil
	case bytes.Buffer:
		return r.Bytes(), nil
	default:
		return nil, ErrorInvalidType
	}
}
