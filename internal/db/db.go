package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
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

	// DefaultMaxPoolSize is the default MongoDB connection pool size.
	DefaultMaxPoolSize uint64 = 100
)

type DBInterface interface {
	Connect(ctx context.Context, uri string) error
	Disconnect(ctx context.Context) error
	EnsureIndexes(ctx context.Context) error
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
	WorkspaceTag(ctx context.Context, workspaceName string) (WorkspaceTag, error)
	UpsertWorkspaceTag(ctx context.Context, tag WorkspaceTag) error
}

type DB struct {
	Client *mongo.Client
}

func getMaxPoolSize() uint64 {
	v := os.Getenv("MONGO_MAX_POOL_SIZE")
	if v == "" {
		return DefaultMaxPoolSize
	}
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		logrus.WithError(err).Warn("invalid MONGO_MAX_POOL_SIZE, using default")
		return DefaultMaxPoolSize
	}
	if i == 0 {
		return DefaultMaxPoolSize
	}
	return i
}

func (db *DB) Connect(ctx context.Context, uri string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.Connect")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()

	c, err := mongo.Connect(ctx, clientOptions(uri))
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

func clientOptions(uri string) *options.ClientOptions {
	return options.Client().ApplyURI(uri).SetMaxPoolSize(getMaxPoolSize()).SetRetryReads(true).SetRetryWrites(true)
}

// EnsureIndexes creates required MongoDB indexes.
func (db *DB) EnsureIndexes(ctx context.Context) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.EnsureIndexes")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	coll := db.Client.Database(DatabaseName).Collection(ProjectsCollection)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "ado_project_name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error creating project index: %v", err)
	}

	return nil
}
