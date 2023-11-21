package main

import (
	"context"
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status       int
	wroteHeader  bool
	bytesWritten int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) Write(b []byte) (n int, err error) {
	n, err = rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true

	return
}

// LoggingMiddleware logs the incoming HTTP request & its duration.
func LoggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Error("Recovered from panic")
				}
			}()

			// Inject a function to allow a request handler to pass the request error to this logger
			var err error
			ctx := context.WithValue(r.Context(), "_set_error_func", func(e error) {
				err = e
			})

			// Record response
			responseRecorder := wrapResponseWriter(w)

			// Perform request with timing
			start := time.Now()
			next.ServeHTTP(responseRecorder, r.WithContext(ctx))
			duration := time.Since(start)

			l := logger.With(
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("method", r.Method),
				zap.String("path", r.URL.EscapedPath()),
				zap.Int64("content_length", r.ContentLength),
				zap.Int("status", responseRecorder.status),
				zap.Duration("duration", duration.Round(time.Millisecond)),
				zap.Int("response_length", responseRecorder.bytesWritten),
			)

			if err != nil {
				l.Errorw("HTTP Request", zap.Error(err))
			} else {
				l.Info("HTTP Request")
			}
		}

		return http.HandlerFunc(fn)
	}
}
