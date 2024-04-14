package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Process struct {
	PID    int    `json:"pid"`
	Status string `json:"status"`
}

var processes = make(map[int]Process)

func main() {
	http.HandleFunc("/start", startProcess)
	http.HandleFunc("/terminate", terminateProcess)
	http.HandleFunc("/status", getProcessStatus)

	fmt.Println("Kernel running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startProcess(w http.ResponseWriter, r *http.Request) {
	var process Process
	process.PID = len(processes)
	process.Status = "RUNNING"
	processes[process.PID] = process

	json.NewEncoder(w).Encode(process)
}

func terminateProcess(w http.ResponseWriter, r *http.Request) {
	pid := 0 // Aquí obtendrías el PID desde la solicitud, pero para simplificar está hardcodeado
	if process, exists := processes[pid]; exists {
		process.Status = "TERMINATED"
		processes[pid] = process
		fmt.Fprintf(w, "Process with PID %d terminated", pid)
	} else {
		fmt.Fprintf(w, "Process with PID %d not found", pid)
	}
}

func getProcessStatus(w http.ResponseWriter, r *http.Request) {
	pid := 0 // Aquí obtendrías el PID desde la solicitud
	if process, exists := processes[pid]; exists {
		json.NewEncoder(w).Encode(process)
	} else {
		fmt.Fprintf(w, "Process with PID %d not found", pid)
	}
}
