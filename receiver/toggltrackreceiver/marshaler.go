package toggltrackreceiver

import (
	"strconv"
	"time"

	toggl "github.com/jason0x43/go-toggl"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	scopeName    = "github.com/zmoog/collector/receiver/toggltrackreceiver"
	scopeVersion = "v0.1.0"
)

type timeEntryMarshaler struct{}

func (m *timeEntryMarshaler) UnmarshalLogs(timeEntries []toggl.TimeEntry) (plog.Logs, error) {
	l := plog.NewLogs()

	resourceLogs := l.ResourceLogs().AppendEmpty()

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName(scopeName)
	scopeLogs.Scope().SetVersion(scopeVersion)
	logRecords := scopeLogs.LogRecords()

	for _, e := range timeEntries {
		if e.IsRunning() {
			// We don't care about running entries
			continue
		}

		lr := logRecords.AppendEmpty()
		lr.SetTimestamp(pcommon.NewTimestampFromTime(*e.Stop))
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		// lr.Body().FromRaw(map[string]any{
		// 	"_id": e.ID,
		// })

		a := lr.Attributes()
		a.PutStr("id", strconv.Itoa(e.ID))
		a.PutStr("workspace_id", strconv.Itoa(e.Wid))
		a.PutStr("description", e.Description)
		a.PutStr("start", e.Start.Format(time.RFC3339))
		a.PutStr("end", e.Stop.Format(time.RFC3339)) // `end` is ECS compliant
		a.PutInt("duration", e.Duration)
		if e.Pid != nil {
			a.PutStr("project_id", strconv.Itoa(*e.Pid))
		}
		if e.Tid != nil {
			a.PutStr("task_id", strconv.Itoa(*e.Tid))
		}

		tags := a.PutEmptySlice("tags")
		for _, tag := range e.Tags {
			tags.AppendEmpty().SetStr(tag)
		}
	}

	return l, nil
}
