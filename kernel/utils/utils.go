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
	"sync"
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

// Semaforos
var planificadorCortoPlazo sync.Mutex
var planificadorLargoPlazo sync.Mutex
var dispositivoGenerico sync.Mutex
var mutexColaListos sync.Mutex
var mutexColaWaiting sync.Mutex
var mutexColaNuevos sync.Mutex
var semProcesosListos chan int

// Variables
var contadorPID int
var planificando bool
var colaDeNuevos []PCB
var colaDeListos []PCB
var colaDeWaiting []PCB
var estadosProcesos map[int]string
var recursos map[string]int
var puertosDispGenericos map[string]int
var puertosDispSTDIN map[string]int
var puertosDispSTDOUT map[string]int
var listaEsperaRecursos map[string][]int
var listaEsperaGenericos map[string][]BodyIO
var listaEsperaSTDIN map[string][]int
var listaEsperaSTDOUT map[string][]int

type BodyIO struct {
	PID        int
	CantidadIO int
}

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
	contadorPID = 0
	planificando = true
	semProcesosListos = make(chan int)
	estadosProcesos = make(map[int]string)
	recursos = make(map[string]int)
	puertosDispGenericos = make(map[string]int)
	puertosDispSTDIN = make(map[string]int)
	puertosDispSTDOUT = make(map[string]int)
	listaEsperaRecursos = make(map[string][]int)
	listaEsperaGenericos = make(map[string][]BodyIO)
	listaEsperaSTDIN = make(map[string][]int)
	listaEsperaSTDOUT = make(map[string][]int)

	for i := 0; i < len(globals.ClientConfig.Resources); i++ {
		recursos[globals.ClientConfig.Resources[i]] = globals.ClientConfig.Resource_instances[i]
	}

}

func InicializarPlanificador() {
	switch globals.ClientConfig.PlanningAlgorithm {
	case "FIFO":
		go planificarFIFO()
	case "RR":
		go planificarRR()
	}
}

func planificarFIFO() {
	semProcesosListos <- 0
	planificadorCortoPlazo.Lock()
	// Selecciona el primer proceso en la lista de procesos
	proceso := colaDeListos[0]

	// Cambia el estado del proceso a EXEC
	proceso.Estado = Exec
	// Enviarlo a ejecutar a la CPU
	//mensaje := EnviarProcesoACPU(&proceso)

	planificadorCortoPlazo.Unlock()
	planificadorCortoPlazo.Lock() //Estos semaforos es por si se ejecuto "detenerPlanificacion"

	mutexColaListos.Lock()
	colaDeListos = colaDeListos[1:]
	//Agregar el proceso modificado por la CPU si corresponde
	if estadosProcesos[proceso.PID] == "EXIT" {
		delete(estadosProcesos, proceso.PID)
	} else {
		colaDeListos = append(colaDeListos, proceso)
	}
	mutexColaListos.Unlock()
	planificadorCortoPlazo.Unlock()
}

// Función para planificar un proceso usando Round Robin (RR)
func planificarRR() {
	semProcesosListos <- 0
	planificadorCortoPlazo.Lock()
	// Selecciona el primer proceso en la lista de procesos
	proceso := colaDeListos[0]

	// Cambia el estado del proceso a EXEC
	proceso.Estado = Exec
	// Enviarlo a ejecutar a la CPU
	time.Sleep(time.Duration(globals.ClientConfig.Quantum) * time.Millisecond)

	//wait(planificadorCortoPlazo) -> Manejar Interrupcion -> Signal
	mutexColaListos.Lock()
	colaDeListos = colaDeListos[1:]
	//Agregar el proceso modificado por la CPU
	mutexColaListos.Unlock()
	planificadorCortoPlazo.Unlock()
}

// iniciarProceso inicia un nuevo proceso
func IniciarProceso(w http.ResponseWriter, r *http.Request) {
	planificadorLargoPlazo.Lock()

	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return
	}
	//Quizas se podria omitir este proceso de decodificar y luego codificar de nuevo
	body, err := json.Marshal(request)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return
	}

	cliente := &http.Client{}
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/process"
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := cliente.Do(req)
	if err != nil {
		log.Printf("error enviando el Path: %s", err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("error en la respuesta de la consulta: %s", resp.Status)
		return
	}

	nuevoProceso := PCB{
		PID:            contadorPID,
		ProgramCounter: 0,
		Quantum:        globals.ClientConfig.Quantum, // Valor por defecto
		Estado:         New,
		RegistrosCPU:   Registros{},
	}

	contadorPID++

	mutexColaNuevos.Lock()
	if len(colaDeNuevos) > 0 {
		colaDeNuevos = append(colaDeNuevos, nuevoProceso)
		mutexColaNuevos.Unlock()
	} else {
		mutexColaNuevos.Unlock()

		mutexColaListos.Lock()
		mutexColaWaiting.Lock()

		if (len(colaDeListos) + len(colaDeWaiting)) < globals.ClientConfig.Multiprogramming {
			nuevoProceso.Estado = Ready
			colaDeListos = append(colaDeListos, nuevoProceso)
			<-semProcesosListos
			mutexColaWaiting.Unlock()
			mutexColaListos.Unlock()
		} else {
			mutexColaWaiting.Unlock()
			mutexColaListos.Unlock()

			mutexColaNuevos.Lock()
			colaDeNuevos = append(colaDeNuevos, nuevoProceso)
			mutexColaNuevos.Unlock()
		}
	}
	log.Printf("%+v\n", colaDeNuevos)
	log.Printf("%+v\n", colaDeListos)
	log.Printf("%+v\n", colaDeWaiting)

	var response = BodyRequestPid{PID: nuevoProceso.PID}

	respuesta, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	planificadorLargoPlazo.Unlock()
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

func EnviarProcesoACPU(pcb *PCB) string {

	body, err := json.Marshal(pcb)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return "error"
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortCPU) + "/PCB"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando PCB: %s", err.Error())
		return "error"
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("error en la respuesta de la consulta: %s", resp.Status)
		return "error"
	}

	var resultadoCPU BodyReqExec

	err = json.NewDecoder(resp.Body).Decode(&resultadoCPU)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return "error"
	}

	*pcb = resultadoCPU.Pcb

	log.Printf("Llego el proceso modificado")
	log.Printf("%+v\n", resultadoCPU.Pcb)
	log.Printf(resultadoCPU.Mensaje)

	return resultadoCPU.Mensaje
}

func ManejarInterrupcion(interrupcion string) {

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
	planificadorLargoPlazo.Lock()
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	valor, ok := estadosProcesos[pidInt]
	if !ok {
		valor = "El PID ingresado no existe"
	} else {
		valor = "El proceso fue eliminado con exito"
	}

	respuesta, err := json.Marshal(valor)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	planificadorLargoPlazo.Unlock()
}

func ListarProcesos(w http.ResponseWriter, r *http.Request) {
	var listaProcesos []BodyResponsePCB
	var proceso BodyResponsePCB

	for pid, estado := range estadosProcesos {
		proceso.PID = pid
		proceso.State = estado
		listaProcesos = append(listaProcesos, proceso)
	}

	respuesta, err := json.Marshal(listaProcesos)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func IniciarPlanificacion(w http.ResponseWriter, r *http.Request) {
	if !planificando {
		planificadorCortoPlazo.Unlock()
		planificadorLargoPlazo.Unlock()
		planificando = true
	}
}

// A desarrollar
func DetenerPlanificacion(w http.ResponseWriter, r *http.Request) {
	if planificando {
		planificadorCortoPlazo.Lock()
		planificadorLargoPlazo.Lock()
		planificando = false
	}
}

type BodyRequestTime struct {
	Dispositivo string `json:"dispositivo"`
	CantidadIO  int    `json:"cantidad_io"`
	PID         int    `json:"pid"`
	Instruccion string `json:"instruccion"`
}

// pedir io a entradasalid
func PedirIO(w http.ResponseWriter, r *http.Request) {
	var request BodyRequestTime

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return
	}

	switch request.Instruccion {
	case "SLEEP":
		var datosIO BodyIO
		datosIO.PID = request.PID
		datosIO.CantidadIO = request.CantidadIO
		dispositivoGenerico.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispGenericos[request.Dispositivo]
		if ok {
			listaEsperaGenericos[request.Dispositivo] = append(listaEsperaGenericos[request.Dispositivo], datosIO)
			if len(listaEsperaGenericos[request.Dispositivo]) == 1 {
				go Sleep(request.Dispositivo, puerto)
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoGenerico.Unlock()
			return
		}
		log.Printf("%+v\n", listaEsperaGenericos[request.Dispositivo])
		dispositivoGenerico.Unlock()
	}

	log.Println("me llegó un Proceso")
	log.Printf("%+v\n", request)

	w.WriteHeader(http.StatusOK)
}

func Sleep(nombreDispositivo string, puerto int) {
	dispositivoGenerico.Lock()
	for len(listaEsperaGenericos[nombreDispositivo]) > 0 {
		proceso := listaEsperaGenericos[nombreDispositivo][0]
		dispositivoGenerico.Unlock()
		url := "http://localhost:" + strconv.Itoa(puerto) + "/sleep/" + strconv.Itoa(proceso.CantidadIO)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			dispositivoGenerico.Lock()
			for _, elemento := range listaEsperaGenericos[nombreDispositivo] {
				estadosProcesos[elemento.PID] = "EXIT"
				log.Printf("%+v\n", estadosProcesos[elemento.PID])
			}
			delete(listaEsperaGenericos, nombreDispositivo)
			delete(puertosDispGenericos, nombreDispositivo)
			dispositivoGenerico.Unlock()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}
		log.Printf("respuesta del servidor: %s", resp.Status)
	}
}

type BodyRequestIO struct {
	Nombre    string `json:"nombre_dispositivo"`
	Puerto    int    `json:"puerto_dispositivo"`
	Categoria string `json:"categoria_dispositivo"`
}

func RegistrarIO(w http.ResponseWriter, r *http.Request) {
	var request BodyRequestIO

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return
	}

	switch request.Categoria {
	case "Generico":
		puertosDispGenericos[request.Nombre] = request.Puerto
	}

	log.Printf("%+v\n", puertosDispGenericos)
}
