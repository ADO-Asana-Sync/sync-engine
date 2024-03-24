package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
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
	glog.Infof("Sync process started. Version: %v, commit: %v, date: %v", version, commit, date)
	app := &App{}
	err := app.setup()
	if err != nil {
		glog.Fatalf("error setting up the app: %v", err)
	}
	defer func() {
		err := app.DB.Client.Disconnect(context.Background())
		if err != nil {
			glog.Fatalf("error disconnecting from the DB: %v", err)
		}
		err = app.UptraceShutdown(context.Background())
		if err != nil {
			glog.Fatalf("error shutting down Uptrace: %v", err)
		}
	}()

	// List all projects
	projects, err := app.DB.Projects()
	if err != nil {
		glog.Fatalf("error listing projects: %v", err)
	}
	spew.Dump(projects)
}

func (app *App) setup() error {
	// Uptrace setup.
	glog.Infof("Connecting to Uptrace")
	dsn := os.Getenv("UPTRACE_DSN")
	environment := os.Getenv("UPTRACE_ENVIRONMENT")
	uptrace.ConfigureOpentelemetry(
		uptrace.WithServiceName("sync-engine"),
		uptrace.WithDSN(dsn),
		uptrace.WithServiceVersion(version),
		uptrace.WithDeploymentEnvironment(environment),
	)
	app.UptraceShutdown = uptrace.Shutdown
	app.Tracer = otel.Tracer("")

	ctx := context.Background()
	ctx, span := app.Tracer.Start(ctx, "sync.setup")
	defer span.End()

	// Database setup.
	glog.Infof("connecting to the DB")
	app.DB = &db.DB{}
	connstr := os.Getenv("MONGO_URI")
	err := app.DB.Connect(ctx, connstr)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		return fmt.Errorf("error connecting to the DB: %v", err)
	}

	// Azure DevOps setup.
	glog.Infof("connecting to Azure DevOps")
	app.Azure = &azure.Azure{}
	org := os.Getenv("ADO_ORG_URL")
	pat := os.Getenv("ADO_PAT")
	app.Azure.Connect(org, pat)

	// Asana setup.
	glog.Infof("connecting to Asana")
	app.Asana = &asana.Asana{}
	app.Asana.Connect(os.Getenv("ASANA_PAT"))

	return nil
}
