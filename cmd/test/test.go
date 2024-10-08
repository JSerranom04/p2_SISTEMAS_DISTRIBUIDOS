package main

import (
	"fmt"
	"os"
	"strconv"

	functions "practica2/cmd/test/functions"
	utils "practica2/cmd/test/utils"
)

// Función para analizar los argumentos de línea de comandos
func parseArguments() int {
	if len(os.Args) != 2 {
		fmt.Println("Uso: go run test.go <numero_procesos>")
		os.Exit(1)
	}

	numProcesses, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("El argumento debe ser un número entero.")
		os.Exit(1)
	}

	return numProcesses
}

// Función para inicializar el estado de los procesos
func initializeProcessStates(numProcesses int) map[int]*utils.ProcessState {
	processStates := make(map[int]*utils.ProcessState)
	for pid := 1; pid <= numProcesses; pid++ {
		processStates[pid] = &utils.ProcessState{
			State:          "Idle",
			Operation:      "",
			PendingReplies: 0,
		}
	}
	return processStates
}

// Función para procesar cada evento
func processEvent(event utils.Event, processStates map[int]*utils.ProcessState, numProcesses int) {
	state := processStates[event.PID]

	switch event.Action {
	case "SEND_REQUEST":
		functions.HandleSendRequest(event, state, numProcesses)
	case "RECEIVE_REPLY":
		functions.HandleReceiveReply(event, state, processStates, numProcesses)
	}
}

// Función para finalizar el estado de los procesos al terminar el análisis
func finalizeProcessStates(processStates map[int]*utils.ProcessState) {
	for pid, state := range processStates {
		if state.State == "InCS" {
			fmt.Printf("Proceso %d sale de la sección crítica.\n", pid)
			state.State = "Idle"
			state.Operation = ""
		}
	}
}

// Función principal
func main() {
	numProcesses := parseArguments()
	processStates := initializeProcessStates(numProcesses)
	events := functions.ParseLogFile("salida.txt")

	for _, event := range events {
		processEvent(event, processStates, numProcesses)
	}

	finalizeProcessStates(processStates)

	fmt.Println("Test exitoso: Todas las verificaciones pasaron correctamente.")
}
