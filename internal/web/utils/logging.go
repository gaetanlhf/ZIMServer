package utils

import (
	"log"
	"net/http"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"

	symbolSuccess = "✓"
	symbolWarning = "⚠"
	symbolError   = "✗"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		var logSymbol string
		var statusColor string

		if rw.statusCode >= 500 {
			logSymbol = symbolError
			statusColor = colorRed
		} else if rw.statusCode >= 400 {
			logSymbol = symbolWarning
			statusColor = colorYellow
		} else {
			logSymbol = symbolSuccess
			statusColor = colorGreen
		}

		log.Printf("%s%s%s %s%s%s %s %s%d%s %v",
			statusColor, logSymbol, colorReset,
			colorGray, r.Method, colorReset,
			r.URL.Path,
			statusColor, rw.statusCode, colorReset,
			duration.Round(time.Millisecond),
		)
	})
}
