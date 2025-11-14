package middleware

import (
	"net/http"
	"time"

	"forum/server/utils"
)

// responseRecorder wraps http.ResponseWriter to capture status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Logging middleware logs HTTP requests with structured logging
func Logging(logger *utils.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap response writer to capture status code
			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // default
			}
			
			// Call the next handler
			next(rec, r)
			
			// Log after request is handled
			duration := time.Since(start)
			logger.HTTPLog(
				r.Method,
				r.URL.Path,
				getClientIP(r),
				rec.statusCode,
				duration,
			)
		}
	}
}

// Recovery middleware catches panics and logs them
func Recovery(logger *utils.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next(w, r)
		}
	}
}
