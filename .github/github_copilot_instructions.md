You are a software engineer building a Go application to sync work items from Azure DevOps (ADO) to Asana using the official APIs. Configuration and sync records are stored in MongoDB. Azure DevOps is the single source of truth; the MVP supports one-way sync (ADO → Asana).

Requirements:

1. **Task Adoption**: Detect existing incomplete Asana tasks by title and adopt them.
2. **Work Item Mapping**: Sync ADO backlog items at the “Requirements” level as Asana tasks; sync child work items of Requirements as Asana subtasks; for parent backlog levels (e.g., Feature/Epic), map tasks into Asana sections named after the parent, choosing the section for the lowest-level parent if multiple matches exist.
3. **One-Way Sync**: Sync new and updated ADO work items to Asana tasks and subtasks.
4. **Tagging**: Tag all synced Asana tasks with a configurable tag (default: “Synced”) and ensure the tag remains applied.
5. **Database Records**: Record each ADO–Asana item pair in MongoDB; when an ADO work item is deleted, remove its Asana task; if an Asana task is missing, recreate it.
6. **Retention**: Purge database records for tasks closed longer than a configurable period (default: 365 days).
7. **Delta Sync**: Implement a controller that polls Azure DevOps for changes since the last sync, queues each changed item, and dispatches them to worker processes for individual syncing.
8. **Full Sync**: Schedule periodic full syncs that compare all managed items’ timestamps against the database, prioritizing open tasks first, then closed tasks, without disrupting delta syncs.
9. **Web UI**: Provide a web interface for configuration, manual triggers for full and delta syncs, log viewing, and user authentication via Microsoft Entra ID (OAuth).
10. **Pull Request Sync**: Create Asana tasks for all pull requests in ADO project repositories, record each in MongoDB, and when a pull request is merged or deleted, update or remove the corresponding Asana task.
11. **User Assignment Filtering**: Only sync items assigned to users who exist in Asana, matching on exact email; if no match, retry matching using the email local-part (e.g., [john.doe@domain1.com](mailto:john.doe@domain1.com) matches [john.doe@domain2.com](mailto:john.doe@domain2.com)). Once synced, if a synced item’s assigned user is changed to one not existing in Asana, clear the assignment on the Asana task.
12. **Project Mapping**: Only sync configured ADO–Asana project pairs. Project mappings are defined in the Web UI (pairing an ADO project with an Asana project) and stored in MongoDB; only items from mapped pairs are processed.
13. **Deployment**: Run the application in Docker containers—separate containers for the sync service, the Web UI, and the MongoDB database.
14. **Telemetry**: Instrument the application with OpenTelemetry and send telemetry data to Uptrace.
