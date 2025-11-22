package toggltrackreceiver

import (
	"testing"
	"time"

	"github.com/jason0x43/go-toggl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestTimeEntryMarshaler_UnmarshalLogs(t *testing.T) {
	tests := []struct {
		name              string
		account           toggl.Account
		initialLastTime   time.Time
		expectedLogCount  int
		validateLogRecord func(t *testing.T, lr plog.LogRecord)
		validateState     func(t *testing.T, m *timeEntryMarshaler)
	}{
		{
			name: "marshal complete time entry with all fields",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					{
						ID:          12345,
						Wid:         100,
						Pid:         intPtr(200),
						Tid:         intPtr(300),
						Description: "Testing feature X",
						Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
						Duration:    5400,
						Billable:    true,
						Tags:        []string{"development", "testing"},
					},
				},
				Workspaces: []toggl.Workspace{
					{ID: 100, Name: "My Workspace"},
				},
				Projects: []toggl.Project{
					{ID: 200, Name: "Project Alpha"},
				},
				Tasks: []toggl.Task{
					{ID: 300, Name: "Feature Implementation"},
				},
			},
			expectedLogCount: 1,
			validateLogRecord: func(t *testing.T, lr plog.LogRecord) {
				attrs := lr.Attributes()

				id, ok := attrs.Get("id")
				require.True(t, ok)
				assert.Equal(t, "12345", id.Str())

				workspaceID, ok := attrs.Get("workspace.id")
				require.True(t, ok)
				assert.Equal(t, "100", workspaceID.Str())

				workspaceName, ok := attrs.Get("workspace.name")
				require.True(t, ok)
				assert.Equal(t, "My Workspace", workspaceName.Str())

				projectID, ok := attrs.Get("project.id")
				require.True(t, ok)
				assert.Equal(t, "200", projectID.Str())

				projectName, ok := attrs.Get("project.name")
				require.True(t, ok)
				assert.Equal(t, "Project Alpha", projectName.Str())

				taskID, ok := attrs.Get("task.id")
				require.True(t, ok)
				assert.Equal(t, "300", taskID.Str())

				taskName, ok := attrs.Get("task.name")
				require.True(t, ok)
				assert.Equal(t, "Feature Implementation", taskName.Str())

				desc, ok := attrs.Get("description")
				require.True(t, ok)
				assert.Equal(t, "Testing feature X", desc.Str())

				start, ok := attrs.Get("start")
				require.True(t, ok)
				assert.Equal(t, "2024-01-15T09:00:00Z", start.Str())

				end, ok := attrs.Get("end")
				require.True(t, ok)
				assert.Equal(t, "2024-01-15T10:30:00Z", end.Str())

				duration, ok := attrs.Get("duration")
				require.True(t, ok)
				assert.Equal(t, int64(5400), duration.Int())

				billable, ok := attrs.Get("billable")
				require.True(t, ok)
				assert.Equal(t, "true", billable.Str())

				tags, ok := attrs.Get("tags")
				require.True(t, ok)
				assert.Equal(t, 2, tags.Slice().Len())
				assert.Equal(t, "development", tags.Slice().At(0).Str())
				assert.Equal(t, "testing", tags.Slice().At(1).Str())
			},
		},
		{
			name: "skip running entries",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					{
						ID:          1,
						Wid:         100,
						Description: "Running entry",
						Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Stop:        nil, // Running entry has no stop time
						Duration:    -1,
					},
					{
						ID:          2,
						Wid:         100,
						Description: "Completed entry",
						Start:       timePtr(time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
				},
				Workspaces: []toggl.Workspace{
					{ID: 100, Name: "My Workspace"},
				},
			},
			expectedLogCount: 1,
			validateLogRecord: func(t *testing.T, lr plog.LogRecord) {
				attrs := lr.Attributes()
				id, ok := attrs.Get("id")
				require.True(t, ok)
				assert.Equal(t, "2", id.Str(), "Should only process completed entry")
			},
		},
		{
			name: "skip already processed entries",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					{
						ID:          3,
						Wid:         100,
						Description: "New entry",
						Start:       timePtr(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
					{
						ID:          2,
						Wid:         100,
						Description: "Old entry",
						Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
				},
				Workspaces: []toggl.Workspace{
					{ID: 100, Name: "My Workspace"},
				},
			},
			initialLastTime:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedLogCount: 1,
			validateLogRecord: func(t *testing.T, lr plog.LogRecord) {
				attrs := lr.Attributes()
				id, ok := attrs.Get("id")
				require.True(t, ok)
				assert.Equal(t, "3", id.Str(), "Should only process new entry")
			},
			validateState: func(t *testing.T, m *timeEntryMarshaler) {
				expectedLastTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
				assert.Equal(t, expectedLastTime, m.lastTimeEntryTime, "Should update lastTimeEntryTime to latest entry")
			},
		},
		{
			name: "handle entries without project and task",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					{
						ID:          4,
						Wid:         100,
						Pid:         nil, // No project
						Tid:         nil, // No task
						Description: "Personal task",
						Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
						Duration:    3600,
						Billable:    false,
						Tags:        []string{},
					},
				},
				Workspaces: []toggl.Workspace{
					{ID: 100, Name: "My Workspace"},
				},
			},
			expectedLogCount: 1,
			validateLogRecord: func(t *testing.T, lr plog.LogRecord) {
				attrs := lr.Attributes()

				// Should have workspace
				workspaceID, ok := attrs.Get("workspace.id")
				require.True(t, ok)
				assert.Equal(t, "100", workspaceID.Str())

				// Should not have project
				_, ok = attrs.Get("project.id")
				assert.False(t, ok, "Should not have project.id when Pid is nil")

				_, ok = attrs.Get("project.name")
				assert.False(t, ok, "Should not have project.name when Pid is nil")

				// Should not have task
				_, ok = attrs.Get("task.id")
				assert.False(t, ok, "Should not have task.id when Tid is nil")

				_, ok = attrs.Get("task.name")
				assert.False(t, ok, "Should not have task.name when Tid is nil")

				// Should have empty tags slice
				tags, ok := attrs.Get("tags")
				require.True(t, ok)
				assert.Equal(t, 0, tags.Slice().Len())
			},
		},
		{
			name: "handle unknown workspace, project, and task IDs",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					{
						ID:          5,
						Wid:         999, // Unknown workspace
						Pid:         intPtr(888),
						Tid:         intPtr(777),
						Description: "Entry with unknown references",
						Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
				},
				Workspaces: []toggl.Workspace{},
				Projects:   []toggl.Project{},
				Tasks:      []toggl.Task{},
			},
			expectedLogCount: 1,
			validateLogRecord: func(t *testing.T, lr plog.LogRecord) {
				attrs := lr.Attributes()

				workspaceName, ok := attrs.Get("workspace.name")
				require.True(t, ok)
				assert.Equal(t, "Unknown (999)", workspaceName.Str())

				projectName, ok := attrs.Get("project.name")
				require.True(t, ok)
				assert.Equal(t, "Unknown (888)", projectName.Str())

				taskName, ok := attrs.Get("task.name")
				require.True(t, ok)
				assert.Equal(t, "Unknown (777)", taskName.Str())
			},
		},
		{
			name: "process multiple entries in correct order",
			account: toggl.Account{
				TimeEntries: []toggl.TimeEntry{
					// Entries are sorted latest first (as per API behavior)
					{
						ID:          3,
						Wid:         100,
						Description: "Latest entry",
						Start:       timePtr(time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
					{
						ID:          2,
						Wid:         100,
						Description: "Middle entry",
						Start:       timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
					{
						ID:          1,
						Wid:         100,
						Description: "Earliest entry",
						Start:       timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
						Stop:        timePtr(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)),
						Duration:    3600,
					},
				},
				Workspaces: []toggl.Workspace{
					{ID: 100, Name: "My Workspace"},
				},
			},
			expectedLogCount: 3,
			validateState: func(t *testing.T, m *timeEntryMarshaler) {
				expectedLastTime := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)
				assert.Equal(t, expectedLastTime, m.lastTimeEntryTime, "Should track the latest stop time")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTimeEntryMarshaler(Mappings{})
			if !tt.initialLastTime.IsZero() {
				m.lastTimeEntryTime = tt.initialLastTime
			}

			logs, err := m.UnmarshalLogs(tt.account)
			require.NoError(t, err)

			// Validate structure
			require.Equal(t, 1, logs.ResourceLogs().Len(), "Should have one ResourceLogs")
			resourceLogs := logs.ResourceLogs().At(0)
			require.Equal(t, 1, resourceLogs.ScopeLogs().Len(), "Should have one ScopeLogs")
			scopeLogs := resourceLogs.ScopeLogs().At(0)

			// Validate scope metadata
			assert.Equal(t, scopeName, scopeLogs.Scope().Name())
			assert.Equal(t, scopeVersion, scopeLogs.Scope().Version())

			// Validate log count
			logRecords := scopeLogs.LogRecords()
			assert.Equal(t, tt.expectedLogCount, logRecords.Len())

			// Validate individual log records
			if tt.validateLogRecord != nil && logRecords.Len() > 0 {
				tt.validateLogRecord(t, logRecords.At(0))
			}

			// Validate marshaler state
			if tt.validateState != nil {
				tt.validateState(t, m)
			}
		})
	}
}

func TestTimeEntryMarshaler_EmptyAccount(t *testing.T) {
	m := newTimeEntryMarshaler(Mappings{})
	account := toggl.Account{
		TimeEntries: []toggl.TimeEntry{},
	}

	logs, err := m.UnmarshalLogs(account)
	require.NoError(t, err)

	resourceLogs := logs.ResourceLogs()
	require.Equal(t, 1, resourceLogs.Len())
	scopeLogs := resourceLogs.At(0).ScopeLogs()
	require.Equal(t, 1, scopeLogs.Len())
	logRecords := scopeLogs.At(0).LogRecords()
	assert.Equal(t, 0, logRecords.Len(), "Should produce no log records for empty account")
}

func TestTimeEntryMarshaler_StatePersistence(t *testing.T) {
	m := newTimeEntryMarshaler(Mappings{})

	// First batch
	account1 := toggl.Account{
		TimeEntries: []toggl.TimeEntry{
			{
				ID:          1,
				Wid:         100,
				Description: "First batch",
				Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
				Stop:        timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
				Duration:    3600,
			},
		},
		Workspaces: []toggl.Workspace{{ID: 100, Name: "Workspace"}},
	}

	logs1, err := m.UnmarshalLogs(account1)
	require.NoError(t, err)
	assert.Equal(t, 1, logs1.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().Len())

	// Second batch with overlapping entry
	account2 := toggl.Account{
		TimeEntries: []toggl.TimeEntry{
			{
				ID:          2,
				Wid:         100,
				Description: "New entry",
				Start:       timePtr(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)),
				Stop:        timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
				Duration:    3600,
			},
			{
				ID:          1,
				Wid:         100,
				Description: "First batch",
				Start:       timePtr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
				Stop:        timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
				Duration:    3600,
			},
		},
		Workspaces: []toggl.Workspace{{ID: 100, Name: "Workspace"}},
	}

	logs2, err := m.UnmarshalLogs(account2)
	require.NoError(t, err)
	logRecords := logs2.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	assert.Equal(t, 1, logRecords.Len(), "Should only process new entry")

	attrs := logRecords.At(0).Attributes()
	id, _ := attrs.Get("id")
	assert.Equal(t, "2", id.Str(), "Should process the new entry")
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
