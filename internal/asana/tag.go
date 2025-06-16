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

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	asanaapi "github.com/qw4n7y/go-asana/asana"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tag represents minimal information about an Asana tag.
type Tag struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// TagByName finds a tag in the given workspace by its name.
func (a *Asana) TagByName(ctx context.Context, workspaceName, tagName string) (Tag, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.TagByName")
	defer span.End()

	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return Tag{}, err
	}
	var wsID int64
	found := false
	for _, ws := range workspaces {
		if ws.Name == workspaceName {
			wsID = ws.ID
			found = true
			break
		}
	}
	if !found {
		err := fmt.Errorf("workspace not found")
		span.SetStatus(codes.Error, err.Error())
		return Tag{}, err
	}

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tags, err := client.ListTags(ctx, &asanaapi.Filter{Workspace: wsID})
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return Tag{}, err
	}

	var (
		foundTag Tag
		minID    int64 = 1<<63 - 1
	)
	for _, t := range tags {
		if t.Name != tagName {
			continue
		}
		var id int64
		if t.GID != "" {
			if v, err := strconv.ParseInt(t.GID, 10, 64); err == nil {
				id = v
			}
		}
		if id == 0 {
			id = t.ID
		}
		if id < minID {
			minID = id
			if t.GID != "" {
				foundTag = Tag{GID: t.GID, Name: t.Name}
			} else {
				foundTag = Tag{GID: fmt.Sprint(t.ID), Name: t.Name}
			}
		}
	}
	if foundTag.GID != "" {
		return foundTag, nil
	}
	err = fmt.Errorf("tag not found")
	span.SetStatus(codes.Error, err.Error())
	return Tag{}, err
}

// AddTagToTask adds a tag to the specified task.
func (a *Asana) AddTagToTask(ctx context.Context, taskGID, tagGID string) error {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.AddTagToTask")
	defer span.End()

	client := asanaapi.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := map[string]map[string]string{
		"data": {"tag": tagGID},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	u := client.BaseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tasks/%s/addTag", taskGID)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana add tag failed: %s", string(body))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
