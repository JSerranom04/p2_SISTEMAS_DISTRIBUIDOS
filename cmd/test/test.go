package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/DistributedClocks/GoVector/govec"
)

func main() {
	sharedRWFile := "shared_test_file.txt"
	endpointsFile := "./data/endpoints/endpoints0.txt"
	// Crear el archivo compartido
	os.Create(sharedRWFile)
	defer os.Remove(sharedRWFile)

	// Configuración de GoVector
	loggerConfig := govec.GetDefaultConfig()
	goLogger := govec.InitGoVector("TestLectoresEscritores", "logs/test_logs.log", loggerConfig)
	logOptions := govec.GetDefaultLogOptions()

	// Leer los endpoints desde el archivo
	endpoints, err := leerEndpoints(endpointsFile)
	if err != nil {
		log.Fatalf("Error al leer el archivo de endpoints: %v", err)
	}

	// Inicializar canales de finalización
	finLectores := make(chan bool)
	finEscritores := make(chan bool)

	// Ejecutar lectores y escritores en las máquinas remotas
	for i, endpoint := range endpoints {
		if i < 3 {
			// Ejecutar lectores
			go func(pid int, endpoint string) {
				goLogger.LogLocalEvent(fmt.Sprintf("Lector %d iniciando", pid), logOptions)
				executeRemoteCommand(endpoint, fmt.Sprintf("cd /path/to/project && go run cmd/lector/lector.go %d data/endpoints/endpoints0.txt shared_test_file logs/lector_%d.log", pid, pid))
				finLectores <- true
			}(i+1, endpoint)
		} else {
			// Ejecutar escritores
			go func(pid int, endpoint string) {
				goLogger.LogLocalEvent(fmt.Sprintf("Escritor %d iniciando", pid), logOptions)
				executeRemoteCommand(endpoint, fmt.Sprintf("cd /path/to/project && go run cmd/escritor/escritor.go %d data/endpoints/endpoints0.txt shared_test_file logs/escritor_%d.log data/content/contentRFile.txt", pid, pid))
				finEscritores <- true
			}(i+1, endpoint)
		}
	}

	// Esperar a que todos los procesos terminen
	for i := 0; i < 3; i++ {
		<-finLectores
	}
	for i := 0; i < 2; i++ {
		<-finEscritores
	}

	goLogger.LogLocalEvent("Finalización de la prueba de lectores y escritores distribuidos", logOptions)
}

// leerEndpoints lee los endpoints del archivo de texto y devuelve un slice con las direcciones IP y puertos.
func leerEndpoints(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			endpoints = append(endpoints, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return endpoints, nil
}

// executeRemoteCommand se conecta a la máquina remota mediante SSH y ejecuta el comando proporcionado.
func executeRemoteCommand(endpoint string, command string) {
	// Construir el comando SSH
	sshCommand := fmt.Sprintf("ssh %s %s", endpoint, command)

	// Ejecutar el comando
	cmd := exec.Command("bash", "-c", sshCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error al ejecutar el comando en %s: %v\nSalida: %s", endpoint, err, string(output))
	} else {
		log.Printf("Comando ejecutado correctamente en %s\nSalida: %s", endpoint, string(output))
	}
}
