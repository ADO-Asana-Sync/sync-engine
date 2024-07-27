package db

import (
	"context"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
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
