package middleware

import (
	"html"
	"net/http"
	"net/url"
)

// Sanitize middleware automatically escapes all form inputs to prevent XSS attacks
func Sanitize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only sanitize POST/PUT requests with form data
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			
			// Sanitize all form values
			sanitized := make(url.Values)
			for key, values := range r.Form {
				for _, value := range values {
					// Escape HTML special characters
					sanitized.Add(key, html.EscapeString(value))
				}
			}
			
			// Replace form with sanitized version
			r.Form = sanitized
			r.PostForm = sanitized
		}
		
		next(w, r)
	}
}
