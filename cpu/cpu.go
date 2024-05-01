package main

import (
	"encoding/json"
	"log"
	"net/http"
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

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /PCB", recibirPCB)

	err := http.ListenAndServe(":8006", mux)
	if err != nil {
		panic(err)
	}
}

func recibirPCB(w http.ResponseWriter, r *http.Request) {
	var paquete PCB

	err := json.NewDecoder(r.Body).Decode(&paquete)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	log.Println("me llegó un PCB")
	log.Printf("%+v\n", paquete)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
