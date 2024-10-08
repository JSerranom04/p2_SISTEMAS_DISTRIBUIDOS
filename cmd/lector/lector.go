/*******************************************************************************
 * File:		cmd/lector/lector.go
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
	"time"
)

/*
 *	@Pre:		The shared_RW_file exists and the process has read permissions.
 *	@Post: 		The content of the shared_RW_file is read, with a sleep between each line.
 *
 *	@Returns:	A string containing the content of the file, or an empty string if the file couldn't be read.
 */
func LeerFichero(shared_RW_file string, PID int) string {
	file, err := os.Open(shared_RW_file)
	if err != nil {
		utils.LogWithColor(utils.Red, fmt.Sprintf("[PID %v] File \"%s\" not created yet, could not read anything\n", PID, shared_RW_file))
		return ""
	}
	defer file.Close()

	content := ""
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		content += scanner.Text() + "\n"
		time.Sleep(utils.GetRandomSleepDuration(1, 3))
	}

	return string(content)
}

/*
 *	@Pre:		true.
 *	@Post: 		Executes the reader process, reading from a shared file.
 *
 *	@Returns:	None.
 */
func main() {
	execFormat := "go run lector.go <line_number> <endpoints_file> <shared_RW_file_prefix>"
	PID, endpoints_file, shared_RW_file_pref, logs_file, endSigChan := utils.ParseAndCheckArgs(os.Args, execFormat, 4)
	shared_RW_file := shared_RW_file_pref + strconv.Itoa(PID) + ".txt"

	// Initialize the Ricart-Agrawala object
	// ra := ra.New(PID, endpoints_file, logs_file, utils.READ)
	ra := Vra.New(PID, endpoints_file, logs_file, utils.READ)

	// Defer the stop
	defer ra.Stop()

	// sleepTime := utils.GetRandomSleepDuration(10, 80)

	for i := 0; i < 8; i++ {
		// Sleep for a random amount of time (milliseconds)
		sleepTime := utils.GetRandomSleepDuration(10, 80)
		utils.LogWithColor(utils.BrCyan, fmt.Sprintf("[PID %v] Sleeping for %v...", PID, sleepTime))
		time.Sleep(sleepTime)

		// Request access to the critical section
		ra.PreProtocol()

		// Execute the read operation
		data := LeerFichero(shared_RW_file, PID)

		// End access to the critical section
		ra.PostProtocol()

		utils.LogWithColor(utils.BrCyan, fmt.Sprintf("\n[PID %v] Content read from file \"%s\":", PID, shared_RW_file))
		utils.LogWithColor(utils.BrCyan, string(data)+"\n")
	}

	log.Printf("[PID %v] Finished read operations\n", PID)
	<-endSigChan
}
