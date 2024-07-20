package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type App struct {
	Asana           *asana.Asana
	Azure           *azure.Azure
	DB              *db.DB
	Tracer          trace.Tracer
	UptraceShutdown func(ctx context.Context) error
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Infof("Web-ui process started. Version: %v, commit: %v, date: %v", version, commit, date)
	app := &App{}
	err := app.setup(ctx)
	if err != nil {
		log.WithError(err).Fatal("error setting up the app")
	}
	defer func(ctx context.Context) {
		err := app.DB.Client.Disconnect(ctx)
		if err != nil {
			log.WithError(err).Fatal("error disconnecting from the DB")
		}
		err = app.UptraceShutdown(ctx)
		if err != nil {
			log.WithError(err).Fatal("error shutting down Uptrace")
		}
	}(ctx)

	// Create a new ServeMux.
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(app, w, r)
	})
	mux.HandleFunc("/add-project", func(w http.ResponseWriter, r *http.Request) {
		addProjectHandler(app, w, r)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		faviconHandler(app, w, r)
	})
	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		projectsHandler(app, w, r)
	})

	// Handler for static content
	fs := http.FileServer(http.Dir(filepath.Join(".", "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Wrap the entire mux with otelhttp.NewHandler.
	handler := otelhttp.NewHandler(mux, "myServer")

	// Create a new http.Server instance with the wrapped handler.
	httpServer := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      handler,
	}
	log.Infof("listening on http://localhost:%v", port)
	log.Fatal(httpServer.ListenAndServe())
}

func (app *App) setup(ctx context.Context) error {
	// Uptrace setup.
	log.Info("connecting to Uptrace")
	dsn := os.Getenv("UPTRACE_DSN")
	environment := os.Getenv("UPTRACE_ENVIRONMENT")
	uptrace.ConfigureOpentelemetry(
		uptrace.WithServiceName("web-ui"),
		uptrace.WithDSN(dsn),
		uptrace.WithServiceVersion(version),
		uptrace.WithDeploymentEnvironment(environment),
	)
	app.UptraceShutdown = uptrace.Shutdown
	app.Tracer = otel.Tracer("")

	ctx, span := app.Tracer.Start(ctx, "web-ui.setup")
	defer span.End()

	// Database setup.
	log.Info("connecting to the DB")
	app.DB = &db.DB{}
	connstr := os.Getenv("MONGO_URI")
	err := app.DB.Connect(ctx, connstr)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error connecting to the DB: %v", err)
	}

	// Azure DevOps setup.
	log.Info("connecting to Azure DevOps")
	app.Azure = &azure.Azure{}
	org := os.Getenv("ADO_ORG_URL")
	pat := os.Getenv("ADO_PAT")
	app.Azure.Connect(ctx, org, pat)

	// Asana setup.
	log.Info("connecting to Asana")
	app.Asana = &asana.Asana{}
	app.Asana.Connect(ctx, os.Getenv("ASANA_PAT"))

	return nil
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

func faviconHandler(app *App, w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("static", "favicon.ico"))
}

func homeHandler(app *App, w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", map[string]interface{}{
		"Title":       "Dashboard",
		"CurrentPage": "dashboard",
	})
}

func projectsHandler(app *App, w http.ResponseWriter, r *http.Request) {
	// Fetch projects from the database
	projects, err := app.DB.Projects()
	if err != nil {
		http.Error(w, "Unable to fetch projects", http.StatusInternalServerError)
		return
	}

	// Render the template with the projects data
	data := struct {
		Title       string
		CurrentPage string
		Projects    []db.Project
		Success     bool
		Error       string
	}{
		Title:       "Projects",
		CurrentPage: "projects",
		Projects:    projects,
	}
	renderTemplate(w, "projects", data)
}

func addProjectHandler(app *App, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	adoProjectName := r.FormValue("ado_project_name")
	adoTeamName := r.FormValue("ado_team_name")
	asanaProjectName := r.FormValue("asana_project_name")
	asanaWorkspaceName := r.FormValue("asana_workspace_name")

	project := db.Project{
		ID:                 primitive.NewObjectID(),
		ADOProjectName:     adoProjectName,
		ADOTeamName:        adoTeamName,
		AsanaProjectName:   asanaProjectName,
		AsanaWorkspaceName: asanaWorkspaceName,
	}

	err := app.DB.AddProject(project)
	if err != nil {
		projects, err := app.DB.Projects()
		if err != nil {
			http.Error(w, "Unable to fetch projects", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, "projects", map[string]interface{}{
			"Title":       "Projects",
			"CurrentPage": "projects",
			"Projects":    projects,
			"Error":       err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
