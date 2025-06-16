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

// TagsCollection defines the collection storing synced tags per workspace.
var TagsCollection = "tags"

// WorkspaceTag stores a tag associated with a workspace.
type WorkspaceTag struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	WorkspaceName string             `bson:"workspace_name" json:"workspace_name"`
	Name          string             `bson:"name" json:"name"`
	GID           string             `bson:"gid" json:"gid"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// WorkspaceTag retrieves the tag for the given workspace.
func (db *DB) WorkspaceTag(ctx context.Context, workspaceName string) (WorkspaceTag, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.WorkspaceTag")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var tag WorkspaceTag
	coll := db.Client.Database(DatabaseName).Collection(TagsCollection)
	err := coll.FindOne(ctx, bson.M{"workspace_name": workspaceName}).Decode(&tag)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return tag, err
	}
	return tag, nil
}

// UpsertWorkspaceTag stores a workspace tag in the database.
func (db *DB) UpsertWorkspaceTag(ctx context.Context, tag WorkspaceTag) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.UpsertWorkspaceTag")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	coll := db.Client.Database(DatabaseName).Collection(TagsCollection)
	tag.UpdatedAt = time.Now()
	filter := bson.M{"workspace_name": tag.WorkspaceName}
	update := bson.M{"$set": bson.M{"name": tag.Name, "gid": tag.GID, "updated_at": tag.UpdatedAt}}
	_, err := coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
