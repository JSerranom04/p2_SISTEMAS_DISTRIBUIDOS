package testing

import (
	"fmt"
	"os"
	"practica2/cmd/test/readerwriter"
	"testing"

	"github.com/DistributedClocks/GoVector/govec"
)

func TestLectoresEscritoresDistribuido(t *testing.T) {
	sharedRWFile := "shared_test_file.txt"
	endpointsFile := "data/endpoints/endpoints0.txt"

	os.Create(sharedRWFile)
	defer os.Remove(sharedRWFile)

	loggerConfig := govec.GetDefaultConfig()
	goLogger := govec.InitGoVector("TestLectoresEscritores", "logs/test_logs.log", loggerConfig)

	finLectores := make(chan bool)
	finEscritores := make(chan bool)

	for i := 0; i < 3; i++ {
		go func(pid int) {
			goLogger.LogLocalEvent(fmt.Sprintf("Lector %d iniciando", pid))
			readerwriter.EjecutarLector(pid, endpointsFile, "test_shared_rw", "logs/lector_"+fmt.Sprint(pid)+".log")
			finLectores <- true
		}(i + 1)
	}

	for i := 0; i < 2; i++ {
		go func(pid int) {
			goLogger.LogLocalEvent(fmt.Sprintf("Escritor %d iniciando", pid))
			readerwriter.EjecutarEscritor(pid, endpointsFile, "test_shared_rw", "logs/escritor_"+fmt.Sprint(pid)+".log", "data/content/contentRFile.txt")
			finEscritores <- true
		}(i + 4)
	}

	for i := 0; i < 3; i++ {
		<-finLectores
	}

	for i := 0; i < 2; i++ {
		<-finEscritores
	}

	goLogger.LogLocalEvent("Finalización de la prueba de lectores y escritores distribuidos")
}
