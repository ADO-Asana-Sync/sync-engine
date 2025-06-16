package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/asana"
	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		err := app.handleTask(ctx, wlog, task)
		if err != nil {
			wlog.WithError(err).Error("task sync failed")
		}
		if task.Result != nil {
			task.Result <- err
		}
	}
}

func (app *App) handleTask(ctx context.Context, wlog *log.Entry, task SyncTask) error {
	tctx, span := app.Tracer.Start(ctx, "sync.worker.taskItem")
	defer span.End()

	wlog.Infof("syncing ADO work item %v", task.ADOTaskID)

	mapping, wi, name, desc, err := app.prepWorkItem(tctx, task.ADOTaskID)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		wlog.WithError(err).Error("failure preparing work item")
		return err
	}

	if mapping != nil {
		return app.updateExistingTask(tctx, wi, *mapping, name, desc)
	}

	asanaProj, workspace, err := app.asanaProjectForADO(tctx, wi.TeamProject)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		wlog.WithError(err).WithField("project", wi.TeamProject).Error("error getting Asana project for ADO project")
		return err
	}
	if asanaProj == "" {
		wlog.WithField("project", wi.TeamProject).Debug("project not mapped to Asana, skipping")
		return nil
	}

	updated, err := app.tryUpdateExistingAsanaTask(tctx, asanaProj, workspace, wi, name, desc)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		wlog.WithError(err).WithField("project", wi.TeamProject).Error("error updating existing Asana task")
		return err
	}
	if updated {
		return nil
	}

	return app.createAndMapTask(tctx, asanaProj, workspace, wi, name, desc)
}

func (app *App) prepWorkItem(ctx context.Context, id int) (*db.TaskMapping, azure.WorkItem, string, string, error) {
	mapping, err := app.DB.TaskByADOTaskID(ctx, id)
	found := err == nil

	wi, err := app.Azure.GetWorkItem(ctx, id)
	if err != nil {
		return nil, azure.WorkItem{}, "", "", err
	}

	name, err := wi.FormatTitle()
	if err != nil {
		return nil, azure.WorkItem{}, "", "", err
	}

	desc, err := wi.FormatTitleWithLink()
	if err != nil {
		return nil, azure.WorkItem{}, "", "", err
	}

	if found {
		return &mapping, wi, name, desc, nil
	}
	return nil, wi, name, desc, nil
}

func (app *App) updateExistingTask(ctx context.Context, wi azure.WorkItem, mapping db.TaskMapping, name, desc string) error {
	cf, ok := app.getLinkCustomField(ctx, mapping.AsanaProjectID)
	customFields := map[string]string{}
	if ok {
		customFields[cf.GID] = wi.URL
	}

	if len(customFields) > 0 {
		if err := app.Asana.UpdateTaskWithCustomFields(ctx, mapping.AsanaTaskID, name, desc, customFields); err != nil {
			return err
		}
	} else {
		if err := app.Asana.UpdateTask(ctx, mapping.AsanaTaskID, name, desc); err != nil {
			return err
		}
	}
	mapping.ADOLastUpdated = wi.ChangedDate
	mapping.AsanaLastUpdated = time.Now()
	if err := app.DB.UpdateTask(ctx, mapping); err != nil {
		return err
	}
	ws, _ := app.workspaceForADO(ctx, mapping.ADOProjectID)
	app.addSyncedTag(ctx, ws, mapping.AsanaTaskID)
	return nil
}

func (app *App) asanaProjectForADO(ctx context.Context, adoProj string) (string, string, error) {
	projects, err := app.DB.Projects(ctx)
	if err != nil {
		return "", "", err
	}
	for _, p := range projects {
		if p.ADOProjectName == adoProj {
			gid, err := app.Asana.ProjectGIDByName(ctx, p.AsanaWorkspaceName, p.AsanaProjectName)
			return gid, p.AsanaWorkspaceName, err
		}
	}
	log.WithField("project", adoProj).Debug("no project mapping found")
	return "", "", nil
}

func (app *App) tryUpdateExistingAsanaTask(ctx context.Context, asanaProj, workspace string, wi azure.WorkItem, name, desc string) (bool, error) {
	tasks, err := app.Asana.ListProjectTasks(ctx, asanaProj)
	if err != nil {
		return false, err
	}
	for _, t := range tasks {
		if t.Name != name {
			continue
		}
		if err := app.updateExistingByName(ctx, t.GID, asanaProj, workspace, wi, name, desc); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// updateExistingByName updates an Asana task and records a new mapping entry.
func (app *App) updateExistingByName(ctx context.Context, taskID, projectID, workspace string, wi azure.WorkItem, name, desc string) error {
	cf, ok := app.getLinkCustomField(ctx, projectID)
	customFields := map[string]string{}
	if ok {
		customFields[cf.GID] = wi.URL
	}

	var err error
	if len(customFields) > 0 {
		err = app.Asana.UpdateTaskWithCustomFields(ctx, taskID, name, desc, customFields)
	} else {
		err = app.Asana.UpdateTask(ctx, taskID, name, desc)
	}
	if err != nil {
		return err
	}

	m := db.TaskMapping{
		ADOProjectID:     wi.TeamProject,
		ADOTaskID:        wi.ID,
		ADOLastUpdated:   wi.ChangedDate,
		AsanaProjectID:   projectID,
		AsanaTaskID:      taskID,
		AsanaLastUpdated: time.Now(),
	}
	if err := app.DB.AddTask(ctx, m); err != nil {
		return err
	}
	app.addSyncedTag(ctx, workspace, taskID)
	return nil
}

func (app *App) createAndMapTask(ctx context.Context, asanaProj, workspace string, wi azure.WorkItem, name, desc string) error {
	cf, ok := app.getLinkCustomField(ctx, asanaProj)
	customFields := map[string]string{}
	if ok {
		customFields[cf.GID] = wi.URL
	}

	var (
		newTask asana.Task
		err     error
	)
	if len(customFields) > 0 {
		newTask, err = app.Asana.CreateTaskWithCustomFields(ctx, asanaProj, name, desc, customFields)
	} else {
		newTask, err = app.Asana.CreateTask(ctx, asanaProj, name, desc)
	}
       if err != nil {
               return err
       }
       m := db.TaskMapping{
                ADOProjectID:     wi.TeamProject,
                ADOTaskID:        wi.ID,
                ADOLastUpdated:   wi.ChangedDate,
                AsanaProjectID:   asanaProj,
                AsanaTaskID:      newTask.GID,
                AsanaLastUpdated: time.Now(),
                CreatedAt:        time.Now(),
                UpdatedAt:        time.Now(),
       }
       if err := app.DB.AddTask(ctx, m); err != nil {
               return err
       }
       app.addSyncedTag(ctx, workspace, newTask.GID)
       return nil
}

func (app *App) addSyncedTag(ctx context.Context, workspace, taskID string) {
	tag, ok := app.SyncedTags[workspace]
	if !ok {
		return
	}
	if err := app.Asana.AddTagToTask(ctx, taskID, tag.GID); err != nil {
		log.WithError(err).WithFields(log.Fields{"workspace": workspace, "task": taskID}).Warn("failed to add synced tag")
	}
}

// getLinkCustomField retrieves the "link" custom field for the specified
// project, using a cached value when available. The boolean return indicates
// whether the field was found.
func (app *App) getLinkCustomField(ctx context.Context, projectID string) (asana.CustomField, bool) {
	key := fmt.Sprintf("project:%s:link_field", projectID)
	item, err := app.DB.GetCacheItem(ctx, key)
	if err == nil && time.Since(item.UpdatedAt) < app.CacheTTL {
		gid, _ := item.Value["gid"].(string)
		name, _ := item.Value["name"].(string)
		if gid != "" {
			return asana.CustomField{GID: gid, Name: name}, true
		}
	}

	cf, err := app.Asana.ProjectCustomFieldByName(ctx, projectID, "link")
	if err != nil {
		return asana.CustomField{}, false
	}
	_ = app.DB.UpsertCacheItem(ctx, db.CacheItem{
		Key:   key,
		Value: map[string]interface{}{"gid": cf.GID, "name": cf.Name},
	})
	return cf, true
}

func (app *App) workspaceForADO(ctx context.Context, adoProj string) (string, error) {
	projects, err := app.DB.Projects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.ADOProjectName == adoProj {
			return p.AsanaWorkspaceName, nil
		}
	}
	return "", nil
}
