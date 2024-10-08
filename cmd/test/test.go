/*******************************************************************************
 * File:        cmd/test/test.go
 * Authors:     Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:        06-10-2024
*******************************************************************************/
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run verifier.go <number_of_processes>")
		os.Exit(1)
	}

	numProcesses, err := strconv.Atoi(os.Args[1])
	if err != nil || numProcesses <= 0 {
		fmt.Println("Invalid number of processes.")
		os.Exit(1)
	}

	logFile := "salida.txt"

	// Open the log file for reading
	file, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer file.Close()

	// Variables to track the critical section entries
	inCriticalSection := make(map[int]bool) // Tracks if a process is in CS
	csEntries := make([]int, 0)             // Keeps track of the order of CS entries

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNumber++

		if strings.Contains(line, "Entering critical section") {
			pid := extractPID(line)
			vectorClock := extractVectorClock(line)
			if pid >= 0 {
				// Check if another process is already in CS
				for otherPID, inCS := range inCriticalSection {
					if inCS && otherPID != pid {
						log.Printf("Error at line %d: Processes %d and %d are in the critical section simultaneously.", lineNumber, pid, otherPID)
						fmt.Println("Execution failed: Mutual exclusion violated.")
						os.Exit(1)
					}
				}
				inCriticalSection[pid] = true
				csEntries = append(csEntries, pid)
				log.Printf("Process %d entered CS at vector clock %v", pid, vectorClock)
			}
		} else if strings.Contains(line, "Exiting critical section") {
			pid := extractPID(line)
			vectorClock := extractVectorClock(line)
			if pid >= 0 {
				if !inCriticalSection[pid] {
					log.Printf("Error at line %d: Process %d is exiting CS but was not recorded as being in CS.", lineNumber, pid)
					fmt.Println("Execution failed: Process state inconsistency.")
					os.Exit(1)
				}
				inCriticalSection[pid] = false
				log.Printf("Process %d exited CS at vector clock %v", pid, vectorClock)
			}
		}
		// Ignore other messages
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading log file: %v", err)
	}

	// Final check to ensure no process is left in CS
	for pid, inCS := range inCriticalSection {
		if inCS {
			log.Printf("Error: Process %d is still in the critical section at the end of execution.", pid)
			fmt.Println("Execution failed: Some processes did not exit the critical section.")
			os.Exit(1)
		}
	}

	// If we reach here, all checks passed
	fmt.Println("Execution successful: All checks passed.")
}

// extractPID extracts the PID from a log line
func extractPID(line string) int {
	var pid int
	_, err := fmt.Sscanf(line, "[PID %d]", &pid)
	if err != nil {
		return -1 // Error in extraction
	}
	return pid
}

// extractVectorClock extracts the vector clock from a log line
func extractVectorClock(line string) []int {
	start := strings.Index(line, "{[")
	end := strings.Index(line, "]}")
	if start != -1 && end != -1 {
		clockStr := line[start+2 : end]
		return parseClockString(clockStr)
	}
	return nil
}

// parseClockString converts a clock string into a slice of integers
func parseClockString(clockStr string) []int {
	parts := strings.Fields(clockStr)
	clock := make([]int, len(parts))
	for i, part := range parts {

		_, err := fmt.Sscanf(part, "%d", &clock[i])
		if err != nil {
			log.Fatalf("Error parsing vector clock: %v", err)
		}
	}
	return clock
}
