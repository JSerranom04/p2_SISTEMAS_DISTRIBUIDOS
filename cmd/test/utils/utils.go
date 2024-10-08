/*******************************************************************************
 * File:		cmd/test/utils/utils.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package utils

// Estructuras de datos
type Event struct {
	LineNumber int
	PID        int
	Operation  string // "READ" o "WRITE"
	Action     string // "SEND_REQUEST", "RECEIVE_REPLY", "DEFERRED_REPLY"
	TargetPID  int    // PID del proceso destino u origen según corresponda
}

type ProcessState struct {
	State           string // "Idle", "Requesting", "InCS"
	Operation       string // "READ" o "WRITE"
	PendingReplies  int
	DeferredReplies []int // Lista de PIDs a los que se les ha diferido la respuesta
}
