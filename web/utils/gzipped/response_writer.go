package gzipped

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipFSPool = sync.Pool{
	New: func() any {
		gzw, err := gzip.NewWriterLevel(io.Discard, flate.BestCompression)
		if err != nil {
			return err
		}

		return gzw
	},
}

// http.ResponseWriter wrapped by gzip
type GzipResponseWriter struct {
	Rw http.ResponseWriter
	Gw *gzip.Writer
}

func NewGzipResponseWriter(rw http.ResponseWriter) *GzipResponseWriter {
	grw := &GzipResponseWriter{
		Rw: rw,
		Gw: gzip.NewWriter(rw),
	}
	return grw
}

func (grw *GzipResponseWriter) Header() http.Header {
	return grw.Rw.Header()
}

func (grw *GzipResponseWriter) Write(b []byte) (int, error) {
	return grw.Gw.Write(b)
}

func (grw *GzipResponseWriter) WriteHeader(statusCode int) {
	grw.Rw.WriteHeader(statusCode)
}

// optional, just to make sure it actually write
func (grw *GzipResponseWriter) Flush() {
	grw.Gw.Flush()
	grw.Gw.Close()
}

// middleware for file server
func GzipFileServer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzw, ok := gzipFSPool.Get().(*gzip.Writer)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		defer gzipFSPool.Put(gzw)
		defer gzw.Flush()

		gzw.Reset(w)
		w.Header().Add("Content-Encoding", "gzip")
		next.ServeHTTP(&GzipResponseWriter{Rw: w, Gw: gzw}, r)
	})
}
