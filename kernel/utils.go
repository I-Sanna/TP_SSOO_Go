package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
	/*var reqBody struct {
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

	json.NewEncoder(w).Encode(map[string]int{"pid": pid})*/

	log.Println("Se solicito crear un proceso")
	json.NewEncoder(w).Encode(0)
}

func estadoProceso(w http.ResponseWriter, r *http.Request) {
	/*pid := obtenerPID(r)
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

	json.NewEncoder(w).Encode(resp)*/
	log.Println("Se solicito el estado de un proceso")
	json.NewEncoder(w).Encode("EXIT")
}

// finalizarProceso finaliza un proceso
func finalizarProceso(w http.ResponseWriter, r *http.Request) {
	//pid := obtenerPID(r)

	//delete(procesos, pid)

	//fmt.Printf("Finaliza el proceso %d - Motivo: SUCCESS\n", pid)

	log.Println("Se solicito finalizar un proceso")
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
	log.Println("Se solicito listar un proceso")
}

// iniciarPlanificacion inicia la planificación de procesos
func iniciarPlanificacion(w http.ResponseWriter, r *http.Request) {
	log.Println("Iniciar planificación...")
}

// detenerPlanificacion detiene la planificación de procesos
func detenerPlanificacion(w http.ResponseWriter, r *http.Request) {
	log.Println("Detener planificación...")
}

// obtenerPID obtiene el PID desde la URL
func obtenerPID(r *http.Request) int {
	//var pid int
	//fmt.Sscanf(r.URL.Path, "/process/%d", &pid)
	//return pid
	log.Println("Se solicito un PID")
	return 0
}
