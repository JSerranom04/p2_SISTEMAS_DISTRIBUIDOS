package utils

// Estructuras de datos
type Event struct {
	LineNumber int
	PID        int
	Operation  string // "READ" o "WRITE"
	Action     string // "SEND_REQUEST", "RECEIVE_REPLY"
}

type ProcessState struct {
	State          string // "Idle", "Requesting", "InCS"
	Operation      string // "READ" o "WRITE"
	PendingReplies int
}
