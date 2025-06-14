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

// createTaskListResponse creates a mock HTTP response for a list of tasks.
func createTaskListResponse(tasks []asanaapi.Task, respErr asanaapi.Errors) *http.Response {
	mockResp := asanaapi.Response{
		Data:     tasks,
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

// createTaskResponse creates a mock HTTP response for a single task.
func createTaskResponse(task asanaapi.Task, respErr asanaapi.Errors) *http.Response {
	mockResp := asanaapi.Response{
		Data:     task,
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

// TestAsanaListProjectTasks verifies that ListProjectTasks correctly parses the API response
// and handles error scenarios including invalid project IDs and API errors.
func TestAsanaListProjectTasks(t *testing.T) {
	tests := []struct {
		name       string
		projectGID string
		tasks      []asanaapi.Task
		respErr    asanaapi.Errors
		expected   []Task
		wantErr    bool
	}{
		{
			name:       "Single task",
			projectGID: "42",
			tasks:      []asanaapi.Task{{GID: "1", Name: "Task 1"}},
			respErr:    nil,
			expected:   []Task{{GID: "1", Name: "Task 1"}},
			wantErr:    false,
		},
		{
			name:       "Multiple tasks",
			projectGID: "42",
			tasks: []asanaapi.Task{
				{GID: "1", Name: "Task 1"},
				{GID: "2", Name: "Task 2"},
			},
			respErr: nil,
			expected: []Task{
				{GID: "1", Name: "Task 1"},
				{GID: "2", Name: "Task 2"},
			},
			wantErr: false,
		},
		{
			name:       "Invalid projectGID",
			projectGID: "abc",
			wantErr:    true,
		},
		{
			name:       "API error",
			projectGID: "42",
			respErr:    asanaapi.Errors{asanaapi.Error{Message: "bad request"}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *http.Client
			if tt.projectGID != "abc" {
				resp := createTaskListResponse(tt.tasks, tt.respErr)
				client = testutil.NewTestClient(resp, nil)
			}
			a := &Asana{Client: client}
			got, err := a.ListProjectTasks(context.Background(), tt.projectGID)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

// TestAsanaCreateTask verifies CreateTask correctly handles success and error scenarios.
func TestAsanaCreateTask(t *testing.T) {
	tests := []struct {
		name    string
		task    asanaapi.Task
		respErr asanaapi.Errors
		want    Task
		wantErr bool
	}{
		{
			name:    "Success",
			task:    asanaapi.Task{GID: "1", Name: "Created"},
			respErr: nil,
			want:    Task{GID: "1", Name: "Created"},
			wantErr: false,
		},
		{
			name:    "API error",
			task:    asanaapi.Task{},
			respErr: asanaapi.Errors{asanaapi.Error{Message: "failed"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createTaskResponse(tt.task, tt.respErr)
			var req *http.Request
			client := testutil.NewTestClientWithRequest(resp, nil, &req)
			a := &Asana{Client: client}
			got, err := a.CreateTask(context.Background(), "42", tt.task.Name, "notes")
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NotNil(t, req)
			require.Equal(t, http.MethodPost, req.Method)
			require.NoError(t, req.ParseForm())
			require.Equal(t, "42", req.Form.Get("projects"))
			require.Equal(t, tt.task.Name, req.Form.Get("name"))
			require.Equal(t, "<body>notes</body>", req.Form.Get("html_notes"))
		})
	}
}

// TestAsanaUpdateTask verifies that UpdateTask sends the request and handles HTTP status codes correctly.
func TestAsanaUpdateTask(t *testing.T) {
	successResp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}
	failResp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("oops")), Header: make(http.Header)}

	tests := []struct {
		name    string
		resp    *http.Response
		respErr error
		wantErr bool
	}{
		{
			name:    "Success",
			resp:    successResp,
			respErr: nil,
			wantErr: false,
		},
		{
			name:    "HTTP error",
			resp:    failResp,
			respErr: nil,
			wantErr: true,
		},
		{
			name:    "Client error",
			resp:    nil,
			respErr: context.DeadlineExceeded,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			client := testutil.NewTestClientWithRequest(tt.resp, tt.respErr, &req)
			a := &Asana{Client: client}
			err := a.UpdateTask(context.Background(), "1", "name", "notes")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, req)
			body, _ := io.ReadAll(req.Body)
			var payload struct {
				Data struct {
					HTML string `json:"html_notes"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Equal(t, "<body>notes</body>", payload.Data.HTML)
		})
	}
}
