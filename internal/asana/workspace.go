package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/range-labs/go-asana/asana"
)

var (
	timeout = 60 * time.Second
)

func ListWorkspaces(ctx context.Context, c *http.Client) ([]asana.Workspace, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListWorkspaces")
	defer span.End()

	client := asana.NewClient(c)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}
