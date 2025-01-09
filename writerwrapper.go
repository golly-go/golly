package golly

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// NewWrapResponseWriter wraps an http.ResponseWriter, returning a proxy that allows you to
// hook into various parts of the response process.
func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) WrapResponseWriter {
	bw := basicWriter{ResponseWriter: w}

	_, isFlusher := w.(http.Flusher)
	switch protoMajor {
	case 2:
		if _, ok := w.(http.Pusher); isFlusher && ok {
			return &http2FancyWriter{bw}
		}
	default:
		if _, ok := w.(http.Hijacker); ok {
			if _, ok := w.(io.ReaderFrom); isFlusher && ok {
				return &httpFancyWriter{bw}
			}
			if isFlusher {
				return &flushHijackWriter{bw}
			}
			return &hijackWriter{bw}
		}
	}

	if isFlusher {
		return &flushWriter{bw}
	}

	return &bw
}

// WrapResponseWriter proxies an http.ResponseWriter, allowing hooks into the response process.
type WrapResponseWriter interface {
	http.ResponseWriter
	Status() int
	BytesWritten() int
	Tee(io.Writer)
	Unwrap() http.ResponseWriter
	Discard()
}

// basicWriter implements the core functionality for WrapResponseWriter.
type basicWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
	tee         io.Writer
	discard     bool
}

func (b *basicWriter) WriteHeader(code int) {
	if b.wroteHeader || (code >= 100 && code <= 199 && code != http.StatusSwitchingProtocols) {
		return
	}

	b.code = code
	b.wroteHeader = true

	if !b.discard {
		b.ResponseWriter.WriteHeader(code)
	}
}

func (b *basicWriter) Write(buf []byte) (int, error) {
	b.maybeWriteHeader()

	if b.discard {
		return io.Discard.Write(buf)
	}

	n, err := b.ResponseWriter.Write(buf)
	if b.tee != nil {
		_, teeErr := b.tee.Write(buf[:n])
		if err == nil {
			err = teeErr
		}
	}

	b.bytes += n
	return n, err
}

func (b *basicWriter) maybeWriteHeader() {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
}

func (b *basicWriter) Status() int                 { return b.code }
func (b *basicWriter) BytesWritten() int           { return b.bytes }
func (b *basicWriter) Tee(w io.Writer)             { b.tee = w }
func (b *basicWriter) Unwrap() http.ResponseWriter { return b.ResponseWriter }
func (b *basicWriter) Discard()                    { b.discard = true }

// flushWriter implements flushing for basicWriter.
type flushWriter struct{ basicWriter }

func (f *flushWriter) Flush() { f.flush() }

// hijackWriter implements hijacking for basicWriter.
type hijackWriter struct{ basicWriter }

func (f *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.hijack()
}

// flushHijackWriter implements both flushing and hijacking.
type flushHijackWriter struct{ basicWriter }

func (f *flushHijackWriter) Flush() { f.flush() }
func (f *flushHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.hijack()
}

// httpFancyWriter satisfies Flusher, Hijacker, and ReaderFrom.
type httpFancyWriter struct{ basicWriter }

func (f *httpFancyWriter) Flush() { f.flush() }
func (f *httpFancyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.hijack()
}
func (f *httpFancyWriter) ReadFrom(r io.Reader) (int64, error) {
	if f.tee != nil {
		n, err := io.Copy(&f.basicWriter, r)
		f.bytes += int(n)
		return n, err
	}
	rf := f.ResponseWriter.(io.ReaderFrom)
	f.maybeWriteHeader()
	n, err := rf.ReadFrom(r)
	f.bytes += int(n)
	return n, err
}

// http2FancyWriter satisfies Flusher and Pusher.
type http2FancyWriter struct{ basicWriter }

func (f *http2FancyWriter) Flush() { f.flush() }
func (f *http2FancyWriter) Push(target string, opts *http.PushOptions) error {
	return f.ResponseWriter.(http.Pusher).Push(target, opts)
}

// Helper methods for flush and hijack.
func (b *basicWriter) flush() {
	if flusher, ok := b.ResponseWriter.(http.Flusher); ok {
		b.wroteHeader = true
		flusher.Flush()
	}
}

func (b *basicWriter) hijack() (net.Conn, *bufio.ReadWriter, error) {
	return b.ResponseWriter.(http.Hijacker).Hijack()
}

var (
	_ http.Flusher  = &flushWriter{}
	_ http.Flusher  = &flushHijackWriter{}
	_ http.Hijacker = &hijackWriter{}
	_ http.Hijacker = &flushHijackWriter{}
	_ http.Flusher  = &httpFancyWriter{}
	_ http.Hijacker = &httpFancyWriter{}
	_ io.ReaderFrom = &httpFancyWriter{}
	_ http.Flusher  = &http2FancyWriter{}
	_ http.Pusher   = &http2FancyWriter{}
)
