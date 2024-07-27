package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/trace"
)

type ProjectsViewData struct {
	Title       string
	CurrentPage string
	Projects    []db.Project
	Error       string
}

func fetchProjectsData(ctx context.Context, app *App) (data ProjectsViewData, err error) {
	ctx, span := app.Tracer.Start(ctx, "projects.fetchProjectsData")
	defer span.End()

	projects, err := app.DB.Projects(ctx)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		return data, err
	}

	sort.SliceStable(projects, func(i, j int) bool {
		return projects[i].ADOProjectName < projects[j].ADOProjectName
	})

	data = ProjectsViewData{
		Title:       "Projects",
		CurrentPage: "projects",
		Projects:    projects,
	}
	span.AddEvent(fmt.Sprintf("%v projects fetched", len(projects)))
	return data, nil
}

func projectsHandler(app *App, w http.ResponseWriter, r *http.Request) {
	data, err := fetchProjectsData(app)
	if err != nil {
		http.Error(w, "Unable to fetch projects", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "projects", data)
}

func addProjectHandler(app *App, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	adoProjectName := r.FormValue("ado_project_name")
	adoTeamName := r.FormValue("ado_team_name")
	asanaProjectName := r.FormValue("asana_project_name")
	asanaWorkspaceName := r.FormValue("asana_workspace_name")

	project := db.Project{
		ID:                 primitive.NewObjectID(),
		ADOProjectName:     adoProjectName,
		ADOTeamName:        adoTeamName,
		AsanaProjectName:   asanaProjectName,
		AsanaWorkspaceName: asanaWorkspaceName,
	}

	err := app.DB.AddProject(ctx, project)
	if err != nil {
		appErr := fmt.Errorf("error adding project: %v", err)
		data, err := fetchProjectsData(app)
		if err != nil {
			http.Error(w, "unable to fetch projects after adding new project", http.StatusInternalServerError)
			return
		}
		data.Error = appErr.Error()
		renderTemplate(w, "projects", data)
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func deleteProjectHandler(app *App, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID := r.URL.Query().Get("id")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	// Convert projectID to ObjectID.
	objID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		http.Error(w, "invalid project ID", http.StatusBadRequest)
		return
	}

	// Delete the project from the database.
	err = app.DB.RemoveProject(ctx, objID)
	if err != nil {
		http.Error(w, "failed to delete project", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
