package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"kernel/globals"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// PCB representa la estructura de control del proceso
type PCB struct {
	PID            int          `json:"pid"`
	ProgramCounter int          `json:"program_counter"`
	Quantum        int          `json:"quantum"`
	Estado         ProcessState `json:"estado"`
	RegistrosCPU   Registros    `json:"registros_cpu"`
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

var planificando bool
var colaDeNuevos []PCB
var colaDeListos []PCB
var estadosProcesos map[int]string
var recursos map[string]int
var puertosDispGenericos map[string]int
var puertosDispSTDIN map[string]int
var puertosDispSTDOUT map[string]int
var listaEsperaRecursos map[string][]PCB
var listaEsperaGenericos map[string][]PCB
var listaEsperaSTDIN map[string][]PCB
var listaEsperaSTDOUT map[string][]PCB

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

func InicializarVariables() {
	planificando = false
	estadosProcesos = make(map[int]string)
	recursos = make(map[string]int)
	puertosDispGenericos = make(map[string]int)
	puertosDispSTDIN = make(map[string]int)
	puertosDispSTDOUT = make(map[string]int)
	listaEsperaRecursos = make(map[string][]PCB)
	listaEsperaGenericos = make(map[string][]PCB)
	listaEsperaSTDIN = make(map[string][]PCB)
	listaEsperaSTDOUT = make(map[string][]PCB)

	for i := 0; i < len(globals.ClientConfig.Resources); i++ {
		recursos[globals.ClientConfig.Resources[i]] = globals.ClientConfig.Resource_instances[i]
	}
}

func planificarFIFO() {
	//Semaforo para que espre hasta que haya procesos en la cola de listos
	for len(colaDeListos) > 0 && planificando {
		// Selecciona el primer proceso en la lista de procesos
		proceso := colaDeListos[0]

		// Remueve el proceso de la lista de procesos
		colaDeListos = colaDeListos[1:]
		// Cambia el estado del proceso a EXEC
		proceso.Estado = Exec
		// Enviarlo a ejecutar a la CPU
	}
}

// Función para planificar un proceso usando Round Robin (RR)
func planificarRR() {
	//Semaforo para que espre hasta que haya procesos en la cola de listos
	for len(colaDeListos) > 0 && planificando {

		// Selecciona el primer proceso en la lista de procesos
		proceso := colaDeListos[0]

		// Remueve el proceso de la lista de procesos
		colaDeListos = colaDeListos[1:]
		// Cambia el estado del proceso a EXEC
		proceso.Estado = Exec
		// Enviarlo a ejecutar a la CPU
		time.Sleep(time.Duration(globals.ClientConfig.Quantum) * time.Millisecond)
	}
}

// iniciarProceso inicia un nuevo proceso
func IniciarProceso(w http.ResponseWriter, r *http.Request) {

	//for i := 0; i < 10; i++ {

	nuevoProceso := PCB{
		PID:            len(colaDeListos) + 1,
		ProgramCounter: 0,
		Quantum:        100, // Valor por defecto
		Estado:         New,
		RegistrosCPU:   Registros{},
	}

	colaDeListos = append(colaDeListos, nuevoProceso)

	var response = BodyRequestPid{PID: nuevoProceso.PID}

	respuesta, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func ProbarKernel(w http.ResponseWriter, r *http.Request) {
	var pcb1 PCB
	pcb1.PID = 1
	pcb1.ProgramCounter = 0
	pcb1.Quantum = 100
	pcb1.Estado = New
	var pcb2 PCB
	pcb2.PID = 2
	pcb2.ProgramCounter = 0
	pcb2.Quantum = 100
	pcb2.Estado = Ready
	var pcb3 PCB
	pcb3.PID = 3
	pcb3.ProgramCounter = 0
	pcb3.Quantum = 100
	pcb3.Estado = Exec
	estadosProcesos[pcb1.PID] = string(pcb1.Estado)
	estadosProcesos[pcb2.PID] = string(pcb2.Estado)
	estadosProcesos[pcb3.PID] = string(pcb3.Estado)
	log.Printf("Llego el proceso modificado")
	log.Printf("%+v\n", estadosProcesos)
}

type BodyReqExec struct {
	Pcb     PCB    `json:"pcb"`
	Mensaje string `json:"mensaje"`
}

func EnviarProcesoACPU(pcb *PCB) {

	body, err := json.Marshal(pcb)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortCPU) + "/PCB"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando PCB: %s", err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("error en la respuesta de la consulta: %s", resp.Status)
		return
	}

	var resultadoCPU BodyReqExec

	err = json.NewDecoder(resp.Body).Decode(&resultadoCPU)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return
	}

	*pcb = resultadoCPU.Pcb

	log.Printf("Llego el proceso modificado")
	log.Printf("%+v\n", pcb)
	log.Printf(resultadoCPU.Mensaje)
}

func EstadoProceso(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	valor, ok := estadosProcesos[pidInt]
	if !ok {
		valor = "El PID ingresado no existe"
	}

	respuesta, err := json.Marshal(valor)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// A desarrollar
func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	respuesta, err := json.Marshal("Se solicito finalizar un proceso")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// A desarrollar
func ListarProcesos(w http.ResponseWriter, r *http.Request) {

}

// A desarrollar
func IniciarPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito iniciar planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	for i := 0; i < len(colaDeListos); i++ {
		proceso := colaDeListos[0]
		proceso.Estado = Ready
		colaDeListos = append(colaDeListos[1:], proceso)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

// A desarrollar
func DetenerPlanificacion(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("Se solicito detener planificación")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

type BodyRequestTime struct {
	Dispositivo string `json:"dispositivo"`
	CantidadIO  int    `json:"cantidad_io"`
}

// pedir io a entradasalid
func PedirIO(w http.ResponseWriter, r *http.Request) {
	var request BodyRequestTime

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	go Sleep(puertosDispGenericos[request.Dispositivo], request.CantidadIO)

	log.Println("me llegó un Proceso")
	log.Printf("%+v\n", request)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func Sleep(puerto int, cantidadIO int) {
	url := "http://localhost:" + strconv.Itoa(puerto) + "/sleep/" + strconv.Itoa(cantidadIO)
	resp, err := http.Get(url) // Enviando nil como el cuerpo
	if err != nil {
		log.Printf("error enviando: %s", err.Error())
		return
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

type BodyRequestIO struct {
	NombreDispositivo    string `json:"nombre_dispositivo"`
	PuertoDispositivo    int    `json:"puerto_dispositivo"`
	CategoriaDispositivo string `json:"categoria_dispositivo"`
}

func RegistrarIO(w http.ResponseWriter, r *http.Request) {
	var request BodyRequestIO

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	if puertosDispGenericos == nil {
		puertosDispGenericos = make(map[string]int)
	}

	switch request.CategoriaDispositivo {
	case "Generico":
		puertosDispGenericos[request.NombreDispositivo] = request.PuertoDispositivo
	}

	log.Printf("%+v\n", puertosDispGenericos)
}
