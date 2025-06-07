package db

import (
	"fmt"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

var (
	// LastSyncCollection is the name of the collection in the database
	LastSyncCollection = "last_sync"
)

// LastSync represents a lastSync record.
type LastSync struct {
	Time time.Time `json:"time" bson:"time"`
}

// LastSync retrieves the last sync time.
func (db *DB) LastSync(ctx context.Context) LastSync {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.LastSync")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var lastSync LastSync
	collection := db.Client.Database(DatabaseName).Collection(LastSyncCollection)
	err := collection.FindOne(ctx, bson.D{}).Decode(&lastSync)
	if err != nil {
		lastSync.Time = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return lastSync
}

// WriteLastSync updates the last sync time in the database.
func (db *DB) WriteLastSync(ctx context.Context, timestamp time.Time) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.WriteLastSync")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	collection := db.Client.Database(DatabaseName).Collection(LastSyncCollection)

	// Prepare the update operation
	update := bson.M{"$set": bson.M{"time": timestamp}}

	// Upsert option to insert if not exists
	opts := options.Update().SetUpsert(true)

	// Perform the update operation
	_, err := collection.UpdateOne(ctx, bson.D{}, update, opts)
	if err != nil {
		return fmt.Errorf("error updating last sync time: %v", err)
	}

	return nil
}
