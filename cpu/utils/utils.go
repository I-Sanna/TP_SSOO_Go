package utils

import (
	"cpu/globals"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type PCB struct {
	PID            int
	ProgramCounter int
	Quantum        int
	Estado         ProcessState
	RegistrosCPU   map[string]int
}

type ProcessState string

const (
	New   ProcessState = "NEW"
	Ready ProcessState = "READY"
	Exec  ProcessState = "EXEC"
	Block ProcessState = "BLOCK"
	Exit  ProcessState = "EXIT"
)

func IniciarConfiguracion(filePath string) *globals.Config {
	var config *globals.Config

	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func ProbarCPU(w http.ResponseWriter, r *http.Request) {
	log.Println("API Probar!! Todo ok")
}

func RecibirProceso(w http.ResponseWriter, r *http.Request) {
	var paquete PCB

	err := json.NewDecoder(r.Body).Decode(&paquete)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	log.Println("me llegó un Proceso")
	log.Printf("%+v\n", paquete)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
