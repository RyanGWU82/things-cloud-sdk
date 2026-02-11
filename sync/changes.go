package sync

import (
	"time"

	things "github.com/nicolai86/things-cloud-sdk"
)

// Change represents a semantic change event from sync
type Change interface {
	// ChangeType returns the type of change (e.g., "TaskCreated", "TaskCompleted")
	ChangeType() string
	// EntityType returns the type of entity (e.g., "Task", "Area", "Tag")
	EntityType() string
	// EntityUUID returns the UUID of the affected entity
	EntityUUID() string
	// ServerIndex returns the server index at which this change occurred
	ServerIndex() int
	// Timestamp returns when this change was synced
	Timestamp() time.Time
}

// TaskLocation represents where a task is located in the Things UI
type TaskLocation int

const (
	LocationUnknown TaskLocation = iota
	LocationInbox
	LocationToday
	LocationAnytime
	LocationSomeday
	LocationUpcoming
	LocationProject
)

// String returns the string representation of TaskLocation
func (l TaskLocation) String() string {
	switch l {
	case LocationInbox:
		return "Inbox"
	case LocationToday:
		return "Today"
	case LocationAnytime:
		return "Anytime"
	case LocationSomeday:
		return "Someday"
	case LocationUpcoming:
		return "Upcoming"
	case LocationProject:
		return "Project"
	default:
		return "Unknown"
	}
}

// baseChange provides common fields for all change types
type baseChange struct {
	serverIndex int
	timestamp   time.Time
}

// ServerIndex returns the server index at which this change occurred
func (b baseChange) ServerIndex() int {
	return b.serverIndex
}

// Timestamp returns when this change was synced
func (b baseChange) Timestamp() time.Time {
	return b.timestamp
}

// taskChange provides common fields for task-related changes
type taskChange struct {
	baseChange
	Task *things.Task
}

// EntityType returns "Task" for all task changes
func (t taskChange) EntityType() string {
	return "Task"
}

// EntityUUID returns the UUID of the task
func (t taskChange) EntityUUID() string {
	if t.Task == nil {
		return ""
	}
	return t.Task.UUID
}

// TaskCreated indicates a new task was created
type TaskCreated struct {
	taskChange
}

// ChangeType returns "TaskCreated"
func (c TaskCreated) ChangeType() string {
	return "TaskCreated"
}

// TaskDeleted indicates a task was permanently deleted
type TaskDeleted struct {
	taskChange
}

// ChangeType returns "TaskDeleted"
func (c TaskDeleted) ChangeType() string {
	return "TaskDeleted"
}

// TaskCompleted indicates a task was marked as completed
type TaskCompleted struct {
	taskChange
}

// ChangeType returns "TaskCompleted"
func (c TaskCompleted) ChangeType() string {
	return "TaskCompleted"
}

// TaskUncompleted indicates a completed task was marked as incomplete
type TaskUncompleted struct {
	taskChange
}

// ChangeType returns "TaskUncompleted"
func (c TaskUncompleted) ChangeType() string {
	return "TaskUncompleted"
}

// TaskCanceled indicates a task was canceled
type TaskCanceled struct {
	taskChange
}

// ChangeType returns "TaskCanceled"
func (c TaskCanceled) ChangeType() string {
	return "TaskCanceled"
}

// TaskTitleChanged indicates a task's title was modified
type TaskTitleChanged struct {
	taskChange
	OldTitle string
}

// ChangeType returns "TaskTitleChanged"
func (c TaskTitleChanged) ChangeType() string {
	return "TaskTitleChanged"
}

// TaskNoteChanged indicates a task's note was modified
type TaskNoteChanged struct {
	taskChange
	OldNote string
}

// ChangeType returns "TaskNoteChanged"
func (c TaskNoteChanged) ChangeType() string {
	return "TaskNoteChanged"
}

// TaskMovedToInbox indicates a task was moved to the Inbox
type TaskMovedToInbox struct {
	taskChange
	From TaskLocation
}

// ChangeType returns "TaskMovedToInbox"
func (c TaskMovedToInbox) ChangeType() string {
	return "TaskMovedToInbox"
}

// TaskMovedToToday indicates a task was moved to Today
type TaskMovedToToday struct {
	taskChange
	From TaskLocation
}

// ChangeType returns "TaskMovedToToday"
func (c TaskMovedToToday) ChangeType() string {
	return "TaskMovedToToday"
}

// TaskMovedToAnytime indicates a task was moved to Anytime
type TaskMovedToAnytime struct {
	taskChange
	From TaskLocation
}

// ChangeType returns "TaskMovedToAnytime"
func (c TaskMovedToAnytime) ChangeType() string {
	return "TaskMovedToAnytime"
}

// TaskMovedToSomeday indicates a task was moved to Someday
type TaskMovedToSomeday struct {
	taskChange
	From TaskLocation
}

// ChangeType returns "TaskMovedToSomeday"
func (c TaskMovedToSomeday) ChangeType() string {
	return "TaskMovedToSomeday"
}

// TaskMovedToUpcoming indicates a task was scheduled for a future date
type TaskMovedToUpcoming struct {
	taskChange
	From         TaskLocation
	ScheduledFor time.Time
}

// ChangeType returns "TaskMovedToUpcoming"
func (c TaskMovedToUpcoming) ChangeType() string {
	return "TaskMovedToUpcoming"
}

// TaskDeadlineChanged indicates a task's deadline was modified
type TaskDeadlineChanged struct {
	taskChange
	OldDeadline *time.Time
}

// ChangeType returns "TaskDeadlineChanged"
func (c TaskDeadlineChanged) ChangeType() string {
	return "TaskDeadlineChanged"
}

// TaskAssignedToProject indicates a task was assigned to a project
type TaskAssignedToProject struct {
	taskChange
	Project    *things.Task
	OldProject *things.Task
}

// ChangeType returns "TaskAssignedToProject"
func (c TaskAssignedToProject) ChangeType() string {
	return "TaskAssignedToProject"
}

// TaskAssignedToArea indicates a task was assigned to an area
type TaskAssignedToArea struct {
	taskChange
	Area    *things.Area
	OldArea *things.Area
}

// ChangeType returns "TaskAssignedToArea"
func (c TaskAssignedToArea) ChangeType() string {
	return "TaskAssignedToArea"
}

// TaskTrashed indicates a task was moved to trash
type TaskTrashed struct {
	taskChange
}

// ChangeType returns "TaskTrashed"
func (c TaskTrashed) ChangeType() string {
	return "TaskTrashed"
}

// TaskRestored indicates a task was restored from trash
type TaskRestored struct {
	taskChange
}

// ChangeType returns "TaskRestored"
func (c TaskRestored) ChangeType() string {
	return "TaskRestored"
}

// TaskTagsChanged indicates a task's tags were modified
type TaskTagsChanged struct {
	taskChange
	Added   []string
	Removed []string
}

// ChangeType returns "TaskTagsChanged"
func (c TaskTagsChanged) ChangeType() string {
	return "TaskTagsChanged"
}

// Compile-time interface implementation checks
var (
	_ Change = (*TaskCreated)(nil)
	_ Change = (*TaskDeleted)(nil)
	_ Change = (*TaskCompleted)(nil)
	_ Change = (*TaskUncompleted)(nil)
	_ Change = (*TaskCanceled)(nil)
	_ Change = (*TaskTitleChanged)(nil)
	_ Change = (*TaskNoteChanged)(nil)
	_ Change = (*TaskMovedToInbox)(nil)
	_ Change = (*TaskMovedToToday)(nil)
	_ Change = (*TaskMovedToAnytime)(nil)
	_ Change = (*TaskMovedToSomeday)(nil)
	_ Change = (*TaskMovedToUpcoming)(nil)
	_ Change = (*TaskDeadlineChanged)(nil)
	_ Change = (*TaskAssignedToProject)(nil)
	_ Change = (*TaskAssignedToArea)(nil)
	_ Change = (*TaskTrashed)(nil)
	_ Change = (*TaskRestored)(nil)
	_ Change = (*TaskTagsChanged)(nil)
)
