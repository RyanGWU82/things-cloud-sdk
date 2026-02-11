# things cloud sdk

[Things](https://culturedcode.com/things/) comes with a cloud based API, which can
be used to synchronize data between devices.
This is a golang SDK to interact with that API, opening the API so that you
can enhance your Things experience on iOS and Mac.

[![Go](https://github.com/arthursoares/things-cloud-sdk/actions/workflows/go.yml/badge.svg)](https://github.com/arthursoares/things-cloud-sdk/actions/workflows/go.yml)

## Features

- **Verify Credentials** — validate account access
- **Account Management** — signup, confirmation, password change, deletion
- **History Management** — list, create, delete, sync histories
- **Item Read/Write** — full event-sourced CRUD for tasks, areas, tags, checklist items
- **Task Types** — tasks, projects, and headings (action groups within projects)
- **Structured Notes** — full-text and delta patch support for task notes
- **Recurring Tasks** — neverending, end on date, end after N times
- **Tombstone Deletion** — explicit deletion records via `Tombstone2` entities
- **Device Registration** — register app instances for APNS push notifications
- **Alarm/Reminders** — alarm time offset support on tasks
- **State Aggregation** — in-memory state built from history items, with queries for projects, headings, subtasks, areas, tags, and checklist items
- **Persistent Sync Engine** — SQLite-backed incremental sync with semantic change detection

## CLI

`things-cli` is a command-line tool for interacting with Things Cloud directly.

### Setup

```bash
export THINGS_USERNAME='your@email.com'
export THINGS_PASSWORD='yourpassword'
go build -o things-cli ./cmd/things-cli/
```

### Commands

```bash
# Read
things-cli list [--today] [--inbox] [--area NAME] [--project NAME]
things-cli show <uuid>
things-cli areas
things-cli projects
things-cli tags

# Create
things-cli create "Title" [--note ...] [--when today|anytime|someday|inbox] \
  [--deadline YYYY-MM-DD] [--scheduled YYYY-MM-DD] \
  [--project UUID] [--heading UUID] [--area UUID] \
  [--tags UUID,...] [--type task|project|heading]
things-cli create-area "Name"
things-cli create-tag "Name" [--shorthand KEY] [--parent UUID]

# Modify
things-cli edit <uuid> [--title ...] [--note ...] [--when ...] [--deadline ...]
things-cli complete <uuid>
things-cli trash <uuid>
things-cli purge <uuid>
things-cli move-to-today <uuid>
```

### Examples

```bash
# Create a project with tasks
things-cli create "My Project" --type project --when anytime
# → {"status":"created","uuid":"BXmAcvS6yK1eDhW31MuZrL","title":"My Project"}

things-cli create "First Task" --project BXmAcvS6yK1eDhW31MuZrL --when today --note "Details here"

# Create an area and assign tasks
things-cli create-area "Work"
things-cli create "Review PR" --area <area-uuid> --when today --deadline 2026-02-15
```

## SDK Usage

```go
package main

import (
    "fmt"
    "os"

    thingscloud "github.com/nicolai86/things-cloud-sdk"
)

func main() {
    c := thingscloud.New(
        thingscloud.APIEndpoint,
        os.Getenv("THINGS_USERNAME"),
        os.Getenv("THINGS_PASSWORD"),
    )

    resp, err := c.Verify()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Account: %s (status: %s)\n", resp.Email, resp.Status)
}
```

See the `example/` directory for a more complete example including history sync, task creation, and state aggregation.

## Persistent Sync Engine

The `sync` package provides a SQLite-backed sync engine that tracks "what changed since last sync" — perfect for building agents, automations, or dashboards that react to Things changes.

```go
package main

import (
    "fmt"
    thingscloud "github.com/nicolai86/things-cloud-sdk"
    "github.com/nicolai86/things-cloud-sdk/sync"
)

func main() {
    client := thingscloud.New(thingscloud.APIEndpoint, email, password)

    // Open persistent sync database
    syncer, _ := sync.Open("things.db", client)
    defer syncer.Close()

    // Fetch changes since last sync
    changes, _ := syncer.Sync()

    for _, c := range changes {
        switch v := c.(type) {
        case sync.TaskCreated:
            fmt.Printf("New task: %s\n", v.Task.Title)
        case sync.TaskCompleted:
            fmt.Printf("Completed: %s\n", v.Task.Title)
        case sync.TaskMovedToToday:
            fmt.Printf("Moved to Today: %s\n", v.Task.Title)
        }
    }

    // Query current state
    state := syncer.State()
    inbox, _ := state.TasksInInbox(sync.QueryOpts{})
    projects, _ := state.AllProjects(sync.QueryOpts{})
}
```

### Semantic Change Types

The sync engine detects 40+ semantic change types:

| Category | Changes |
|----------|---------|
| **Task Lifecycle** | `TaskCreated`, `TaskCompleted`, `TaskUncompleted`, `TaskTrashed`, `TaskDeleted` |
| **Task Movement** | `TaskMovedToInbox`, `TaskMovedToToday`, `TaskMovedToAnytime`, `TaskMovedToSomeday`, `TaskMovedToUpcoming` |
| **Task Organization** | `TaskMovedToProject`, `TaskMovedToArea`, `TaskMovedUnderHeading`, `TaskTagsChanged` |
| **Task Details** | `TaskTitleChanged`, `TaskNoteChanged`, `TaskDeadlineSet`, `TaskDeadlineRemoved` |
| **Projects** | `ProjectCreated`, `ProjectCompleted`, `ProjectTrashed`, `ProjectDeleted` |
| **Areas & Tags** | `AreaCreated`, `AreaDeleted`, `TagCreated`, `TagDeleted` |
| **Checklists** | `ChecklistItemCreated`, `ChecklistItemCompleted`, `ChecklistItemDeleted` |

### State Queries

```go
state := syncer.State()

// Query by location
inbox, _ := state.TasksInInbox(sync.QueryOpts{})
today, _ := state.TasksInToday(sync.QueryOpts{})
anytime, _ := state.TasksInAnytime(sync.QueryOpts{})

// Query by container
tasks, _ := state.TasksInProject(projectUUID, sync.QueryOpts{})
tasks, _ := state.TasksInArea(areaUUID, sync.QueryOpts{})

// List all
projects, _ := state.AllProjects(sync.QueryOpts{})
areas, _ := state.AllAreas(sync.QueryOpts{})
tags, _ := state.AllTags(sync.QueryOpts{})
```

### Change Log Queries

```go
// Changes in last hour
changes, _ := syncer.ChangesSince(time.Now().Add(-1 * time.Hour))

// Changes for a specific task
changes, _ := syncer.ChangesForEntity(taskUUID)

// Changes since server index
changes, _ := syncer.ChangesSinceIndex(150)
```

## Wire Format Notes

Key findings from reverse engineering the Things Cloud sync protocol:

- **UUIDs must be Base58-encoded** (Bitcoin alphabet: `123456789ABCDEFGH...`). Standard UUID strings or other encodings will crash Things.app during sync.
- **`md` (modification date) must be `null` on creates.** Set timestamps only on updates.
- **Schedule field (`st`)**: `0` = Inbox, `1` = Anytime/Today (with `sr`/`tir` dates = Today), `2` = Someday/Upcoming (with dates = Upcoming).
- **Status field (`ss`)**: `0` = Pending, `2` = Canceled, `3` = Completed. Don't confuse with `st` (schedule)!
- **Headings (`tp=2`) must have `st=1`** (anytime). `st=0` (inbox) crashes Things.app.
- **Tasks in projects, headings, or areas** should default to `st=1` (anytime) — they've been triaged out of inbox.
- **Kind strings**: `Task6`, `Tag4`, `ChecklistItem3`, `Area3`, `Tombstone2`

See `docs/client-side-bugs.md` for the full investigation and crash analysis.

## Architecture

The SDK models all changes as immutable Items (events). A History is a sync stream identified by a UUID. The client pushes/pulls Items through Histories, inspired by [operational transformations and Git's internals](https://www.swift.org/blog/how-swifts-server-support-powers-things-cloud/).

## TODO

- [ ] Repeat after completion
- [x] Persistent state storage (see `sync` package)

## Note

As there is no official API documentation available all requests need to be reverse engineered,
which takes some time. Feel free to contribute and improve & extend this implementation.
