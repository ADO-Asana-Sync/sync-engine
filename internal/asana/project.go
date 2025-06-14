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
