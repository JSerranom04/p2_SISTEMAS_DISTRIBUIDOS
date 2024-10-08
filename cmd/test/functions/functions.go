/*******************************************************************************
 * File:		cmd/test/functions/functions.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package functions

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"practica2/cmd/test/utils"
)

/*
 *  @Pre:      The log file specified by 'logFile' exists and is readable.
 *  @Post:     Parses the log file and extracts a list of events.
 *
 *  @Returns:  A slice of utils.Event representing the events extracted from the log file.
 */
func ParseLogFile(logFile string) []utils.Event {
	file, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("Error opening the log file: %v", err)
	}
	defer file.Close()

	var events []utils.Event
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Read the log file line by line
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// Parse each line to extract events
		event, err := parseLogLine(line, lineNumber)
		if err == nil {
			events = append(events, event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading the log file: %v", err)
	}

	return events
}

/*
 *  @Pre:      'line' is a string containing a line from the log file.
 *  @Post:     Parses the line and extracts an event if it matches known patterns.
 *
 *  @Returns:  A utils.Event if the line corresponds to a known event, otherwise an error.
 */
func parseLogLine(line string, lineNumber int) (utils.Event, error) {
	// Regular expressions to detect events
	sendRequestRegex := regexp.MustCompile(`\[PID (\d+)\] Sending CS request to process (\d+), payload: {\[.*?\] \d+ (\d+)}`)
	receiveReplyRegex := regexp.MustCompile(`\[PID (\d+)\] Received ra_vector\.VReply`)
	sendImmediatePermissionRegex := regexp.MustCompile(`\[PID \d+\] Sending immediate CS permission to process (\d+)`)
	deferReplyRegex := regexp.MustCompile(`\[PID (\d+)\] Deferred process (\d+), RepDefd: \[.*\]`)

	// Match the line against each regex pattern
	if matches := sendRequestRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		opTypeInt, _ := strconv.Atoi(matches[3])
		opTypeStr := getOperationType(opTypeInt)
		return utils.Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "SEND_REQUEST",
			Operation:  opTypeStr,
		}, nil
	} else if matches := receiveReplyRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		return utils.Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "RECEIVE_REPLY",
		}, nil
	} else if matches := sendImmediatePermissionRegex.FindStringSubmatch(line); matches != nil {
		targetPID, _ := strconv.Atoi(matches[1])
		return utils.Event{
			LineNumber: lineNumber,
			PID:        targetPID,
			Action:     "RECEIVE_REPLY",
		}, nil
	} else if matches := deferReplyRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		targetPID, _ := strconv.Atoi(matches[2])
		return utils.Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "DEFERRED_REPLY",
			TargetPID:  targetPID,
		}, nil
	}

	return utils.Event{}, fmt.Errorf("Non-relevant line")
}

/*
 *  @Pre:      'opTypeInt' is an integer representing the operation type.
 *  @Post:     Converts the integer operation type to its string representation.
 *
 *  @Returns:  A string representing the operation type ("READ", "WRITE", or "UNKNOWN").
 */
func getOperationType(opTypeInt int) string {
	switch opTypeInt {
	case 0:
		return "READ"
	case 1:
		return "WRITE"
	default:
		return "UNKNOWN"
	}
}

/*
 *  @Pre:      An event of type "SEND_REQUEST" is provided, along with the current state of the process.
 *  @Post:     Updates the process state to reflect the sending of a request to enter the critical section.
 *
 *  @Returns:  None.
 */
func HandleSendRequest(event utils.Event, state *utils.ProcessState, processStates map[int]*utils.ProcessState, numProcesses int) {
	fmt.Printf("[DEBUG] Line %d: Process %d handling SEND_REQUEST\n", event.LineNumber, event.PID)
	if state.State == "Idle" {
		fmt.Printf("Process %d requests to enter the critical section (%s).\n", event.PID, event.Operation)
		state.State = "Requesting"
		state.Operation = event.Operation
		state.PendingReplies = numProcesses - 1 // Initialize to the total number of other processes
		fmt.Printf("[DEBUG] Process %d: PendingReplies initialized to %d\n", event.PID, state.PendingReplies)
	} else if state.State == "InCS" {
		// Process exits the critical section
		fmt.Printf("Process %d exits the critical section.\n", event.PID)
		state.State = "Idle"
		state.Operation = ""
		// Simulate sending deferred replies
		for _, targetPID := range state.DeferredReplies {
			targetState := processStates[targetPID]
			fmt.Printf("[DEBUG] Process %d sends deferred reply to process %d\n", event.PID, targetPID)
			HandleReceiveReply(utils.Event{
				LineNumber: event.LineNumber,
				PID:        targetPID,
				Action:     "RECEIVE_REPLY",
			}, targetState, processStates, numProcesses)
		}
		state.DeferredReplies = []int{}
		// Now the process requests to enter the critical section again
		fmt.Printf("Process %d requests to enter the critical section (%s).\n", event.PID, event.Operation)
		state.State = "Requesting"
		state.Operation = event.Operation
		state.PendingReplies = numProcesses - 1
		fmt.Printf("[DEBUG] Process %d: PendingReplies initialized to %d\n", event.PID, state.PendingReplies)
	} else {
		// If already requesting, it might be resending requests
		fmt.Printf("[DEBUG] Process %d is already in state %s\n", event.PID, state.State)
	}
}

/*
 *  @Pre:      An event of type "RECEIVE_REPLY" is provided, along with the current state of the process.
 *  @Post:     Updates the process state based on receiving a reply.
 *
 *  @Returns:  None.
 */
func HandleReceiveReply(event utils.Event, state *utils.ProcessState, processStates map[int]*utils.ProcessState, numProcesses int) {
	fmt.Printf("[DEBUG] Line %d: Process %d handling RECEIVE_REPLY\n", event.LineNumber, event.PID)
	if state.State == "Requesting" {
		if state.PendingReplies > 0 {
			state.PendingReplies -= 1
			fmt.Printf("[DEBUG] Process %d: PendingReplies decremented to %d\n", event.PID, state.PendingReplies)
			if state.PendingReplies == 0 {
				// All replies received, enter the critical section
				fmt.Printf("Process %d enters the critical section (%s).\n", event.PID, state.Operation)
				state.State = "InCS"
				// Verify mutual exclusion and readers-writers rules
				checkMutualExclusion(processStates, event.PID, state.Operation, event.LineNumber)
			}
		} else {
			logError(event.LineNumber, event.PID, "received more replies than expected")
			fmt.Println("Test failed: Received replies without having sent requests.")
			os.Exit(1)
		}
	} else {
		logError(event.LineNumber, event.PID, "received a reply without requesting")
		fmt.Println("Test failed: Received replies without having sent requests.")
		os.Exit(1)
	}
}

/*
 *  @Pre:      An event of type "DEFERRED_REPLY" is provided, along with the current state of the process.
 *  @Post:     Adds the target PID to the list of deferred replies.
 *
 *  @Returns:  None.
 */
func HandleDeferredReply(event utils.Event, state *utils.ProcessState) {
	fmt.Printf("[DEBUG] Line %d: Process %d defers reply to process %d\n", event.LineNumber, event.PID, event.TargetPID)
	state.DeferredReplies = append(state.DeferredReplies, event.TargetPID)
}

/*
 *  @Pre:      The process is attempting to enter the critical section.
 *  @Post:     Checks for violations of mutual exclusion or readers-writers rules.
 *
 *  @Returns:  None. Exits the program if a violation is detected.
 */
func checkMutualExclusion(processStates map[int]*utils.ProcessState, enteringPID int, enteringOp string, lineNumber int) {
	for pid, ps := range processStates {
		if ps.State == "InCS" && pid != enteringPID {
			fmt.Printf("[DEBUG] Checking conflict between process %d and process %d\n", enteringPID, pid)
			// Another process is in the critical section
			if enteringOp == "WRITE" || ps.Operation == "WRITE" {
				// Violation: A writer cannot be in the critical section with another process
				logViolation(lineNumber, enteringPID, pid, "Violation of mutual exclusion or readers-writers rules")
				fmt.Println("Test failed: Violation of mutual exclusion or readers-writers rules.")
				os.Exit(1)
			}
		}
	}
}

/*
 *  @Pre:      A violation has been detected.
 *  @Post:     Logs the violation and displays an error message.
 *
 *  @Returns:  None.
 */
func logViolation(lineNumber, pid1, pid2 int, message string) {
	log.Printf("Violation at line %d: %s. Processes involved: %d and %d.", lineNumber, message, pid1, pid2)
}

/*
 *  @Pre:      An error has been detected.
 *  @Post:     Logs the error.
 *
 *  @Returns:  None.
 */
func logError(lineNumber, pid int, message string) {
	log.Printf("Error at line %d: Process %d %s.", lineNumber, pid, message)
}
