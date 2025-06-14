package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"
	asanaapi "github.com/qw4n7y/go-asana/asana"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// CustomField represents minimal information about an Asana custom field.
type CustomField struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// CustomFieldByName finds a custom field in the given workspace by its name.
func (a *Asana) CustomFieldByName(ctx context.Context, workspaceName, fieldName string) (CustomField, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.CustomFieldByName")
	defer span.End()

	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return CustomField{}, err
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
		return CustomField{}, err
	}

	client := asanaapi.NewClient(a.Client)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	u := client.BaseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("workspaces/%d/custom_fields", wsID)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana custom fields request failed: %s", string(body))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	var result struct {
		Data []CustomField `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	for _, f := range result.Data {
		if f.Name == fieldName {
			return f, nil
		}
	}
	err = fmt.Errorf("custom field not found")
	span.SetStatus(codes.Error, err.Error())
	return CustomField{}, err
}
