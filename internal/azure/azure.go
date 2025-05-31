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
	GetWorkItem(ctx context.Context, args workitemtracking.GetWorkItemArgs) (*workitemtracking.WorkItem, error)
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
		projects = append(projects, (*responseValue).Value...)

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

// GetWorkItemDetails retrieves the details of a specific work item from Azure DevOps.
// It populates a WorkItem struct with the relevant fields.
func (a *Azure) GetWorkItemDetails(ctx context.Context, workItemID int) (*WorkItem, error) {
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "azure.GetWorkItemDetails")
	defer span.End()

	workClient, err := a.newWorkItemClient(ctx, a.Client)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create work item client: %w", err)
	}

	expand := workitemtracking.WorkItemExpandValues.All
	resp, err := workClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:     &workItemID,
		Expand: &expand,
		Fields: &[]string{
			"System.Id",
			"System.Title",
			"System.State",
			"System.ChangedDate",
			"System.CreatedDate",
			"System.WorkItemType",
			"System.AssignedTo",
			// We also need the URL, which is typically available in the _links section or needs to be constructed.
			// The GetWorkItem method usually includes this in the response.
		},
	})
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		// TODO: Check for a specific "not found" error type if the SDK provides one
		return nil, fmt.Errorf("failed to get work item %d: %w", workItemID, err)
	}

	if resp == nil || resp.Fields == nil {
		err := fmt.Errorf("received nil work item or nil fields for work item ID %d", workItemID)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	fields := resp.Fields.(map[string]interface{})

	var assignedTo string
	if assignedToField, ok := fields["System.AssignedTo"].(map[string]interface{}); ok {
		if displayName, ok := assignedToField["displayName"].(string); ok {
			assignedTo = displayName
		}
	}

	// Construct the URL to the work item
	// Example: https://dev.azure.com/{organization}/{project}/_workitems/edit/{id}
	// This might need adjustment based on how project/org info is available or if the URL is directly in _links
	workItemURL := ""
	if links, ok := resp.Links.(map[string]interface{}); ok {
		if htmlLink, ok := links["html"].(map[string]interface{}); ok {
			if href, ok := htmlLink["href"].(string); ok {
				workItemURL = href
			}
		}
	}
	// Fallback or ensure org/project context if link not directly available.
	// For now, relying on _links.html.href which is common.

	id := 0
	if resp.Id != nil {
		id = *resp.Id
	} else {
		// This case should ideally be caught by resp == nil check earlier,
		// but as a safeguard if fields are present but Id is missing.
		err := fmt.Errorf("work item ID is nil in response for input ID %d", workItemID)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	getStringField := func(key string) (string, error) {
		val, ok := fields[key].(string)
		if !ok {
			return "", fmt.Errorf("field %s not found or not a string", key)
		}
		return val, nil
	}

	getTimeField := func(key string) (time.Time, error) {
		fieldVal, exists := fields[key]
		if !exists {
			return time.Time{}, fmt.Errorf("field %s not found", key)
		}

		// Check if it's already a time.Time
		if t, ok := fieldVal.(time.Time); ok {
			return t, nil
		}

		// Check if it's a string and try to parse it
		if s, ok := fieldVal.(string); ok {
			t, err := time.Parse(time.RFC3339Nano, s) // ADO often uses ISO 8601 with nanoseconds
			if err != nil {
				// Fallback to parsing without nanoseconds
				t, errSimple := time.Parse("2006-01-02T15:04:05Z", s)
				if errSimple != nil {
					return time.Time{}, fmt.Errorf("field %s is a string but could not be parsed as time (RFC3339Nano: %v / Simple: %v)", key, err, errSimple)
				}
				return tSimple, nil
			}
			return t, nil
		}
		return time.Time{}, fmt.Errorf("field %s is not a time.Time or string representation", key)
	}

	title, err := getStringField("System.Title")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	state, err := getStringField("System.State")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	workItemType, err := getStringField("System.WorkItemType")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	changedDate, err := getTimeField("System.ChangedDate")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	createdDate, err := getTimeField("System.CreatedDate")
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	workItem := &WorkItem{
		ID:           id,
		Title:        title,
		State:        state,
		ChangedDate:  changedDate,
		CreatedDate:  createdDate,
		WorkItemType: workItemType,
		AssignedTo:   assignedTo, // AssignedTo already has a safe lookup
		URL:          workItemURL,  // URL already has a safe lookup
	}

	// Validate required fields
	if err := workItem.checkRequiredProperties("ID", "Title", "WorkItemType", "URL"); err != nil {
		// URL might not be strictly required by all callers, but it's good to have.
		// Adjust checkRequiredProperties if URL is optional.
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("populated work item is missing required fields: %w", err)
	}

	return workItem, nil
}
