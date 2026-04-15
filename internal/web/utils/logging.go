package utils

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	ColorReset  = ""
	ColorRed    = ""
	ColorGreen  = ""
	ColorYellow = ""
	ColorBlue   = ""
	ColorGray   = ""

	SymbolSuccess  = "✓"
	SymbolWarning  = "⚠"
	SymbolError    = "✗"
	SymbolRedirect = "➜"
)

var (
	colorReset  = ""
	colorRed    = ""
	colorGreen  = ""
	colorYellow = ""
	colorBlue   = ""
	colorGray   = ""
)

func init() {
	if isTerminal(os.Stderr) {
		colorReset = "\033[0m"
		colorRed = "\033[31m"
		colorGreen = "\033[32m"
		colorYellow = "\033[33m"
		colorBlue = "\033[34m"
		colorGray = "\033[90m"
	}
	startLogWriter()
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

const logChannelSize = 1024

var (
	logCh   chan string
	logOnce sync.Once
)

func startLogWriter() {
	logOnce.Do(func() {
		logCh = make(chan string, logChannelSize)
		go func() {
			for line := range logCh {
				fmt.Fprintln(os.Stderr, line)
			}
		}()
	})
}

func writeLog(line string) {
	select {
	case logCh <- line:
	default:
		fmt.Fprintln(os.Stderr, line)
	}
}

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

		duration := time.Since(start).Round(time.Millisecond)

		var logSymbol, statusColor string
		switch {
		case rw.statusCode >= 500:
			logSymbol, statusColor = SymbolError, colorRed
		case rw.statusCode >= 400:
			logSymbol, statusColor = SymbolWarning, colorYellow
		case rw.statusCode >= 300:
			logSymbol, statusColor = SymbolRedirect, colorBlue
		default:
			logSymbol, statusColor = SymbolSuccess, colorGreen
		}

		writeLog(fmt.Sprintf("%s%s%s %s %s%d%s %v %s%s%s",
			statusColor, logSymbol, colorReset,
			r.Method,
			statusColor, rw.statusCode, colorReset,
			duration,
			colorGray, r.URL.Path, colorReset,
		))
	})
}
