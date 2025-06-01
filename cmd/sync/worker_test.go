package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"go.opentelemetry.io/otel/trace"
)

// --- Model definitions (assuming they are not in easily importable packages) ---
// These should mirror the definitions in worker.go or a shared models.go

type TaskMapping struct { // Ensure this matches the definition in worker.go
	ADOTaskID   int
	AsanaTaskID string
	SyncedAt    time.Time
}

type SyncTask struct { // Ensure this matches the definition in worker.go
	ADOTaskID       int
	AsanaProjectGID string // Asana Project GID
	ADOProjectID    string // ADO Project ID
}

// --- Interface definitions for mocks ---

type MockDBInterface interface {
	GetTaskMappingByADOTaskID(ctx context.Context, adoTaskID int) (*TaskMapping, error)
	UpdateTaskMapping(ctx context.Context, tm *TaskMapping) error
	CreateTaskMapping(ctx context.Context, tm *TaskMapping) error
}

// --- Mock Implementations ---

// MockDB is a mock implementation of DBInterface
type MockDB struct {
	GetTaskMappingByADOTaskIDFunc func(ctx context.Context, adoTaskID int) (*TaskMapping, error)
	UpdateTaskMappingFunc         func(ctx context.Context, tm *TaskMapping) error
	CreateTaskMappingFunc         func(ctx context.Context, tm *TaskMapping) error

	// Call trackers
	GetTaskMappingByADOTaskIDCalledWith int
	UpdateTaskMappingCalledWith         *TaskMapping
	CreateTaskMappingCalledWith         *TaskMapping
}

func (m *MockDB) GetTaskMappingByADOTaskID(ctx context.Context, adoTaskID int) (*TaskMapping, error) {
	m.GetTaskMappingByADOTaskIDCalledWith = adoTaskID
	if m.GetTaskMappingByADOTaskIDFunc != nil {
		return m.GetTaskMappingByADOTaskIDFunc(ctx, adoTaskID)
	}
	return nil, fmt.Errorf("GetTaskMappingByADOTaskIDFunc not set")
}

func (m *MockDB) UpdateTaskMapping(ctx context.Context, tm *TaskMapping) error {
	m.UpdateTaskMappingCalledWith = tm
	if m.UpdateTaskMappingFunc != nil {
		return m.UpdateTaskMappingFunc(ctx, tm)
	}
	return fmt.Errorf("UpdateTaskMappingFunc not set")
}

func (m *MockDB) CreateTaskMapping(ctx context.Context, tm *TaskMapping) error {
	m.CreateTaskMappingCalledWith = tm
	if m.CreateTaskMappingFunc != nil {
		return m.CreateTaskMappingFunc(ctx, tm)
	}
	return fmt.Errorf("CreateTaskMappingFunc not set")
}

// MockAzureClient is a mock implementation of azure.AzureInterface
type MockAzureClient struct {
	GetWorkItemDetailsFunc func(ctx context.Context, workItemID int) (*azure.WorkItem, error)
	// Call trackers
	GetWorkItemDetailsCalledWith int
}

func (m *MockAzureClient) GetWorkItemDetails(ctx context.Context, workItemID int) (*azure.WorkItem, error) {
	m.GetWorkItemDetailsCalledWith = workItemID
	if m.GetWorkItemDetailsFunc != nil {
		return m.GetWorkItemDetailsFunc(ctx, workItemID)
	}
	return nil, fmt.Errorf("GetWorkItemDetailsFunc not set")
}
// Dummy methods to satisfy azure.AzureInterface if it has more methods
func (m *MockAzureClient) Connect(ctx context.Context, orgUrl, pat string) {}
// Remove other methods like GetChangedWorkItems, GetProjects if they cause import issues and are not used by worker

// MockAsanaClient is a mock implementation of asana.AsanaInterface
type MockAsanaClient struct {
	UpdateTaskFunc      func(ctx context.Context, taskID string, name string, htmlDescription string) (*asana.Task, error)
	FindTaskByTitleFunc func(ctx context.Context, projectID string, title string) (*asana.Task, error)
	CreateTaskFunc      func(ctx context.Context, projectID string, name string, htmlDescription string) (*asana.Task, error)

	// Call trackers
	UpdateTaskCalledWithID          string
	UpdateTaskCalledWithName        string
	UpdateTaskCalledWithDescription string
	FindTaskByTitleCalledWithProjectID string
	FindTaskByTitleCalledWithTitle  string
	CreateTaskCalledWithProjectID   string
	CreateTaskCalledWithName        string
	CreateTaskCalledWithDescription string
}

func (m *MockAsanaClient) UpdateTask(ctx context.Context, taskID string, name string, htmlDescription string) (*asana.Task, error) {
	m.UpdateTaskCalledWithID = taskID
	m.UpdateTaskCalledWithName = name
	m.UpdateTaskCalledWithDescription = htmlDescription
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, taskID, name, htmlDescription)
	}
	return nil, fmt.Errorf("UpdateTaskFunc not set")
}

func (m *MockAsanaClient) FindTaskByTitle(ctx context.Context, projectID string, title string) (*asana.Task, error) {
	m.FindTaskByTitleCalledWithProjectID = projectID
	m.FindTaskByTitleCalledWithTitle = title
	if m.FindTaskByTitleFunc != nil {
		return m.FindTaskByTitleFunc(ctx, projectID, title)
	}
	return nil, fmt.Errorf("FindTaskByTitleFunc not set")
}

func (m *MockAsanaClient) CreateTask(ctx context.Context, projectID string, name string, htmlDescription string) (*asana.Task, error) {
	m.CreateTaskCalledWithProjectID = projectID
	m.CreateTaskCalledWithName = name
	m.CreateTaskCalledWithDescription = htmlDescription
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, projectID, name, htmlDescription)
	}
	return nil, fmt.Errorf("CreateTaskFunc not set")
}
// Dummy methods to satisfy asana.AsanaInterface
func (m *MockAsanaClient) Connect(ctx context.Context, pat string) {}

// Minimal App struct for testing (redefined for tests to use mock types directly)
type App struct {
	db          *MockDB // Use concrete mock type
	azureClient *MockAzureClient // Use concrete mock type
	asanaClient *MockAsanaClient // Use concrete mock type
	Tracer      trace.Tracer
}


func TestWorker_ExistingMapping_UpdateSuccess(t *testing.T) {
	mockDb := &MockDB{}
	mockAzure := &MockAzureClient{}
	mockAsana := &MockAsanaClient{}

	// Ensure the test App uses the concrete mock types
	app := &App{
		db:          mockDb,
		azureClient: mockAzure,
		asanaClient: mockAsana,
		Tracer:      trace.NewNoopTracerProvider().Tracer("test"),
	}

	syncTask := SyncTask{ADOTaskID: 123, AsanaProjectGID: "as_project1", ADOProjectID: "ado_proj_abc"}
	taskMapping := &TaskMapping{ADOTaskID: 123, AsanaTaskID: "asana_task1", SyncedAt: time.Now().Add(-time.Hour)}
	adoWorkItem := &azure.WorkItem{
		ID:           123,
		Title:        "ADO Task Title",
		WorkItemType: "Bug",
		URL:          "http://ado/bug/123",
		// Fields for FormatTitle and FormatTitleWithLink must be set
		ChangedDate: time.Now(),
		CreatedDate: time.Now().Add(-2 * time.Hour),
		State:       "Active",
	}
	// Expected formatted titles based on azure.WorkItem methods
	expectedAsanaName, _ := adoWorkItem.FormatTitle()
	expectedAsanaDesc, _ := adoWorkItem.FormatTitleWithLink()

	mockDb.GetTaskMappingByADOTaskIDFunc = func(ctx context.Context, adoTaskID int) (*TaskMapping, error) {
		if adoTaskID != syncTask.ADOTaskID {
			t.Errorf("Expected GetTaskMappingByADOTaskID to be called with %d, got %d", syncTask.ADOTaskID, adoTaskID)
		}
		return taskMapping, nil
	}
	mockAzure.GetWorkItemDetailsFunc = func(ctx context.Context, workItemID int) (*azure.WorkItem, error) {
		if workItemID != syncTask.ADOTaskID {
			t.Errorf("Expected GetWorkItemDetails to be called with %d, got %d", syncTask.ADOTaskID, workItemID)
		}
		return adoWorkItem, nil
	}
	mockAsana.UpdateTaskFunc = func(ctx context.Context, taskID string, name string, htmlDescription string) (*asana.Task, error) {
		if taskID != taskMapping.AsanaTaskID {
			t.Errorf("Expected UpdateTask Asana ID %s, got %s", taskMapping.AsanaTaskID, taskID)
		}
		if name != expectedAsanaName {
			t.Errorf("Expected UpdateTask name '%s', got '%s'", expectedAsanaName, name)
		}
		if htmlDescription != expectedAsanaDesc {
			t.Errorf("Expected UpdateTask description '%s', got '%s'", expectedAsanaDesc, htmlDescription)
		}
		return &asana.Task{GID: taskID, Name: name, HTMLNotes: htmlDescription}, nil
	}
	mockDb.UpdateTaskMappingFunc = func(ctx context.Context, tm *TaskMapping) error {
		if tm.ADOTaskID != syncTask.ADOTaskID {
			t.Errorf("Expected UpdateTaskMapping ADOTaskID %d, got %d", syncTask.ADOTaskID, tm.ADOTaskID)
		}
		if tm.AsanaTaskID != taskMapping.AsanaTaskID {
			t.Errorf("Expected UpdateTaskMapping AsanaTaskID %s, got %s", taskMapping.AsanaTaskID, tm.AsanaTaskID)
		}
		return nil
	}

	syncTasksChan := make(chan SyncTask, 1)
	syncTasksChan <- syncTask
	close(syncTasksChan) // Close channel to make worker exit after processing one task

	app.worker(context.Background(), 1, syncTasksChan)

	// Verify calls (basic check, could be more robust with call counters or testify/mock)
	if mockDb.GetTaskMappingByADOTaskIDCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetTaskMappingByADOTaskID was not called correctly or at all")
	}
	if mockAzure.GetWorkItemDetailsCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetWorkItemDetails was not called correctly or at all")
	}
	if mockAsana.UpdateTaskCalledWithID != taskMapping.AsanaTaskID {
		t.Errorf("UpdateTask was not called correctly or at all")
	}
	if mockDb.UpdateTaskMappingCalledWith == nil || mockDb.UpdateTaskMappingCalledWith.ADOTaskID != syncTask.ADOTaskID {
		t.Errorf("UpdateTaskMapping was not called correctly or at all")
	}
}

// TODO: Implement other test cases:
// TestWorker_NoMapping_ExistingAsanaTask_UpdateSuccess
// TestWorker_NoMapping_NoAsanaTask_CreateSuccess
// Error handling tests

func TestWorker_NoMapping_ExistingAsanaTask_UpdateSuccess(t *testing.T) {
	mockDb := &MockDB{}
	mockAzure := &MockAzureClient{}
	mockAsana := &MockAsanaClient{}

	app := &App{
		db:          mockDb,
		azureClient: mockAzure,
		asanaClient: mockAsana,
		Tracer:      trace.NewNoopTracerProvider().Tracer("test"),
	}

	syncTask := SyncTask{ADOTaskID: 456, AsanaProjectGID: "as_project2", ADOProjectID: "ado_proj_def"}
	adoWorkItem := &azure.WorkItem{
		ID:           456,
		Title:        "ADO Task For Existing Asana",
		WorkItemType: "Task",
		URL:          "http://ado/task/456",
		ChangedDate:  time.Now(),
		CreatedDate:  time.Now().Add(-3 * time.Hour),
		State:        "New",
	}
	existingAsanaTask := &asana.Task{
		GID:  "existing_asana_task_gid",
		Name: "Old Asana Task Name", // This will be updated
	}

	expectedAsanaName, _ := adoWorkItem.FormatTitle()
	expectedAsanaDesc, _ := adoWorkItem.FormatTitleWithLink()

	mockDb.GetTaskMappingByADOTaskIDFunc = func(ctx context.Context, adoTaskID int) (*TaskMapping, error) {
		return nil, nil // No mapping found
	}
	mockAzure.GetWorkItemDetailsFunc = func(ctx context.Context, workItemID int) (*azure.WorkItem, error) {
		return adoWorkItem, nil
	}
	// FindTaskByTitle should be called with adoWorkItem.Title (raw title)
	mockAsana.FindTaskByTitleFunc = func(ctx context.Context, projectID string, title string) (*asana.Task, error) {
		if projectID != syncTask.AsanaProjectGID {
			t.Errorf("Expected FindTaskByTitle projectID %s, got %s", syncTask.AsanaProjectGID, projectID)
		}
		if title != adoWorkItem.Title { // Search by raw ADO title
			t.Errorf("Expected FindTaskByTitle title '%s', got '%s'", adoWorkItem.Title, title)
		}
		return existingAsanaTask, nil
	}
	mockAsana.UpdateTaskFunc = func(ctx context.Context, taskID string, name string, htmlDescription string) (*asana.Task, error) {
		if taskID != existingAsanaTask.GID {
			t.Errorf("Expected UpdateTask Asana ID %s, got %s", existingAsanaTask.GID, taskID)
		}
		if name != expectedAsanaName {
			t.Errorf("Expected UpdateTask name '%s', got '%s'", expectedAsanaName, name)
		}
		if htmlDescription != expectedAsanaDesc {
			t.Errorf("Expected UpdateTask description '%s', got '%s'", expectedAsanaDesc, htmlDescription)
		}
		return &asana.Task{GID: taskID, Name: name, HTMLNotes: htmlDescription}, nil
	}
	mockDb.CreateTaskMappingFunc = func(ctx context.Context, tm *TaskMapping) error {
		if tm.ADOTaskID != syncTask.ADOTaskID {
			t.Errorf("Expected CreateTaskMapping ADOTaskID %d, got %d", syncTask.ADOTaskID, tm.ADOTaskID)
		}
		if tm.AsanaTaskID != existingAsanaTask.GID {
			t.Errorf("Expected CreateTaskMapping AsanaTaskID %s, got %s", existingAsanaTask.GID, tm.AsanaTaskID)
		}
		return nil
	}

	syncTasksChan := make(chan SyncTask, 1)
	syncTasksChan <- syncTask
	close(syncTasksChan)

	app.worker(context.Background(), 1, syncTasksChan)

	if mockDb.GetTaskMappingByADOTaskIDCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetTaskMappingByADOTaskID was not called correctly")
	}
	if mockAzure.GetWorkItemDetailsCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetWorkItemDetails was not called correctly")
	}
	if mockAsana.FindTaskByTitleCalledWithTitle != adoWorkItem.Title {
		t.Errorf("FindTaskByTitle was not called with correct title")
	}
	if mockAsana.UpdateTaskCalledWithID != existingAsanaTask.GID {
		t.Errorf("UpdateTask was not called correctly for existing Asana task")
	}
	if mockDb.CreateTaskMappingCalledWith == nil || mockDb.CreateTaskMappingCalledWith.ADOTaskID != syncTask.ADOTaskID {
		t.Errorf("CreateTaskMapping was not called correctly")
	}
}

func TestWorker_Error_GetWorkItemDetailsFails(t *testing.T) {
	mockDb := &MockDB{}
	mockAzure := &MockAzureClient{}
	mockAsana := &MockAsanaClient{} // Needed for App struct, though not expected to be called

	app := &App{
		db:          mockDb,
		azureClient: mockAzure,
		asanaClient: mockAsana,
		Tracer:      trace.NewNoopTracerProvider().Tracer("test"),
	}

	syncTask := SyncTask{ADOTaskID: 999, ProjectID: "as_project_err"}
	expectedError := fmt.Errorf("azure API is down")

	// Scenario: Existing mapping, but GetWorkItemDetails fails
	taskMapping := &TaskMapping{ADOTaskID: 999, AsanaTaskID: "asana_task_err", SyncedAt: time.Now().Add(-time.Hour)}

	mockDb.GetTaskMappingByADOTaskIDFunc = func(ctx context.Context, adoTaskID int) (*TaskMapping, error) {
		return taskMapping, nil // Simulate mapping exists
	}
	mockAzure.GetWorkItemDetailsFunc = func(ctx context.Context, workItemID int) (*azure.WorkItem, error) {
		return nil, expectedError // Simulate error
	}

	syncTasksChan := make(chan SyncTask, 1)
	syncTasksChan <- syncTask
	close(syncTasksChan)

	// We don't have a direct way to check logs here without more setup,
	// but we can verify that no Asana or DB update/create calls were made.
	app.worker(context.Background(), 1, syncTasksChan)

	if mockDb.GetTaskMappingByADOTaskIDCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetTaskMappingByADOTaskID was not called correctly")
	}
	if mockAzure.GetWorkItemDetailsCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetWorkItemDetails was not called correctly")
	}

	// Crucially, these should NOT have been called
	if mockAsana.UpdateTaskCalledWithID != "" {
		t.Errorf("AsanaClient.UpdateTask should not have been called, but was called with ID: %s", mockAsana.UpdateTaskCalledWithID)
	}
	if mockAsana.CreateTaskCalledWithProjectID != "" {
		t.Errorf("AsanaClient.CreateTask should not have been called, but was called for project: %s", mockAsana.CreateTaskCalledWithProjectID)
	}
	if mockDb.UpdateTaskMappingCalledWith != nil {
		t.Errorf("DB.UpdateTaskMapping should not have been called")
	}
	if mockDb.CreateTaskMappingCalledWith != nil {
		t.Errorf("DB.CreateTaskMapping should not have been called")
	}
}

func TestWorker_NoMapping_NoAsanaTask_CreateSuccess(t *testing.T) {
	mockDb := &MockDB{}
	mockAzure := &MockAzureClient{}
	mockAsana := &MockAsanaClient{}

	app := &App{
		db:          mockDb,
		azureClient: mockAzure,
		asanaClient: mockAsana,
		Tracer:      trace.NewNoopTracerProvider().Tracer("test"),
	}

	syncTask := SyncTask{ADOTaskID: 789, AsanaProjectGID: "as_project3", ADOProjectID: "ado_proj_ghi"}
	adoWorkItem := &azure.WorkItem{
		ID:           789,
		Title:        "ADO Task For New Asana",
		WorkItemType: "Feature",
		URL:          "http://ado/feature/789",
		ChangedDate:  time.Now(),
		CreatedDate:  time.Now().Add(-4 * time.Hour),
		State:        "Active",
	}
	createdAsanaTaskGID := "new_asana_task_gid"

	expectedAsanaName, _ := adoWorkItem.FormatTitle()
	expectedAsanaDesc, _ := adoWorkItem.FormatTitleWithLink()

	mockDb.GetTaskMappingByADOTaskIDFunc = func(ctx context.Context, adoTaskID int) (*TaskMapping, error) {
		return nil, nil // No mapping found
	}
	mockAzure.GetWorkItemDetailsFunc = func(ctx context.Context, workItemID int) (*azure.WorkItem, error) {
		return adoWorkItem, nil
	}
	mockAsana.FindTaskByTitleFunc = func(ctx context.Context, projectID string, title string) (*asana.Task, error) {
		if projectID != syncTask.AsanaProjectGID {
			t.Errorf("Expected FindTaskByTitle projectID %s, got %s", syncTask.AsanaProjectGID, projectID)
		}
		if title != adoWorkItem.Title { // Search by raw ADO title
			t.Errorf("Expected FindTaskByTitle title '%s', got '%s'", adoWorkItem.Title, title)
		}
		return nil, nil // No Asana task found by title
	}
	mockAsana.CreateTaskFunc = func(ctx context.Context, projectID string, name string, htmlDescription string) (*asana.Task, error) {
		if projectID != syncTask.AsanaProjectGID {
			t.Errorf("Expected CreateTask projectID %s, got %s", syncTask.AsanaProjectGID, projectID)
		}
		if name != expectedAsanaName {
			t.Errorf("Expected CreateTask name '%s', got '%s'", expectedAsanaName, name)
		}
		if htmlDescription != expectedAsanaDesc {
			t.Errorf("Expected CreateTask description '%s', got '%s'", expectedAsanaDesc, htmlDescription)
		}
		return &asana.Task{GID: createdAsanaTaskGID, Name: name, HTMLNotes: htmlDescription}, nil
	}
	mockDb.CreateTaskMappingFunc = func(ctx context.Context, tm *TaskMapping) error {
		if tm.ADOTaskID != syncTask.ADOTaskID {
			t.Errorf("Expected CreateTaskMapping ADOTaskID %d, got %d", syncTask.ADOTaskID, tm.ADOTaskID)
		}
		if tm.AsanaTaskID != createdAsanaTaskGID {
			t.Errorf("Expected CreateTaskMapping AsanaTaskID %s, got %s", createdAsanaTaskGID, tm.AsanaTaskID)
		}
		return nil
	}

	syncTasksChan := make(chan SyncTask, 1)
	syncTasksChan <- syncTask
	close(syncTasksChan)

	app.worker(context.Background(), 1, syncTasksChan)

	if mockDb.GetTaskMappingByADOTaskIDCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetTaskMappingByADOTaskID was not called correctly")
	}
	if mockAzure.GetWorkItemDetailsCalledWith != syncTask.ADOTaskID {
		t.Errorf("GetWorkItemDetails was not called correctly")
	}
	if mockAsana.FindTaskByTitleCalledWithTitle != adoWorkItem.Title {
		t.Errorf("FindTaskByTitle was not called with correct title")
	}
	if mockAsana.CreateTaskCalledWithProjectID != syncTask.ProjectID || mockAsana.CreateTaskCalledWithName != expectedAsanaName {
		t.Errorf("CreateTask was not called correctly")
	}
	if mockDb.CreateTaskMappingCalledWith == nil || mockDb.CreateTaskMappingCalledWith.ADOTaskID != syncTask.ADOTaskID {
		t.Errorf("CreateTaskMapping was not called correctly")
	}
}
