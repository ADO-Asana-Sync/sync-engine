package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/range-labs/go-asana/asana"
)

// ListProjectsByWorkspace lists all projects in the provided workspace.
func ListProjectsByWorkspace(ctx context.Context, c *http.Client, workspaceID string) ([]asana.Project, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListProjects")
	defer span.End()

	client := asana.NewClient(c)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	allProjects, err := client.ListProjects(ctx, &asana.Filter{WorkspaceGID: workspaceID})
	if err != nil {
		return nil, err
	}

	return allProjects, nil
}
