package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// gzipMinBytes leaves small bodies alone: below roughly one packet, compression
// costs more than it saves.
const gzipMinBytes = 1400

// gzipPool reuses the compressor's window across requests.
var gzipPool = sync.Pool{New: func() any { return gzip.NewWriter(io.Discard) }}

// compressibleTypes are the media types worth gzipping. Everything else the
// server sends (agent binaries, PNG and WebP icons, woff2 fonts) is already
// compressed, so re-compressing it only burns CPU.
var compressibleTypes = []string{
	"application/json",
	"application/javascript",
	"application/manifest+json",
	"text/",
	"image/svg+xml",
}

func compressible(contentType string) bool {
	for _, t := range compressibleTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

// gzipResponses compresses text responses for clients that accept gzip. Metric
// series and the SPA bundle are the payloads that dominate this server's
// bandwidth, and both are highly compressible JSON and JavaScript.
func gzipResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Accept-Encoding")
		cw := &compressWriter{ResponseWriter: w}
		defer cw.close()
		next.ServeHTTP(cw, r)
	})
}

// compressWriter decides at header-write time whether the response is worth
// compressing, since the content type is only known then.
type compressWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (c *compressWriter) WriteHeader(code int) {
	if c.wroteHeader {
		c.ResponseWriter.WriteHeader(code)
		return
	}
	c.wroteHeader = true
	if c.shouldCompress(code) {
		h := c.Header()
		h.Set("Content-Encoding", "gzip")
		h.Del("Content-Length")
		c.gz = gzipPool.Get().(*gzip.Writer)
		c.gz.Reset(c.ResponseWriter)
	}
	c.ResponseWriter.WriteHeader(code)
}

// shouldCompress skips bodyless statuses, partial content, already-encoded
// responses, media that does not shrink, and bodies too small to be worth a
// gzip frame.
func (c *compressWriter) shouldCompress(code int) bool {
	if code == http.StatusNoContent || code == http.StatusNotModified {
		return false
	}
	h := c.Header()
	// A range response describes a byte window of the identity encoding, so
	// gzipping it would leave Content-Range describing bytes that are not
	// what was sent. http.ServeFile answers ranges for the SPA bundle.
	if code == http.StatusPartialContent || h.Get("Content-Range") != "" {
		return false
	}
	if h.Get("Content-Encoding") != "" || !compressible(h.Get("Content-Type")) {
		return false
	}
	if n, err := strconv.Atoi(h.Get("Content-Length")); err == nil && n < gzipMinBytes {
		return false
	}
	return true
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.gz != nil {
		return c.gz.Write(p)
	}
	return c.ResponseWriter.Write(p)
}

func (c *compressWriter) Flush() {
	if c.gz != nil {
		c.gz.Flush()
	}
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *compressWriter) close() {
	if c.gz == nil {
		return
	}
	c.gz.Close()
	c.gz.Reset(io.Discard)
	gzipPool.Put(c.gz)
	c.gz = nil
}
