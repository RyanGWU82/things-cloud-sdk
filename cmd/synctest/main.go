package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	// Show detailed changes (last 10)
	if len(changes) > 0 {
		fmt.Println("\n--- New Changes ---")
		limit := 10
		if len(changes) < limit {
			limit = len(changes)
		}
		for i := len(changes) - limit; i < len(changes); i++ {
			printChange(changes[i])
		}
	}

	// Daily summary
	dailySummary(syncer)

	// Inbox alert
	inboxAlert(syncer)

	// Today view with context
	todayWithContext(syncer)

	// Reschedule patterns for inbox items
	reschedulePatterns(syncer)

	fmt.Println("\n✓ Sync complete!")
}

// dailySummary shows what happened today
func dailySummary(syncer *sync.Syncer) {
	today := time.Now().Truncate(24 * time.Hour)
	changes, _ := syncer.ChangesSince(today)

	if len(changes) == 0 {
		return
	}

	fmt.Println("\n--- Today's Activity ---")

	completed := filterChanges(changes, "TaskCompleted")
	created := filterChanges(changes, "TaskCreated")
	movedToToday := filterChanges(changes, "TaskMovedToToday")
	movedAway := filterChangesPrefix(changes, "TaskMovedTo")

	fmt.Printf("  ✓ Completed: %d\n", len(completed))
	fmt.Printf("  + Created: %d\n", len(created))
	fmt.Printf("  → Moved to Today: %d\n", len(movedToToday))
	fmt.Printf("  ↔ Rescheduled: %d\n", len(movedAway)-len(movedToToday))

	// Show completed tasks
	if len(completed) > 0 {
		fmt.Println("\n  Completed today:")
		for _, c := range completed {
			if tc, ok := c.(sync.TaskCompleted); ok && tc.Task != nil {
				fmt.Printf("    ✓ %s\n", tc.Task.Title)
			}
		}
	}
}

// inboxAlert warns if inbox is growing too large
func inboxAlert(syncer *sync.Syncer) {
	state := syncer.State()
	inbox, _ := state.TasksInInbox(sync.QueryOpts{})

	fmt.Println("\n--- Inbox Status ---")
	fmt.Printf("  📥 %d items\n", len(inbox))

	if len(inbox) > 10 {
		fmt.Println("  ⚠️  Inbox has grown large — consider triaging")
	} else if len(inbox) == 0 {
		fmt.Println("  ✨ Inbox zero!")
	}

	// Show inbox items
	if len(inbox) > 0 {
		fmt.Println("\n  Items:")
		for _, t := range inbox {
			age := taskAge(t)
			if age > 0 {
				fmt.Printf("    - %s (%dd old)\n", t.Title, age)
			} else {
				fmt.Printf("    - %s\n", t.Title)
			}
		}
	}
}

// todayWithContext shows Today view with task history
func todayWithContext(syncer *sync.Syncer) {
	state := syncer.State()
	today, _ := state.TasksInToday(sync.QueryOpts{})

	if len(today) == 0 {
		fmt.Println("\n--- Today ---")
		fmt.Println("  No tasks scheduled for today")
		return
	}

	fmt.Println("\n--- Today ---")
	fmt.Printf("  %d tasks scheduled\n\n", len(today))

	for _, task := range today {
		changes, _ := syncer.ChangesForEntity(task.UUID)
		age := daysSinceCreated(changes)
		moves := countMoves(changes)

		// Build context string
		var context []string
		if age > 1 {
			context = append(context, fmt.Sprintf("%dd old", age))
		}
		if moves > 0 {
			context = append(context, fmt.Sprintf("moved %dx", moves))
		}

		if len(context) > 0 {
			fmt.Printf("  - %s (%s)\n", task.Title, strings.Join(context, ", "))
		} else {
			fmt.Printf("  - %s\n", task.Title)
		}

		// Warn about frequently rescheduled tasks
		if moves >= 3 {
			fmt.Printf("    ⚠️  Rescheduled %d times — consider breaking down or delegating\n", moves)
		}
	}
}

// reschedulePatterns checks for tasks that keep getting pushed
func reschedulePatterns(syncer *sync.Syncer) {
	state := syncer.State()

	// Check inbox items
	inbox, _ := state.TasksInInbox(sync.QueryOpts{})
	var problematic []struct {
		title string
		moves int
		age   int
	}

	for _, task := range inbox {
		changes, _ := syncer.ChangesForEntity(task.UUID)
		moves := countMoves(changes)
		age := daysSinceCreated(changes)

		if moves >= 3 || age >= 7 {
			problematic = append(problematic, struct {
				title string
				moves int
				age   int
			}{task.Title, moves, age})
		}
	}

	// Also check today items
	today, _ := state.TasksInToday(sync.QueryOpts{})
	for _, task := range today {
		changes, _ := syncer.ChangesForEntity(task.UUID)
		moves := countMoves(changes)
		age := daysSinceCreated(changes)

		if moves >= 3 {
			problematic = append(problematic, struct {
				title string
				moves int
				age   int
			}{task.Title, moves, age})
		}
	}

	if len(problematic) > 0 {
		fmt.Println("\n--- Accountability Check ---")
		fmt.Printf("  ⚠️  %d tasks need attention:\n\n", len(problematic))
		for _, p := range problematic {
			var reasons []string
			if p.moves >= 3 {
				reasons = append(reasons, fmt.Sprintf("moved %dx", p.moves))
			}
			if p.age >= 7 {
				reasons = append(reasons, fmt.Sprintf("%dd old", p.age))
			}
			fmt.Printf("  - %s (%s)\n", p.title, strings.Join(reasons, ", "))
		}
	}
}

// Helper functions

func printChange(c sync.Change) {
	uuid := c.EntityUUID()
	if len(uuid) > 8 {
		uuid = uuid[:8]
	}
	fmt.Printf("  [%s] %s", c.ChangeType(), uuid)

	switch v := c.(type) {
	case sync.TaskCreated:
		if v.Task != nil {
			fmt.Printf(" - %q", v.Task.Title)
		}
	case sync.TaskCompleted:
		if v.Task != nil {
			fmt.Printf(" - %q ✓", v.Task.Title)
		}
	case sync.TaskTitleChanged:
		if v.Task != nil {
			fmt.Printf(" → %q", v.Task.Title)
		}
	case sync.TaskNoteChanged:
		if v.Task != nil {
			fmt.Printf(" - %q (note)", v.Task.Title)
		}
	case sync.TaskMovedToToday:
		if v.Task != nil {
			fmt.Printf(" - %q → Today", v.Task.Title)
		}
	case sync.TaskMovedToInbox:
		if v.Task != nil {
			fmt.Printf(" - %q → Inbox", v.Task.Title)
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

func filterChanges(changes []sync.Change, changeType string) []sync.Change {
	var result []sync.Change
	for _, c := range changes {
		if c.ChangeType() == changeType {
			result = append(result, c)
		}
	}
	return result
}

func filterChangesPrefix(changes []sync.Change, prefix string) []sync.Change {
	var result []sync.Change
	for _, c := range changes {
		if strings.HasPrefix(c.ChangeType(), prefix) {
			result = append(result, c)
		}
	}
	return result
}

func daysSinceCreated(changes []sync.Change) int {
	for _, c := range changes {
		if c.ChangeType() == "TaskCreated" {
			return int(time.Since(c.Timestamp()).Hours() / 24)
		}
	}
	return 0
}

func countMoves(changes []sync.Change) int {
	count := 0
	for _, c := range changes {
		if strings.HasPrefix(c.ChangeType(), "TaskMovedTo") {
			count++
		}
	}
	return count
}

func taskAge(task *things.Task) int {
	if task.CreationDate.IsZero() {
		return 0
	}
	return int(time.Since(task.CreationDate).Hours() / 24)
}
