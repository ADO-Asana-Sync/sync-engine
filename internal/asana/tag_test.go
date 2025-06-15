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

func createTagListResponse(tags []asanaapi.Tag, respErr asanaapi.Errors) *http.Response {
	mockResp := asanaapi.Response{
		Data:     tags,
		NextPage: nil,
		Errors:   respErr,
	}
	b, err := json.Marshal(mockResp)
	if err != nil {
		panic(err)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}
}

func TestAsanaTagByName(t *testing.T) {
	wsResp := createWorkspaceResponse([]asanaapi.Workspace{{ID: 1, Name: "Acme"}}, nil)
	tagResp := createTagListResponse([]asanaapi.Tag{{GID: "1", Name: "synced"}}, nil)
	badResp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("oops")), Header: make(http.Header)}

	tests := []struct {
		name          string
		wsResp        *http.Response
		tagResp       *http.Response
		tagErr        error
		workspaceName string
		tagName       string
		want          Tag
		wantErr       bool
	}{
		{name: "found", wsResp: wsResp, tagResp: tagResp, workspaceName: "Acme", tagName: "synced", want: Tag{GID: "1", Name: "synced"}},
		{name: "workspace not found", wsResp: createWorkspaceResponse([]asanaapi.Workspace{}, nil), tagResp: tagResp, workspaceName: "Missing", tagName: "synced", wantErr: true},
		{name: "tag missing", wsResp: wsResp, tagResp: createTagListResponse([]asanaapi.Tag{}, nil), workspaceName: "Acme", tagName: "synced", wantErr: true},
		{name: "api error", wsResp: wsResp, tagResp: badResp, workspaceName: "Acme", tagName: "synced", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var call int
			client := &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				call++
				if call == 1 {
					return tt.wsResp, nil
				}
				return tt.tagResp, tt.tagErr
			})}
			a := &Asana{Client: client}
			got, err := a.TagByName(context.Background(), tt.workspaceName, tt.tagName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAsanaAddTagToTask(t *testing.T) {
	successResp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}
	failResp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("oops")), Header: make(http.Header)}

	tests := []struct {
		name    string
		resp    *http.Response
		respErr error
		wantErr bool
	}{
		{name: "success", resp: successResp},
		{name: "http error", resp: failResp, wantErr: true},
		{name: "client error", resp: nil, respErr: context.DeadlineExceeded, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			client := testutil.NewTestClientWithRequest(tt.resp, tt.respErr, &req)
			a := &Asana{Client: client}
			err := a.AddTagToTask(context.Background(), "1", "2")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, req)
			require.Equal(t, http.MethodPost, req.Method)
		})
	}
}
