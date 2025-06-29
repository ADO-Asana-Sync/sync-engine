# ADO-Asana-Sync - AI Assistant Instructions

This document provides instructions for AI assistants (OpenAI Codex, GitHub Copilot, Gemini) working with the ADO-Asana-Sync codebase.

## Project Summary

**Purpose**: Go application synchronizing Azure DevOps work items to Asana tasks (one-way sync)  
**Data Storage**: MongoDB for configuration and sync records  
**Source of Truth**: Azure DevOps (ADO)

## Key Components

```
sync-engine/
├── cmd/
│   ├── sync/           # Sync engine (controller-worker architecture)
│   └── web-ui/         # Web UI for configuration
├── internal/
│   ├── asana/          # Asana API client
│   ├── azure/          # Azure DevOps API client  
│   ├── db/             # MongoDB interactions
│   └── helpers/        # Utility functions
└── deploy/            # Deployment configuration
```

## Core Functionality

1. **Controller** (`cmd/sync/controller.go`):
   - Polls ADO for changed work items since last sync
   - Queues changed items to a channel for worker processing
   - Handles full and delta sync scheduling

2. **Worker** (`cmd/sync/worker.go`):
   - Processes items from the queue
   - Creates/updates corresponding Asana tasks
   - Records mappings in MongoDB

3. **Data Models**:
   - `ProjectMapping`: Links ADO project to Asana project
   - `TaskMapping`: Tracks ADO work item to Asana task relationship
   - `SyncTask`: Work unit in the sync queue

## Sync Requirements

- **Task Adoption**: Match existing Asana tasks by title
- **Item Hierarchy**: Map ADO hierarchy to Asana tasks/subtasks/sections
- **One-Way Sync**: ADO → Asana only
- **Tagging**: Apply configurable tag to all synced items
- **Retention**: Auto-purge closed items after configurable period
- **Delta + Full Sync**: Both incremental and full synchronization

## Code Patterns

When writing code for this project:

```go
// Example controller-worker pattern
func controller(ctx context.Context, queue chan<- SyncTask) {
    // Get last sync timestamp from MongoDB
    lastSync := getLastSyncTime(ctx)
    
    // Query ADO for changes
    changedItems := queryADOChanges(ctx, lastSync)
    
    // Queue items for processing
    for _, item := range changedItems {
        queue <- SyncTask{
            WorkItemID: item.ID,
            ProjectID: item.ProjectID,
        }
    }
}

func worker(ctx context.Context, queue <-chan SyncTask) {
    for task := range queue {
        // Process each sync task
        processTask(ctx, task)
    }
}

// Always instrument with OpenTelemetry
func processTask(ctx context.Context, task SyncTask) {
    ctx, span := tracer.Start(ctx, "processTask")
    defer span.End()
    
    // Error handling pattern
    adoItem, err := getADOWorkItem(ctx, task.WorkItemID)
    if err != nil {
        span.RecordError(err)
        log.WithError(err).Error("Failed to get ADO work item")
        return
    }
    
    // ...
}
```

## Development Guidelines

- **Go Standards**: Follow standard Go idioms and structure
- **Error Handling**: Use proper error wrapping/propagation
- **Logging**: Structured logging with logrus
- **Telemetry**: OpenTelemetry instrumentation
- **Testing**: Unit test core functionality
- **Linting**: Run golangci-lint against all code changes
- **Configuration**: Environment variables for all configurable parts
- **MongoDB**: Use appropriate indexes and efficient queries

## AI Assistant Instructions

When helping with this codebase:

1. **Suggest code** that follows Go idioms and project patterns
2. **Consider API limits** for both Azure DevOps and Asana
3. **Maintain error handling** with proper logging and tracing
4. **Check for race conditions** in the controller-worker model
5. **Structure MongoDB queries** for efficiency
6. **Reference existing patterns** in the codebase
7. **Include telemetry** in any new functionality
8. **Implement graceful failure modes** for sync operations
9. **Follow the rules in the .editorconfig file**

## Environment Configuration

```
# Required environment variables
ADO_ORG_URL=https://dev.azure.com/organization
ADO_PAT=personal_access_token
ASANA_PAT=personal_access_token
MONGO_URI=mongodb://localhost:27017
SLEEP_TIME=5m
UPTRACE_DSN=https://token@api.uptrace.dev/project_id
UPTRACE_ENVIRONMENT=development
```

## Codebase Status

This is a work-in-progress project. When suggesting changes or additions, focus on:
1. Completing core sync functionality
2. Improving error handling and reliability
3. Enhancing the web UI for configuration
4. Adding telemetry for operational visibility
