package main

import (
	"encoding/json"
	"fmt"

	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/PUT/process", iniciarProceso)
	http.HandleFunc("/DELETE/process/fin", finalizarProceso)
	http.HandleFunc("/GET/process/est", estadoProceso)
	http.HandleFunc("/GET/process/list", listarProcesos)
	http.HandleFunc("/PUT/plani", iniciarPlanificacion)
	http.HandleFunc("/DELETE/plani", detenerPlanificacion)

	fmt.Println("Kernel escuchando en el puerto 8080...")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))

}

// PCB representa la estructura de control del proceso
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

// Recurso representa un recurso del sistema
type Recurso struct {
	Nombre     string
	Instancias int
}

var procesos = make(map[int]*PCB)

//var recursos = make(map[string]*Recurso)

// iniciarProceso inicia un nuevo proceso
func iniciarProceso(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Recibida solicitud para iniciar proceso")
	var reqBody struct {
		Path string `json:"path"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Error en la solicitud", http.StatusBadRequest)
		return
	}

	pid := len(procesos) + 1
	proceso := &PCB{
		PID:            pid,
		ProgramCounter: 0,
		Quantum:        100, // Valor por defecto
		Estado:         New,
		RegistrosCPU:   make(map[string]int),
	}

	procesos[pid] = proceso

	fmt.Printf("Se crea el proceso %d en NEW\n", pid)

	resp := map[string]int{"pid": pid}
	json.NewEncoder(w).Encode(resp)

	log.Printf("Se crea el proceso %d en NEW", pid)

}

func estadoProceso(w http.ResponseWriter, r *http.Request) {
	pid := obtenerPID(r)
	proceso, ok := procesos[pid]
	if !ok {
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		return
	}

	resp := struct {
		State string `json:"state"`
	}{
		State: string(proceso.Estado),
	}

	json.NewEncoder(w).Encode(resp)
}

// finalizarProceso finaliza un proceso
func finalizarProceso(w http.ResponseWriter, r *http.Request) {
	pid := obtenerPID(r)

	delete(procesos, pid)

	fmt.Printf("Finaliza el proceso %d - Motivo: SUCCESS\n", pid)
}

// listarProcesos lista todos los procesos
func listarProcesos(w http.ResponseWriter, r *http.Request) {
	var lista []map[string]interface{}
	for pid, proceso := range procesos {
		lista = append(lista, map[string]interface{}{
			"pid":   pid,
			"state": proceso.Estado,
		})
	}

	json.NewEncoder(w).Encode(lista)
}

// iniciarPlanificacion inicia la planificación de procesos
func iniciarPlanificacion(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Iniciar planificación...")
}

// detenerPlanificacion detiene la planificación de procesos
func detenerPlanificacion(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Detener planificación...")
}

// obtenerPID obtiene el PID desde la URL
func obtenerPID(r *http.Request) int {
	var pid int
	fmt.Sscanf(r.URL.Path, "/process/%d", &pid)
	return pid
}
