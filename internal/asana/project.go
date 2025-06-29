package asana

import (
	"context"
	"fmt"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	asanaapi "github.com/qw4n7y/go-asana/asana"
)

// Project represents minimal info about an Asana project.
type Project struct {
	GID  string
	Name string
}

// ProjectGIDByName resolves an Asana project GID using workspace and project names.
func (a *Asana) ProjectGIDByName(ctx context.Context, workspaceName, projectName string) (string, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ProjectGIDByName")
	defer span.End()

	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return "", err
	}
	var wsID int64
	found := false
	for _, ws := range workspaces {
		if ws.Name == workspaceName {
			wsID = ws.ID
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("workspace not found")
	}

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	projs, err := client.ListProjects(ctx, &asanaapi.Filter{Workspace: wsID})
	if err != nil {
		return "", err
	}
	for _, p := range projs {
		if p.Name == projectName {
			if p.GID != "" {
				return p.GID, nil
			}
			return fmt.Sprint(p.ID), nil
		}
	}
	return "", fmt.Errorf("project not found")
}

// ListProjects returns all projects within the given workspace.
func (a *Asana) ListProjects(ctx context.Context, workspaceName string) ([]Project, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListProjects")
	defer span.End()

	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	var wsID int64
	found := false
	for _, ws := range workspaces {
		if ws.Name == workspaceName {
			wsID = ws.ID
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("workspace not found")
	}

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	projs, err := client.ListProjects(ctx, &asanaapi.Filter{Workspace: wsID})
	if err != nil {
		return nil, err
	}
	var result []Project
	for _, p := range projs {
		if p.GID != "" {
			result = append(result, Project{GID: p.GID, Name: p.Name})
		} else {
			result = append(result, Project{GID: fmt.Sprint(p.ID), Name: p.Name})
		}
	}
	return result, nil
}
