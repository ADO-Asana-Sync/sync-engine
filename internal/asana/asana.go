package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"

	"golang.org/x/oauth2"
)

// AsanaInterface defines the methods that the Asana client must implement.
type AsanaInterface interface {
	Connect(ctx context.Context, pat string)
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
	// ListProjects returns the projects in the given workspace.
	ListProjects(ctx context.Context, workspaceName string) ([]Project, error)
	ProjectGIDByName(ctx context.Context, workspaceName, projectName string) (string, error)
	// CustomFieldByName returns the custom field in the workspace's library
	// matching the provided name.
	CustomFieldByName(ctx context.Context, workspaceName, fieldName string) (CustomField, error)
	// ProjectHasCustomField checks if the given project has a custom field with the provided name.
	ProjectHasCustomField(ctx context.Context, projectGID, fieldName string) (bool, error)
	// ProjectCustomFieldByName returns the custom field with the provided name
	// from the given project. The search is case-insensitive. If the field is
	// not found, an error is returned.
	ProjectCustomFieldByName(ctx context.Context, projectGID, fieldName string) (CustomField, error)
	ListProjectTasks(ctx context.Context, projectGID string) ([]Task, error)
	// CreateTask creates a task in the given project. The notes parameter
	// should contain HTML wrapped in a <body> element which will be stored as
	// the task description.
	CreateTask(ctx context.Context, projectGID, name, notes string) (Task, error)
	// UpdateTask updates an existing task. The notes parameter should
	// contain HTML wrapped in a <body> element for the description.
	UpdateTask(ctx context.Context, taskGID, name, notes string) error
	// CreateTaskWithCustomFields creates a task with additional custom fields.
	CreateTaskWithCustomFields(ctx context.Context, projectGID, name, notes string, customFields map[string]string) (Task, error)
	// UpdateTaskWithCustomFields updates a task and sets custom field values.
	UpdateTaskWithCustomFields(ctx context.Context, taskGID, name, notes string, customFields map[string]string) error
	// TagByName returns the tag with the specified name in the workspace.
	TagByName(ctx context.Context, workspaceName, tagName string) (Tag, error)
	// AddTagToTask adds a tag to the specified task.
	AddTagToTask(ctx context.Context, taskGID, tagGID string) error
}

type Asana struct {
	Client *http.Client
}

func (a *Asana) Connect(ctx context.Context, pat string) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.Connect")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	a.Client = conf.Client(ctx, tok)
}
