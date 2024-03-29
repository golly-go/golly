package golly

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/golly-go/golly/errors"
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

	ErrorInvalidType = fmt.Errorf("invalid resposne type provided")

	DoubleRenderError = errors.Error{Key: "ERROR.DOUBLE_RENDER_ERROR", Status: 500}

	marshalers = map[FormatOption]Marshaler{
		FormatTypeJSON: {json.Marshal, "application/json"},
		FormatTypeXML:  {xml.Marshal, "application/xml"},
		FormatTypeData: {marshalData, ""},
		FormatTypeText: {marshalText, "text/none"},
		FormatTypeHTML: {marshalText, "text/html; charset=UTF-8"},
	}
)

func RegisterMarshaler(tpe FormatOption, marshal marshalFunc, contentType string) {
	marshalers[tpe] = Marshaler{marshal, contentType}
}

// RenderOptions Holds render options for mutiple format
type RenderOptions struct {
	Format FormatOption
}

func RenderExt(wctx WebContext, resp interface{}, err error, format FormatOption) {
	if err != nil {
		resp = errors.WrapGeneric(err)
	}

	wctx.Render(resp, RenderOptions{Format: format})
}

func Render(wctx WebContext, resp interface{}) {
	format := wctx.format
	if format == 0 {
		format = FormatTypeJSON
	}

	var err error
	if e, ok := resp.(error); ok {
		err = e
		resp = nil
	}

	RenderExt(wctx, resp, err, format)
}

func (wctx WebContext) Render(resp interface{}, options RenderOptions) {
	if wctx.rendered {
		panic(DoubleRenderError.NewError(fmt.Errorf("render called twice")))
	}

	if marshaler, found := marshalers[options.Format]; found {
		RenderResponse(wctx, marshaler, resp)
	} else {
		wctx.Logger().Errorf("invalid format provided (format: %v)", options.Format)
		wctx.Response().WriteHeader(http.StatusInternalServerError)
	}
}

// func (wctx WebContext) RenderAttachment(name string, data []byte) {
// 	wctx.Response().Header().Set("")
// }

func (wctx WebContext) RenderJSON(resp interface{}) {
	wctx.Render(resp, RenderOptions{Format: FormatTypeJSON})
}

func (wctx WebContext) RenderXML(resp interface{}) {
	wctx.Render(resp, RenderOptions{Format: FormatTypeXML})
}

func (wctx WebContext) RenderData(data []byte) {
	wctx.Render(data, RenderOptions{Format: FormatTypeData})
}

func (wctx WebContext) RenderText(data string) {
	wctx.Render(data, RenderOptions{Format: FormatTypeText})
}

func (wctx WebContext) RenderHTML(data string) {
	wctx.Render(data, RenderOptions{Format: FormatTypeHTML})
}

func marshalData(v interface{}) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	return []byte{}, ErrorInvalidType
}

func marshalText(v interface{}) ([]byte, error) {
	if r, ok := v.(string); ok {
		return []byte(r), nil
	} else if b, ok := v.([]byte); ok {
		return b, nil
	} else {
		return []byte{}, ErrorInvalidType
	}
}

// TODO Support Streaming

// RenderResponse this should make it so we can deprecate the WrapRequest and maybe we will do someting
// like ctx := app.UseContext(r)
func RenderResponse(wctx WebContext, marshal Marshaler, res interface{}) {
	status := http.StatusOK
	var b []byte

	l := wctx.Logger()

	// If the response is an error
	if err, ok := res.(error); ok {
		status = http.StatusInternalServerError

		if ae, ok := err.(errors.Error); ok {
			l = l.WithFields(ae.ToLogFields())

			if ae.Status != 0 {
				status = ae.Status
			}

			res = err

			if status != http.StatusUnauthorized && status != http.StatusNotFound {
				l.Error(err)
			} else {
				l.Warn(err)
			}
		}
	}

	if status == 0 {
		status = http.StatusOK
	}

	if wctx.Request().Method == "HEAD" {
		wctx.Response().WriteHeader(status)
		return
	}

	if d, ok := res.([]byte); ok {
		b = d
	} else {
		if d, mErr := marshal.Handler(res); mErr == nil {
			b = d
		} else {
			wctx.Response().WriteHeader(http.StatusInternalServerError)
			wctx.Logger().Error(mErr.Error())
			return
		}
	}

	ct := marshal.ContentType
	if ct == "" {
		ct = http.DetectContentType(b)
	}

	wctx.AddHeader("Content-Type", ct)
	wctx.Response().WriteHeader(status)

	wctx.Write(b)
}
