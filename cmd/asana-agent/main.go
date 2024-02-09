package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type Config struct {
	Asana struct {
		PAT string
	}
	Port string
}

func setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/", pingHandler)
	r.HandleFunc("/api/ping", pingHandler)
	// workspace routes
	r.HandleFunc("/api/workspace", workspaceGetHandler).Methods("GET")
	return r
}

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	r := setupRouter()
	fmt.Printf("Web server starting at http://localhost:%s\n", config.Port)
	err := http.ListenAndServe(":"+config.Port, r)
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
