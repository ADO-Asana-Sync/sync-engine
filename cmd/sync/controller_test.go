package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel"
)

type mockDB struct{ wrote bool }

func (m *mockDB) Connect(ctx context.Context, uri string) error                  { return nil }
func (m *mockDB) Disconnect(ctx context.Context) error                           { return nil }
func (m *mockDB) Projects(ctx context.Context) ([]db.Project, error)             { return nil, nil }
func (m *mockDB) AddProject(ctx context.Context, project db.Project) error       { return nil }
func (m *mockDB) RemoveProject(ctx context.Context, id primitive.ObjectID) error { return nil }
func (m *mockDB) UpdateProject(ctx context.Context, project db.Project) error    { return nil }
func (m *mockDB) LastSync(ctx context.Context) db.LastSync                       { return db.LastSync{} }
func (m *mockDB) WriteLastSync(ctx context.Context, timestamp time.Time) error {
	m.wrote = true
	return nil
}
func (m *mockDB) TaskByADOTaskID(ctx context.Context, id int) (db.TaskMapping, error) {
	return db.TaskMapping{}, nil
}
func (m *mockDB) AddTask(ctx context.Context, task db.TaskMapping) error    { return nil }
func (m *mockDB) UpdateTask(ctx context.Context, task db.TaskMapping) error { return nil }
func (m *mockDB) GetCacheItem(ctx context.Context, key string) (db.CacheItem, error) {
	return db.CacheItem{}, fmt.Errorf("not found")
}
func (m *mockDB) UpsertCacheItem(ctx context.Context, item db.CacheItem) error { return nil }
func (m *mockDB) WorkspaceTag(ctx context.Context, workspaceName string) (db.WorkspaceTag, error) {
	return db.WorkspaceTag{}, fmt.Errorf("not found")
}
func (m *mockDB) UpsertWorkspaceTag(ctx context.Context, tag db.WorkspaceTag) error { return nil }

type mockAzure struct{}

func (m *mockAzure) Connect(ctx context.Context, orgUrl, pat string) {}
func (m *mockAzure) GetChangedWorkItems(ctx context.Context, lastSync time.Time) ([]workitemtracking.WorkItemReference, error) {
	return nil, nil
}
func (m *mockAzure) GetWorkItem(ctx context.Context, id int) (azure.WorkItem, error) {
	return azure.WorkItem{}, nil
}
func (m *mockAzure) GetProjects(ctx context.Context) ([]core.TeamProjectReference, error) {
	return nil, nil
}

func TestControllerWritesLastSync(t *testing.T) {
	app := &App{
		Azure:  &mockAzure{},
		DB:     &mockDB{},
		Tracer: otel.Tracer("test"),
	}
	taskCh := make(chan SyncTask)
	go func() {
		// drain channel if any tasks sent
		for range taskCh {
		}
	}()
	ctx := context.Background()
	app.controller(ctx, taskCh)
	md := app.DB.(*mockDB)
	if !md.wrote {
		t.Fatalf("expected WriteLastSync to be called")
	}
}
