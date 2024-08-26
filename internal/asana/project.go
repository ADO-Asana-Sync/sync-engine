package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/range-labs/go-asana/asana"
)

// ListProjects lists all projects in the provided workspace, handling pagination.
func ListProjects(ctx context.Context, c *http.Client, workspaceID string) ([]asana.Project, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListProjects")
	defer span.End()

	client := asana.NewClient(c)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var allProjects []asana.Project
	var offset string

	for {
		projects, nextOffset, err := client.ListProjects(ctx, workspaceID, &asana.ListProjectsOptions{Offset: offset})
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if nextOffset == "" {
			break
		}
		offset = nextOffset
	}

	return allProjects, nil
}
