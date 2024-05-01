package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

type BodyRequest struct {
	Path string `json:"path"`
}

type BodyRequestPid struct {
	PID int `json:"pid"`
}

type BodyResponsePCB struct {
	PID   int    `json:"pid"`
	State string `json:"state"`
}

type BodyResponsePCBArray struct {
	Processes []BodyResponsePCB `json:"processes"`
}
type Config struct {
	PortKernel         int               `json:"port_kernel"`
	IpMemory           string            `json:"ip_memory"`
	PortMemory         int               `json:"port_memory"`
	IpCPU              string            `json:"ip_cpu"`
	PortCPU            int               `json:"port_cpu"`
	PortIO             int               `json:"port_io"`
	PlanningAlgorithm  string            `json:"planning_algorithm"`
	Quantum            int               `json:"quantum"`
	Resources          map[string]string `json:"resources"`
	Resource_instances []int             `json:"resource_instances"`
	Multiprogramming   int               `json:"multiprogramming"`
}

var ClientConfig *Config

func configurar() {

	ClientConfig = IniciarConfiguracion("config.json")
	log.Println(ClientConfig.PortKernel)
	log.Println(ClientConfig.PortCPU)
	log.Println(ClientConfig.PortMemory)
}
func IniciarConfiguracion(filePath string) *Config {
	var config *Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

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
	var request BodyRequest
	var response BodyRequestPid

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response.PID = 48

	respuesta, err := json.Marshal(response.PID)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
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
	pid := r.PathValue("pid")
	log.Println(pid)

	var response BodyRequest
	response.Path = "EXIT"

	respuesta, err := json.Marshal(response.Path)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// finalizarProceso finaliza un proceso
func finalizarProceso(w http.ResponseWriter, r *http.Request) {
	//pid := obtenerPID(r)

	//delete(procesos, pid)

	//fmt.Printf("Finaliza el proceso %d - Motivo: SUCCESS\n", pid)

	respuesta, err := json.Marshal("Se solicito finalizar un proceso")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// listarProcesos lista todos los procesos
func listarProcesos(w http.ResponseWriter, r *http.Request) {
	/*var lista []map[string]interface{}
	for pid, proceso := range procesos {
		lista = append(lista, map[string]interface{}{
			"pid":   pid,
			"state": proceso.Estado,
		})
	}

	json.NewEncoder(w).Encode(lista)*/

	var proceso1 BodyResponsePCB
	proceso1.PID = 1
	proceso1.State = "Ready"

	var proceso2 BodyResponsePCB
	proceso2.PID = 2
	proceso2.State = "EXIT"

	var procesos BodyResponsePCBArray
	procesos.Processes = append(procesos.Processes, proceso1)
	procesos.Processes = append(procesos.Processes, proceso2)

	respuesta, err := json.Marshal(procesos.Processes)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// iniciarPlanificacion inicia la planificación de procesos
func iniciarPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito iniciar planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// detenerPlanificacion detiene la planificación de procesos
func detenerPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito detener planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// obtenerPID obtiene el PID desde la URL
func obtenerPID(r *http.Request) int {
	//var pid int
	//fmt.Sscanf(r.URL.Path, "/process/%d", &pid)
	//return pid
	log.Println("Se solicito un PID")
	return 0
}
