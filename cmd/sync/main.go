package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/davecgh/go-spew/spew"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type App struct {
	DB *db.DB
}

func main() {
	log.Printf("Sync process started. Version: %v, commit: %v, date: %v", version, commit, date)
	app := &App{}
	err := app.setup()
	if err != nil {
		log.Fatalf("error setting up the app: %v", err)
	}
	defer func() {
		err := app.DB.Client.Disconnect(nil)
		if err != nil {
			log.Fatalf("error disconnecting from the DB: %v", err)
		}
	}()

	// List all projects
	projects, err := app.DB.Projects()
	if err != nil {
		log.Fatalf("error listing projects: %v", err)
	}
	spew.Dump(projects)
}

func (app *App) setup() error {
	log.Printf("connecting to the DB")
	app.DB = &db.DB{}
	connstr := os.Getenv("MONGO_URI")
	err := app.DB.Connect(connstr)
	if err != nil {
		return fmt.Errorf("error connecting to the DB: %v", err)
	}
	return nil
}
