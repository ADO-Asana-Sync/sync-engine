package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/viper"

	"github.com/gorilla/mux"
)

type Config struct {
	PAT  string `mapstructure:"PAT"`
	Port string `mapstructure:"PORT"`
}

func loadConfig() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("asana_agent") // use ASANA_AGENT_ as prefix for environment variables
	err := viper.BindEnv("PAT")
	if err != nil {
		return nil, err
	}
	err = viper.BindEnv("PORT")
	if err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
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
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	r := setupRouter()
	if config.Port == "" {
		log.Fatal("Port must be set")
	}
	fmt.Printf("Web server starting at http://localhost:%s\n", config.Port)
	err = http.ListenAndServe(":"+config.Port, r)
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
