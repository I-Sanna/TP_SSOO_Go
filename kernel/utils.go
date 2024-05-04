package main

import (
	"bytes"
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
	RegistrosCPU   Registros
}

type Registros struct {
	PC  uint32 // Program Counter, indica la próxima instrucción a ejecutar
	AX  uint8  // Registro Numérico de propósito general
	BX  uint8  // Registro Numérico de propósito general
	CX  uint8  // Registro Numérico de propósito general
	DX  uint8  // Registro Numérico de propósito general
	EAX uint32 // Registro Numérico de propósito general
	EBX uint32 // Registro Numérico de propósito general
	ECX uint32 // Registro Numérico de propósito general
	EDX uint32 // Registro Numérico de propósito general
	SI  uint32 // Contiene la dirección lógica de memoria de origen desde donde se va a copiar un string
	DI  uint32 // Contiene la dirección lógica de memoria de destino a donde se va a copiar un string
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

type Kernel struct {
	Procesos []*PCB
}

func (k *Kernel) planificarFIFO() *PCB {
	if len(k.Procesos) == 0 {
		return nil
	}
	// Selecciona el primer proceso en la lista de procesos
	proceso := k.Procesos[0]
	// Remueve el proceso de la lista de procesos
	k.Procesos = k.Procesos[1:]
	// Cambia el estado del proceso a EXEC
	proceso.Estado = Exec
	return proceso
}

// Función para planificar un proceso usando Round Robin (RR)
func (k *Kernel) planificarRR() *PCB {
	if len(k.Procesos) == 0 {
		return nil
	}
	// Selecciona el primer proceso en la lista de procesos
	proceso := k.Procesos[0]
	// Remueve el proceso de la lista de procesos
	k.Procesos = append(k.Procesos[1:], proceso)
	// Cambia el estado del proceso a EXEC
	proceso.Estado = Exec
	return proceso
}

var k *Kernel

func init() {
	k = &Kernel{
		Procesos: make([]*PCB, 0),
	}
}

// iniciarProceso inicia un nuevo proceso
func iniciarProceso(w http.ResponseWriter, r *http.Request) {

	nuevoProceso := &PCB{
		PID:            len(k.Procesos) + 1,
		ProgramCounter: 0,
		Quantum:        100, // Valor por defecto
		Estado:         New,
		RegistrosCPU:   Registros{},
	}

	k.Procesos = append(k.Procesos, nuevoProceso)

	var request BodyRequest
	var response BodyRequestPid

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response = BodyRequestPid{PID: nuevoProceso.PID}

	respuesta, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func planificadoCortoPlazo(w http.ResponseWriter, r *http.Request) {

	var request BodyRequest
	var response BodyRequestPid

	procesoPlanificado := k.planificarFIFO()
	if procesoPlanificado == nil {
		procesoPlanificado = k.planificarRR()
	}

	enviarProcesoACPU(procesoPlanificado)

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respuesta, err := json.Marshal(response.PID)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func enviarProcesoACPU(pcb *PCB) {
	body, err := json.Marshal(pcb)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return
	}

	url := "http://localhost:8006/PCB"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando PCB: %s", err.Error())
		return
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
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

	respuesta, err := json.Marshal(k.Procesos)
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
