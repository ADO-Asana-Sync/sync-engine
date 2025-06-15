package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	var fields []CustomField

	if err := client.Request(ctx, fmt.Sprintf("workspaces/%d/custom_fields", wsID), nil, &fields); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	for _, f := range fields {
		if f.Name == fieldName {
			return f, nil
		}
	}
	err = fmt.Errorf("custom field not found")
	span.SetStatus(codes.Error, err.Error())
	return CustomField{}, err
}

// ProjectHasCustomField reports whether the given project contains a custom field
// with the specified name.
func (a *Asana) ProjectHasCustomField(ctx context.Context, projectGID, fieldName string) (bool, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ProjectHasCustomField")
	defer span.End()

	client := asanaapi.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var settings []struct {
		CustomField CustomField `json:"custom_field"`
	}

	if err := client.Request(ctx, fmt.Sprintf("projects/%s/custom_field_settings", projectGID), nil, &settings); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	for _, s := range settings {
		if s.CustomField.Name == fieldName {
			return true, nil
		}
	}
	return false, nil
}

// ProjectCustomFieldByName retrieves the custom field matching fieldName from
// the given project. The name comparison is case-insensitive.
func (a *Asana) ProjectCustomFieldByName(ctx context.Context, projectGID, fieldName string) (CustomField, error) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.ProjectCustomFieldByName")
	defer span.End()

	client := asanaapi.NewClient(a.Client)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	u := client.BaseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("projects/%s/custom_field_settings", projectGID)})
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

	if resp.StatusCode == http.StatusPaymentRequired {
		return CustomField{}, fmt.Errorf("custom fields unavailable")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("asana request failed: %s", string(body))
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	var payload struct {
		Data []struct {
			CustomField CustomField `json:"custom_field"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return CustomField{}, err
	}

	lname := strings.ToLower(fieldName)
	for _, s := range payload.Data {
		if strings.ToLower(s.CustomField.Name) == lname {
			return s.CustomField, nil
		}
	}
	err = fmt.Errorf("custom field not found")
	span.SetStatus(codes.Error, err.Error())
	return CustomField{}, err
}
