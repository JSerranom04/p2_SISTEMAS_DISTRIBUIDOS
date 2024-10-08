/*******************************************************************************
 * File:        pkg/readerwriter/readerwriter.go
 * Authors:     Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:        06-10-2024
*******************************************************************************/

package readerwritter

import (
	"bufio"
	"fmt"
	"log"
	"os"
	Vra "practica2/ra/vector"
	"practica2/utils"
	"strconv"
	"strings"
	"time"
)

func LeerFichero(sharedRWFile string, PID int) string {
	file, err := os.Open(sharedRWFile)
	if err != nil {
		utils.LogWithColor(utils.Red, fmt.Sprintf("[PID %v] File \"%s\" not created yet, could not read anything\n", PID, sharedRWFile))
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

func EscribirFichero(sharedRWFile string, fragmento string, PID int) {
	splitFrag := strings.Split(fragmento, " ")
	f, err := os.OpenFile(sharedRWFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
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

func EjecutarLector(PID int, endpoints_file string, shared_RW_file_pref string, logs_file string) {
	sharedRWFile := shared_RW_file_pref + strconv.Itoa(PID) + ".txt"

	ra := Vra.New(PID, endpoints_file, logs_file, utils.READ)
	defer ra.Stop()

	for i := 0; i < 8; i++ {
		sleepTime := utils.GetRandomSleepDuration(10, 80)
		utils.LogWithColor(utils.BrCyan, fmt.Sprintf("[PID %v] Sleeping for %v...", PID, sleepTime))
		time.Sleep(sleepTime)

		ra.PreProtocol()
		data := LeerFichero(sharedRWFile, PID)
		ra.PostProtocol()

		utils.LogWithColor(utils.BrCyan, fmt.Sprintf("\n[PID %v] Content read from file \"%s\":", PID, sharedRWFile))
		utils.LogWithColor(utils.BrCyan, string(data)+"\n")
	}

	log.Printf("[PID %v] Finished read operations\n", PID)
}

func EjecutarEscritor(PID int, endpoints_file string, shared_RW_file_pref string, logs_file string, contentRFile string) {
	totalPeers, _ := utils.CountNonEmptyLines(endpoints_file)
	lines_file_reader, _ := os.Open(contentRFile)
	defer lines_file_reader.Close()
	scanner := bufio.NewScanner(lines_file_reader)

	ra := Vra.New(PID, endpoints_file, logs_file, utils.WRITE)
	defer ra.Stop()

	sleepTime := utils.GetRandomSleepDuration(2, 100)

	for scanner.Scan() {
		utils.LogWithColor(utils.Pink, fmt.Sprintf("[PID %v] Sleeping for %v...", PID, sleepTime))
		time.Sleep(sleepTime)

		line := scanner.Text()

		ra.PreProtocol()
		for i := 1; i <= totalPeers; i++ {
			EscribirFichero(shared_RW_file_pref+strconv.Itoa(i)+".txt", line, PID)
		}
		ra.PostProtocol()

		utils.LogWithColor(utils.Pink, fmt.Sprintf("\n[PID %v] Content written to files \"%s\" (1, 2, ...):", PID, shared_RW_file_pref))
		utils.LogWithColor(utils.Pink, line+"\n")
	}

	log.Printf("[PID %v] Finished write operations\n", PID)
}
