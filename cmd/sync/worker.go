package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		_, span := app.Tracer.Start(ctx, "sync.worker.taskItem")
		wlog.Infof("syncing %v", task.ADOTaskID)

		// Check for existing TaskMapping
		taskMapping, err := app.db.GetTaskMappingByADOTaskID(ctx, task.ADOTaskID)
		if err != nil {
			// TODO: Handle error properly - maybe retry or send to a dead-letter queue
			wlog.Errorf("error getting task mapping: %v", err)
			span.End()
			continue
		}

		if taskMapping != nil {
			// Mapping exists - Update existing Asana task
			wlog.Infof("mapping exists for ADO task %d, Asana task %s", task.ADOTaskID, taskMapping.AsanaTaskID)

			// Fetch Azure DevOps work item details
			adoWorkItem, err := app.azureClient.GetWorkItemDetails(ctx, task.ADOTaskID)
			if err != nil {
				wlog.Errorf("error getting ADO work item details for ID %d: %v", task.ADOTaskID, err)
				span.End()
				continue
			}

			// Prepare Asana task details using FormatTitle and FormatTitleWithLink
			asanaTaskName, err := adoWorkItem.FormatTitle()
			if err != nil {
				wlog.Errorf("error formatting ADO work item title for ID %d: %v", adoWorkItem.ID, err)
				span.End()
				continue
			}
			asanaTaskDescription, err := adoWorkItem.FormatTitleWithLink()
			if err != nil {
				wlog.Errorf("error formatting ADO work item description for ID %d: %v", adoWorkItem.ID, err)
				span.End()
				continue
			}

			// Update Asana task
			_, err = app.asanaClient.UpdateTask(ctx, taskMapping.AsanaTaskID, asanaTaskName, asanaTaskDescription)
			if err != nil {
				wlog.Errorf("error updating Asana task %s for ADO task %d: %v", taskMapping.AsanaTaskID, task.ADOTaskID, err)
				span.End()
				continue
			}

			// Update TaskMapping in the database
			taskMapping.SyncedAt = time.Now()
			err = app.db.UpdateTaskMapping(ctx, taskMapping)
			if err != nil {
				wlog.Errorf("error updating task mapping for ADO task %d: %v", task.ADOTaskID, err)
				span.End()
				continue
			}
			wlog.Infof("successfully updated Asana task %s for ADO task %d", taskMapping.AsanaTaskID, task.ADOTaskID)

		} else {
			// No mapping exists
			wlog.Infof("no mapping exists for ADO task %d", task.ADOTaskID)

			// Fetch Azure DevOps work item details
			adoWorkItem, err := app.azureClient.GetWorkItemDetails(ctx, task.ADOTaskID)
			if err != nil {
				wlog.Errorf("error getting ADO work item details for ID %d: %v", task.ADOTaskID, err)
				span.End()
				continue
			}

			// Prepare Asana task details using FormatTitle and FormatTitleWithLink
			asanaTaskName, err := adoWorkItem.FormatTitle()
			if err != nil {
				wlog.Errorf("error formatting ADO work item title for ID %d: %v", adoWorkItem.ID, err)
				span.End()
				continue
			}
			asanaTaskDescription, err := adoWorkItem.FormatTitleWithLink()
			if err != nil {
				wlog.Errorf("error formatting ADO work item description for ID %d: %v", adoWorkItem.ID, err)
				span.End()
				continue
			}

			// Search for existing Asana task by title
			// Note: worker.go was previously passing adoWorkItem.Title, now using formatted title.
			// The subtask description implies searching by the ADO title, not the formatted one.
			// For now, I'll stick to adoWorkItem.Title for search, as FormatTitle() adds type prefix like "Bug 123: Title".
			// This might need clarification if the intent was to search by the fully formatted title.
			// For the test description, it says "Verify that asanaClient.FindTaskByTitle is called." - does not specify which title.
			// The previous subtask for implementing worker.go said "Search for an existing Asana task by title within the linked project."
			// Let's assume it means the raw title from ADO.
			existingAsanaTask, err := app.asanaClient.FindTaskByTitle(ctx, task.ProjectID, adoWorkItem.Title)
			if err != nil {
				wlog.Errorf("error finding Asana task by title '%s' for ADO task %d: %v", adoWorkItem.Title, task.ADOTaskID, err)
				span.End()
				continue
			}

			if existingAsanaTask != nil {
				// Matching Asana task found
				wlog.Infof("found matching Asana task %s for ADO task %d by title '%s'", existingAsanaTask.GID, task.ADOTaskID, adoWorkItem.Title)

				// Update Asana task
				_, err = app.asanaClient.UpdateTask(ctx, existingAsanaTask.GID, asanaTaskName, asanaTaskDescription)
				if err != nil {
					wlog.Errorf("error updating existing Asana task %s for ADO task %d: %v", existingAsanaTask.GID, task.ADOTaskID, err)
					span.End()
					continue
				}

				// Create a new TaskMapping
				newTaskMapping := &TaskMapping{ // Assuming TaskMapping struct is defined in this package or imported
					ADOTaskID:   task.ADOTaskID,
					AsanaTaskID: existingAsanaTask.GID,
					SyncedAt:    time.Now(),
				}
				err = app.db.CreateTaskMapping(ctx, newTaskMapping)
				if err != nil {
					wlog.Errorf("error creating task mapping for ADO task %d and Asana task %s: %v", task.ADOTaskID, existingAsanaTask.GID, err)
					span.End()
					continue
				}
				wlog.Infof("successfully created task mapping for ADO task %d and Asana task %s", task.ADOTaskID, existingAsanaTask.GID)

			} else {
				// No matching Asana task found - Create a new Asana task
				wlog.Infof("no matching Asana task found for ADO task %d by title '%s', creating new Asana task", task.ADOTaskID, adoWorkItem.Title)

				// Create new Asana task
				newAsanaTask, err := app.asanaClient.CreateTask(ctx, task.ProjectID, asanaTaskName, asanaTaskDescription)
				if err != nil {
					wlog.Errorf("error creating new Asana task for ADO task %d: %v", task.ADOTaskID, err)
					span.End()
					continue
				}

				// Create a new TaskMapping
				newTaskMapping := &TaskMapping{ // Assuming TaskMapping struct is defined
					ADOTaskID:   task.ADOTaskID,
					AsanaTaskID: newAsanaTask.GID,
					SyncedAt:    time.Now(),
				}
				err = app.db.CreateTaskMapping(ctx, newTaskMapping)
				if err != nil {
					wlog.Errorf("error creating task mapping for ADO task %d and new Asana task %s: %v", task.ADOTaskID, newAsanaTask.GID, err)
					span.End()
					continue
				}
				wlog.Infof("successfully created new Asana task %s and task mapping for ADO task %d", newAsanaTask.GID, task.ADOTaskID)
			}
		}
		span.End()
	}
}
