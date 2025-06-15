package db

import (
	"context"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	// ConnectionTimeout is the default timeout for connecting to the database.
	ConnectionTimeout = 60 * time.Second

	// DatabaseName is the name of the database
	DatabaseName = "ado-asana-sync"

	// Timeout is the default timeout for database operations.
	Timeout = 10 * time.Second
)

type DBInterface interface {
	Connect(ctx context.Context, uri string) error
	Disconnect(ctx context.Context) error
	Projects(ctx context.Context) ([]Project, error)
	AddProject(ctx context.Context, project Project) error
	RemoveProject(ctx context.Context, id primitive.ObjectID) error
	UpdateProject(ctx context.Context, project Project) error
	LastSync(ctx context.Context) LastSync
	WriteLastSync(ctx context.Context, timestamp time.Time) error
	TaskByADOTaskID(ctx context.Context, id int) (TaskMapping, error)
	AddTask(ctx context.Context, task TaskMapping) error
	UpdateTask(ctx context.Context, task TaskMapping) error
	GetCacheItem(ctx context.Context, key string) (CacheItem, error)
	UpsertCacheItem(ctx context.Context, item CacheItem) error
}

type DB struct {
	Client *mongo.Client
}

func (db *DB) Connect(ctx context.Context, uri string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.Connect")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	db.Client = c

	err = db.Client.Ping(ctx, nil)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (db *DB) Disconnect(ctx context.Context) error {
	return db.Client.Disconnect(ctx)
}
