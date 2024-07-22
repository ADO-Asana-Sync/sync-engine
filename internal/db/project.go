package db

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/net/context"
)

var (
	// ProjectsCollection is the name of the collection in the database
	ProjectsCollection = "projects"
)

// Project represents a project with its corresponding names in ADO and Asana.
type Project struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id"`
	ADOProjectName     string             `json:"ado_project_name" bson:"ado_project_name"`
	ADOTeamName        string             `json:"ado_team_name" bson:"ado_team_name"`
	AsanaProjectName   string             `json:"asana_project_name" bson:"asana_project_name"`
	AsanaWorkspaceName string             `json:"asana_workspace_name" bson:"asana_workspace_name"`
}

// Projects retrieves all projects from the database.
// It returns a slice of Project structs and an error, if any.
func (db *DB) Projects() ([]Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var projects []Project
	collection := db.Client.Database(DatabaseName).Collection(ProjectsCollection)
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return projects, fmt.Errorf("error finding projects: %v", err)
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &projects)
	if err != nil {
		return projects, fmt.Errorf("error decoding projects: %v", err)
	}
	return projects, nil
}

// AddProject adds a new project to the database.
// It takes a Project struct as input and returns an error, if any.
func (db *DB) AddProject(ctx context.Context, project Project) error {
	// Start a new span for the operation.
	tracer := otel.GetTracerProvider().Tracer("db")
	ctx, span := tracer.Start(ctx, "db.AddProject")
	defer span.End()

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(ProjectsCollection)

	// Check if the project already exists.
	filter := bson.M{
		"ado_project_name":     project.ADOProjectName,
		"ado_team_name":        project.ADOTeamName,
		"asana_project_name":   project.AsanaProjectName,
		"asana_workspace_name": project.AsanaWorkspaceName,
	}
	count, err := collection.CountDocuments(dbCtx, filter)
	if err != nil {
		err = fmt.Errorf("error checking for existing project: %v", err)
		span.RecordError(err)
		return err
	}

	// If the project exists, throw an error.
	if count > 0 {
		err = fmt.Errorf("project already exists")
		span.RecordError(err)
		return err
	}

	// Insert the new project with a unique ID.
	_, err = collection.InsertOne(dbCtx, project)
	if err != nil {
		err = fmt.Errorf("error inserting project: %v", err)
		span.RecordError(err)
		return err
	}
	return nil
}

// RemoveProject removes a project from the database based on its ID.
// It takes an ID ObjectID as input and returns an error, if any.
func (db *DB) RemoveProject(ctx context.Context, id primitive.ObjectID) error {
	// Start a new span for the operation.
	tracer := otel.GetTracerProvider().Tracer("db")
	ctx, span := tracer.Start(ctx, "db.RemoveProject")
	defer span.End()

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(ProjectsCollection)

	// Remove the project directly using the ID.
	span.SetAttributes(attribute.String("project_id", id.String()))
	filter := bson.M{"_id": id}
	result, err := collection.DeleteOne(dbCtx, filter)
	if err != nil {
		err = fmt.Errorf("error removing project: %v", err)
		span.RecordError(err)
		return err
	}

	// Check if any project was deleted.
	if result.DeletedCount == 0 {
		err = fmt.Errorf("project does not exist")
		span.RecordError(err)
		return err
	}

	return nil
}
