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

// createCustomFieldResponse creates a mock HTTP response for a list of custom fields.
func createCustomFieldResponse(fields []CustomField) *http.Response {
	payload := struct {
		Data []CustomField `json:"data"`
	}{Data: fields}
	b, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(b))),
		Header:     make(http.Header),
	}
}

func TestAsanaCustomFieldByName(t *testing.T) {
	wsResp := createWorkspaceResponse([]asanaapi.Workspace{{ID: 1, Name: "Acme"}}, nil)
	fieldResp := createCustomFieldResponse([]CustomField{{GID: "f1", Name: "Priority"}})
	badResp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("oops")), Header: make(http.Header)}

	tests := []struct {
		name          string
		wsResp        *http.Response
		fieldResp     *http.Response
		fieldErr      error
		workspaceName string
		fieldName     string
		want          CustomField
		wantErr       bool
	}{
		{
			name:          "found",
			wsResp:        wsResp,
			fieldResp:     fieldResp,
			workspaceName: "Acme",
			fieldName:     "Priority",
			want:          CustomField{GID: "f1", Name: "Priority"},
		},
		{
			name:          "workspace not found",
			wsResp:        createWorkspaceResponse([]asanaapi.Workspace{}, nil),
			fieldResp:     fieldResp,
			workspaceName: "Missing",
			fieldName:     "Priority",
			wantErr:       true,
		},
		{
			name:          "api error",
			wsResp:        wsResp,
			fieldResp:     badResp,
			workspaceName: "Acme",
			fieldName:     "Priority",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var call int
			client := &http.Client{
				Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return tt.wsResp, nil
					}
					return tt.fieldResp, tt.fieldErr
				}),
			}
			a := &Asana{Client: client}
			got, err := a.CustomFieldByName(context.Background(), tt.workspaceName, tt.fieldName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
