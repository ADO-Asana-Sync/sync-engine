package asana

import (
	"context"
)

// AsanaInterface defines the methods that the Asana client must implement.
type AsanaInterface interface {
	Connect(ctx context.Context, pat string)
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
}

type Workspace struct {
	ID   int64
	Name string
}
