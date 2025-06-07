package main

import (
	"context"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/db"
	log "github.com/sirupsen/logrus"
)

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		tctx, span := app.Tracer.Start(ctx, "sync.worker.taskItem")
		wlog.Infof("syncing %v", task.ADOTaskID)

		mapping, err := app.DB.TaskByADOTaskID(tctx, task.ADOTaskID)
		found := err == nil

		wi, err := app.Azure.GetWorkItem(tctx, task.ADOTaskID)
		if err != nil {
			wlog.WithError(err).Error("failed to get work item")
			span.End()
			continue
		}

		name, err := wi.FormatTitle()
		if err != nil {
			wlog.WithError(err).Error("failed to format title")
			span.End()
			continue
		}
		desc, err := wi.FormatTitleWithLink()
		if err != nil {
			wlog.WithError(err).Error("failed to format description")
			span.End()
			continue
		}

		if found {
			if err := app.Asana.UpdateTask(tctx, mapping.AsanaTaskID, name, desc); err != nil {
				wlog.WithError(err).Error("failed to update asana task")
			} else {
				mapping.ADOLastUpdated = wi.ChangedDate
				mapping.AsanaLastUpdated = time.Now()
				_ = app.DB.UpdateTask(tctx, mapping)
			}
			span.End()
			continue
		}

		projects, err := app.DB.Projects(tctx)
		if err != nil {
			wlog.WithError(err).Error("failed to list projects")
			span.End()
			continue
		}

		var asanaProj string
		for _, p := range projects {
			if p.ADOProjectName == wi.TeamProject {
				asanaProj = p.AsanaProjectName
				break
			}
		}
		if asanaProj == "" {
			wlog.WithField("project", wi.TeamProject).Warn("no project mapping found")
			span.End()
			continue
		}

		tasks, err := app.Asana.ListProjectTasks(tctx, asanaProj)
		if err == nil {
			for _, tsk := range tasks {
				if tsk.Name == name {
					if err := app.Asana.UpdateTask(tctx, tsk.GID, name, desc); err == nil {
						m := db.TaskMapping{
							ADOProjectID:     wi.TeamProject,
							ADOTaskID:        wi.ID,
							ADOLastUpdated:   wi.ChangedDate,
							AsanaProjectID:   asanaProj,
							AsanaTaskID:      tsk.GID,
							AsanaLastUpdated: time.Now(),
						}
						_ = app.DB.AddTask(tctx, m)
					}
					span.End()
					continue
				}
			}
		}

		newTask, err := app.Asana.CreateTask(tctx, asanaProj, name, desc)
		if err != nil {
			wlog.WithError(err).Error("failed to create asana task")
			span.End()
			continue
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
		_ = app.DB.AddTask(tctx, m)

		span.End()
	}
}
