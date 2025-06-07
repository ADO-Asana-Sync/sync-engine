package asana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	asanaapi "github.com/range-labs/go-asana/asana"
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

func (a *Asana) CreateTask(ctx context.Context, projectGID, name, notes string) (Task, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.CreateTask")
	defer span.End()

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fields := map[string]string{
		"projects": projectGID,
		"name":     name,
		"notes":    notes,
	}
	t, err := client.CreateTask(ctx, fields, nil)
	if err != nil {
		return Task{}, err
	}
	return Task{GID: t.GID, Name: t.Name}, nil
}

func (a *Asana) UpdateTask(ctx context.Context, taskGID, name, notes string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.UpdateTask")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := map[string]map[string]string{"data": {"name": name, "notes": notes}}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("https://app.asana.com/api/1.0/tasks/%s", taskGID), bytes.NewReader(b))
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
