package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/gin-gonic/contrib/renders/multitemplate"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	serviceName = "web-ui"
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

	log.Infof("Web-ui process started. Version: %v, commit: %v, date: %v", Version, Commit, Date)
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

	router := gin.Default()
	router.Use(otelgin.Middleware(serviceName))
	router.HTMLRender = loadTemplates(ctx, app)
	registerRoutes(router, app)

	log.Infof("listening on http://localhost:%v", port)
	listenAddress := fmt.Sprintf(":%v", port)
	if err := router.Run(listenAddress); err != nil {
		log.WithError(err).Fatal("error running the router")
	}
}

func (app *App) setup(ctx context.Context) error {
	// Uptrace setup.
	log.Info("connecting to Uptrace")
	dsn := os.Getenv("UPTRACE_DSN")
	environment := os.Getenv("UPTRACE_ENVIRONMENT")
	uptrace.ConfigureOpentelemetry(
		uptrace.WithServiceName(serviceName),
		uptrace.WithDSN(dsn),
		uptrace.WithServiceVersion(Version),
		uptrace.WithDeploymentEnvironment(environment),
	)
	app.UptraceShutdown = uptrace.Shutdown
	app.Tracer = otel.Tracer("web-ui")

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

func loadTemplates(ctx context.Context, app *App) multitemplate.Render {
	_, span := app.Tracer.Start(ctx, "web-ui.loadTemplates")
	defer span.End()

	templates := multitemplate.New()

	// Read all files in the templates folder.
	files, err := os.ReadDir("templates")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		log.Fatalf("Failed to read templates directory: %v", err)
	}

	span.AddEvent(fmt.Sprintf("Found %v files in the templates directory", len(files)))

	// Iterate over each file in the templates folder.
	for _, file := range files {
		// Skip directories and the main.html file.
		if file.IsDir() || file.Name() == "main.html" {
			continue
		}

		// Get the file name without the extension to use as the template name.
		templateName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		// Add the template using main.html and the current file.
		templates.AddFromFiles(templateName,
			filepath.Join("templates", "main.html"),
			filepath.Join("templates", file.Name()),
		)
	}

	return templates
}
