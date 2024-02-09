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
type Config struct {
	Asana struct {
		PAT string `mapstructure:"ASANA_PAT"`
	}
	Port string `mapstructure:"PORT"`
}

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var config Config
	viper.SetDefault("PORT", "8080") // Set a default port if none is specified
	viper.BindEnv("ASANA.PAT")      // Bind the ASANA_PAT environment variable
	viper.BindEnv("PORT")           // Bind the PORT environment variable
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	r := setupRouter()
	if config.Port == "" {
		log.Fatal("Port must be set")
	}
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
