package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana" // Required for asana.Task type hint if used in function signatures directly
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure" // Required for azure.WorkItem
	log "github.com/sirupsen/logrus"
)

// TaskMapping represents the link between an Azure DevOps task and an Asana task.
type TaskMapping struct {
	ADOTaskID   int       `json:"ado_task_id"`
	AsanaTaskID string    `json:"asana_task_id"`
	SyncedAt    time.Time `json:"synced_at"`
	// Potentially other fields like a GID for the mapping itself, ADO/Asana project GIDs
}

// SyncTask represents a task that needs to be synchronized.
type SyncTask struct {
	ADOTaskID       int    `json:"ado_task_id"`
	AsanaProjectGID string `json:"asana_project_gid"` // Asana Project GID
	ADOProjectID    string `json:"ado_project_id"`    // ADO Project ID (optional for current logic but good for future)
	// ProjectID was used before, now replaced by AsanaProjectGID for clarity
}

// App holds application-wide dependencies.
// This struct might be defined elsewhere in package main, but is included here
// to show what `app` refers to in the worker and helper functions.
// Ensure this matches the actual App struct used by the application.
/*
type App struct {
	db          somepackage.DBInterface // Replace with actual DB interface
	azureClient azure.AzureInterface
	asanaClient asana.AsanaInterface
	Tracer      trace.Tracer
	// Other app dependencies
}
*/

func handleExistingTaskSync(ctx context.Context, app *App, wlog *log.Entry, taskMapping *TaskMapping, syncTask SyncTask) error {
	wlog.Infof("mapping exists for ADO task %d, Asana task %s. Updating existing.", taskMapping.ADOTaskID, taskMapping.AsanaTaskID)

	adoWorkItem, err := app.azureClient.GetWorkItemDetails(ctx, taskMapping.ADOTaskID)
	if err != nil {
		wlog.Errorf("error getting ADO work item details for ID %d: %v", taskMapping.ADOTaskID, err)
		return err
	}

	asanaTaskName, err := adoWorkItem.FormatTitle()
	if err != nil {
		wlog.Errorf("error formatting ADO work item title for ID %d: %v", adoWorkItem.ID, err)
		return err
	}
	asanaTaskDescription, err := adoWorkItem.FormatTitleWithLink()
	if err != nil {
		wlog.Errorf("error formatting ADO work item description for ID %d: %v", adoWorkItem.ID, err)
		return err
	}

	_, err = app.asanaClient.UpdateTask(ctx, taskMapping.AsanaTaskID, asanaTaskName, asanaTaskDescription)
	if err != nil {
		wlog.Errorf("error updating Asana task %s for ADO task %d: %v", taskMapping.AsanaTaskID, taskMapping.ADOTaskID, err)
		return err
	}

	taskMapping.SyncedAt = time.Now()
	err = app.db.UpdateTaskMapping(ctx, taskMapping)
	if err != nil {
		wlog.Errorf("error updating task mapping for ADO task %d: %v", taskMapping.ADOTaskID, err)
		return err
	}
	wlog.Infof("successfully updated Asana task %s for ADO task %d", taskMapping.AsanaTaskID, taskMapping.ADOTaskID)
	return nil
}

func handleNewTaskSync(ctx context.Context, app *App, wlog *log.Entry, syncTask SyncTask) error {
	wlog.Infof("no mapping exists for ADO task %d. Processing as new.", syncTask.ADOTaskID)

	adoWorkItem, err := app.azureClient.GetWorkItemDetails(ctx, syncTask.ADOTaskID)
	if err != nil {
		wlog.Errorf("error getting ADO work item details for ID %d: %v", syncTask.ADOTaskID, err)
		return err
	}

	asanaTaskName, err := adoWorkItem.FormatTitle()
	if err != nil {
		wlog.Errorf("error formatting ADO work item title for ID %d: %v", adoWorkItem.ID, err)
		return err
	}
	asanaTaskDescription, err := adoWorkItem.FormatTitleWithLink()
	if err != nil {
		wlog.Errorf("error formatting ADO work item description for ID %d: %v", adoWorkItem.ID, err)
		return err
	}

	existingAsanaTask, err := app.asanaClient.FindTaskByTitle(ctx, syncTask.AsanaProjectGID, adoWorkItem.Title)
	if err != nil {
		wlog.Errorf("error finding Asana task by title '%s' in project %s for ADO task %d: %v", adoWorkItem.Title, syncTask.AsanaProjectGID, syncTask.ADOTaskID, err)
		return err
	}

	var asanaTaskGID string
	if existingAsanaTask != nil {
		wlog.Infof("found matching Asana task %s for ADO task %d by title '%s'", existingAsanaTask.GID, syncTask.ADOTaskID, adoWorkItem.Title)
		_, err = app.asanaClient.UpdateTask(ctx, existingAsanaTask.GID, asanaTaskName, asanaTaskDescription)
		if err != nil {
			wlog.Errorf("error updating existing Asana task %s for ADO task %d: %v", existingAsanaTask.GID, syncTask.ADOTaskID, err)
			return err
		}
		asanaTaskGID = existingAsanaTask.GID
	} else {
		wlog.Infof("no matching Asana task found for ADO task %d by title '%s', creating new Asana task in project %s", syncTask.ADOTaskID, adoWorkItem.Title, syncTask.AsanaProjectGID)
		newAsanaTask, err_create := app.asanaClient.CreateTask(ctx, syncTask.AsanaProjectGID, asanaTaskName, asanaTaskDescription)
		if err_create != nil { // renamed err to err_create to avoid conflict
			wlog.Errorf("error creating new Asana task in project %s for ADO task %d: %v", syncTask.AsanaProjectGID, syncTask.ADOTaskID, err_create)
			return err_create
		}
		asanaTaskGID = newAsanaTask.GID
		wlog.Infof("successfully created new Asana task %s", newAsanaTask.GID)
	}

	newTaskMapping := &TaskMapping{
		ADOTaskID:   syncTask.ADOTaskID,
		AsanaTaskID: asanaTaskGID,
		SyncedAt:    time.Now(),
	}
	err = app.db.CreateTaskMapping(ctx, newTaskMapping)
	if err != nil {
		wlog.Errorf("error creating task mapping for ADO task %d and Asana task %s: %v", syncTask.ADOTaskID, asanaTaskGID, err)
		return err
	}
	wlog.Infof("successfully created task mapping for ADO task %d and Asana task %s", syncTask.ADOTaskID, asanaTaskGID)
	return nil
}

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		_, span := app.Tracer.Start(ctx, "sync.worker.taskItem") // Make sure span is ended appropriately
		wlog.Infof("syncing ADO task %d (Asana Project GID: %s, ADO Project ID: %s)", task.ADOTaskID, task.AsanaProjectGID, task.ADOProjectID)

		taskMapping, err := app.db.GetTaskMappingByADOTaskID(ctx, task.ADOTaskID) // This now uses the ADOTaskID from the enriched SyncTask
		if err != nil {
			wlog.Errorf("error getting task mapping for ADO task ID %d: %v", task.ADOTaskID, err)
			span.End() // End span on error before continuing
			continue
		}

		if taskMapping != nil {
			err = handleExistingTaskSync(ctx, app, wlog, taskMapping, task)
			if err != nil {
				// Error is already logged by handleExistingTaskSync, just end span and continue
				span.End()
				continue
			}
		} else {
			err = handleNewTaskSync(ctx, app, wlog, task)
			if err != nil {
				// Error is already logged by handleNewTaskSync, just end span and continue
				span.End()
				continue
			}
		}
		span.End() // End span on successful completion of processing for this task
	}
}
