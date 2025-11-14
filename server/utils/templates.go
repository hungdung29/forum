package utils

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"text/template"

	"forum/server/config"
	"forum/server/models"
)

// Template cache - parse once, reuse forever
var (
	templateCache = make(map[string]*template.Template)
	cacheMutex    sync.RWMutex
)

type GlobalData struct {
	IsAuthenticated bool
	Data            any
	UserName        string
	Categories      []models.Category
}

type Error struct {
	Code    int
	Message string
	Details string
}

// RenderError handles error responses
func RenderError(db *sql.DB, w http.ResponseWriter, r *http.Request, statusCode int, isauth bool, username string) {
	typeError := Error{
		Code:    statusCode,
		Message: http.StatusText(statusCode),
	}
	if err := RenderTemplate(db, w, r, "error", statusCode, typeError, isauth, username); err != nil {
		http.Error(w, "500 | Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
	}
}

func ParseTemplates(tmpl string) (*template.Template, error) {
	// Parse the template files
	t, err := template.ParseFiles(
		config.BasePath+"web/templates/partials/header.html",
		config.BasePath+"web/templates/partials/footer.html",
		config.BasePath+"web/templates/partials/navbar.html",
		config.BasePath+"web/templates/"+tmpl+".html",
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing template files: %w", err)
	}
	return t, nil
}

func RenderTemplate(db *sql.DB, w http.ResponseWriter, r *http.Request, tmpl string, statusCode int, data any, isauth bool, username string) error {
	// Try to get cached template first
	cacheMutex.RLock()
	t, exists := templateCache[tmpl]
	cacheMutex.RUnlock()
	
	// If not cached, parse and cache it
	if !exists {
		cacheMutex.Lock()
		// Double-check after acquiring write lock
		t, exists = templateCache[tmpl]
		if !exists {
			var err error
			t, err = ParseTemplates(tmpl)
			if err != nil {
				cacheMutex.Unlock()
				return err
			}
			templateCache[tmpl] = t
		}
		cacheMutex.Unlock()
	}
	
	categories, err := models.FetchCategories(db)
	if err != nil {
		categories = nil
	}

	globalData := GlobalData{
		IsAuthenticated: isauth,
		Data:            data,
		UserName:        username,
		Categories:      categories,
	}
	w.WriteHeader(statusCode)
	// Execute the template with the provided data
	err = t.ExecuteTemplate(w, tmpl+".html", globalData)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
}
