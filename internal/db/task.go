package db

import (
	"fmt"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/net/context"
)

// ErrorFmtFindingTask is the error format for when a task cannot be found
const ErrorFmtFindingTask = "error finding task: %v"

// TasksCollection is the name of the collection in the database for synced tasks.
var TasksCollection = "tasks"

// TaskMapping represents a mapping between an ADO work item and an Asana task.
type TaskMapping struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ADOProjectID     string             `bson:"ado_project_id" json:"ado_project_id"`
	ADOTaskID        int                `bson:"ado_task_id" json:"ado_task_id"`
	ADOLastUpdated   time.Time          `bson:"ado_last_updated" json:"ado_last_updated"`
	AsanaProjectID   string             `bson:"asana_project_id" json:"asana_project_id"`
	AsanaTaskID      string             `bson:"asana_task_id" json:"asana_task_id"`
	AsanaLastUpdated time.Time          `bson:"asana_last_updated" json:"asana_last_updated"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

// Tasks retrieves all tasks from the database.
// If projectIDs is provided, it will filter the tasks by ADO project ID.
// It returns a slice of TaskMapping structs and an error, if any.
func (db *DB) Tasks(ctx context.Context, projectIDs ...string) ([]TaskMapping, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.Tasks")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var filter bson.M
	if len(projectIDs) > 0 {
		filter = bson.M{"ado_project_id": bson.M{"$in": projectIDs}}
		span.SetAttributes(attribute.StringSlice("project_ids", projectIDs))
	} else {
		filter = bson.M{}
	}

	var tasks []TaskMapping
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		err = fmt.Errorf("error finding tasks: %v", err)
		span.RecordError(err)
		return tasks, err
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &tasks)
	if err != nil {
		err = fmt.Errorf("error decoding tasks: %v", err)
		span.RecordError(err)
		return tasks, err
	}

	return tasks, nil
}

// TaskByID retrieves a task from the database by its ID.
// It returns the TaskMapping struct and an error, if any.
func (db *DB) TaskByID(ctx context.Context, id primitive.ObjectID) (TaskMapping, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.TaskByID")
	defer span.End()

	span.SetAttributes(attribute.String("task_id", id.String()))

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var task TaskMapping
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		err = fmt.Errorf(ErrorFmtFindingTask, err)
		span.RecordError(err)
		return task, err
	}

	return task, nil
}

// TaskByIDs retrieves a task from the database by ADO and Asana task IDs.
// It returns the TaskMapping struct and an error, if any.
func (db *DB) TaskByIDs(ctx context.Context, adoProjectID string, adoTaskID int, asanaProjectID, asanaTaskID string) (TaskMapping, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.TaskByIDs")
	defer span.End()

	span.SetAttributes(
		attribute.String("ado_project_id", adoProjectID),
		attribute.Int("ado_task_id", adoTaskID),
		attribute.String("asana_project_id", asanaProjectID),
		attribute.String("asana_task_id", asanaTaskID),
	)

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var task TaskMapping
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)

	filter := bson.M{
		"ado_project_id":   adoProjectID,
		"ado_task_id":      adoTaskID,
		"asana_project_id": asanaProjectID,
		"asana_task_id":    asanaTaskID,
	}

	err := collection.FindOne(ctx, filter).Decode(&task)
	if err != nil {
		err = fmt.Errorf(ErrorFmtFindingTask, err)
		span.RecordError(err)
		return task, err
	}

	return task, nil
}

// TaskByADOTaskID retrieves a task from the database by its ADO task ID.
func (db *DB) TaskByADOTaskID(ctx context.Context, id int) (TaskMapping, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.TaskByADOTaskID")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var task TaskMapping
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)
	err := collection.FindOne(ctx, bson.M{"ado_task_id": id}).Decode(&task)
	if err != nil {
		err = fmt.Errorf(ErrorFmtFindingTask, err)
		span.RecordError(err)
		return task, err
	}
	return task, nil
}

// AddTask adds a new task mapping to the database.
// It takes a TaskMapping struct as input and returns an error, if any.
func (db *DB) AddTask(ctx context.Context, task TaskMapping) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.AddTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("ado_project_id", task.ADOProjectID),
		attribute.Int("ado_task_id", task.ADOTaskID),
		attribute.String("asana_project_id", task.AsanaProjectID),
		attribute.String("asana_task_id", task.AsanaTaskID),
	)

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)

	// Check if the task mapping already exists.
	filter := bson.M{
		"ado_project_id":   task.ADOProjectID,
		"ado_task_id":      task.ADOTaskID,
		"asana_project_id": task.AsanaProjectID,
		"asana_task_id":    task.AsanaTaskID,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		err = fmt.Errorf("error checking for existing task mapping: %v", err)
		span.RecordError(err)
		return err
	}

	// If the task mapping exists, throw an error.
	if count > 0 {
		err = fmt.Errorf("task mapping already exists")
		span.RecordError(err)
		return err
	}

	// Set creation and update timestamps.
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	// Insert the new task mapping with a unique ID.
	if task.ID.IsZero() {
		task.ID = primitive.NewObjectID()
	}

	_, err = collection.InsertOne(ctx, task)
	if err != nil {
		err = fmt.Errorf("error inserting task mapping: %v", err)
		span.RecordError(err)
		return err
	}

	return nil
}

// RemoveTask removes a task mapping from the database based on its ID.
// It takes an ID ObjectID as input and returns an error, if any.
func (db *DB) RemoveTask(ctx context.Context, id primitive.ObjectID) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.RemoveTask")
	defer span.End()

	span.SetAttributes(attribute.String("task_id", id.String()))

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)

	// Remove the task mapping directly using the ID.
	filter := bson.M{"_id": id}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		err = fmt.Errorf("error removing task mapping: %v", err)
		span.RecordError(err)
		return err
	}

	// Check if any task mapping was deleted.
	if result.DeletedCount == 0 {
		err = fmt.Errorf("task mapping does not exist")
		span.RecordError(err)
		return err
	}

	return nil
}

// UpdateTask updates an existing task mapping in the database.
// It takes a TaskMapping struct as input and returns an error, if any.
func (db *DB) UpdateTask(ctx context.Context, task TaskMapping) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "db.UpdateTask")
	defer span.End()

	span.SetAttributes(attribute.String("task_id", task.ID.String()))

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(TasksCollection)

	// Update the last modified time.
	task.UpdatedAt = time.Now()

	// Update the task mapping using the ID.
	filter := bson.M{"_id": task.ID}
	update := bson.M{
		"$set": bson.M{
			"ado_project_id":     task.ADOProjectID,
			"ado_task_id":        task.ADOTaskID,
			"ado_last_updated":   task.ADOLastUpdated,
			"asana_project_id":   task.AsanaProjectID,
			"asana_task_id":      task.AsanaTaskID,
			"asana_last_updated": task.AsanaLastUpdated,
			"updated_at":         task.UpdatedAt,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		err = fmt.Errorf("error updating task mapping: %v", err)
		span.RecordError(err)
		return err
	}

	return nil
}
