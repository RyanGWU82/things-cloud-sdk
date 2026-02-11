package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	things "github.com/nicolai86/things-cloud-sdk"
	"github.com/nicolai86/things-cloud-sdk/sync"
)

func main() {
	username := os.Getenv("THINGS_USERNAME")
	password := os.Getenv("THINGS_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("THINGS_USERNAME and THINGS_PASSWORD must be set")
	}

	fmt.Printf("Connecting as: %s\n", username)

	// Create client
	client := things.New(things.APIEndpoint, username, password)

	// Use persistent database to test incremental sync
	dbPath := filepath.Join(os.TempDir(), "things-sync-test.db")
	fmt.Printf("Database: %s\n", dbPath)

	// Open syncer
	syncer, err := sync.Open(dbPath, client)
	if err != nil {
		log.Fatalf("Open failed: %v", err)
	}
	defer syncer.Close()

	// Show last synced index
	fmt.Printf("Last synced index: %d\n", syncer.LastSyncedIndex())

	// Sync
	fmt.Println("\nSyncing...")
	changes, err := syncer.Sync()
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}
	fmt.Printf("New changes: %d\n", len(changes))

	// Show changes by type
	typeCounts := make(map[string]int)
	for _, c := range changes {
		typeCounts[c.ChangeType()]++
	}
	if len(typeCounts) > 0 {
		fmt.Println("\nChanges by type:")
		for t, count := range typeCounts {
			fmt.Printf("  %s: %d\n", t, count)
		}
	}

	// Show detailed changes (last 10)
	if len(changes) > 0 {
		fmt.Println("\n--- Recent Changes (detailed) ---")
		limit := 10
		if len(changes) < limit {
			limit = len(changes)
		}
		for i := len(changes) - limit; i < len(changes); i++ {
			c := changes[i]
			fmt.Printf("  [%s] %s", c.ChangeType(), c.EntityUUID()[:8])

			// Type assert to get task details
			switch v := c.(type) {
			case sync.TaskCreated:
				if v.Task != nil {
					fmt.Printf(" - %q", v.Task.Title)
				}
			case sync.TaskCompleted:
				if v.Task != nil {
					fmt.Printf(" - %q completed", v.Task.Title)
				}
			case sync.TaskTitleChanged:
				if v.Task != nil {
					fmt.Printf(" → %q", v.Task.Title)
				}
			case sync.TaskNoteChanged:
				if v.Task != nil {
					fmt.Printf(" - %q note updated", v.Task.Title)
				}
			case sync.ProjectCreated:
				if v.Project != nil {
					fmt.Printf(" - %q", v.Project.Title)
				}
			case sync.AreaCreated:
				if v.Area != nil {
					fmt.Printf(" - %q", v.Area.Title)
				}
			case sync.TagCreated:
				if v.Tag != nil {
					fmt.Printf(" - %q", v.Tag.Title)
				}
			}
			fmt.Println()
		}
	}

	// Query state
	state := syncer.State()

	fmt.Println("\n--- Current State ---")

	inbox, _ := state.TasksInInbox(sync.QueryOpts{})
	fmt.Printf("Inbox: %d tasks\n", len(inbox))
	for _, t := range inbox {
		fmt.Printf("  - %s\n", t.Title)
	}

	projects, _ := state.AllProjects(sync.QueryOpts{})
	fmt.Printf("\nProjects: %d\n", len(projects))
	for _, p := range projects {
		tasks, _ := state.TasksInProject(p.UUID, sync.QueryOpts{})
		fmt.Printf("  - %s (%d tasks)\n", p.Title, len(tasks))
	}

	// Query change log
	fmt.Println("\n--- Change Log Queries ---")

	// Changes in last hour
	hourAgo := time.Now().Add(-1 * time.Hour)
	recentChanges, _ := syncer.ChangesSince(hourAgo)
	fmt.Printf("Changes in last hour: %d\n", len(recentChanges))

	// Changes for a specific task (first inbox task)
	if len(inbox) > 0 {
		taskChanges, _ := syncer.ChangesForEntity(inbox[0].UUID)
		fmt.Printf("Changes for '%s': %d\n", inbox[0].Title, len(taskChanges))
		for _, c := range taskChanges {
			fmt.Printf("  - %s\n", c.ChangeType())
		}
	}

	fmt.Println("\nTest complete!")
}
