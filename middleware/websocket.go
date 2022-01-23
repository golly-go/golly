package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/slimloans/golly"
)

type wsClient struct {
	c net.Conn
}

type Websocket struct {
	HandshakeTimeout time.Duration

	OnConnect func()
	OnMessage func()
	OnError   func()
}

func NewWebsocket() Websocket {
	return Websocket{}
}

func (ws Websocket) hijacker(next golly.HandlerFunc) golly.HandlerFunc {
	return func(c golly.WebContext) {

		if !golly.HeaderTokenContains(c.Request().Header, "connection", "upgrade") {
			c.Logger().Warn("ws: client upgrade token not found in connection header")
			c.RenderStatus(http.StatusBadRequest)
			return
		}

		if !golly.HeaderTokenContains(c.Request().Header, "upgrade", "websocket") {
			c.Logger().Warn("ws: client websocket token not found in upgrade header")
			c.RenderStatus(http.StatusBadRequest)
			return
		}

		h, ok := c.Response().(http.Hijacker)
		if !ok {

		}

		var brw *bufio.ReadWriter

		conn, brw, _ := h.Hijack()
		if brw.Reader.Buffered() > 0 {
			conn.Close()
			c.Logger().Warn("ws: client sent data before handshake ended")
			return
		}
		// var br *bufio.Reader = brw.Reader

		// bw := brw.Writer
		// if
	}
}

func (ws Websocket) Routes(r *golly.Route) {
	r.Use(ws.hijacker)
}

// bufioReaderSize size returns the size of a bufio.Reader.
func bufioReaderSize(originalReader io.Reader, br *bufio.Reader) int {
	br.Reset(originalReader)
	return br.Size()
}

type writeHook struct {
	p []byte
}

func (wh *writeHook) Write(p []byte) (int, error) {
	wh.p = p
	return len(p), nil
}

// bufioWriterBuffer grabs the buffer from a bufio.Writer.
func bufioWriterBuffer(originalWriter io.Writer, bw *bufio.Writer) []byte {
	var wh writeHook
	bw.Reset(&wh)
	bw.WriteByte(0)
	bw.Flush()

	bw.Reset(originalWriter)

	return wh.p[:cap(wh.p)]
}

// Type assertion
var _ = golly.Controller(Websocket{})
