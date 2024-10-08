package main

import (
	"fmt"
	"os"
	"strconv"

	"practica2/cmd/test/functions"
	"practica2/cmd/test/utils"
)

/*
 *  @Pre:      The program is called with a single argument specifying the number of processes.
 *  @Post:     Processes the log file, updates process states, and prints the test result.
 *
 *  @Returns:  None.
 */
func main() {
	numProcesses := parseArguments()

	// Initialize the states of all processes.
	processStates := initializeProcessStates(numProcesses)

	// Parse the log file and obtain the list of events.
	events := functions.ParseLogFile("salida.txt")

	// Process each event and update process states accordingly.
	for _, event := range events {
		processEvent(event, processStates, numProcesses)
	}

	// Finalize the states of processes after all events have been processed.
	finalizeProcessStates(processStates)

	fmt.Println("Test successful: All checks passed correctly.")
}

/*
 *  @Pre:      The program is called with a single command-line argument.
 *  @Post:     Parses the command-line argument and returns the number of processes.
 *
 *  @Returns:  The number of processes as an integer.
 */
func parseArguments() int {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run test.go <number_of_processes>")
		os.Exit(1)
	}

	numProcesses, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("The argument must be an integer.")
		os.Exit(1)
	}

	return numProcesses
}

/*
 *  @Pre:      The number of processes is provided.
 *  @Post:     Initializes and returns a map of process states.
 *
 *  @Returns:  A map[int]*utils.ProcessState with initial states for each process.
 */
func initializeProcessStates(numProcesses int) map[int]*utils.ProcessState {
	// Create a map to hold the state of each process.
	processStates := make(map[int]*utils.ProcessState)
	for pid := 1; pid <= numProcesses; pid++ {
		// Initialize each process state to default values.
		processStates[pid] = &utils.ProcessState{
			State:           "Idle",
			Operation:       "",
			PendingReplies:  0,
			DeferredReplies: []int{},
		}
	}
	return processStates
}

/*
 *  @Pre:      An event, the process states map, and the number of processes are provided.
 *  @Post:     Processes the event and updates the corresponding process state.
 *
 *  @Returns:  None.
 */
func processEvent(event utils.Event, processStates map[int]*utils.ProcessState, numProcesses int) {
	state := processStates[event.PID]

	// Handle the event based on its action type.
	switch event.Action {
	case "SEND_REQUEST":
		functions.HandleSendRequest(event, state, processStates, numProcesses)
	case "RECEIVE_REPLY":
		functions.HandleReceiveReply(event, state, processStates, numProcesses)
	case "DEFERRED_REPLY":
		functions.HandleDeferredReply(event, state)
	default:
		fmt.Printf("Unknown event at line %d: %+v\n", event.LineNumber, event)
	}
}

/*
 *  @Pre:      The process states map is provided.
 *  @Post:     Finalizes the states of processes after analysis.
 *
 *  @Returns:  None.
 */
func finalizeProcessStates(processStates map[int]*utils.ProcessState) {
	// Iterate over all processes and reset their states if necessary.
	for pid, state := range processStates {
		if state.State == "InCS" {
			fmt.Printf("Process %d exits the critical section.\n", pid)
			state.State = "Idle"
			state.Operation = ""
		}
	}
}
