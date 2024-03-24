# Sync Plan

## Overview

* Store a timestamp of the last successful sync.
* Using sync timestamp, fetch all changes since last sync.
* Compare the task IDs in the delta sync with the DB IDs.
  * If task ID is not in the DB, create a new sync task.
  * If task ID is in the DB, update the sync task.
* TODO: Figure out orphaned task handling.
