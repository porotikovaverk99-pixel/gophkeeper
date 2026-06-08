package gzip

import (
	"compress/gzip"
	"net/http"
	"io"
	"strings"
)

type compressWriter struct {
	w http.ResponseWriter
	zw *gzip.Writer
}

type compressReader struct {
	r io.ReadCloser
	zr *gzip.Reader
}

func NewCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w: w,
		zw: gzip.NewWriter(w),
	}
}

func NewCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{
		r: r,
		zr: zr,
	}, nil
}

func (cw *compressWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	return cw.zw.Write(b)
}

func (cw *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		cw.w.Header().Set("Content-Encoding", "gzip")
	}
	cw.w.WriteHeader(statusCode)
}

func (cw *compressWriter) Close() error {
	return cw.zw.Close()
}

func (cr *compressReader) Read(b []byte) (int, error) {
	return cr.zr.Read(b)
}

func (cr *compressReader) Close() error {
	if err := cr.r.Close(); err != nil {
        return err
    }
    return cr.zr.Close()
}

func GzipMiddleware(h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		ow := w

		if supportsGzip {

			cwPointer := NewCompressWriter(w)
			defer cwPointer.Close()
			ow = cwPointer

		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip {

			crPointer, err := NewCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}

			defer crPointer.Close()

			r.Body = crPointer

		}

		h.ServeHTTP(ow, r)

	})

}