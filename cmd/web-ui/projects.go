package main

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectsViewData struct {
	Title       string
	CurrentPage string
	Projects    []db.Project
	Error       string
}

func fetchProjectsData(app *App) (data ProjectsViewData, err error) {
	projects, err := app.DB.Projects()
	if err != nil {
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

	err := app.DB.AddProject(project)
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
