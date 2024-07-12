package db

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/net/context"
)

var (
	// ProjectsCollection is the name of the collection in the database
	ProjectsCollection = "projects"
)

// Project represents a project with its corresponding names in ADO and Asana.
type Project struct {
	ADOProjectName   string `json:"ado_project_name" bson:"ado_project_name"`
	ADOTeamName      string `json:"ado_team_name" bson:"ado_team_name"`
	AsanaProjectName string `json:"asana_project_name" bson:"asana_project_name"`
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
func (db *DB) AddProject(project Project) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	collection := db.Client.Database(DatabaseName).Collection(ProjectsCollection)

	// Check if the project already exists
	filter := bson.M{"ado_project_name": project.ADOProjectName, "ado_team_name": project.ADOTeamName, "asana_project_name": project.AsanaProjectName}
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("error checking for existing project: %v", err)
	}

	// If the project exists, return without inserting
	if count > 0 {
		return nil
	}

	// Insert the new project
	_, err = collection.InsertOne(ctx, project)
	if err != nil {
		return fmt.Errorf("error inserting project: %v", err)
	}
	return nil
}


