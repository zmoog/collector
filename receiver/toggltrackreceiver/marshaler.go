package toggltrackreceiver

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jason0x43/go-toggl"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	scopeName    = "github.com/zmoog/collector/receiver/toggltrackreceiver"
	scopeVersion = "v0.2.0"
)

type timeEntryMarshaler struct {
	mappings          Mappings `mapstructure:"mappings"`
	lastTimeEntryTime time.Time
}

func newTimeEntryMarshaler(mappings Mappings) *timeEntryMarshaler {
	return &timeEntryMarshaler{mappings: mappings}
}

func (m *timeEntryMarshaler) UnmarshalLogs(account toggl.Account) (plog.Logs, error) {
	l := plog.NewLogs()

	resourceLogs := l.ResourceLogs().AppendEmpty()

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName(scopeName)
	scopeLogs.Scope().SetVersion(scopeVersion)
	logRecords := scopeLogs.LogRecords()

	// Unify the observed timestamp for all log records.
	observedTimestamp := pcommon.NewTimestampFromTime(time.Now())

	// account.TimeEntries is sorted with the latest entries first, so we need
	// to walk backwards to make skipping already processed entries easier.
	for i := len(account.TimeEntries) - 1; i >= 0; i-- {
		e := account.TimeEntries[i]

		if e.IsRunning() {
			// We don't care about running entries
			continue
		}

		if !e.Stop.After(m.lastTimeEntryTime) {
			// We've already processed this entry
			continue
		}
		m.lastTimeEntryTime = *e.Stop

		lr := logRecords.AppendEmpty()
		lr.SetTimestamp(pcommon.NewTimestampFromTime(*e.Stop))
		lr.SetObservedTimestamp(observedTimestamp)

		a := lr.Attributes()
		a.PutStr("id", strconv.Itoa(e.ID))
		a.PutStr("workspace.id", strconv.Itoa(e.Wid))
		a.PutStr("description", e.Description)
		a.PutStr("start", e.Start.Format(time.RFC3339))
		a.PutStr("end", e.Stop.Format(time.RFC3339)) // `end` is ECS compliant
		a.PutInt("duration", e.Duration)
		a.PutStr("billable", strconv.FormatBool(e.Billable))
		if e.Pid != nil {
			a.PutStr("project.id", strconv.Itoa(*e.Pid))
		}
		if e.Tid != nil {
			a.PutStr("task.id", strconv.Itoa(*e.Tid))
		}

		a.PutStr("workspace.name", lookupName(account.Workspaces, e.Wid))
		if e.Pid != nil {
			a.PutStr("project.name", lookupName(account.Projects, *e.Pid))
		}
		if e.Tid != nil {
			a.PutStr("task.name", lookupName(account.Tasks, *e.Tid))
		}

		tags := a.PutEmptySlice("tags")
		for _, tag := range e.Tags {
			tags.AppendEmpty().SetStr(tag)
		}
	}

	return l, nil
}

// entityWithIDAndName is a constraint for types that have ID and Name fields.
type entityWithIDAndName interface {
	toggl.Workspace | toggl.Project | toggl.Task
}

// lookupName is a generic function that looks up an entity by ID and returns its name.
func lookupName[T entityWithIDAndName](entities []T, id int) string {
	for _, entity := range entities {
		// Use type assertion to access ID and Name fields
		switch e := any(entity).(type) {
		case toggl.Workspace:
			if e.ID == id {
				return e.Name
			}
		case toggl.Project:
			if e.ID == id {
				return e.Name
			}
		case toggl.Task:
			if e.ID == id {
				return e.Name
			}
		}
	}
	return fmt.Sprintf("Unknown (%d)", id)
}
