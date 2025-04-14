package client

import (
	"bytes"
	"fmt"
	"time"

	"github.com/olekukonko/tablewriter"
)

// TaskStatus represents the status of the task
// used to display the task information to the caller
type TaskStatus struct {
	TaskID            string
	Status            string
	StartTime         time.Time
	EndTime           time.Time
	ExitCode          *int32
	ProcessID         int32
	TerminationSignal string
	TerminationSource string
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatExitCode(code *int32) string {
	if code == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *code)
}

func formatString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (t *TaskStatus) String() string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{
		"TASK ID", "START TIME", "PID", "STATUS", "EXIT CODE", "SIGNAL", "STOP SOURCE", "END TIME",
	})
	table.SetAutoWrapText(true)
	table.SetBorder(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	row := []string{
		t.TaskID,
		t.StartTime.Format("2006-01-02 15:04:05"),
		fmt.Sprintf("%d", t.ProcessID),
		t.Status,
		formatExitCode(t.ExitCode),
		formatString(t.TerminationSignal),
		formatString(t.TerminationSource),
		formatTime(t.EndTime),
	}

	table.Append(row)
	table.Render()
	return buf.String()
}
