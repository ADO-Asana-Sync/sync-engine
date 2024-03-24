package main

import "time"

type SyncTask struct {
	ADOTaskID        int
	ADOLastUpdated   time.Time
	AsanaTaskID      string
	AsanaLastUpdated time.Time
}
