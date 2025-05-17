package main

import (
	"context"
	"time"
)

type SyncTask struct {
	ADOTaskID        int
	ADOLastUpdated   time.Time
	AsanaTaskID      string
	AsanaLastUpdated time.Time
}

func (t *SyncTask) Sync(app *App, ctx context.Context) error {
	// TODO: implement sync logic between ADO and Asana
	return nil
}
