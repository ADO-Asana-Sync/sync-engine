package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SyncedTask represents a record of a task that has been synced between ADO and Asana.
type SyncedTask struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	ADOTaskID        int                `bson:"ado_task_id"`
	ADOLastUpdated   time.Time          `bson:"ado_last_updated"`
	AsanaTaskID      string             `bson:"asana_task_id"`
	AsanaLastUpdated time.Time          `bson:"asana_last_updated"`
	Status           string             `bson:"status"` // e.g., synced, failed, deleted
	SyncedAt         time.Time          `bson:"synced_at"`
}

const SyncedTasksCollection = "synced_tasks"

// AddOrUpdateSyncedTask inserts or updates a synced task record.
func (db *DB) AddOrUpdateSyncedTask(ctx context.Context, task SyncedTask) error {
	collection := db.Client.Database(DatabaseName).Collection(SyncedTasksCollection)
	filter := bson.M{"ado_task_id": task.ADOTaskID, "asana_task_id": task.AsanaTaskID}
	update := bson.M{"$set": task}
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetAllSyncedTasks retrieves all synced task records.
func (db *DB) GetAllSyncedTasks(ctx context.Context) ([]SyncedTask, error) {
	collection := db.Client.Database(DatabaseName).Collection(SyncedTasksCollection)
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []SyncedTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}
