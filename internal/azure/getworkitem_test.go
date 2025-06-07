package azure

import (
	"context"
	"testing"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ADO-Asana-Sync/sync-engine/internal/testutil"
)

func TestAzureGetWorkItem(t *testing.T) {
	t.Parallel()

	fields := map[string]interface{}{
		"System.Title":        "Test Item",
		"System.WorkItemType": "Bug",
		"System.State":        "Active",
		"System.AssignedTo":   "bob@example.com",
		"System.ChangedDate":  time.Now(),
		"System.CreatedDate":  time.Now(),
	}
	wi := &workitemtracking.WorkItem{
		Id:     testutil.Ptr(123),
		Url:    testutil.Ptr("https://dev.azure.com/org/proj/_workitems/edit/123"),
		Fields: &fields,
	}

	mockWI := new(MockWIClient)
	mockWI.On("GetWorkItem", mock.Anything, mock.MatchedBy(func(args workitemtracking.GetWorkItemArgs) bool {
		return args.Id != nil && *args.Id == 123
	})).Return(wi, nil)

	a := &Azure{
		newWorkItemClient: func(ctx context.Context, c *azuredevops.Connection) (WIClient, error) {
			return mockWI, nil
		},
	}

	got, err := a.GetWorkItem(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, 123, got.ID)
	require.Equal(t, "Test Item", got.Title)
	require.Equal(t, "Bug", got.WorkItemType)
	require.Equal(t, "https://dev.azure.com/org/proj/_workitems/edit/123", got.URL)

	mockWI.AssertExpectations(t)
}

func TestAzureGetWorkItemError(t *testing.T) {
	t.Parallel()
	a := &Azure{
		newWorkItemClient: func(ctx context.Context, c *azuredevops.Connection) (WIClient, error) {
			return nil, context.DeadlineExceeded
		},
	}
	_, err := a.GetWorkItem(context.Background(), 123)
	require.Error(t, err)
}
