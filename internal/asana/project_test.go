package asana

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ADO-Asana-Sync/sync-engine/internal/testutil"
	asanaapi "github.com/qw4n7y/go-asana/asana"
	"github.com/stretchr/testify/require"
)

func createProjectListResponse(projects []asanaapi.Project, respErr asanaapi.Errors) *http.Response {
	mockResp := asanaapi.Response{Data: projects, NextPage: nil, Errors: respErr}
	b, err := json.Marshal(mockResp)
	if err != nil {
		panic(err)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}
}

func TestAsanaListProjects(t *testing.T) {
	projResp := createProjectListResponse([]asanaapi.Project{{GID: "p1", Name: "Proj"}}, nil)
	wsResp := createWorkspaceResponse([]asanaapi.Workspace{{ID: 1, GID: "1", Name: "Acme"}}, nil)
	badResp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("oops")), Header: make(http.Header)}

	tests := []struct {
		name          string
		wsResp        *http.Response
		projResp      *http.Response
		projErr       error
		workspaceName string
		want          []Project
		wantErr       bool
	}{
		{
			name:          "success",
			wsResp:        wsResp,
			projResp:      projResp,
			workspaceName: "Acme",
			want:          []Project{{GID: "p1", Name: "Proj"}},
		},
		{
			name:          "workspace not found",
			wsResp:        createWorkspaceResponse([]asanaapi.Workspace{}, nil),
			projResp:      projResp,
			workspaceName: "Missing",
			wantErr:       true,
		},
		{
			name:          "api error",
			wsResp:        wsResp,
			projResp:      badResp,
			workspaceName: "Acme",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var call int
			client := &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				call++
				if call == 1 {
					return tt.wsResp, nil
				}
				return tt.projResp, tt.projErr
			})}
			a := &Asana{Client: client}
			got, err := a.ListProjects(context.Background(), tt.workspaceName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
