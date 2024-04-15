package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// PCB representa la estructura de control del proceso
type PCB struct {
	PID            int
	ProgramCounter int
	Quantum        int
	Estado         string
	RegistrosCPU   map[string]int
}

// Recurso representa un recurso del sistema
type Recurso struct {
	Nombre     string
	Instancias int
}

var procesos = make(map[int]*PCB)

//var recursos = make(map[string]*Recurso)

// iniciarProceso inicia un nuevo proceso
func iniciarProceso(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Path string `json:"path"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Error en la solicitud", http.StatusBadRequest)
		return
	}

	pid := len(procesos)
	proceso := &PCB{
		PID:            pid,
		ProgramCounter: 0,
		Quantum:        100, // Valor por defecto
		Estado:         "NEW",
		RegistrosCPU:   make(map[string]int),
	}

	procesos[pid] = proceso

	fmt.Printf("Se crea el proceso %d en NEW\n", pid)

	json.NewEncoder(w).Encode(map[string]int{"pid": pid})

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
		State: proceso.Estado,
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
	// Implementación de la lógica de planificación
	fmt.Println("Iniciar planificación...")
}

// detenerPlanificacion detiene la planificación de procesos
func detenerPlanificacion(w http.ResponseWriter, r *http.Request) {
	// Implementación de la lógica de detención de planificación
	fmt.Println("Detener planificación...")
}

// obtenerPID obtiene el PID desde la URL
func obtenerPID(r *http.Request) int {
	var pid int
	fmt.Sscanf(r.URL.Path, "/process/%d", &pid)
	return pid
}
