package asana

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ADO-Asana-Sync/sync-engine/internal/testutil"
	"github.com/range-labs/go-asana/asana"
	"github.com/stretchr/testify/require"
)

// createWorkspaceResponse is a helper function to create a mock HTTP response from a provided object.
func createWorkspaceResponse(workspaces []asana.Workspace, respErr asana.Errors) *http.Response {
	mockResp := asana.Response{
		Data:     workspaces,
		NextPage: nil,
		Errors:   respErr,
	}
	jsonBytes, err := json.Marshal(mockResp)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(jsonBytes))),
		Header:     make(http.Header),
	}
}

// TestAsanaListWorkspaces tests the ListWorkspaces method of the Asana client.
// It verifies that the method correctly handles various scenarios, including:
// - Returning a single workspace
// - Returning multiple workspaces
// - Handling error responses from the Asana API
// - Handling empty workspace lists
// The test uses a table-driven approach to cover these cases and asserts that
// the returned workspaces and errors match the expected results.
func TestAsanaListWorkspaces(t *testing.T) {
	tests := []struct {
		name       string
		workspaces []asana.Workspace
		respErr    asana.Errors
		expected   []Workspace
	}{
		{
			name: "Single workspace",
			workspaces: []asana.Workspace{
				{ID: 42, GID: "012345647", Name: "AcmeCorp"},
			},
			respErr: nil,
			expected: []Workspace{
				{ID: 42, Name: "AcmeCorp"},
			},
		},
		{
			name: "Multiple workspaces",
			workspaces: []asana.Workspace{
				{ID: 42, GID: "012345647", Name: "AcmeCorp"},
				{ID: 43, GID: "012345648", Name: "TechCorp"},
			},
			respErr: nil,
			expected: []Workspace{
				{ID: 42, Name: "AcmeCorp"},
				{ID: 43, Name: "TechCorp"},
			},
		},
		{
			name:       "Error response",
			workspaces: nil,
			respErr:    asana.Errors{asana.Error{Message: "Invalid token"}},
			expected:   nil,
		},
		{
			name:       "Empty response",
			workspaces: []asana.Workspace{},
			respErr:    nil,
			expected:   []Workspace{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockResp := createWorkspaceResponse(test.workspaces, test.respErr)
			fakeClient := testutil.NewTestClient(mockResp, nil)
			a := &Asana{Client: fakeClient}
			wss, err := a.ListWorkspaces(context.Background())
			if test.respErr != nil {
				require.Error(t, err)
				require.Nil(t, wss)
				return
			}
			require.NoError(t, err)
			require.Len(t, wss, len(test.expected))
			for i, ws := range wss {
				require.Equal(t, test.expected[i].ID, ws.ID)
				require.Equal(t, test.expected[i].Name, ws.Name)
			}
		})
	}
}
