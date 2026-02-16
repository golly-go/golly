package golly

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync"
)

// UniversalResponseWriter implements ALL interfaces and checks capability at runtime.
// This allows a single struct type to be pooled and reused for any response writer.
type UniversalResponseWriter struct {
	basicWriter
}

// writerPool recycles UniversalResponseWriter instances.
var writerPool = sync.Pool{
	New: func() any {
		return &UniversalResponseWriter{}
	},
}

// NewWrapResponseWriter wraps an http.ResponseWriter.
// Returns a pooled instance if possible.
//
// Note: We ignore protoMajor for interface selection now as we implement all.
// The runtime checks in Flush/Hijack/Push/ReadFrom handle capability.
func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) WrapResponseWriter {
	wr := writerPool.Get().(*UniversalResponseWriter)
	wr.Reset(w)
	return wr
}

// FreeWrapResponseWriter returns a WrapResponseWriter to the pool if it's a UniversalResponseWriter.
func FreeWrapResponseWriter(w WrapResponseWriter) {
	if u, ok := w.(*UniversalResponseWriter); ok {
		writerPool.Put(u)
	}
}

// WrapResponseWriter proxies an http.ResponseWriter, allowing hooks into the response process.
type WrapResponseWriter interface {
	http.ResponseWriter
	Status() int
	BytesWritten() int
	Tee(io.Writer)
	Unwrap() http.ResponseWriter
	Discard()
	Reset(w http.ResponseWriter)
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
func (b *basicWriter) Reset(w http.ResponseWriter) {
	b.ResponseWriter = w
	b.wroteHeader = false
	b.code = 0
	b.bytes = 0
	b.tee = nil
	b.discard = false
}

// Flush implements http.Flusher
func (u *UniversalResponseWriter) Flush() {
	if f, ok := u.ResponseWriter.(http.Flusher); ok {
		u.wroteHeader = true
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (u *UniversalResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := u.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher
func (u *UniversalResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := u.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// ReadFrom implements io.ReaderFrom
func (u *UniversalResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if u.tee != nil {
		n, err := io.Copy(&u.basicWriter, r)
		u.bytes += int(n)
		return n, err
	}
	if rf, ok := u.ResponseWriter.(io.ReaderFrom); ok {
		u.maybeWriteHeader()
		n, err := rf.ReadFrom(r)
		u.bytes += int(n)
		return n, err
	}
	return io.Copy(&u.basicWriter, r)
}

// Compile time checks
var (
	_ http.Flusher       = &UniversalResponseWriter{}
	_ http.Hijacker      = &UniversalResponseWriter{}
	_ http.Pusher        = &UniversalResponseWriter{}
	_ io.ReaderFrom      = &UniversalResponseWriter{}
	_ WrapResponseWriter = &UniversalResponseWriter{}
)
