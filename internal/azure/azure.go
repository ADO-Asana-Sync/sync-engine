package azure

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"go.opentelemetry.io/otel"
)

type Azure struct {
	Client *azuredevops.Connection
}

func (a *Azure) Connect(ctx context.Context, orgUrl, pat string) {
	tracer := otel.GetTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "azure.Connect")
	defer span.End()

	clt := azuredevops.NewPatConnection(orgUrl, pat)
	a.Client = clt
}

func (a *Azure) GetChangedWorkItems(ctx context.Context) {
	// Get the changed work items.
}

func (a *Azure) GetProjects(ctx context.Context) {
	// Get the projects.
}
