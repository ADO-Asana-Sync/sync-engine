package main

import "net/http"

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", map[string]interface{}{
		"Title":       "Dashboard",
		"CurrentPage": "home",
	})
}
