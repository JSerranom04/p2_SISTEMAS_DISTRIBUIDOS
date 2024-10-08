/*******************************************************************************
 * File:		cmd/escritor/escritor.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	Vra "practica2/ra/vector"

	//ra "practica2/ra/scalar"
	"practica2/utils"
	"strconv"
	"strings"
	"time"
)

/*
 *	@Pre:		The shared_RW_file exists or can be created, and the process has write permissions.
 *	@Post: 		The fragmento is written to the shared_RW_file, with each word separated by a sleep.
 *
 *	@Returns:	None.
 */
func EscribirFichero(shared_RW_file string, fragmento string, PID int) {
	splitFrag := strings.Split(fragmento, " ")
	f, err := os.OpenFile(shared_RW_file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("[PID %v] Fatal error while opening shared RW file to write: %v\n", PID, err)
	}
	defer f.Close()

	for _, word := range splitFrag {
		_, err := f.WriteString(word + " ")
		if err != nil {
			log.Fatalf("[PID %v] Fatal error while writing to shared RW file: %v\n", PID, err)
		}
		time.Sleep(utils.GetRandomSleepDuration(2, 20))
	}

	_, err = f.WriteString("\n")
	if err != nil {
		log.Fatalf("[PID %v] Fatal error while writing to shared RW file: %v\n", PID, err)
	}
}

/*
 *	@Pre:		true.
 *	@Post: 		Executes the writer process, reading from a content file and writing to a shared file.
 *
 *	@Returns:	None.
 */
func main() {
	execFormat := "go run escritor.go <line_number> <endpoints_file> <shared_RW_file_prefix> <shared_Rcontent_file>"
	PID, endpoints_file, shared_RW_file_pref, logs_file, endSigChan := utils.ParseAndCheckArgs(os.Args, execFormat, 5)
	totalPeers, _ := utils.CountNonEmptyLines(endpoints_file)
	contentRFile := os.Args[4]

	// Initialize the Ricart-Agrawala object
	// ra := ra.New(PID, endpoints_file, logs_file, utils.WRITE)
	ra := Vra.New(PID, endpoints_file, logs_file, utils.WRITE)

	lines_file_reader, _ := os.Open(contentRFile)
	defer lines_file_reader.Close()
	scanner := bufio.NewScanner(lines_file_reader)

	// Defer the stop
	defer ra.Stop()

	sleepTime := utils.GetRandomSleepDuration(2, 100)

	for scanner.Scan() {
		// Sleep for a random amount of time (milliseconds)
		// sleepTime := utils.GetRandomSleepDuration(2, 100)
		utils.LogWithColor(utils.Pink, fmt.Sprintf("[PID %v] Sleeping for %v...", PID, sleepTime))
		time.Sleep(sleepTime)

		// Read the next line from the input file
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Request access to the critical section
		ra.PreProtocol()

		// Write to the files inside the critical section
		for i := 1; i <= totalPeers; i++ {
			EscribirFichero(shared_RW_file_pref+strconv.Itoa(i)+".txt", line, PID)
		}

		// End access to the critical section
		ra.PostProtocol()

		utils.LogWithColor(utils.Pink, fmt.Sprintf("\n[PID %v] Content written to all shared RW files \"%s...\":", PID, shared_RW_file_pref))
		utils.LogWithColor(utils.Pink, line+"\n")
	}

	log.Printf("[PID %v] Finished write operations\n", PID)
	<-endSigChan
}
