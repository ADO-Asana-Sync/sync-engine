package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/azure"
	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	log "github.com/sirupsen/logrus"
)

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		if err := app.handleTask(ctx, wlog, task); err != nil {
			wlog.WithError(err).Error("task sync failed")
		}
	}
}

func (app *App) handleTask(ctx context.Context, wlog *log.Entry, task SyncTask) error {
	tctx, span := app.Tracer.Start(ctx, "sync.worker.taskItem")
	defer span.End()

	wlog.Infof("syncing %v", task.ADOTaskID)

	mapping, wi, name, desc, err := app.prepWorkItem(tctx, task.ADOTaskID)
	if err != nil {
		return err
	}

	if mapping != nil {
		return app.updateExistingTask(tctx, wi, *mapping, name, desc)
	}

	asanaProj, err := app.asanaProjectForADO(tctx, wi.TeamProject)
	if err != nil {
		return err
	}

	updated, err := app.tryUpdateExistingAsanaTask(tctx, asanaProj, wi, name, desc)
	if err != nil {
		return err
	}
	if updated {
		return nil
	}

	return app.createAndMapTask(tctx, asanaProj, wi, name, desc)
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
	if err := app.Asana.UpdateTask(ctx, mapping.AsanaTaskID, name, desc); err != nil {
		return err
	}
	mapping.ADOLastUpdated = wi.ChangedDate
	mapping.AsanaLastUpdated = time.Now()
	return app.DB.UpdateTask(ctx, mapping)
}

func (app *App) asanaProjectForADO(ctx context.Context, adoProj string) (string, error) {
	projects, err := app.DB.Projects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.ADOProjectName == adoProj {
			return p.AsanaProjectName, nil
		}
	}
	log.WithField("project", adoProj).Warn("no project mapping found")
	return "", fmt.Errorf("no project mapping")
}

func (app *App) tryUpdateExistingAsanaTask(ctx context.Context, asanaProj string, wi azure.WorkItem, name, desc string) (bool, error) {
	tasks, err := app.Asana.ListProjectTasks(ctx, asanaProj)
	if err != nil {
		return false, err
	}
	for _, t := range tasks {
		if t.Name == name {
			if err := app.Asana.UpdateTask(ctx, t.GID, name, desc); err != nil {
				return false, err
			}
			m := db.TaskMapping{
				ADOProjectID:     wi.TeamProject,
				ADOTaskID:        wi.ID,
				ADOLastUpdated:   wi.ChangedDate,
				AsanaProjectID:   asanaProj,
				AsanaTaskID:      t.GID,
				AsanaLastUpdated: time.Now(),
			}
			if err := app.DB.AddTask(ctx, m); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (app *App) createAndMapTask(ctx context.Context, asanaProj string, wi azure.WorkItem, name, desc string) error {
	newTask, err := app.Asana.CreateTask(ctx, asanaProj, name, desc)
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
	return app.DB.AddTask(ctx, m)
}
