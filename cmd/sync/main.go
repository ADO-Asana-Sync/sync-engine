package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type App struct {
	Asana           asana.AsanaInterface
	Azure           azure.AzureInterface
	DB              db.DBInterface
	CacheTTL        time.Duration
	SyncedTags      map[string]asana.Tag
	Tracer          trace.Tracer
	UptraceShutdown func(ctx context.Context) error
}

func init() {
	// Configure logrus to include file and line number in logs
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
	// Enable caller reporting
	log.SetReportCaller(true)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Infof("Sync process started. Version: %v, commit: %v, date: %v", Version, Commit, Date)
	app := &App{}
	err := app.setup(ctx)
	if err != nil {
		log.WithError(err).Fatal("error setting up the app")
	}
	defer func(ctx context.Context) {
		err := app.DB.Disconnect(ctx)
		if err != nil {
			log.WithError(err).Fatal("error disconnecting from the DB")
		}
		err = app.UptraceShutdown(ctx)
		if err != nil {
			log.WithError(err).Fatal("error shutting down Uptrace")
		}
	}(ctx)

	// Create the worker pool.
	numWorkers := 10 // Set the number of concurrent workers here or use an environment variable
	taskCh := make(chan SyncTask)

	for i := 0; i < numWorkers; i++ {
		go app.worker(ctx, i, taskCh)
	}

	// Create the controller.
	var st time.Duration
	for {
		ctx, span := app.Tracer.Start(ctx, "sync.main")
		app.controller(ctx, taskCh)

		st = getSleepTime()
		span.SetAttributes(attribute.Int64("sleepTimeSec", int64(st.Seconds())))
		span.End()

		log.Infof("sleeping for %v", st)
		time.Sleep(st)
	}
}

func getSleepTime() time.Duration {
	sleepTime := os.Getenv("SLEEP_TIME")
	if sleepTime == "" {
		return 5 * time.Minute
	}

	d, err := time.ParseDuration(sleepTime)
	if err != nil {
		log.WithError(err).Warn("unable to parse SLEEP_TIME variable, defaulting to 5m")
		return 5 * time.Minute
	}
	return d
}

func getCacheTTL() time.Duration {
	ttl := os.Getenv("PROPERTY_CACHE_TTL")
	if ttl == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(ttl)
	if err != nil {
		log.WithError(err).Warn("unable to parse PROPERTY_CACHE_TTL, defaulting to 24h")
		return 24 * time.Hour
	}
	return d
}

func (app *App) setup(ctx context.Context) error {
	// Uptrace setup.
	log.Info("connecting to Uptrace")
	dsn := os.Getenv("UPTRACE_DSN")
	environment := os.Getenv("UPTRACE_ENVIRONMENT")
	uptrace.ConfigureOpentelemetry(
		uptrace.WithServiceName("sync-engine"),
		uptrace.WithDSN(dsn),
		uptrace.WithServiceVersion(Version),
		uptrace.WithDeploymentEnvironment(environment),
	)
	app.UptraceShutdown = uptrace.Shutdown
	app.Tracer = otel.Tracer("sync.main")

	ctx, span := app.Tracer.Start(ctx, "sync.setup")
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

	if err := app.DB.EnsureIndexes(ctx); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error ensuring DB indexes: %v", err)
	}

	// Azure DevOps setup.
	log.Info("connecting to Azure DevOps")
	app.Azure = azure.NewAzure()
	org := os.Getenv("ADO_ORG_URL")
	pat := os.Getenv("ADO_PAT")
	app.Azure.Connect(ctx, org, pat)

	// Asana setup.
	log.Info("connecting to Asana")
	app.Asana = &asana.Asana{}
	app.Asana.Connect(ctx, os.Getenv("ASANA_PAT"))

	app.CacheTTL = getCacheTTL()
	app.SyncedTags = make(map[string]asana.Tag)
	app.loadSyncedTags(ctx)

	return nil
}

func (app *App) loadSyncedTags(ctx context.Context) {
	projects, err := app.DB.Projects(ctx)
	if err != nil {
		log.WithError(err).Warn("unable to load projects for tag lookup")
		return
	}
	seen := make(map[string]bool)
	for _, p := range projects {
		ws := p.AsanaWorkspaceName
		if seen[ws] {
			continue
		}
		seen[ws] = true
		rec, err := app.DB.WorkspaceTag(ctx, ws)
		if err == nil && rec.GID != "" {
			app.SyncedTags[ws] = asana.Tag{GID: rec.GID, Name: rec.Name}
		}
	}
}
