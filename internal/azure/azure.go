package azure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Azure struct {
	Client *azuredevops.Connection
}

// Connect establishes a connection to Azure DevOps using the provided organization URL and personal access token (PAT).
// It configures tracing and sets up the Azure DevOps client for further operations.
func (a *Azure) Connect(ctx context.Context, orgUrl, pat string) {
	// Configure the tracing.
	tracer := otel.GetTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "azure.Connect")
	defer span.End()

	clt := azuredevops.NewPatConnection(orgUrl, pat)
	a.Client = clt
}

// GetChangedWorkItems retrieves the changed work items from Azure.
// It configures the tracing and starts a span for the operation.
//
// https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/workitemtracking/client.go#L2676
// https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/wiql/query-by-wiql?view=azure-devops-rest-7.2&tabs=HTTP
func (a *Azure) GetChangedWorkItems(ctx context.Context, lastSync time.Time) ([]workitemtracking.WorkItemReference, error) {
	// Configure the tracing.
	tracer := otel.GetTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "azure.GetChangedWorkItems")
	defer span.End()

	var tasks []workitemtracking.WorkItemReference

	workClient, err := workitemtracking.NewClient(ctx, a.Client)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return tasks, err
	}

	qs := fmt.Sprintf(
		"SELECT [System.Id], [System.Title], [System.State] FROM workitems WHERE [System.ChangedDate] > '%s' ORDER BY [System.ChangedDate] DESC",
		lastSync.Format(time.RFC3339),
	)

	// Get the first page of work items.
	responseValue, err := workClient.QueryByWiql(ctx, workitemtracking.QueryByWiqlArgs{
		Wiql: &workitemtracking.Wiql{
			Query: &qs,
		},
	})
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return tasks, err
	}

	for _, item := range *(*responseValue).WorkItems {
		tasks = append(tasks, item)
	}

	return tasks, nil
}

// GetProjects retrieves a list of team projects from Azure DevOps.
// It returns a slice of core.TeamProjectReference and an error if any.
// The function uses the provided context for cancellation or timeout.
func (a *Azure) GetProjects(ctx context.Context) ([]core.TeamProjectReference, error) {
	// Configure the tracing.
	tracer := otel.GetTracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "azure.GetProjects")
	defer span.End()

	var projects []core.TeamProjectReference

	// Get the projects.
	coreClient, err := core.NewClient(ctx, a.Client)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return projects, err
	}

	// Get first page of the list of team projects for your organization
	responseValue, err := coreClient.GetProjects(ctx, core.GetProjectsArgs{})
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return projects, err
	}

	index := 0
	for responseValue != nil {
		// Create the slice of team projects.
		for _, teamProjectReference := range (*responseValue).Value {
			projects = append(projects, teamProjectReference)
			index++
		}

		// if continuationToken has a value, then there is at least one more page of projects to get.
		if responseValue.ContinuationToken != "" {

			continuationToken, err := strconv.Atoi(responseValue.ContinuationToken)
			if err != nil {
				span.RecordError(err, trace.WithStackTrace(true))
				span.SetStatus(codes.Error, err.Error())
				return projects, err
			}

			// Get next page of team projects.
			projectArgs := core.GetProjectsArgs{
				ContinuationToken: &continuationToken,
			}
			responseValue, err = coreClient.GetProjects(ctx, projectArgs)
			if err != nil {
				span.RecordError(err, trace.WithStackTrace(true))
				span.SetStatus(codes.Error, err.Error())
				return projects, err
			}
		} else {
			responseValue = nil
		}
	}

	return projects, nil
}
