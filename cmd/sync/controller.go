package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (app *App) controller(ctx context.Context, syncTasks chan<- SyncTask) {
	// Configure the tracing.
	ctx, span := app.Tracer.Start(ctx, "sync.controller")
	defer span.End()

	log.Info("controller started")

	// List all ADO projects
	lastSync := app.DB.LastSync(ctx)
	log.WithField("lastSyncTime", lastSync.Time).Info("get items modified since last sync")
	items, err := app.Azure.GetChangedWorkItems(ctx, lastSync.Time)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Fatal("error getting projects")
	}

	resultCh := make(chan error, len(items))
	for _, t := range items {
		syncTasks <- SyncTask{ADOTaskID: *t.Id, AsanaTaskID: "", Result: resultCh}
	}

	success := true
	for i := 0; i < len(items); i++ {
		if err := <-resultCh; err != nil {
			success = false
		}
	}

	if success {
		if err := app.DB.WriteLastSync(ctx, time.Now()); err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
			log.WithError(err).Error("error writing last sync time")
		}
	}
}
