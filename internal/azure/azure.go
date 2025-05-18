package azure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AzureInterface defines the methods that the Azure client must implement.
type AzureInterface interface {
	Connect(ctx context.Context, orgUrl, pat string)
	GetChangedWorkItems(ctx context.Context, lastSync time.Time) ([]workitemtracking.WorkItemReference, error)
	GetProjects(ctx context.Context) ([]core.TeamProjectReference, error)
}

// WIClient defines the methods that the Azure Work Item client must implement.
type WIClient interface {
	QueryByWiql(ctx context.Context, args workitemtracking.QueryByWiqlArgs) (*workitemtracking.WorkItemQueryResult, error)
}

// CoreClient defines the methods that the Azure Core client must implement.
type CoreClient interface {
	GetProjects(ctx context.Context, args core.GetProjectsArgs) (*core.GetProjectsResponseValue, error)
}

type Azure struct {
	Client            *azuredevops.Connection
	newCoreClient     func(context.Context, *azuredevops.Connection) (CoreClient, error)
	newWorkItemClient func(context.Context, *azuredevops.Connection) (WIClient, error)
}

func NewAzure() *Azure {
	return &Azure{
		newWorkItemClient: func(ctx context.Context, c *azuredevops.Connection) (WIClient, error) {
			return workitemtracking.NewClient(ctx, c)
		},
		newCoreClient: func(ctx context.Context, c *azuredevops.Connection) (CoreClient, error) {
			return core.NewClient(ctx, c)
		},
	}
}

// Connect establishes a connection to Azure DevOps using the provided organization URL and personal access token (PAT).
// It configures tracing and sets up the Azure DevOps client for further operations.
func (a *Azure) Connect(ctx context.Context, orgUrl, pat string) {
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "azure.Connect")
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
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "azure.GetChangedWorkItems")
	defer span.End()

	var tasks []workitemtracking.WorkItemReference

	workClient, err := a.newWorkItemClient(ctx, a.Client)
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

	if responseValue.WorkItems != nil {
		tasks = append(tasks, *responseValue.WorkItems...)
	}

	return tasks, nil
}

// GetProjects retrieves a list of team projects from Azure DevOps.
// It returns a slice of core.TeamProjectReference and an error if any.
// The function uses the provided context for cancellation or timeout.
func (a *Azure) GetProjects(ctx context.Context) ([]core.TeamProjectReference, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "azure.GetProjects")
	defer span.End()

	var projects []core.TeamProjectReference

	// Get the projects.
	coreClient, err := a.newCoreClient(ctx, a.Client)
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

	for responseValue != nil {
		// Create the slice of team projects.
		for _, teamProjectReference := range (*responseValue).Value {
			projects = append(projects, teamProjectReference)
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
