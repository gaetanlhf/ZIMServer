package utils

import (
	"log"
	"net/http"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGray   = "\033[90m"

	SymbolSuccess  = "✓"
	SymbolWarning  = "⚠"
	SymbolError    = "✗"
	SymbolRedirect = "➜"
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
			logSymbol = SymbolError
			statusColor = ColorRed
		} else if rw.statusCode >= 400 {
			logSymbol = SymbolWarning
			statusColor = ColorYellow
		} else if rw.statusCode >= 300 && rw.statusCode < 400 {
			logSymbol = SymbolRedirect
			statusColor = ColorBlue
		} else {
			logSymbol = SymbolSuccess
			statusColor = ColorGreen
		}

		log.Printf("%s%s%s %s%s%s %s %s%d%s %v",
			statusColor, logSymbol, ColorReset,
			ColorGray, r.Method, ColorReset,
			r.URL.Path,
			statusColor, rw.statusCode, ColorReset,
			duration.Round(time.Millisecond),
		)
	})
}
