# AI Agents Guidelines for the ADO-Asana-Sync Project

This document provides guidance for AI agents working with the ADO-Asana-Sync sync-engine codebase.

## Project Overview

ADO-Asana-Sync is a Go application that synchronizes work items from Azure DevOps (ADO) to Asana using the official APIs. The application follows a one-way sync model, with Azure DevOps serving as the single source of truth. Configuration and sync records are stored in MongoDB.

### Key Components

- **Sync Engine**: A controller-worker architecture that handles the synchronization process
- **Web UI**: A configuration interface for managing project mappings and triggering syncs
- **Database**: MongoDB for storing configuration and sync state

## Architecture

The project follows a standard Go project layout as recommended by [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

### Main Components

1. **Sync Service (`cmd/sync/`)**: 
   - `controller.go`: Polls Azure DevOps for changes and queues work items
   - `worker.go`: Processes individual sync tasks
   - `main.go`: Application entry point and setup

2. **Web UI (`cmd/web-ui/`)**:
   - Provides configuration interface
   - Manages project mappings
   - Triggers sync operations

3. **Core Libraries (`internal/`)**:
   - `asana/`: Asana API client
   - `azure/`: Azure DevOps API client
   - `db/`: MongoDB interactions
   - `helpers/`: Utility functions

## Development Focus Areas

When working on this codebase, agents should focus on implementing the following requirements:

1. **Task Adoption**: Detect existing incomplete Asana tasks by title and adopt them
2. **Work Item Mapping**: Map ADO items to Asana tasks according to hierarchy rules
3. **One-Way Sync**: Sync new and updated ADO work items to Asana
4. **Tagging**: Tag all synced Asana tasks with a configurable tag
5. **Database Records**: Track each ADO-Asana item pair in MongoDB
6. **Retention**: Purge database records for closed tasks after a configurable period
7. **Delta Sync**: Poll Azure DevOps for changes since the last sync
8. **Full Sync**: Compare all managed items' timestamps against the database
9. **Web UI**: Provide configuration, manual triggers, and authentication
10. **Pull Request Sync**: Create Asana tasks for ADO pull requests
11. **User Assignment Filtering**: Only sync items assigned to users who exist in Asana
12. **Project Mapping**: Only sync configured ADO-Asana project pairs
13. **Deployment**: Docker containers for sync service, Web UI, and MongoDB
14. **Telemetry**: OpenTelemetry instrumentation with Uptrace integration

## Coding Guidelines

1. **Go Standards**: Follow standard Go coding conventions and idioms
2. **Error Handling**: Properly handle errors and provide meaningful error messages
3. **Logging**: Use structured logging via logrus
4. **Tracing**: Instrument code with OpenTelemetry for observability
5. **Testing**: Write unit tests for all core functionality
6. **Commit Messages**: Follow [Conventional Commits](https://www.conventionalcommits.org/)
7. **Complexity Scanning**: Scan the code with gocognit and keep the complexity below 15 where possible

## Synchronization Logic

The sync engine operates as follows:

1. **Controller** (`controller.go`):
   - Retrieves the timestamp of the last successful sync from MongoDB
   - Queries Azure DevOps for all items changed since that timestamp
   - Queues changed items for processing by workers

2. **Workers** (`worker.go`):
   - Process items from the queue
   - Retrieve full details from Azure DevOps
   - Check if the item exists in Asana (by title for new items, or by stored mapping)
   - Create or update the corresponding Asana task
   - Store the mapping in MongoDB

## Data Models

Key data structures include:

1. **Project Mapping**: Links an Azure DevOps project to an Asana project
2. **Task Mapping**: Tracks the relationship between an ADO work item and its Asana task
3. **SyncTask**: Represents a unit of work in the sync queue

## Environment Setup

The application requires the following environment variables:

- `ADO_ORG_URL`: Azure DevOps organization URL
- `ADO_PAT`: Azure DevOps Personal Access Token
- `ASANA_PAT`: Asana Personal Access Token
- `MONGO_URI`: MongoDB connection string
- `SLEEP_TIME`: Duration between sync cycles
- `UPTRACE_DSN`: Uptrace Data Source Name
- `UPTRACE_ENVIRONMENT`: Environment name for Uptrace

## Docker Deployment

The application is containerized with separate services:
- `sync`: The sync engine service
- `web-ui`: The web interface
- `mongo`: The MongoDB database

Run the application with Docker Compose:
```
docker compose up --build --remove-orphans
```

## Resource Links

- GitHub Repository: [https://github.com/ADO-Asana-Sync/sync-engine](https://github.com/ADO-Asana-Sync/sync-engine)
- Documentation: See README.md and inline code comments

## Advice for AI Assistance

When helping with this codebase:

1. **Context Awareness**: Understand the differences between ADO and Asana concepts/models
2. **Completion Status**: Note that this is a work-in-progress project
3. **API Understanding**: Familiarize yourself with both the Azure DevOps and Asana API capabilities
4. **Concurrency Model**: Be aware of the controller-worker pattern and channel usage
5. **Database Interactions**: Consider MongoDB document structure when designing queries
6. **Telemetry**: Maintain proper tracing throughout any new code
7. **Error Handling**: Ensure robust error handling in sync operations
8. **Authentication**: Handle API tokens securely
9. **Performance Considerations**: Be mindful of API rate limits and batch operations where possible
