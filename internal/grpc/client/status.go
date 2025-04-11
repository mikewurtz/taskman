package client

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"
)

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
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTART TIME\tPID\tSTATUS\tEXIT CODE\tSIGNAL\tSTOP SOURCE\tEND TIME")
	fmt.Fprintln(w, "-------\t----------\t---\t------\t---------\t------\t-----------\t--------")

	startTime := t.StartTime.Format("2006-01-02 15:04:05")

	endTime := formatTime(t.EndTime)
	exitStr := formatExitCode(t.ExitCode)
	signal := formatString(t.TerminationSignal)
	source := formatString(t.TerminationSource)

	fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
		t.TaskID,
		startTime,
		t.ProcessID,
		t.Status,
		exitStr,
		signal,
		source,
		endTime,
	)
	w.Flush()
	return buf.String()
}
