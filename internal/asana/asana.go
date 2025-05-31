package asana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

// AsanaInterface defines the methods that the Asana client must implement.
type AsanaInterface interface {
	Connect(ctx context.Context, pat string)
	// ListWorkspaces(ctx context.Context) ([]Workspace, error) // Assuming Workspace is defined elsewhere or will be
	UpdateTask(ctx context.Context, taskID string, name string, htmlDescription string) (*Task, error)
	FindTaskByTitle(ctx context.Context, projectID string, title string) (*Task, error)
	CreateTask(ctx context.Context, projectID string, name string, htmlDescription string) (*Task, error)
}

type Asana struct {
	Client *http.Client
}

func (a *Asana) Connect(ctx context.Context, pat string) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.Connect")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	a.Client = conf.Client(ctx, tok)
}

const (
	asanaAPIBaseURL = "https://app.asana.com/api/1.0"
)

// Task represents a simplified Asana task.
type Task struct {
	GID          string `json:"gid"`
	Name         string `json:"name"`
	HTMLNotes    string `json:"html_notes,omitempty"`
	PermalinkURL string `json:"permalink_url,omitempty"`
	// Consider adding 'Projects' if needed for context, though not strictly required by prompt for return
}

// UpdateTask updates an existing Asana task's name and description.
// API: PUT /tasks/{task_gid}
func (a *Asana) UpdateTask(ctx context.Context, taskID string, name string, htmlDescription string) (*Task, error) {
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.UpdateTask")
	defer span.End()

	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/tasks/%s", asanaAPIBaseURL, taskID)

	requestBody := map[string]interface{}{
		"data": map[string]interface{}{
			"name":       name,
			"html_notes": htmlDescription,
		},
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var responsePayload struct {
		Data Task `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responsePayload); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &responsePayload.Data, nil
}

// FindTaskByTitle searches for an Asana task within a specific project by its exact title.
// API: GET /projects/{project_gid}/tasks?opt_fields=name,permalink_url,projects (or more fields as needed)
// or GET /tasks?project={project_gid}&opt_fields=... and filter client-side.
// For simplicity, we'll fetch tasks for the project and filter.
func (a *Asana) FindTaskByTitle(ctx context.Context, projectID string, title string) (*Task, error) {
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.FindTaskByTitle")
	defer span.End()

	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty")
	}
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	// Asana API typically requires opt_fields to get non-default fields.
	// permalink_url is an important one for our Task struct.
	queryParams := url.Values{}
	queryParams.Add("opt_fields", "name,html_notes,permalink_url")
	// We could also filter by completed_since, modified_since, etc. if needed.

	apiURL := fmt.Sprintf("%s/projects/%s/tasks?%s", asanaAPIBaseURL, projectID, queryParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Project not found, or tasks endpoint not found (less likely)
		err := fmt.Errorf("asana API error: project %s not found (status %d)", projectID, resp.StatusCode)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err // Or a specific not found error type
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var responsePayload struct {
		Data []Task `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responsePayload); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	for _, task := range responsePayload.Data {
		if task.Name == title {
			span.SetAttributes(trace.StringAttribute("asana.task.found_gid", task.GID))
			return &task, nil // Found the task
		}
	}

	// No task with the exact title found
	span.SetAttributes(trace.BoolAttribute("asana.task.found", false))
	return nil, nil // As per requirement: If no task is found, return nil for the task and no error.
}

// CreateTask creates a new Asana task in a specific project.
// API: POST /tasks
func (a *Asana) CreateTask(ctx context.Context, projectID string, name string, htmlDescription string) (*Task, error) {
	_, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.CreateTask")
	defer span.End()

	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty for creating a task")
	}
	if name == "" {
		// Asana might allow tasks with empty names, but it's generally not desired.
		// Depending on requirements, this check can be enforced or removed.
		return nil, fmt.Errorf("task name cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/tasks", asanaAPIBaseURL)

	requestBody := map[string]interface{}{
		"data": map[string]interface{}{
			"name":       name,
			"html_notes": htmlDescription,
			"projects":   []string{projectID}, // Assign to the specified project
		},
	}
	// Note: opt_fields in POST body is a way to specify response fields for some Asana endpoints.
	// Alternatively, some POST responses return the full object by default, or one might make a GET request after POST.
	// Checking Asana docs: for POST /tasks, the created task object is returned. opt_fields can be passed as query params too.
	// For consistency and to ensure we get what we need, let's pass opt_fields as query parameters.

	queryParams := url.Values{}
	queryParams.Add("opt_fields", "name,html_notes,permalink_url")
	apiURLWithParams := fmt.Sprintf("%s?%s", apiURL, queryParams.Encode())


	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURLWithParams, bytes.NewBuffer(jsonBody))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana API error creating task: status %d, body: %s", resp.StatusCode, string(bodyBytes))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var responsePayload struct {
		Data Task `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responsePayload); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &responsePayload.Data, nil
}
