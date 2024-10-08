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
	LineNumber  int
	PID         int
	Operation   string // "READ" o "WRITE"
	Action      string // "REQUEST", "REPLY", "SEND_REQUEST", "RECEIVE_REQUEST", "SEND_REPLY", "RECEIVE_REPLY"
	TargetPID   int    // PID del proceso destino o origen según corresponda
	VectorClock []int
}

// Estructura para rastrear el estado de cada proceso
type ProcessState struct {
	InCriticalSection bool
	Operation         string       // "READ" o "WRITE"
	PendingReplies    map[int]bool // PIDs de los procesos de los cuales espera confirmación
}

// Función principal
func main() {
	// Verificar que se ha proporcionado el número correcto de argumentos
	if len(os.Args) != 2 {
		fmt.Println("Uso: go run main.go <numero_procesos>")
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
	events := []Event{}

	// Inicializar el estado de todos los procesos esperados
	for pid := 1; pid <= numProcesses; pid++ {
		processStates[pid] = &ProcessState{
			InCriticalSection: false,
			Operation:         "", // Desconocido al inicio
			PendingReplies:    make(map[int]bool),
		}
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		//log.Printf("%s\n", line)

		// Parsear la línea para extraer información relevante
		event, err := parseLogLine(line, lineNumber, numProcesses)
		if err != nil {
			// Si la línea no es relevante, se ignora
			continue
		}

		events = append(events, event)

		state := processStates[event.PID]

		switch event.Action {
		case "SEND_REQUEST":
			// Cuando un proceso envía una solicitud, marca que espera respuesta del proceso destino
			if state.Operation == "" {
				state.Operation = "WRITE" // Asumimos que es escritura si no se especifica
			}
			state.PendingReplies[event.TargetPID] = true

		case "RECEIVE_REPLY":
			// Cuando un proceso recibe una confirmación, elimina al proceso de la lista de pendientes
			delete(state.PendingReplies, event.TargetPID)

			// Si ya recibió todas las confirmaciones, entra en la sección crítica
			if len(state.PendingReplies) == 0 && !state.InCriticalSection {
				// Verificar reglas de acceso a la sección crítica
				if state.Operation == "WRITE" {
					// Un escritor no puede entrar si hay otro proceso en la sección crítica
					for pid, ps := range processStates {
						if ps.InCriticalSection && pid != event.PID {
							logViolation(lineNumber, event.PID, pid, "Escritor entrando cuando otro proceso está en sección crítica")
							return
						}
					}
				} else if state.Operation == "READ" {
					// Un lector no puede entrar si hay un escritor en la sección crítica
					for pid, ps := range processStates {
						if ps.InCriticalSection && ps.Operation == "WRITE" && pid != event.PID {
							logViolation(lineNumber, event.PID, pid, "Lector entrando cuando un escritor está en sección crítica")
							return
						}
					}
				}
				state.InCriticalSection = true
				// Opcional: puedes registrar que el proceso entró en la sección crítica
				// log.Printf("[PID %d] Enters critical section", event.PID)
			}

		case "EXIT":
			if !state.InCriticalSection {
				log.Printf("Error en la línea %d: Proceso %d intenta salir de la sección crítica sin haber entrado.", lineNumber, event.PID)
				fmt.Println("Test fallido: Estado inconsistente del proceso.")
				return
			}
			state.InCriticalSection = false
			state.Operation = ""
			// Opcional: puedes registrar que el proceso salió de la sección crítica
			// log.Printf("[PID %d] Exits critical section", event.PID)

			// Puedes manejar otros casos si es necesario
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error al leer el archivo de logs: %v", err)
	}

	// Verificar que ningún proceso quedó en la sección crítica
	for pid, state := range processStates {
		if state.InCriticalSection {
			log.Printf("Error: El proceso %d quedó en la sección crítica al finalizar los logs.", pid)
			fmt.Println("Test fallido: Procesos quedaron en la sección crítica.")
			return
		}
	}

	fmt.Println("Test exitoso: Todas las verificaciones pasaron correctamente.")
}

// Función para parsear una línea del log y extraer un evento
func parseLogLine(line string, lineNumber int, numProcesses int) (Event, error) {
	// Expresiones regulares para detectar eventos
	sendRequestRegex := regexp.MustCompile(`\[PID (\d+)\] Sending CS request to process (\d+), payload: .*`)
	receiveReplyRegex := regexp.MustCompile(`\[PID (\d+)\] Received ra_vector\.VReply: .*`)
	exitCSRegex := regexp.MustCompile(`\[PID (\d+)\] Released critical section`)

	if matches := sendRequestRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		targetPID, _ := strconv.Atoi(matches[2])
		return Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "SEND_REQUEST",
			TargetPID:  targetPID,
		}, nil
	} else if matches := receiveReplyRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		// Necesitamos inferir el PID del proceso que envió el reply, pero no está en la línea
		// En este caso, asumimos que no es necesario, ya que eliminamos una confirmación pendiente
		return Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "RECEIVE_REPLY",
			// No conocemos TargetPID aquí
		}, nil
	} else if matches := exitCSRegex.FindStringSubmatch(line); matches != nil {
		pid, _ := strconv.Atoi(matches[1])
		return Event{
			LineNumber: lineNumber,
			PID:        pid,
			Action:     "EXIT",
		}, nil
	}

	return Event{}, fmt.Errorf("Línea no relevante")
}

// Función para registrar una violación y mostrar un mensaje de error
func logViolation(lineNumber, pid1, pid2 int, message string) {
	log.Printf("Violación en la línea %d: %s. Procesos involucrados: %d y %d.", lineNumber, message, pid1, pid2)
	fmt.Printf("Test fallido: %s\n", message)
}
