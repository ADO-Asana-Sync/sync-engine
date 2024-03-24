package db

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
func (db *DB) LastSync() LastSync {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var lastSync LastSync
	collection := db.Client.Database(DatabaseName).Collection(LastSyncCollection)
	err := collection.FindOne(ctx, bson.D{}).Decode(&lastSync)
	if err != nil {
		lastSync.Time = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return lastSync
}
