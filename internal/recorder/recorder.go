package recorder

import "context"

// Status represents the current state of a recorder.
type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusError   Status = "error"
)

// Recorder is the interface that all capture sources must implement.
type Recorder interface {
	// Start begins recording. Returns when recording has started or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully stops recording.
	Stop(ctx context.Context) error

	// Status returns the current recorder state.
	Status() Status
}
