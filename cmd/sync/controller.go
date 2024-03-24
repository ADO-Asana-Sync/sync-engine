package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (app *App) controller(ctx context.Context, syncTasks chan<- SyncTask) {
	// Configure the tracing.
	_, span := app.Tracer.Start(ctx, "sync.controller")
	defer span.End()

	log.Info("controller started")

	// // List all projects in DB
	// projects, err := app.DB.Projects()
	// if err != nil {
	// 	log.WithError(err).Fatal("error listing projects")
	// }
	// spew.Dump(projects)

	// // List all ADO projects
	// proj, err := app.Azure.GetProjects(ctx)
	// if err != nil {
	// 	span.RecordError(err, trace.WithStackTrace(true))
	// 	span.SetStatus(codes.Error, err.Error())
	// 	log.WithError(err).Fatal("error getting projects")
	// }
	// for _, p := range proj {
	// 	syncTasks <- SyncTask{ADOTaskID: p.Id.String(), AsanaTaskID: ""}
	// }

	// List all ADO projects
	lastSync := app.DB.LastSync()
	log.WithField("lastSyncTime", lastSync.Time).Info("get items modified since last sync")
	tasks, err := app.Azure.GetChangedWorkItems(ctx, lastSync.Time)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Fatal("error getting projects")
	}
	for _, t := range tasks {
		syncTasks <- SyncTask{ADOTaskID: *t.Id, AsanaTaskID: ""}
	}
}
