package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PageData struct {
	Title   string
	Error   string
	Cars    []Car
	Car     Car
	IsEdit  bool
	FormAction string
}

var templates *template.Template
var dbapi *DBAPIClient

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

func htmlDir() string {
	if d := os.Getenv("HTML_DIR"); d != "" {
		return d
	}
	// Prefer local html/ next to the binary's working directory / source tree.
	candidates := []string{"html", "src/webapp/html", "/opt/app-root/html"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return "html"
}

func loadTemplates(dir string) (*template.Template, error) {
	pattern := filepath.Join(dir, "*.html")
	return template.New("webapp").Funcs(template.FuncMap{
		"chipClass": chipClass,
	}).ParseGlob(pattern)
}

func chipClass(condition string) string {
	c := strings.ToLower(condition)
	switch {
	case strings.Contains(c, "new"):
		return "chip chip-new"
	case strings.Contains(c, "old"):
		return "chip chip-old"
	default:
		return "chip chip-mid"
	}
}

func render(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func parseCarID(path, suffix string) (int, bool) {
	// path like /cars/12/edit or /cars/12/delete or /cars/12
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "cars" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	if suffix == "" {
		return id, len(parts) == 2
	}
	return id, len(parts) == 3 && parts[2] == suffix
}

func main() {
	dir := htmlDir()
	var err error
	templates, err = loadTemplates(dir)
	if err != nil {
		log.Fatalf("load templates from %s: %v", dir, err)
	}

	dbapi = NewDBAPIClient(getEnv("DBAPI_URL", "http://dbapi:8080"))
	port := getEnv("PORT", "8080")

	staticDir := filepath.Join(dir, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	http.HandleFunc("/", router)

	log.Printf("Starting webapp on port %s (html=%s dbapi=%s)", port, dir, dbapi.BaseURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func router(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/" && r.Method == http.MethodGet:
		handleList(w, r)
	case path == "/cars/new" && r.Method == http.MethodGet:
		handleNewForm(w, r)
	case path == "/cars" && r.Method == http.MethodPost:
		handleCreate(w, r)
	default:
		if id, ok := parseCarID(path, "edit"); ok && r.Method == http.MethodGet {
			handleEditForm(w, r, id)
			return
		}
		if id, ok := parseCarID(path, ""); ok && r.Method == http.MethodPost {
			handleUpdate(w, r, id)
			return
		}
		if id, ok := parseCarID(path, "delete"); ok && r.Method == http.MethodPost {
			handleDelete(w, r, id)
			return
		}
		http.NotFound(w, r)
	}
}
