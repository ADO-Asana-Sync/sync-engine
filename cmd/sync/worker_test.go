package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel"
)

// Enhanced mockDB with realistic behavior for worker tests
type enhancedMockDB struct {
	projects      []db.Project
	tasks         map[int]db.TaskMapping
	cache         map[string]db.CacheItem
	workspaceTags map[string]db.WorkspaceTag
	lastSync      db.LastSync

	// Test tracking
	addTaskCalls     []db.TaskMapping
	updateTaskCalls  []db.TaskMapping
	upsertCacheCalls []db.CacheItem
	errors           map[string]error // Function name → error to return
}

func newEnhancedMockDB() *enhancedMockDB {
	return &enhancedMockDB{
		projects:         []db.Project{},
		tasks:            make(map[int]db.TaskMapping),
		cache:            make(map[string]db.CacheItem),
		workspaceTags:    make(map[string]db.WorkspaceTag),
		addTaskCalls:     []db.TaskMapping{},
		updateTaskCalls:  []db.TaskMapping{},
		upsertCacheCalls: []db.CacheItem{},
		errors:           make(map[string]error),
	}
}

func (m *enhancedMockDB) Connect(ctx context.Context, uri string) error { return nil }
func (m *enhancedMockDB) Disconnect(ctx context.Context) error          { return nil }
func (m *enhancedMockDB) EnsureIndexes(ctx context.Context) error       { return nil }

func (m *enhancedMockDB) Projects(ctx context.Context) ([]db.Project, error) {
	if err := m.errors["Projects"]; err != nil {
		return nil, err
	}
	return m.projects, nil
}

func (m *enhancedMockDB) AddProject(ctx context.Context, project db.Project) error {
	return nil
}

func (m *enhancedMockDB) RemoveProject(ctx context.Context, id primitive.ObjectID) error {
	return nil
}

func (m *enhancedMockDB) UpdateProject(ctx context.Context, project db.Project) error {
	return nil
}

func (m *enhancedMockDB) LastSync(ctx context.Context) db.LastSync {
	return m.lastSync
}

func (m *enhancedMockDB) WriteLastSync(ctx context.Context, timestamp time.Time) error {
	return nil
}

func (m *enhancedMockDB) TaskByADOTaskID(ctx context.Context, id int) (db.TaskMapping, error) {
	if err := m.errors["TaskByADOTaskID"]; err != nil {
		return db.TaskMapping{}, err
	}
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return db.TaskMapping{}, fmt.Errorf("not found")
}

func (m *enhancedMockDB) AddTask(ctx context.Context, task db.TaskMapping) error {
	if err := m.errors["AddTask"]; err != nil {
		return err
	}
	m.addTaskCalls = append(m.addTaskCalls, task)
	m.tasks[task.ADOTaskID] = task
	return nil
}

func (m *enhancedMockDB) UpdateTask(ctx context.Context, task db.TaskMapping) error {
	if err := m.errors["UpdateTask"]; err != nil {
		return err
	}
	m.updateTaskCalls = append(m.updateTaskCalls, task)
	m.tasks[task.ADOTaskID] = task
	return nil
}

func (m *enhancedMockDB) GetCacheItem(ctx context.Context, key string) (db.CacheItem, error) {
	if err := m.errors["GetCacheItem"]; err != nil {
		return db.CacheItem{}, err
	}
	if item, ok := m.cache[key]; ok {
		return item, nil
	}
	return db.CacheItem{}, fmt.Errorf("not found")
}

func (m *enhancedMockDB) UpsertCacheItem(ctx context.Context, item db.CacheItem) error {
	if err := m.errors["UpsertCacheItem"]; err != nil {
		return err
	}
	m.upsertCacheCalls = append(m.upsertCacheCalls, item)
	m.cache[item.Key] = item
	return nil
}

func (m *enhancedMockDB) WorkspaceTag(ctx context.Context, workspaceName string) (db.WorkspaceTag, error) {
	if err := m.errors["WorkspaceTag"]; err != nil {
		return db.WorkspaceTag{}, err
	}
	if tag, ok := m.workspaceTags[workspaceName]; ok {
		return tag, nil
	}
	return db.WorkspaceTag{}, fmt.Errorf("not found")
}

func (m *enhancedMockDB) UpsertWorkspaceTag(ctx context.Context, tag db.WorkspaceTag) error {
	m.workspaceTags[tag.WorkspaceName] = tag
	return nil
}

// Enhanced mockAsana with realistic behavior
type enhancedMockAsana struct {
	projects     map[string]map[string]string   // workspace → project name → GID
	tasks        map[string][]asana.Task        // project GID → tasks
	customFields map[string][]asana.CustomField // project GID → custom fields
	tags         map[string]asana.Tag           // workspace → tag

	// Test tracking
	tasksCreated       []asana.Task
	tasksUpdated       []string            // task GIDs
	tasksUpdatedWithCF []string            // task GIDs updated with custom fields
	tagsAdded          map[string][]string // task GID → tag GIDs
	errors             map[string]error
}

func newEnhancedMockAsana() *enhancedMockAsana {
	return &enhancedMockAsana{
		projects:           make(map[string]map[string]string),
		tasks:              make(map[string][]asana.Task),
		customFields:       make(map[string][]asana.CustomField),
		tags:               make(map[string]asana.Tag),
		tasksCreated:       []asana.Task{},
		tasksUpdated:       []string{},
		tasksUpdatedWithCF: []string{},
		tagsAdded:          make(map[string][]string),
		errors:             make(map[string]error),
	}
}

func (m *enhancedMockAsana) Connect(ctx context.Context, pat string) {}

func (m *enhancedMockAsana) ProjectGIDByName(ctx context.Context, workspace, projectName string) (string, error) {
	if err := m.errors["ProjectGIDByName"]; err != nil {
		return "", err
	}
	if ws, ok := m.projects[workspace]; ok {
		if gid, ok := ws[projectName]; ok {
			return gid, nil
		}
	}
	return "", fmt.Errorf("project not found")
}

func (m *enhancedMockAsana) ListProjects(ctx context.Context, workspace string) ([]asana.Project, error) {
	return nil, nil
}

func (m *enhancedMockAsana) ListProjectTasks(ctx context.Context, projectGID string) ([]asana.Task, error) {
	if err := m.errors["ListProjectTasks"]; err != nil {
		return nil, err
	}
	if tasks, ok := m.tasks[projectGID]; ok {
		return tasks, nil
	}
	return []asana.Task{}, nil
}

func (m *enhancedMockAsana) CreateTask(ctx context.Context, projectGID, name, notes string) (asana.Task, error) {
	if err := m.errors["CreateTask"]; err != nil {
		return asana.Task{}, err
	}
	task := asana.Task{GID: fmt.Sprintf("task-%d", len(m.tasksCreated)+1), Name: name}
	m.tasksCreated = append(m.tasksCreated, task)
	return task, nil
}

func (m *enhancedMockAsana) UpdateTask(ctx context.Context, taskGID, name, notes string) error {
	if err := m.errors["UpdateTask"]; err != nil {
		return err
	}
	m.tasksUpdated = append(m.tasksUpdated, taskGID)
	return nil
}

func (m *enhancedMockAsana) CreateTaskWithCustomFields(ctx context.Context, projectGID, name, notes string, customFields map[string]string) (asana.Task, error) {
	if err := m.errors["CreateTaskWithCustomFields"]; err != nil {
		return asana.Task{}, err
	}
	task := asana.Task{GID: fmt.Sprintf("task-cf-%d", len(m.tasksCreated)+1), Name: name}
	m.tasksCreated = append(m.tasksCreated, task)
	return task, nil
}

func (m *enhancedMockAsana) UpdateTaskWithCustomFields(ctx context.Context, taskGID, name, notes string, customFields map[string]string) error {
	if err := m.errors["UpdateTaskWithCustomFields"]; err != nil {
		return err
	}
	m.tasksUpdatedWithCF = append(m.tasksUpdatedWithCF, taskGID)
	return nil
}

func (m *enhancedMockAsana) ProjectCustomFieldByName(ctx context.Context, projectGID, fieldName string) (asana.CustomField, error) {
	if err := m.errors["ProjectCustomFieldByName"]; err != nil {
		return asana.CustomField{}, err
	}
	if fields, ok := m.customFields[projectGID]; ok {
		for _, field := range fields {
			if field.Name == fieldName {
				return field, nil
			}
		}
	}
	return asana.CustomField{}, fmt.Errorf("field not found")
}

func (m *enhancedMockAsana) ProjectHasCustomField(ctx context.Context, projectGID, fieldGID string) (bool, error) {
	return false, nil
}

func (m *enhancedMockAsana) CustomFieldByName(ctx context.Context, workspace, fieldName string) (asana.CustomField, error) {
	return asana.CustomField{}, nil
}

func (m *enhancedMockAsana) TagByName(ctx context.Context, workspace, tagName string) (asana.Tag, error) {
	if err := m.errors["TagByName"]; err != nil {
		return asana.Tag{}, err
	}
	if tag, ok := m.tags[workspace]; ok {
		return tag, nil
	}
	return asana.Tag{}, fmt.Errorf("tag not found")
}

func (m *enhancedMockAsana) AddTagToTask(ctx context.Context, taskGID, tagGID string) error {
	if err := m.errors["AddTagToTask"]; err != nil {
		return err
	}
	m.tagsAdded[taskGID] = append(m.tagsAdded[taskGID], tagGID)
	return nil
}

func (m *enhancedMockAsana) ListWorkspaces(ctx context.Context) ([]asana.Workspace, error) {
	return nil, nil
}

// Enhanced mockAzure
type enhancedMockAzure struct {
	workItems map[int]azure.WorkItem
	errors    map[string]error
}

func newEnhancedMockAzure() *enhancedMockAzure {
	return &enhancedMockAzure{
		workItems: make(map[int]azure.WorkItem),
		errors:    make(map[string]error),
	}
}

func (m *enhancedMockAzure) Connect(ctx context.Context, orgUrl, pat string) {}

func (m *enhancedMockAzure) GetChangedWorkItems(ctx context.Context, lastSync time.Time) ([]workitemtracking.WorkItemReference, error) {
	return nil, nil
}

func (m *enhancedMockAzure) GetWorkItem(ctx context.Context, id int) (azure.WorkItem, error) {
	if err := m.errors["GetWorkItem"]; err != nil {
		return azure.WorkItem{}, err
	}
	if wi, ok := m.workItems[id]; ok {
		return wi, nil
	}
	return azure.WorkItem{}, fmt.Errorf("work item not found")
}

func (m *enhancedMockAzure) GetProjects(ctx context.Context) ([]core.TeamProjectReference, error) {
	return nil, nil
}

// Test helper functions
func setupTestApp() *App {
	return &App{
		DB:         newEnhancedMockDB(),
		Asana:      newEnhancedMockAsana(),
		Azure:      newEnhancedMockAzure(),
		CacheTTL:   24 * time.Hour,
		SyncedTags: make(map[string]asana.Tag),
		Tracer:     otel.Tracer("test"),
	}
}

func createTestWorkItem(id int, title, project, url string, changedDate time.Time) azure.WorkItem {
	return azure.WorkItem{
		ID:           id,
		Title:        title,
		WorkItemType: "User Story",
		TeamProject:  project,
		URL:          url,
		ChangedDate:  changedDate,
	}
}

// ============================================================================
// Worker Tests
// ============================================================================

func TestWorkerProcessesTasks(t *testing.T) {
	app := setupTestApp()
	taskCh := make(chan SyncTask, 1)
	resultCh := make(chan error, 1)

	// Setup: Add a work item and project mapping
	mockDB := app.DB.(*enhancedMockDB)
	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAsana := app.Asana.(*enhancedMockAsana)

	mockDB.projects = []db.Project{
		{ADOProjectName: "TestProject", AsanaWorkspaceName: "workspace1", AsanaProjectName: "AsanaProj"},
	}
	mockAzure.workItems[123] = createTestWorkItem(123, "Test Task", "TestProject", "http://ado.com/123", time.Now())

	if mockAsana.projects["workspace1"] == nil {
		mockAsana.projects["workspace1"] = make(map[string]string)
	}
	mockAsana.projects["workspace1"]["AsanaProj"] = "proj-gid-1"

	// Start worker
	go app.worker(context.Background(), 1, taskCh)

	// Send task
	taskCh <- SyncTask{ADOTaskID: 123, Result: resultCh}

	// Wait for result
	select {
	case err := <-resultCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for worker to process task")
	}

	// Verify task was created
	assert.Len(t, mockAsana.tasksCreated, 1, "should have created 1 Asana task")
	close(taskCh)
}

func TestWorkerContinuesAfterError(t *testing.T) {
	app := setupTestApp()
	taskCh := make(chan SyncTask, 2)
	resultCh := make(chan error, 2)

	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAzure.errors["GetWorkItem"] = fmt.Errorf("API error")

	go app.worker(context.Background(), 1, taskCh)

	// Send failing task
	taskCh <- SyncTask{ADOTaskID: 999, Result: resultCh}

	// First task should fail
	err := <-resultCh
	assert.Error(t, err)

	// Now fix the error and send another task
	delete(mockAzure.errors, "GetWorkItem")
	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)

	mockDB.projects = []db.Project{
		{ADOProjectName: "TestProject", AsanaWorkspaceName: "workspace1", AsanaProjectName: "AsanaProj"},
	}
	mockAzure.workItems[456] = createTestWorkItem(456, "Second Task", "TestProject", "http://ado.com/456", time.Now())

	if mockAsana.projects["workspace1"] == nil {
		mockAsana.projects["workspace1"] = make(map[string]string)
	}
	mockAsana.projects["workspace1"]["AsanaProj"] = "proj-gid-1"

	taskCh <- SyncTask{ADOTaskID: 456, Result: resultCh}

	// Second task should succeed
	err = <-resultCh
	assert.NoError(t, err)

	close(taskCh)
}

// ============================================================================
// handleTask Tests
// ============================================================================

func TestHandleTaskUpdateExisting(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	wlog := log.WithField("test", "worker")

	mockDB := app.DB.(*enhancedMockDB)
	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: existing mapping
	mockDB.tasks[123] = db.TaskMapping{
		ADOProjectID:   "TestProject",
		ADOTaskID:      123,
		AsanaProjectID: "proj-gid-1",
		AsanaTaskID:    "existing-task-1",
	}
	mockAzure.workItems[123] = createTestWorkItem(123, "Updated Task", "TestProject", "http://ado.com/123", time.Now())

	task := SyncTask{ADOTaskID: 123}
	err := app.handleTask(ctx, wlog, task)

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksUpdated, 1, "should have updated existing task")
	assert.Len(t, mockDB.updateTaskCalls, 1, "should have updated DB mapping")
}

func TestHandleTaskCreateNew(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	wlog := log.WithField("test", "worker")

	mockDB := app.DB.(*enhancedMockDB)
	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: no existing mapping, but project mapping exists
	mockDB.projects = []db.Project{
		{ADOProjectName: "TestProject", AsanaWorkspaceName: "workspace1", AsanaProjectName: "AsanaProj"},
	}
	mockAzure.workItems[123] = createTestWorkItem(123, "New Task", "TestProject", "http://ado.com/123", time.Now())

	if mockAsana.projects["workspace1"] == nil {
		mockAsana.projects["workspace1"] = make(map[string]string)
	}
	mockAsana.projects["workspace1"]["AsanaProj"] = "proj-gid-1"

	task := SyncTask{ADOTaskID: 123}
	err := app.handleTask(ctx, wlog, task)

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksCreated, 1, "should have created new Asana task")
	assert.Len(t, mockDB.addTaskCalls, 1, "should have added DB mapping")
}

func TestHandleTaskSkipUnmappedProject(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	wlog := log.WithField("test", "worker")

	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: work item exists but project not mapped
	mockAzure.workItems[123] = createTestWorkItem(123, "Unmapped Task", "UnmappedProj", "http://ado.com/123", time.Now())

	task := SyncTask{ADOTaskID: 123}
	err := app.handleTask(ctx, wlog, task)

	assert.NoError(t, err, "should not error for unmapped project")
	assert.Len(t, mockAsana.tasksCreated, 0, "should not create task")
}

func TestHandleTaskUpdateExistingByName(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	wlog := log.WithField("test", "worker")

	mockDB := app.DB.(*enhancedMockDB)
	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: no mapping, but task with same name exists in Asana
	mockDB.projects = []db.Project{
		{ADOProjectName: "TestProject", AsanaWorkspaceName: "workspace1", AsanaProjectName: "AsanaProj"},
	}
	mockAzure.workItems[123] = createTestWorkItem(123, "Existing Task Name", "TestProject", "http://ado.com/123", time.Now())

	if mockAsana.projects["workspace1"] == nil {
		mockAsana.projects["workspace1"] = make(map[string]string)
	}
	mockAsana.projects["workspace1"]["AsanaProj"] = "proj-gid-1"
	mockAsana.tasks["proj-gid-1"] = []asana.Task{
		{GID: "existing-task-gid", Name: "User Story 123: Existing Task Name"},
	}

	task := SyncTask{ADOTaskID: 123}
	err := app.handleTask(ctx, wlog, task)

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksUpdated, 1, "should have updated existing task by name")
	assert.Len(t, mockDB.addTaskCalls, 1, "should have created new mapping")
	assert.Len(t, mockAsana.tasksCreated, 0, "should not create new task")
}

// ============================================================================
// prepWorkItem Tests
// ============================================================================

func TestPrepWorkItemExistingMapping(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAzure := app.Azure.(*enhancedMockAzure)

	mockDB.tasks[123] = db.TaskMapping{ADOTaskID: 123, AsanaTaskID: "task-1"}
	mockAzure.workItems[123] = createTestWorkItem(123, "Test Task", "TestProject", "http://ado.com/123", time.Now())

	mapping, wi, name, desc, err := app.prepWorkItem(ctx, 123)

	assert.NoError(t, err)
	assert.NotNil(t, mapping, "should return existing mapping")
	assert.Equal(t, "task-1", mapping.AsanaTaskID)
	assert.Equal(t, 123, wi.ID)
	assert.Contains(t, name, "Test Task")
	assert.Contains(t, desc, "Test Task")
}

func TestPrepWorkItemNewWorkItem(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAzure.workItems[456] = createTestWorkItem(456, "New Task", "Project", "http://ado.com/456", time.Now())

	mapping, wi, name, desc, err := app.prepWorkItem(ctx, 456)

	assert.NoError(t, err)
	assert.Nil(t, mapping, "should return nil mapping for new work item")
	assert.Equal(t, 456, wi.ID)
	assert.NotEmpty(t, name)
	assert.NotEmpty(t, desc)
}

func TestPrepWorkItemAzureError(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAzure := app.Azure.(*enhancedMockAzure)
	mockAzure.errors["GetWorkItem"] = fmt.Errorf("Azure API error")

	_, _, _, _, err := app.prepWorkItem(ctx, 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Azure API error")
}

// ============================================================================
// updateExistingTask Tests
// ============================================================================

func TestUpdateExistingTaskWithCustomFields(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: custom field exists
	mockAsana.customFields["proj-1"] = []asana.CustomField{
		{GID: "cf-link", Name: "link"},
	}

	mapping := db.TaskMapping{
		ADOProjectID:   "TestProject",
		ADOTaskID:      123,
		AsanaProjectID: "proj-1",
		AsanaTaskID:    "task-1",
	}
	wi := createTestWorkItem(123, "Task", "TestProject", "http://ado.com/123", time.Now())

	err := app.updateExistingTask(ctx, wi, mapping, "Task Name", "Task Desc")

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksUpdatedWithCF, 1, "should update with custom fields")
	assert.Len(t, mockDB.updateTaskCalls, 1, "should update DB")
}

func TestUpdateExistingTaskWithoutCustomFields(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)

	mapping := db.TaskMapping{
		AsanaProjectID: "proj-1",
		AsanaTaskID:    "task-1",
	}
	wi := createTestWorkItem(123, "Task", "TestProject", "http://ado.com/123", time.Now())

	err := app.updateExistingTask(ctx, wi, mapping, "Task Name", "Task Desc")

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksUpdated, 1, "should update without custom fields")
	assert.Len(t, mockDB.updateTaskCalls, 1)
}

func TestUpdateExistingTaskAsanaError(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)
	mockAsana.errors["UpdateTask"] = fmt.Errorf("Asana error")

	mapping := db.TaskMapping{AsanaProjectID: "proj-1", AsanaTaskID: "task-1"}
	wi := createTestWorkItem(123, "Task", "TestProject", "http://ado.com/123", time.Now())

	err := app.updateExistingTask(ctx, wi, mapping, "Task Name", "Task Desc")

	assert.Error(t, err)
	assert.Len(t, mockDB.updateTaskCalls, 0, "should not update DB on Asana error")
}

// ============================================================================
// Additional worker function tests
// ============================================================================

func TestAsanaProjectForADOMapped(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)

	mockDB.projects = []db.Project{
		{ADOProjectName: "ProjectA", AsanaWorkspaceName: "workspace1", AsanaProjectName: "AsanaProjectA"},
	}

	if mockAsana.projects["workspace1"] == nil {
		mockAsana.projects["workspace1"] = make(map[string]string)
	}
	mockAsana.projects["workspace1"]["AsanaProjectA"] = "proj-gid-123"

	gid, workspace, err := app.asanaProjectForADO(ctx, "ProjectA")

	assert.NoError(t, err)
	assert.Equal(t, "proj-gid-123", gid)
	assert.Equal(t, "workspace1", workspace)
}

func TestAsanaProjectForADOUnmapped(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	gid, workspace, err := app.asanaProjectForADO(ctx, "UnmappedProject")

	assert.NoError(t, err)
	assert.Empty(t, gid)
	assert.Empty(t, workspace)
}

func TestGetLinkCustomFieldCached(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockAsana := app.Asana.(*enhancedMockAsana)

	// Setup: cached field
	mockDB.cache["project:proj-1:link_field"] = db.CacheItem{
		Key:       "project:proj-1:link_field",
		Value:     map[string]interface{}{"gid": "cf-link-123", "name": "link"},
		UpdatedAt: time.Now(),
	}

	cf, ok := app.getLinkCustomField(ctx, "proj-1")

	assert.True(t, ok)
	assert.Equal(t, "cf-link-123", cf.GID)
	assert.Len(t, mockAsana.customFields, 0, "should not call Asana when cached")
}

func TestGetLinkCustomFieldFetchFromAsana(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAsana := app.Asana.(*enhancedMockAsana)
	mockDB := app.DB.(*enhancedMockDB)

	mockAsana.customFields["proj-1"] = []asana.CustomField{
		{GID: "cf-link-new", Name: "link"},
	}

	cf, ok := app.getLinkCustomField(ctx, "proj-1")

	assert.True(t, ok)
	assert.Equal(t, "cf-link-new", cf.GID)
	assert.Len(t, mockDB.upsertCacheCalls, 1, "should cache the result")
}

func TestResolveSyncedTagFromMemory(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	app.SyncedTags["workspace1"] = asana.Tag{GID: "tag-123", Name: "synced"}

	tag, ok := app.resolveSyncedTag(ctx, "workspace1")

	assert.True(t, ok)
	assert.Equal(t, "tag-123", tag.GID)
}

func TestResolveSyncedTagFromDB(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockDB.workspaceTags["workspace1"] = db.WorkspaceTag{
		WorkspaceName: "workspace1",
		GID:           "tag-db-456",
		Name:          "synced",
	}

	tag, ok := app.resolveSyncedTag(ctx, "workspace1")

	assert.True(t, ok)
	assert.Equal(t, "tag-db-456", tag.GID)
	assert.Equal(t, "tag-db-456", app.SyncedTags["workspace1"].GID, "should cache in memory")
}

func TestResolveSyncedTagFromAsana(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAsana := app.Asana.(*enhancedMockAsana)
	mockAsana.tags["workspace1"] = asana.Tag{GID: "tag-asana-789", Name: "synced"}

	tag, ok := app.resolveSyncedTag(ctx, "workspace1")

	assert.True(t, ok)
	assert.Equal(t, "tag-asana-789", tag.GID)
}

func TestWorkspaceForADO(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockDB := app.DB.(*enhancedMockDB)
	mockDB.projects = []db.Project{
		{ADOProjectName: "ProjectA", AsanaWorkspaceName: "workspace1"},
		{ADOProjectName: "ProjectB", AsanaWorkspaceName: "workspace2"},
	}

	workspace := app.workspaceForADO(ctx, "ProjectA")
	assert.Equal(t, "workspace1", workspace)

	workspace = app.workspaceForADO(ctx, "UnknownProject")
	assert.Empty(t, workspace)
}

func TestCreateAndMapTaskWithCustomFields(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAsana := app.Asana.(*enhancedMockAsana)
	mockDB := app.DB.(*enhancedMockDB)

	mockAsana.customFields["proj-1"] = []asana.CustomField{
		{GID: "cf-link", Name: "link"},
	}

	wi := createTestWorkItem(123, "New Task", "TestProject", "http://ado.com/123", time.Now())

	err := app.createAndMapTask(ctx, "proj-1", "workspace1", wi, "Task Name", "Task Desc")

	assert.NoError(t, err)
	assert.Len(t, mockAsana.tasksCreated, 1)
	assert.Len(t, mockDB.addTaskCalls, 1)
}

func TestAddSyncedTagSuccess(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAsana := app.Asana.(*enhancedMockAsana)
	app.SyncedTags["workspace1"] = asana.Tag{GID: "tag-123", Name: "synced"}

	app.addSyncedTag(ctx, "workspace1", "task-gid-1")

	assert.Len(t, mockAsana.tagsAdded["task-gid-1"], 1)
	assert.Equal(t, "tag-123", mockAsana.tagsAdded["task-gid-1"][0])
}

func TestAddSyncedTagNoTag(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()

	mockAsana := app.Asana.(*enhancedMockAsana)

	// No tag configured - should be a no-op
	app.addSyncedTag(ctx, "workspace-unknown", "task-gid-1")

	assert.Len(t, mockAsana.tagsAdded, 0, "should not add tag when not resolved")
}
