package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/range-labs/go-asana/asana"
)

var (
	timeout = 60 * time.Second
)

func ListWorkspaces(c *http.Client) ([]asana.Workspace, error) {
	client := asana.NewClient(c)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}
