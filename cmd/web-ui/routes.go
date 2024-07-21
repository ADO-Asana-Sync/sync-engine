package main

import (
	"net/http"
	"path/filepath"
)

func registerRoutes(mux *http.ServeMux, app *App) {
	// Handlers for shared content.
	fs := http.FileServer(http.Dir(filepath.Join(".", "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		faviconHandler(w, r)
	})

	// Dashboard routes.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(w, r)
	})

	// Projects routes.
	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		projectsHandler(app, w, r)
	})
	mux.HandleFunc("/add-project", func(w http.ResponseWriter, r *http.Request) {
		addProjectHandler(app, w, r)
	})
	mux.HandleFunc("/delete-project", func(w http.ResponseWriter, r *http.Request) {
		deleteProjectHandler(app, w, r)
	})
}
