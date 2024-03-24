package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (app *App) worker(ctx context.Context, id int, syncTasks <-chan SyncTask) {
	wlog := log.WithField("worker", id)
	wlog.Infof("worker started")

	for task := range syncTasks {
		_, span := app.Tracer.Start(ctx, "sync.worker.taskItem")
		wlog.Infof("syncing %v", task.ADOTaskID)
		span.End()
	}
}
