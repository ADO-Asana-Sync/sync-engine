package asana

import (
	"context"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/range-labs/go-asana/asana"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	timeout = 60 * time.Second
)

// Workspace represents an Asana workspace with minimal required information.
type Workspace struct {
	ID   int64
	Name string
}

func (a *Asana) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListWorkspaces")
	defer span.End()

	client := asana.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var result []Workspace
	for _, ws := range workspaces {
		result = append(result, Workspace{ID: ws.ID, Name: ws.Name})
	}
	return result, nil
}
