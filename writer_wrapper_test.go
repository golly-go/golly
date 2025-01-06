package golly

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Custom ResponseWriter to support Hijack for testing
type HijackableResponseWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (h *HijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true

	// Use an io.Pipe to avoid type assertion issues
	reader, writer := io.Pipe()
	readWriter := bufio.NewReadWriter(
		bufio.NewReader(reader),
		bufio.NewWriter(writer),
	)

	return nil, readWriter, nil
}

func (h *HijackableResponseWriter) Hijacked() bool {
	return h.hijacked
}

// Custom ResponseWriter that implements io.ReaderFrom for testing
type ReaderFromResponseWriter struct {
	http.ResponseWriter
}

func (r *ReaderFromResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(r.ResponseWriter, reader)
}

// TestBasicWriter ensures the basicWriter correctly records status and bytes written.
func TestBasicWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	bw := basicWriter{ResponseWriter: rec}

	bw.WriteHeader(http.StatusCreated)
	if bw.Status() != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, bw.Status())
	}

	bw.Write([]byte("Hello"))
	if bw.BytesWritten() != 5 {
		t.Errorf("expected 5 bytes written, got %d", bw.BytesWritten())
	}

	if rec.Body.String() != "Hello" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

// TestTeeWriter verifies tee works by writing to an additional buffer.
func TestTeeWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	bw := basicWriter{ResponseWriter: rec}

	var buf bytes.Buffer
	bw.Tee(&buf)

	bw.Write([]byte("TeeTest"))
	if buf.String() != "TeeTest" {
		t.Errorf("expected tee buffer to contain 'TeeTest', got %s", buf.String())
	}

	if rec.Body.String() != "TeeTest" {
		t.Errorf("expected response body to contain 'TeeTest', got %s", rec.Body.String())
	}
}

// TestDiscard ensures the discard behavior prevents writes to the original ResponseWriter.
func TestDiscard(t *testing.T) {
	rec := httptest.NewRecorder()
	bw := basicWriter{ResponseWriter: rec}
	bw.Discard()

	bw.Write([]byte("Discarded"))
	if rec.Body.String() != "" {
		t.Errorf("expected no response body, got %s", rec.Body.String())
	}
}

// TestFlushWriter verifies flushing behavior.
func TestFlushWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	fw := flushWriter{basicWriter{ResponseWriter: rec}}

	fw.Flush()
	if !rec.Flushed {
		t.Errorf("expected recorder to be flushed")
	}
}

// TestHttp2FancyWriter ensures http2FancyWriter correctly flushes and pushes.
func TestHttp2FancyWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	h2w := http2FancyWriter{basicWriter{ResponseWriter: rec}}

	h2w.Flush()
	if !rec.Flushed {
		t.Errorf("expected recorder to be flushed")
	}
}

// TestHttpFancyWriter verifies the full capabilities of httpFancyWriter.
func TestHttpFancyWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	fw := httpFancyWriter{basicWriter{ResponseWriter: rec}}

	fw.Flush()
	if !rec.Flushed {
		t.Errorf("expected recorder to be flushed")
	}

	_, err := fw.Write([]byte("FancyWriter"))
	if err != nil {
		t.Errorf("unexpected error during write: %v", err)
	}
	if rec.Body.String() != "FancyWriter" {
		t.Errorf("expected body to contain 'FancyWriter', got %s", rec.Body.String())
	}
}

func TestHijackWriter(t *testing.T) {
	hijackableRec := &HijackableResponseWriter{ResponseWriter: httptest.NewRecorder()}
	hw := hijackWriter{basicWriter{ResponseWriter: hijackableRec}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, err := hw.Hijack()
		if err != nil {
			t.Errorf("unexpected error hijacking: %v", err)
		}
		if !hijackableRec.Hijacked() {
			t.Errorf("expected connection to be hijacked")
		}
	}))
	defer srv.Close()

	_, err := http.Get(srv.URL)
	if err != nil {
		t.Errorf("failed to send request to test server: %v", err)
	}
}

// TestNewWrapResponseWriter verifies the wrapping behavior of NewWrapResponseWriter.
func TestNewWrapResponseWriter(t *testing.T) {
	cases := []struct {
		protoMajor     int
		responseWriter http.ResponseWriter
		expectedType   string
	}{
		{1, httptest.NewRecorder(), "*flushWriter"},
		{2, httptest.NewRecorder(), "*http2FancyWriter"},
		{1, &HijackableResponseWriter{ResponseWriter: httptest.NewRecorder()}, "*flushHijackWriter"},
		{1, &HijackableResponseWriter{ResponseWriter: httptest.NewRecorder()}, "*hijackWriter"},
		{1, httptest.NewRecorder(), "*basicWriter"},
	}

	for _, tc := range cases {
		wr := NewWrapResponseWriter(tc.responseWriter, tc.protoMajor)
		wr.WriteHeader(http.StatusAccepted)

		if wr.Status() != http.StatusAccepted {
			t.Errorf("expected status %d, got %d", http.StatusAccepted, wr.Status())
		}
	}
}

// Benchmark for basicWriter.
func BenchmarkBasicWriter(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"basic writer"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				bw := basicWriter{ResponseWriter: rec}
				bw.WriteHeader(http.StatusOK)
				bw.Write([]byte("Benchmark"))
			}
		})
	}
}

// Benchmark for flushWriter.
func BenchmarkFlushWriter(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"flush writer"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				fw := flushWriter{basicWriter{ResponseWriter: rec}}
				fw.WriteHeader(http.StatusOK)
				fw.Flush()
			}
		})
	}
}

// Benchmark for hijackWriter.
func BenchmarkHijackWriter(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"hijack writer"},
	}

	for _, bm := range benchmarks {
		hijackableRec := &HijackableResponseWriter{ResponseWriter: httptest.NewRecorder()}
		hw := hijackWriter{basicWriter{ResponseWriter: hijackableRec}}

		b.Run(bm.name, func(b *testing.B) {

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = hw.Hijack()
			}
		})
	}
}

// Benchmark for httpFancyWriter.
func BenchmarkHttpFancyWriter(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"http fancy writer"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				fw := httpFancyWriter{basicWriter{ResponseWriter: rec}}
				fw.WriteHeader(http.StatusOK)
				fw.Flush()
			}
		})
	}
}

// Benchmark for http2FancyWriter.
func BenchmarkHttp2FancyWriter(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"http2 fancy writer"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				h2w := http2FancyWriter{basicWriter{ResponseWriter: rec}}
				h2w.WriteHeader(http.StatusOK)
				h2w.Flush()
			}
		})
	}
}

// Benchmark for NewWrapResponseWriter.
func BenchmarkNewWrapResponseWriter(b *testing.B) {
	benchmarks := []struct {
		name       string
		protoMajor int
	}{
		{"wrap response writer HTTP/1.1", 1},
		{"wrap response writer HTTP/2", 2},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				_ = NewWrapResponseWriter(rec, bm.protoMajor)
			}
		})
	}
}

func BenchmarkHttpFancyWriterReadFrom(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{"http fancy writer ReadFrom"},
	}

	for _, bm := range benchmarks {
		rec := &ReaderFromResponseWriter{ResponseWriter: httptest.NewRecorder()}
		fw := httpFancyWriter{basicWriter{ResponseWriter: rec}}
		data := bytes.NewReader([]byte("Benchmark data for ReadFrom method"))

		b.Run(bm.name, func(b *testing.B) {

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = fw.ReadFrom(data)
				data.Seek(0, io.SeekStart) // Reset the reader for the next iteration
			}
		})
	}
}
