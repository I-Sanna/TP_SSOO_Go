package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"kernel/globals"
	"log"
	"net/http"
	"os"
	"time"
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

func ConfigurarLogger() {
	logFile, err := os.OpenFile("logs/kernel.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

type Kernel struct {
	Procesos []*PCB
}

func (k *Kernel) planificarFIFO() *PCB {
	//log.Print("\nSe planifica por FIFO\n")

	if len(k.Procesos) == 0 {
		return nil
	}
	// Selecciona el primer proceso en la lista de procesos
	proceso := k.Procesos[0]
	if proceso.Estado != Ready {
		return proceso
	}
	// Remueve el proceso de la lista de procesos
	k.Procesos = append(k.Procesos[1:], proceso)
	// Cambia el estado del proceso a EXEC
	proceso.Estado = Exec
	return proceso
}

// Función para planificar un proceso usando Round Robin (RR)
func (k *Kernel) planificarRR() *PCB {
	//log.Print("\nSe planifica por Round Robin\n")
	if len(k.Procesos) == 0 {
		return nil
	}
	// Selecciona el primer proceso en la lista de procesos
	proceso := k.Procesos[0]
	if proceso.Estado != Ready {
		return proceso
	}
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
func IniciarProceso(w http.ResponseWriter, r *http.Request) {

	//for i := 0; i < 10; i++ {

	nuevoProceso := &PCB{
		PID:            len(k.Procesos) + 1,
		ProgramCounter: 0,
		Quantum:        100, // Valor por defecto
		Estado:         New,
		RegistrosCPU:   Registros{},
	}

	k.Procesos = append(k.Procesos, nuevoProceso)

	var response = BodyRequestPid{PID: nuevoProceso.PID}

	respuesta, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	//}
}

var quantumOk = false

func iniciarQuantum() {
	quantumOk = true
	tiempo := globals.ClientConfig.Quantum
	log.Print("\n\nSe inicio el quantum\n\n")
	time.Sleep(time.Duration(tiempo) * time.Millisecond)
	log.Print("\n\nFin de quantum\n\n")
	quantumOk = false

}

func PlanificadoCortoPlazo(w http.ResponseWriter, r *http.Request) {

	var request BodyRequest
	var response BodyRequestPid
	log.Print(globals.ClientConfig.PlanningAlgorithm)
	switch globals.ClientConfig.PlanningAlgorithm {
	case "FIFO":
		for i := 0; i < len(k.Procesos); i++ {
			procesoPlanificado := k.planificarFIFO()
			log.Printf("\nk=%d Planifico por FIFO switch case 2\n", len(k.Procesos)-i)
			if procesoPlanificado.Estado == Exec {
				EnviarProcesoACPU(procesoPlanificado)
			}
		}
	case "RR":
		for i := 0; i < len(k.Procesos); i++ {
			if !quantumOk {
				go iniciarQuantum()
			}
			procesoPlanificado := k.planificarRR()
			log.Printf("\nk=%d Planifico por RR switch case 2\n", len(k.Procesos)-i)
			if procesoPlanificado.Estado == Exec {
				EnviarProcesoACPU(procesoPlanificado)
			}
		}
	default:
		http.Error(w, "Algoritmo de planificación no soportado", http.StatusInternalServerError)
		return
	}

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

func EnviarProcesoACPU(pcb *PCB) {
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

func EstadoProceso(w http.ResponseWriter, r *http.Request) {
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
func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
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
func ListarProcesos(w http.ResponseWriter, r *http.Request) {
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
func IniciarPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito iniciar planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	for i := 0; i < len(k.Procesos); i++ {
		proceso := k.Procesos[0]
		proceso.Estado = Ready
		k.Procesos = append(k.Procesos[1:], proceso)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// detenerPlanificacion detiene la planificación de procesos
func DetenerPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito detener planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// pedir io a entradasalid
func PedirIO(w http.ResponseWriter, r *http.Request) {
	url := "http://localhost:8004/sleep"
	resp, err := http.Post(url, "application/json", nil) // Enviando nil como el cuerpo
	if err != nil {
		log.Printf("error enviando PCB: %s", err.Error())
		return
	}

	defer resp.Body.Close()
	log.Printf("respuesta del servidor: %s", resp.Status)
}
