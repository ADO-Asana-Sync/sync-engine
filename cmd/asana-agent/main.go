package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/", pingHandler)
	r.HandleFunc("/api/ping", pingHandler)
	// workspace routes
	r.HandleFunc("/api/workspace", workspaceGetHandler).Methods("GET")
	return r
}

func main() {
	r := setupRouter()
	if config.Port == "" {
		log.Fatal("Port must be set")
	}
	fmt.Printf("Web server starting at http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Pong!")
}

func workspaceGetHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "workspaces")
}
