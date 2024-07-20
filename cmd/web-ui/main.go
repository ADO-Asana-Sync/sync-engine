package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/projects", projectsHandler)

	// Handler for static content
	fs := http.FileServer(http.Dir(filepath.Join(".", "static")))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	templates := template.Must(template.ParseFiles(
		filepath.Join("templates", "main.html"),
		filepath.Join("templates", tmpl+".html"),
	))

	err := templates.ExecuteTemplate(w, "main", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("static", "favicon.ico"))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", map[string]interface{}{
		"Title":       "Home",
		"CurrentPage": "home",
	})
}
func projectsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "projects", map[string]interface{}{
		"Title":       "Projects",
		"CurrentPage": "projects",
	})
}
