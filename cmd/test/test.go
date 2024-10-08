package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
)

// Estructura para representar eventos relevantes en los logs
type Event struct {
	LineNumber int
	PID        int
	Operation  string // "READ" o "WRITE"
	Action     string // "SEND_REQUEST", "RECEIVE_REPLY"
	TargetPID  int    // PID del proceso destino u origen según corresponda
}

// Estructura para rastrear el estado de cada proceso
type ProcessState struct {
	State          string // "Idle", "Requesting", "InCS"
	Operation      string // "READ" o "WRITE"
	PendingReplies int
}

// Función principal
func main() {
	// Verificar que se ha proporcionado el número correcto de argumentos
	if len(os.Args) != 2 {
		fmt.Println("Uso: go run test.go <numero_procesos>")
		os.Exit(1)
	}

	// Número total de procesos en el sistema
	numProcesses, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("El argumento debe ser un número entero.")
		os.Exit(1)
	}

	// Archivo de logs generado por los procesos
	logFile := "salida.txt"

	// Abrir el archivo de logs
	file, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("Error al abrir el archivo de logs: %v", err)
	}
	defer file.Close()

	// Mapas para rastrear el estado de los procesos
	processStates := make(map[int]*ProcessState)

	// Inicializar el estado de todos los procesos esperados
	for pid := 1; pid <= numProcesses; pid++ {
		processStates[pid] = &ProcessState{
			State:          "Idle",
			Operation:      "", // Desconocido al inicio
			PendingReplies: 0,
		}
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// Parsear la línea para extraer información relevante
		event, err := parseLogLine(line, lineNumber)
		if err != nil {
			// Si la línea no es relevante, se ignora
			continue
		}

		state := processStates[event.PID]

		switch event.Action {
		case "SEND_REQUEST":
			if state.State == "Idle" || state.State == "InCS" {
				// Si estaba en InCS, asumimos que salió de la sección crítica
				if state.State == "InCS" {
					// Proceso sale de la sección crítica
					fmt.Printf("Proceso %d sale de la sección crítica.\n", event.PID)
					state.State = "Idle"
					state.Operation = ""
				}
				// Inicia una nueva solicitud
				fmt.Printf("Proceso %d solicita entrar en la sección crítica (%s).\n", event.PID, event.Operation)
				state.State = "Requesting"
				state.Operation = event.Operation
				state.PendingReplies = 1
			} else if state.State == "Requesting" {
				state.PendingReplies += 1
			}
		case "RECEIVE_REPLY":
			if state.State == "Requesting" {
				if state.PendingReplies > 0 {
					state.PendingReplies -= 1
					if state.PendingReplies == 0 {
						// Todas las respuestas recibidas, entra en la sección crítica
						fmt.Printf("Proceso %d entra en la sección crítica (%s).\n", event.PID, state.Operation)
						state.State = "InCS"
						// Verificar exclusión mutua y reglas de lectores-escritores
						checkMutualExclusion(processStates, event.PID, state.Operation, lineNumber)
					}
				} else {
					log.Printf("Error en la línea %d: Proceso %d recibió una respuesta inesperada.", lineNumber, event.PID)
					fmt.Println("Test fallido: Respuestas recibidas sin haber enviado solicitudes.")
					return
				}
			} else {
				log.Printf("Error en la línea %d: Proceso %d recibió una respuesta sin estar solicitando.", lineNumber, event.PID)
				fmt.Println("Test fallido: Respuestas recibidas sin haber enviado solicitudes.")
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error al leer el archivo de logs: %v", err)
	}

	// Verificar que ningún proceso quedó en la sección crítica
	for pid, state := range processStates {
		if state.State == "InCS" {
			fmt.Printf("Proceso %d sale de la sección crítica.\n", pid)
			state.State = "Idle"
			state.Operation = ""
		}
	}

	fmt.Println("Test exitoso: Todas las verificaciones pasaron correctamente.")
}

// Función para parsear una línea del log y extraer un evento
func parseLogLine(line string, lineNumber int) (Event, error) {
	// Expresiones regulares para detectar eventos
	// [PID 1] Sending CS request to process 2, payload: {[1 0 0 0 0 0 0 0 0 0 0 0] 1 1}
	sendRequestRegex := regexp.MustCompile(`\[PID (\d+)\] Sending CS request to process (\d+), payload: {\[(.*?)\] (\d+) (\d+)}`)
	// [PID 1] Received ra_vector.VReply: {}, my_send_clock: [1 0 0 0 0 0 0 0 0 0 0 0]
	receiveReplyRegex := regexp.MustCompile(`\[PID (\d+)\] Received ra_vector\.VReply: {}, my_send_clock: \[(.*?)\]`)

	if matches := sendRequestRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		targetPID, _ := strconv.Atoi(matches[2])
		opTypeInt, _ := strconv.Atoi(matches[5])
		var opTypeStr string
		if opTypeInt == 0 {
			opTypeStr = "READ"
		} else if opTypeInt == 1 {
			opTypeStr = "WRITE"
		} else {
			opTypeStr = "UNKNOWN"
		}
		return Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "SEND_REQUEST",
			TargetPID:  targetPID,
			Operation:  opTypeStr,
		}, nil
	} else if matches := receiveReplyRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		return Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "RECEIVE_REPLY",
		}, nil
	}
	return Event{}, fmt.Errorf("Línea no relevante")
}

// Función para verificar la exclusión mutua y las reglas de lectores-escritores
func checkMutualExclusion(processStates map[int]*ProcessState, enteringPID int, enteringOp string, lineNumber int) {
	for pid, ps := range processStates {
		if ps.State == "InCS" && pid != enteringPID {
			// Otro proceso está en la sección crítica
			if enteringOp == "WRITE" || ps.Operation == "WRITE" {
				// Violación: Un escritor no puede estar en la sección crítica con otro proceso
				logViolation(lineNumber, enteringPID, pid, "Violación de exclusión mutua o lectores-escritores")
				os.Exit(1)
			}
		}
	}
}

// Función para registrar una violación y mostrar un mensaje de error
func logViolation(lineNumber, pid1, pid2 int, message string) {
	log.Printf("Violación en la línea %d: %s. Procesos involucrados: %d y %d.", lineNumber, message, pid1, pid2)
	fmt.Printf("Test fallido: %s\n", message)
}
