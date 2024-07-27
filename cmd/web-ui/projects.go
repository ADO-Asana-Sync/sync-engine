package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/gin-gonic/gin"
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

func projectsHandler(app *App, c *gin.Context) {
	ctx, span := app.Tracer.Start(c.Request.Context(), "projects.projectsHandler")
	defer span.End()

	data, err := fetchProjectsData(ctx, app)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Unable to fetch projects",
		})
		return
	}
	c.HTML(http.StatusOK, "projects", data)
}

func addProjectHandler(app *App, c *gin.Context) {
	ctx, span := app.Tracer.Start(c.Request.Context(), "projects.addProjectHandler")
	defer span.End()

	adoProjectName := c.Request.FormValue("ado_project_name")
	if adoProjectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ADO project name is required"})
		return
	}
	adoTeamName := c.Request.FormValue("ado_team_name")
	if adoTeamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ADO team name is required"})
		return
	}
	asanaProjectName := c.Request.FormValue("asana_project_name")
	if asanaProjectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asana project name is required"})
		return
	}
	asanaWorkspaceName := c.Request.FormValue("asana_workspace_name")
	if asanaWorkspaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asana workspace name is required"})
		return
	}

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
		data, err := fetchProjectsData(ctx, app)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unable to fetch projects after adding new project",
			})
			return
		}
		data.Error = appErr.Error()
		c.HTML(http.StatusOK, "projects", data)
		return
	}
	c.Redirect(http.StatusSeeOther, "/projects")
}

func deleteProjectHandler(app *App, c *gin.Context) {
	ctx, span := app.Tracer.Start(c.Request.Context(), "projects.deleteProjectHandler")
	defer span.End()

	projectID := c.Query("id")
	if projectID == "" {
		span.RecordError(fmt.Errorf("missing project ID"), trace.WithStackTrace(true))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing project ID",
		})
		return
	}

	// Convert projectID to ObjectID.
	objID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid project ID",
		})
		return
	}

	// Delete the project from the database.
	err = app.DB.RemoveProject(ctx, objID)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete project",
		})
		return
	}
	c.Status(http.StatusNoContent)
}

func editProjectHandler(app *App, c *gin.Context) {
	ctx, span := app.Tracer.Start(c.Request.Context(), "projects.editProjectHandler")
	defer span.End()

	projectID := c.Request.FormValue("id")
	adoProjectName := c.Request.FormValue("ado_project_name")
	adoTeamName := c.Request.FormValue("ado_team_name")
	asanaProjectName := c.Request.FormValue("asana_project_name")
	asanaWorkspaceName := c.Request.FormValue("asana_workspace_name")

	// Convert projectID to ObjectID.
	objID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid project ID",
		})
		return
	}

	project := db.Project{
		ID:                 objID,
		ADOProjectName:     adoProjectName,
		ADOTeamName:        adoTeamName,
		AsanaProjectName:   asanaProjectName,
		AsanaWorkspaceName: asanaWorkspaceName,
	}

	err = app.DB.UpdateProject(ctx, project)
	if err != nil {
		appErr := fmt.Errorf("error updating project: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		data, err := fetchProjectsData(ctx, app)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to fetch projects after updating project",
			})
			return
		}
		data.Error = appErr.Error()
		c.HTML(http.StatusOK, "projects", data)
		return
	}
	c.Redirect(http.StatusSeeOther, "/projects")
}
