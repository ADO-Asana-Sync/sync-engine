package db

import (
	"context"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// CacheCollection defines the collection name used for cached properties.
var CacheCollection = "cache"

// CacheItem stores a cached property value identified by a unique key.
// Value is stored as a generic map to allow reuse for different property types.
type CacheItem struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Key       string                 `bson:"key" json:"key"`
	Value     map[string]interface{} `bson:"value" json:"value"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
}

// GetCacheItem retrieves a cached item by key.
func (db *DB) GetCacheItem(ctx context.Context, key string) (CacheItem, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.GetCacheItem")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var item CacheItem
	coll := db.Client.Database(DatabaseName).Collection(CacheCollection)
	err := coll.FindOne(ctx, bson.M{"key": key}).Decode(&item)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return item, err
	}
	return item, nil
}

// UpsertCacheItem stores a cached item, updating existing entries.
func (db *DB) UpsertCacheItem(ctx context.Context, item CacheItem) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.UpsertCacheItem")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	coll := db.Client.Database(DatabaseName).Collection(CacheCollection)
	item.UpdatedAt = time.Now()
	update := bson.M{"$set": bson.M{"value": item.Value, "updated_at": item.UpdatedAt}}
	_, err := coll.UpdateOne(ctx, bson.M{"key": item.Key}, update, options.Update().SetUpsert(true))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
