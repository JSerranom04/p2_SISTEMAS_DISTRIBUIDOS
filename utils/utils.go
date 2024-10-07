/*******************************************************************************
 * File:		utils/utils.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package utils

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type OpType int

const (
	READ OpType = iota
	WRITE
)

/*
 *	@Pre:		localOpType and remoteOpType are valid OpType values.
 *	@Post: 		Determines if the operations are mutually exclusive.
 *
 *	@Returns:	true if the operations are mutually exclusive, false otherwise.
 */
func ExcludeOps(localOpType, remoteOpType OpType) bool {
	return localOpType == WRITE || remoteOpType == WRITE
}

/*
 *	@Pre:		args is a slice of command-line arguments, execFormat is a string describing the expected format,
 *				argsNum is the expected number of arguments.
 *	@Post: 		Parses and validates command-line arguments.
 *
 *	@Returns:	PID (int), endpointsFile (string), sharedRWFile (string), logsFile (string), and a channel for OS signals.
 */
func ParseAndCheckArgs(args []string, execFormat string, argsNum int) (int, string, string, string, chan os.Signal) {
	var PID int
	var err error
	if len(args) < argsNum {
		log.Fatalf("Error: arguments missing, execution format: %s\n", execFormat)
	} else if PID, err = strconv.Atoi(args[1]); err != nil {
		log.Fatal("Error: the provided process line number cannot be parsed to an int\n")
	}

	endSigChan := make(chan os.Signal, 1)
	signal.Notify(endSigChan, syscall.SIGINT, syscall.SIGTERM)
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	endpointsFile := args[2]
	sharedRWFile := args[3]
	logsFile := fmt.Sprintf("logs/logs%v", PID)

	return PID, endpointsFile, sharedRWFile, logsFile, endSigChan
}

func GetRandomSleepDuration(minDelayMs int, maxDelayMs int) time.Duration {
	if minDelayMs > maxDelayMs {
		panic("minDelayMs should not be greater than maxDelayMs")
	}
	delayMs := rand.Intn(maxDelayMs-minDelayMs+1) + minDelayMs
	return time.Duration(delayMs) * time.Millisecond
}

func CountNonEmptyLines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

// CustomEncoder encodes a given data structure into a byte slice
func CustomEncoder(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	return buf.Bytes(), err
}

// CustomDecoder decodes a byte slice back into a data structure
func CustomDecoder(src []byte, dest interface{}) error {
	// log.Println("DECOOOODINNNNG")
	reader := bytes.NewReader(src)
	decoder := gob.NewDecoder(reader)
	return decoder.Decode(dest)
}

const (
	Reset    = "\033[0m"
	Red      = "\033[31m"
	Green    = "\033[32m"
	Yellow   = "\033[33m"
	Blue     = "\033[34m"
	Orange   = "\033[38;2;255;165;0m"
	Pink     = "\033[38;5;13m"
	BrCyan   = "\033[96m"
	BgLBlue  = "\033[48;5;81m"
	BgOrange = "\033[48;5;208m"
)

func LogWithColor(color, message string) {
	fmt.Println(color + message + Reset)
}
