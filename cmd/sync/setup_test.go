package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
)

// ============================================================================
// LoadSyncedTags Tests
// ============================================================================

func TestLoadSyncedTagsSuccess(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	mockDB := app.DB.(*enhancedMockDB)

	// Setup: DB has projects with tags
	mockDB.projects = []db.Project{
		{AsanaWorkspaceName: "ws1", ADOProjectName: "p1"},
		{AsanaWorkspaceName: "ws2", ADOProjectName: "p2"},
	}

	mockDB.workspaceTags["ws1"] = db.WorkspaceTag{
		WorkspaceName: "ws1",
		GID:           "tag-1",
		Name:          "synced",
	}
	mockDB.workspaceTags["ws2"] = db.WorkspaceTag{
		WorkspaceName: "ws2",
		GID:           "tag-2",
		Name:          "synced",
	}

	app.loadSyncedTags(ctx)

	assert.Len(t, app.SyncedTags, 2)
	assert.Equal(t, "tag-1", app.SyncedTags["ws1"].GID)
	assert.Equal(t, "tag-2", app.SyncedTags["ws2"].GID)
}

func TestLoadSyncedTagsDeduplication(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	mockDB := app.DB.(*enhancedMockDB)

	// Setup: Multiple projects in same workspace
	mockDB.projects = []db.Project{
		{AsanaWorkspaceName: "ws1", ADOProjectName: "p1"},
		{AsanaWorkspaceName: "ws1", ADOProjectName: "p2"},
		{AsanaWorkspaceName: "ws1", ADOProjectName: "p3"},
	}

	mockDB.workspaceTags["ws1"] = db.WorkspaceTag{WorkspaceName: "ws1", GID: "tag-1"}

	// We can track calls by inspecting internal state logic of enhancedMockDB if we wanted,
	// but mostly we care that result is correct and it doesn't crash
	app.loadSyncedTags(ctx)

	assert.Len(t, app.SyncedTags, 1)
	assert.Equal(t, "tag-1", app.SyncedTags["ws1"].GID)
}

func TestLoadSyncedTagsDBError(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	mockDB := app.DB.(*enhancedMockDB)

	mockDB.errors["Projects"] = fmt.Errorf("db connection error")

	app.loadSyncedTags(ctx)

	assert.Empty(t, app.SyncedTags, "should handle db error gracefully")
}

func TestLoadSyncedTagsMissingTag(t *testing.T) {
	app := setupTestApp()
	ctx := context.Background()
	mockDB := app.DB.(*enhancedMockDB)

	mockDB.projects = []db.Project{
		{AsanaWorkspaceName: "ws1", ADOProjectName: "p1"},
	}
	// No tag in mockDB.workspaceTags

	app.loadSyncedTags(ctx)

	assert.Empty(t, app.SyncedTags, "should not load missing tags")
}

// ============================================================================
// Setup Tests
// ============================================================================

func TestSetupSuccess(t *testing.T) {
	// We need to set env vars for setup
	os.Setenv("MONGO_URI", "mongodb://localhost:27017")
	os.Setenv("UPTRACE_DSN", "https://token@uptrace.dev/123")
	os.Setenv("ADO_ORG_URL", "https://dev.azure.com/org")
	os.Setenv("ADO_PAT", "ado-pat")
	os.Setenv("ASANA_PAT", "asana-pat")
	os.Setenv("PROPERTY_CACHE_TTL", "1h")
	defer func() {
		os.Unsetenv("MONGO_URI")
		os.Unsetenv("UPTRACE_DSN")
		os.Unsetenv("ADO_ORG_URL")
		os.Unsetenv("ADO_PAT")
		os.Unsetenv("ASANA_PAT")
		os.Unsetenv("PROPERTY_CACHE_TTL")
	}()

	app := setupTestApp()
	// uptrace setup in main.go calls uptrace.ConfigureOpentelemetry which might interact with globals.
	// We mocked UptraceShutdown in App struct but setup assigns it.
	// The real uptrace call might be tricky.
	// For this test, we accept that uptrace might configure global OTEL.

	err := app.setup(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, app.Tracer)
	assert.NotNil(t, app.UptraceShutdown)
	assert.Equal(t, 1*time.Hour, app.CacheTTL)
}

func TestSetupDBConnectFail(t *testing.T) {
	app := setupTestApp()

	// Create a new mock DB that fails connect
	// mockDB := newEnhancedMockDB()

	// Let's modify the app with a custom DB implementation for this test
	failingDB := &failingConnectDB{enhancedMockDB: newEnhancedMockDB()}
	app.DB = failingDB

	os.Setenv("UPTRACE_DSN", "https://token@uptrace.dev/123")
	defer os.Unsetenv("UPTRACE_DSN")

	// Pre-initialize Tracer as setup() does part of it before DB connect
	app.Tracer = otel.Tracer("test")

	err := app.setup(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error connecting to the DB")
	assert.Contains(t, err.Error(), "connection failed")
}

func TestSetupEnsureIndexesFail(t *testing.T) {
	app := setupTestApp()
	mockDB := app.DB.(*enhancedMockDB)

	// We need to extend enhancedMockDB to support EnsureIndexes error
	// Or use a one-off mock.
	// Since we defined enhancedMockDB in worker_test.go, we can't easily change it there without editing that file.
	// But we can define a struct that embeds it and overrides EnsureIndexes?
	// No, that doesn't work well with Go pointers.

	// Instead, let's look at enhancedMockDB definition given in the prompt/context locally.
	// It doesn't look like it checks errors["EnsureIndexes"].
	// So we might need a specialized mock here.

	failingIndexDB := &failingIndexDB{enhancedMockDB: mockDB}
	app.DB = failingIndexDB

	os.Setenv("UPTRACE_DSN", "https://token@uptrace.dev/123")
	defer os.Unsetenv("UPTRACE_DSN")

	err := app.setup(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error ensuring DB indexes")
	assert.Contains(t, err.Error(), "index error")
}

// Helper structs for failure scenarios
type failingConnectDB struct {
	*enhancedMockDB // Embed to satisfy interface
}

// We need to implement all methods of DBInterface to satisfy the interface,
// embedding *enhancedMockDB (which satisfies it) works, but we need to instantiate it carefully.
// However, *enhancedMockDB has pointer receiver methods.

func (m *failingConnectDB) Connect(ctx context.Context, uri string) error {
	return fmt.Errorf("connection failed")
}

// We need to ensure failingConnectDB satisfies db.DBInterface.
// It does because it embeds *enhancedMockDB which implements it, AND overrides Connect.
// But we need to make sure nil pointers don't panic.
func (m *failingConnectDB) LastSync(ctx context.Context) db.LastSync { return db.LastSync{} } // fallback

// Actually, enhancedMockDB logic for methods not overridden will be used.

type failingIndexDB struct {
	*enhancedMockDB
}

func (m *failingIndexDB) EnsureIndexes(ctx context.Context) error {
	return fmt.Errorf("index error")
}
