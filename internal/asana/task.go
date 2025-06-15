package asana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	asanaapi "github.com/qw4n7y/go-asana/asana"
)

// Task represents minimal information about an Asana task.
type Task struct {
	GID  string
	Name string
}

func (a *Asana) ListProjectTasks(ctx context.Context, projectGID string) ([]Task, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ListProjectTasks")
	defer span.End()

	client := asanaapi.NewClient(a.Client)
	pid, err := strconv.ParseInt(projectGID, 10, 64)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tasks, err := client.ListProjectTasks(ctx, pid, nil)
	if err != nil {
		return nil, err
	}

	var result []Task
	for _, t := range tasks {
		result = append(result, Task{GID: t.GID, Name: t.Name})
	}
	return result, nil
}

// CreateTask creates a new task in the given project using HTML notes for the description.
func (a *Asana) CreateTask(ctx context.Context, projectGID, name, notes string) (Task, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.CreateTask")
	defer span.End()

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	notes = ensureHTMLBody(notes)

	fields := map[string]string{
		"projects":   projectGID,
		"name":       name,
		"html_notes": notes,
	}
	t, err := client.CreateTask(ctx, fields, nil)
	if err != nil {
		return Task{}, err
	}
	return Task{GID: t.GID, Name: t.Name}, nil
}

// UpdateTask updates an existing task using HTML notes for the description.
func (a *Asana) UpdateTask(ctx context.Context, taskGID, name, notes string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.UpdateTask")
	defer span.End()

	client := asanaapi.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	notes = ensureHTMLBody(notes)

	payload := map[string]map[string]string{
		"data": {
			"name":       name,
			"html_notes": notes,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	u := client.BaseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tasks/%s", taskGID)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asana update failed: %s", string(body))
	}
	return nil
}

// CreateTaskWithCustomFields creates a task and sets the provided custom fields.
func (a *Asana) CreateTaskWithCustomFields(ctx context.Context, projectGID, name, notes string, customFields map[string]string) (Task, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.CreateTaskWithCustomFields")
	defer span.End()

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	notes = ensureHTMLBody(notes)

	body := asanaapi.NewTask{
		Name:         name,
		HTMLNotes:    notes,
		Projects:     []string{projectGID},
		CustomFields: customFields,
	}

	t, err := client.CreateTask2(ctx, body, nil)
	if err != nil {
		return Task{}, err
	}
	return Task{GID: t.GID, Name: t.Name}, nil
}

// UpdateTaskWithCustomFields updates a task and sets custom field values.
func (a *Asana) UpdateTaskWithCustomFields(ctx context.Context, taskGID, name, notes string, customFields map[string]string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.UpdateTaskWithCustomFields")
	defer span.End()

	client := asanaapi.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	notes = ensureHTMLBody(notes)

	payload := map[string]map[string]interface{}{
		"data": {
			"name":          name,
			"html_notes":    notes,
			"custom_fields": customFields,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	u := client.BaseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tasks/%s", taskGID)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asana update failed: %s", string(body))
	}
	return nil
}

// ensureHTMLBody wraps the provided notes in a <body> element if one is not already present.
func ensureHTMLBody(notes string) string {
	lower := strings.ToLower(notes)
	if strings.Contains(lower, "<body") {
		return notes
	}
	return "<body>" + notes + "</body>"
}
