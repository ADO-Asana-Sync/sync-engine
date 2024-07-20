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
		filepath.Join("templates", "projects.html"),
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
func projectsHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch projects from the database
	projects, err := dbInstance.Projects()
	if err != nil {
		http.Error(w, "Unable to fetch projects", http.StatusInternalServerError)
		return
	}

	// Render the template with the projects data
	data := struct {
		Title       string
		CurrentPage string
		Projects    []db.Project
	}{
		Title:       "Projects",
		CurrentPage: "projects",
		Projects:    projects,
	}
	renderTemplate(w, "projects", data)
}

func main() {
	http.HandleFunc("/projects", projectsHandler)
