package functions

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	utils "practica2/cmd/test/utils"
)

// Función para analizar el archivo de logs y extraer los eventos
func ParseLogFile(logFile string) []utils.Event {
	file, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("Error al abrir el archivo de logs: %v", err)
	}
	defer file.Close()

	var events []utils.Event
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		event, err := parseLogLine(line, lineNumber)
		if err == nil {
			events = append(events, event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error al leer el archivo de logs: %v", err)
	}

	return events
}

// Función para analizar una línea del log y extraer un evento
func parseLogLine(line string, lineNumber int) (utils.Event, error) {
	sendRequestRegex := regexp.MustCompile(`\[PID (\d+)\] Sending CS request to process \d+, payload: {\[.*?\] \d+ (\d+)}`)
	receiveReplyRegex := regexp.MustCompile(`\[PID (\d+)\] Received ra_vector\.VReply`)
	sendImmediatePermissionRegex := regexp.MustCompile(`\[PID \d+\] Sending immediate CS permission to process (\d+)`)

	if matches := sendRequestRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		opTypeInt, _ := strconv.Atoi(matches[2])
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
	}

	return utils.Event{}, fmt.Errorf("Línea no relevante")
}

// Función para obtener el tipo de operación como cadena
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

// Función para manejar el evento de envío de solicitud
func HandleSendRequest(event utils.Event, state *utils.ProcessState, numProcesses int) {
	if state.State == "Idle" || state.State == "InCS" {
		if state.State == "InCS" {
			// Proceso sale de la sección crítica
			fmt.Printf("Proceso %d sale de la sección crítica.\n", event.PID)
		}
		fmt.Printf("Proceso %d solicita entrar en la sección crítica (%s).\n", event.PID, event.Operation)
		state.State = "Requesting"
		state.Operation = event.Operation
		state.PendingReplies = numProcesses - 1 // Inicializar al número total de procesos menos uno
	}
}

// Función para manejar el evento de recepción de respuesta
func HandleReceiveReply(event utils.Event, state *utils.ProcessState, processStates map[int]*utils.ProcessState, numProcesses int) {
	if state.State == "Requesting" {
		if state.PendingReplies > 0 {
			state.PendingReplies -= 1
			if state.PendingReplies == 0 {
				// Todas las respuestas recibidas, entra en la sección crítica
				fmt.Printf("Proceso %d entra en la sección crítica (%s).\n", event.PID, state.Operation)
				state.State = "InCS"
				// Verificar exclusión mutua y reglas de lectores-escritores
				checkMutualExclusion(processStates, event.PID, state.Operation, event.LineNumber)
			}
		} else {
			logError(event.LineNumber, event.PID, "recibió más respuestas de las esperadas")
			fmt.Println("Test fallido: Respuestas recibidas sin haber enviado solicitudes.")
			os.Exit(1)
		}
	} else {
		logError(event.LineNumber, event.PID, "recibió una respuesta sin estar solicitando")
		fmt.Println("Test fallido: Respuestas recibidas sin haber enviado solicitudes.")
		os.Exit(1)
	}
}

// Función para verificar la exclusión mutua y las reglas de lectores-escritores
func checkMutualExclusion(processStates map[int]*utils.ProcessState, enteringPID int, enteringOp string, lineNumber int) {
	for pid, ps := range processStates {
		if ps.State == "InCS" && pid != enteringPID {
			// Otro proceso está en la sección crítica
			if enteringOp == "WRITE" || ps.Operation == "WRITE" {
				// Violación: Un escritor no puede estar en la sección crítica con otro proceso
				logViolation(lineNumber, enteringPID, pid, "Violación de exclusión mutua o lectores-escritores")
				fmt.Println("Test fallido: Violación de exclusión mutua o lectores-escritores.")
				os.Exit(1)
			}
		}
	}
}

// Función para registrar una violación y mostrar un mensaje de error
func logViolation(lineNumber, pid1, pid2 int, message string) {
	log.Printf("Violación en la línea %d: %s. Procesos involucrados: %d y %d.", lineNumber, message, pid1, pid2)
}

// Función para registrar un error
func logError(lineNumber, pid int, message string) {
	log.Printf("Error en la línea %d: Proceso %d %s.", lineNumber, pid, message)
}
