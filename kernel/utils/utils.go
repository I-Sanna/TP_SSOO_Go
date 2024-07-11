package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kernel/globals"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PCB representa la estructura de control del proceso
type PCB struct {
	PID            int       `json:"pid"`
	ProgramCounter int       `json:"program_counter"`
	Quantum        int       `json:"quantum"`
	Estado         string    `json:"estado"`
	RegistrosCPU   Registros `json:"registros_cpu"`
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

// Semaforos
var planificadorCortoPlazo sync.Mutex
var planificadorLargoPlazo sync.Mutex
var dispositivoGenerico sync.Mutex
var dispositivoLectura sync.Mutex
var dispositivoEscritura sync.Mutex
var dispositivoLecturaFS sync.Mutex
var dispositivoEscrituraFS sync.Mutex
var mutexColaListos sync.Mutex
var mutexColaListosQuantum sync.Mutex
var mutexColaBlocked sync.Mutex
var mutexColaNuevos sync.Mutex
var mutexMapaEstados sync.Mutex
var semProcesosListos chan int
var semProcesoBloqueado chan int

// Variables
var killProcess bool
var contadorPID int
var planificando bool
var colaDeNuevos []PCB
var colaDeListos []PCB
var colaDeListosQuantum []PCB
var colaDeBlocked []PCB
var estadosProcesos map[int]string
var recursos map[string]int
var puertosDispGenericos map[string]int
var puertosDispSTDIN map[string]int
var puertosDispSTDOUT map[string]int
var puertosDispFSR map[string]int
var puertosDispFSW map[string]int
var listaEsperaRecursos map[string][]int
var listaEsperaGenericos map[string][]BodyIO
var listaEsperaSTDIN map[string][]BodySTD
var listaEsperaSTDOUT map[string][]BodySTD
var listaEsperaFSW map[string][]BodyFS
var listaEsperaFSR map[string][]BodyFS

type BodyIO struct {
	PID        int
	CantidadIO int
}

type BodySTD struct {
	PID       int
	Tamaño    int
	Direccion int `json:"direccion"`
}

type BodyFS struct {
	Dispositivo string `json:"dispositivo"`
	PID         int    `json:"pid"`
	Archivo     string `json:"archivo"`
	Tamaño      int    `json:"tamaño"`
	Direccion   int    `json:"direccion"`
	PtrArchivo  int    `json:"ptrarchivo"`
	Instruccion string `json:"instruccion"`
}

type BodyRequestFS struct {
	PID        int    `json:"pid"`
	Archivo    string `json:"archivo"`
	Tamaño     int    `json:"tamaño"`
	Direccion  int    `json:"direccion"`
	PtrArchivo int    `json:"ptrarchivo"`
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
	killProcess = false
	contadorPID = 0
	planificando = true
	semProcesoBloqueado = make(chan int, 1)
	semProcesosListos = make(chan int, globals.ClientConfig.Multiprogramming)
	estadosProcesos = make(map[int]string)
	recursos = make(map[string]int)
	puertosDispGenericos = make(map[string]int)
	puertosDispSTDIN = make(map[string]int)
	puertosDispSTDOUT = make(map[string]int)
	puertosDispFSR = make(map[string]int)
	puertosDispFSW = make(map[string]int)
	listaEsperaRecursos = make(map[string][]int)
	listaEsperaGenericos = make(map[string][]BodyIO)
	listaEsperaSTDIN = make(map[string][]BodySTD)
	listaEsperaSTDOUT = make(map[string][]BodySTD)
	listaEsperaFSW = make(map[string][]BodyFS)
	listaEsperaFSR = make(map[string][]BodyFS)
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
	case "VRR":
		go planificarVRR()
	}
}

func planificarFIFO() {
	for {
		<-semProcesosListos
		planificadorCortoPlazo.Lock()
		// Selecciona el primer proceso en la lista de procesos
		mutexColaListos.Lock()
		proceso := colaDeListos[0]
		mutexColaListos.Unlock()

		cambiarEstado(string(proceso.Estado), "EXEC", &proceso)

		// Enviarlo a ejecutar a la CPU
		mensaje := EnviarProcesoACPU(&proceso)

		planificadorCortoPlazo.Unlock()
		planificadorCortoPlazo.Lock() //Estos semaforos es por si se ejecuto "detenerPlanificacion"

		ManejarInterrupcion(mensaje, proceso, false)

		planificadorCortoPlazo.Unlock()
	}
}

// Función para planificar un proceso usando Round Robin (RR)
func planificarRR() {
	for {
		<-semProcesosListos

		planificadorCortoPlazo.Lock()

		// Selecciona el primer proceso en la lista de procesos
		mutexColaListos.Lock()
		proceso := colaDeListos[0]
		mutexColaListos.Unlock()

		// Cambia el estado del proceso a EXEC
		cambiarEstado(proceso.Estado, "EXEC", &proceso)

		// Enviar el proceso a la CPU para su ejecución

		go quantum(proceso.PID, proceso.Quantum)
		mensaje := EnviarProcesoACPU(&proceso)

		if mensaje == "error" {
			log.Printf("Error ejecutando el proceso %d", proceso.PID)
		}

		// Manejar la interrupción y la actualización de la cola de listos
		planificadorCortoPlazo.Unlock()
		planificadorCortoPlazo.Lock()
		ManejarInterrupcion(mensaje, proceso, false)

		planificadorCortoPlazo.Unlock()
	}
}

func planificarVRR() {
	for {
		<-semProcesosListos

		planificadorCortoPlazo.Lock()

		var proceso PCB
		var colaVRR bool = false

		// Selecciona el primer proceso en la lista de procesos
		mutexColaListos.Lock()
		mutexColaListosQuantum.Lock()
		if len(colaDeListosQuantum) != 0 {
			proceso = colaDeListosQuantum[0]
			colaVRR = true
		} else {
			proceso = colaDeListos[0]
		}
		mutexColaListos.Unlock()
		mutexColaListosQuantum.Unlock()

		// Cambia el estado del proceso a EXEC
		cambiarEstado(proceso.Estado, "EXEC", &proceso)

		// Enviar el proceso a la CPU para su ejecución
		go quantum(proceso.PID, proceso.Quantum)

		start := time.Now()
		mensaje := EnviarProcesoACPU(&proceso)
		elapsed := time.Since(start)

		if mensaje == "error" {
			log.Printf("Error ejecutando el proceso %d", proceso.PID)
		}

		//Le restamos lo que tardo en ejecutar o le reseteamos el quantum si fue desalojado por ello
		if int(int(elapsed)/1000000) < proceso.Quantum {
			proceso.Quantum = proceso.Quantum - int(int(elapsed)/1000000) //Time.Since me devuelve un tiempo con 6 decimales de precision
		} else {
			proceso.Quantum = globals.ClientConfig.Quantum
		}

		// Simula la ejecución durante el quantum

		// Manejar la interrupción y la actualización de la cola de listos
		planificadorCortoPlazo.Unlock()

		planificadorCortoPlazo.Lock()
		ManejarInterrupcion(mensaje, proceso, colaVRR)
		planificadorCortoPlazo.Unlock()
	}
}

func quantum(PID int, quantum int) {
	time.Sleep(time.Duration(quantum) * time.Millisecond)
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortCPU) + "/quantum/" + strconv.Itoa(PID)

	_, err := http.Get(url)

	if err != nil {
		log.Printf("error enviando interrupcion por quantum: %s", err.Error())
		return
	}
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
		planificadorLargoPlazo.Unlock()
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
		Estado:         "NEW",
		RegistrosCPU:   Registros{},
	}

	contadorPID++

	log.Printf("Se crea el proceso %d en NEW", nuevoProceso.PID)

	mutexColaNuevos.Lock()
	colaDeNuevos = append(colaDeNuevos, nuevoProceso)
	mutexColaNuevos.Unlock()
	mutexMapaEstados.Lock()
	estadosProcesos[nuevoProceso.PID] = "NEW"
	mutexMapaEstados.Unlock()
	if len(colaDeNuevos) == 1 {
		agregarProcesosALaColaListos()
	}

	var response = BodyRequestPid{PID: nuevoProceso.PID}

	respuesta, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	planificadorLargoPlazo.Unlock()
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func agregarProcesosALaColaListos() {
	mutexColaListos.Lock()
	mutexColaBlocked.Lock()
	mutexColaNuevos.Lock()

	for (len(colaDeListos)+len(colaDeBlocked)) < globals.ClientConfig.Multiprogramming && len(colaDeNuevos) > 0 {
		proceso := colaDeNuevos[0]
		cambiarEstado(string(proceso.Estado), "READY", &proceso)
		colaDeListos = append(colaDeListos, proceso)
		colaDeNuevos = colaDeNuevos[1:]
		semProcesosListos <- 0
	}

	mutexColaNuevos.Unlock()
	mutexColaBlocked.Unlock()

	var listaPID []int
	for _, proceso := range colaDeListos {
		listaPID = append(listaPID, proceso.PID)
	}

	mutexColaListos.Unlock()

	if len(listaPID) != 0 {
		log.Printf("Cola Ready colaDeListos: %v", listaPID)
	}
}

func rehabilitarProcesoBlocked(PID int) {
	mutexColaListos.Lock()
	mutexColaListosQuantum.Lock()
	mutexColaBlocked.Lock()
	planificadorLargoPlazo.Lock()

	var contador int = 0

	mutexMapaEstados.Lock()
	_, ok := estadosProcesos[PID]
	mutexMapaEstados.Unlock()
	if !ok {
		return
	}

	for _, proceso := range colaDeBlocked {
		if proceso.PID == PID {
			cambiarEstado(proceso.Estado, "READY", &proceso)
			colaDeBlocked = removerIndex(colaDeBlocked, contador)
			if proceso.Quantum == globals.ClientConfig.Quantum {
				colaDeListos = append(colaDeListos, proceso)
			} else {
				colaDeListosQuantum = append(colaDeListosQuantum, proceso)
			}
			semProcesosListos <- 0
			break
		} else {
			contador++
		}
	}

	planificadorLargoPlazo.Unlock()
	mutexColaBlocked.Unlock()
	mutexColaListosQuantum.Unlock()
	mutexColaListos.Unlock()
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

	return resultadoCPU.Mensaje
}

func ManejarInterrupcion(interrupcion string, proceso PCB, colaVRR bool) {
	motivo := strings.Split(strings.TrimRight(interrupcion, "\x00"), " ")

	mutexColaListos.Lock()
	if colaVRR {
		mutexColaListosQuantum.Lock()
		colaDeListosQuantum = colaDeListosQuantum[1:]
		mutexColaListosQuantum.Unlock()
	} else {
		colaDeListos = colaDeListos[1:]
	}

	if killProcess {
		mutexColaListos.Unlock()
		eliminarProceso(proceso, "Se solicito finalizar el proceso")

		if motivo[0] == "BLOCKED" && motivo[1] != "WAIT" && motivo[1] != "SIGNAL" {
			semProcesoBloqueado <- 0
		}

		killProcess = false

		return
	}

	switch motivo[0] {
	case "error":
		mutexColaListos.Unlock()
		eliminarProceso(proceso, "Ocurrio un error durante la ejecución")
	case "EXIT":
		mutexColaListos.Unlock()
		eliminarProceso(proceso, motivo[1])
	case "READY":
		cambiarEstado(string(proceso.Estado), "READY", &proceso)
		colaDeListos = append(colaDeListos, proceso)
		mutexColaListos.Unlock()

		mensaje := ""
		if len(motivo) > 1 {
			mensaje = motivo[1]
		}

		if mensaje == "QUANTUM" {
			log.Printf("PID: %d - Desalojado por fin de Quantum", proceso.PID)
		}

		semProcesosListos <- 0
	case "BLOCKED":
		cambiarEstado(string(proceso.Estado), "BLOCKED", &proceso)

		if motivo[1] == "WAIT" {
			mutexColaListos.Unlock()
			log.Printf("PID: %d - Recurso solicitado: %v", proceso.PID, motivo[2])
			resultado := WAIT(proceso.PID, motivo[2])

			if resultado == "OK" {
				cambiarEstado(string(proceso.Estado), "READY", &proceso)
				var listaTemp []PCB
				listaTemp = append(listaTemp, proceso)
				if proceso.Quantum == globals.ClientConfig.Quantum {
					mutexColaListosQuantum.Lock()
					listaTemp = append(listaTemp, colaDeListosQuantum...)
					colaDeListosQuantum = listaTemp
					mutexColaListosQuantum.Unlock()
				} else {
					mutexColaListos.Lock()
					listaTemp = append(listaTemp, colaDeListos...)
					colaDeListos = listaTemp
					mutexColaListos.Unlock()
				}
				log.Printf("PID: %d - Recurso asignado: %v", proceso.PID, motivo[2])
				semProcesosListos <- 0
				return
			} else if resultado == "NOT_FOUND" {
				eliminarProceso(proceso, "INVALID_RESOURCE")
				return
			}
		} else if motivo[1] == "SIGNAL" {
			log.Printf("PID: %d - Recurso liberado: %v", proceso.PID, motivo[2])
			mutexColaListos.Unlock()
			resultado := SIGNAL(motivo[2])
			if resultado == "NOT_FOUND" {
				eliminarProceso(proceso, "INVALID_RESOURCE")
				return
			}
			cambiarEstado(string(proceso.Estado), "READY", &proceso)
			var listaTemp []PCB
			listaTemp = append(listaTemp, proceso)
			if proceso.Quantum == globals.ClientConfig.Quantum {
				mutexColaListosQuantum.Lock()
				listaTemp = append(listaTemp, colaDeListosQuantum...)
				colaDeListosQuantum = listaTemp
				mutexColaListosQuantum.Unlock()
			} else {
				mutexColaListos.Lock()
				listaTemp = append(listaTemp, colaDeListos...)
				colaDeListos = listaTemp
				mutexColaListos.Unlock()
			}
			semProcesosListos <- 0
			return
		} else {
			mutexColaListos.Unlock()
			semProcesoBloqueado <- 0
		}

		mutexColaBlocked.Lock() // Lo devuelvo como estaba por la funcion Sleep que puede elminiar elementos de la lista
		colaDeBlocked = append(colaDeBlocked, proceso)
		mutexColaBlocked.Unlock()

		mensaje := ""
		if len(motivo) > 1 {
			mensaje = motivo[1]
		}

		log.Printf("PID: %d - Bloqueado por: %v", proceso.PID, mensaje)
	}
}

func eliminarProceso(proceso PCB, motivo string) {
	mutexMapaEstados.Lock()
	delete(estadosProcesos, proceso.PID)
	mutexMapaEstados.Unlock()

	log.Printf("Finaliza el proceso %d - Motivo: %v", proceso.PID, motivo)

	liberarRecursosProceso(proceso.PID)
	agregarProcesosALaColaListos()
}

func cambiarEstado(estadoAnterior string, estadoNuevo string, proceso *PCB) {
	proceso.Estado = estadoNuevo

	mutexMapaEstados.Lock()
	estadosProcesos[proceso.PID] = estadoNuevo
	mutexMapaEstados.Unlock()

	log.Printf("PID: %d - Estado Anterior: %v - Estado Actual: %v", proceso.PID, estadoAnterior, estadoNuevo)
}

func liberarRecursosProceso(pid int) {
	cliente := &http.Client{}
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/process/" + strconv.Itoa(pid)
	req, err := http.NewRequest("DELETE", url, nil)
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
}

func EstadoProceso(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	mutexMapaEstados.Lock()
	valor, ok := estadosProcesos[pidInt]
	mutexMapaEstados.Unlock()
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

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	mutexColaListos.Lock()
	mutexColaBlocked.Lock()
	mutexColaNuevos.Lock()
	mutexColaListosQuantum.Lock()
	planificadorLargoPlazo.Lock()
	mutexMapaEstados.Lock()
	estado, ok := estadosProcesos[pidInt]
	mutexMapaEstados.Unlock()
	if !ok {
		mutexColaListos.Unlock()
		mutexColaBlocked.Unlock()
		mutexColaNuevos.Unlock()
		mutexColaListosQuantum.Unlock()
		planificadorLargoPlazo.Unlock()

		respuesta, err := json.Marshal("No existe el proceso a eliminar")
		if err != nil {
			http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
	}

	switch estado {
	case "NEW":
		mutexColaListos.Unlock()
		mutexColaListosQuantum.Unlock()
		mutexColaBlocked.Unlock()
		removerProcesoDeLista(&colaDeNuevos, pidInt, "Se solicito finalizar el proceso")
		mutexColaNuevos.Unlock()
	case "READY":
		mutexColaBlocked.Unlock()
		mutexColaNuevos.Unlock()
		removerProcesoDeLista(&colaDeListos, pidInt, "Se solicito finalizar el proceso")
		removerProcesoDeLista(&colaDeListosQuantum, pidInt, "Se solicito finalizar el proceso")
		mutexColaListos.Unlock()
		mutexColaListosQuantum.Unlock()
		agregarProcesosALaColaListos()
	case "BLOCKED":
		mutexColaListos.Unlock()
		mutexColaListosQuantum.Unlock()
		mutexColaNuevos.Unlock()
		removerProcesoDeLista(&colaDeBlocked, pidInt, "Se solicito finalizar el proceso")
		mutexColaBlocked.Unlock()
		agregarProcesosALaColaListos()
	case "EXEC":
		mutexColaListos.Unlock()
		mutexColaListosQuantum.Unlock()
		mutexColaBlocked.Unlock()
		mutexColaNuevos.Unlock()
		killProcess = true
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortCPU) + "/desalojar/" + strconv.Itoa(pidInt)

		_, err := http.Get(url)

		if err != nil {
			log.Printf("error enviando interrupcion por quantum: %s", err.Error())
			return
		}
	}

	planificadorLargoPlazo.Unlock()

	respuesta, err := json.Marshal("Se elimino el proceso exitosamente")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func ListarProcesos(w http.ResponseWriter, r *http.Request) {
	var listaProcesos []BodyResponsePCB
	var proceso BodyResponsePCB

	mutexMapaEstados.Lock()
	for pid, estado := range estadosProcesos {
		proceso.PID = pid
		proceso.State = estado
		listaProcesos = append(listaProcesos, proceso)
	}
	mutexMapaEstados.Unlock()

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
	Tamaño      int    `json:"tamaño"`
	Direccion   int    `json:"direccion"`
	Instruccion string `json:"instruccion"`
	Archivo     string `json:"archivo"`
	PtrArchivo  int    `json:"ptrarchivo"`
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
		fmt.Println("Sleep por validar conexionIO")
		if ok && validarConexionIO(puerto) {
			go agregarElemAListaGenericos(request.Dispositivo, puerto, datosIO)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoGenerico.Unlock()
			return
		}
		dispositivoGenerico.Unlock()
	case "READ":
		fmt.Println("Entró en el case de read en kernel")
		var datosSTD BodySTD
		datosSTD.PID = request.PID
		datosSTD.Tamaño = request.Tamaño
		datosSTD.Direccion = request.Direccion

		fmt.Println("Datos en READ ", request.PID, request.Tamaño, request.Direccion)
		dispositivoLectura.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispSTDIN[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaSTDIN(request.Dispositivo, puerto, datosSTD)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoLectura.Unlock()
			return
		}
		dispositivoLectura.Unlock()
	case "WRITE":
		fmt.Println("Entró en el case de write en kernel")
		var datosSTD BodySTD
		datosSTD.PID = request.PID
		datosSTD.Tamaño = request.Tamaño
		datosSTD.Direccion = request.Direccion

		dispositivoEscritura.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispSTDOUT[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaSTDOUT(request.Dispositivo, puerto, datosSTD)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoEscritura.Unlock()
			return
		}
		dispositivoEscritura.Unlock()
	case "CREATE":
		fmt.Println("Entró en el case de write en kernel")
		var datosSTD BodySTD
		datosSTD.PID = request.PID
		datosSTD.Tamaño = request.Tamaño
		datosSTD.Direccion = request.Direccion

		dispositivoEscritura.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispSTDOUT[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaSTDOUT(request.Dispositivo, puerto, datosSTD)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoEscritura.Unlock()
			return
		}
		dispositivoEscritura.Unlock()
	case "DELETE":
		fmt.Println("Entró en el case de write en kernel")
		var datosSTD BodySTD
		datosSTD.PID = request.PID
		datosSTD.Tamaño = request.Tamaño
		datosSTD.Direccion = request.Direccion

		dispositivoEscritura.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispSTDOUT[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaSTDOUT(request.Dispositivo, puerto, datosSTD)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoEscritura.Unlock()
			return
		}
		dispositivoEscritura.Unlock()
	case "TRUNCATE":
		fmt.Println("Entró en el case de write en kernel")
		var datosSTD BodySTD
		datosSTD.PID = request.PID
		datosSTD.Tamaño = request.Tamaño
		datosSTD.Direccion = request.Direccion

		dispositivoEscritura.Lock() //Habria que hacer un semaforo por dispostivo
		puerto, ok := puertosDispSTDOUT[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaSTDOUT(request.Dispositivo, puerto, datosSTD)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoEscritura.Unlock()
			return
		}
		dispositivoEscritura.Unlock()
	case "FSWRITE":
		fmt.Println("Entró en el case de write en kernel")
		var datosFS BodyFS
		datosFS.PID = request.PID
		datosFS.Tamaño = request.Tamaño
		datosFS.Direccion = request.Direccion
		datosFS.Archivo = request.Archivo
		datosFS.PtrArchivo = request.PtrArchivo

		dispositivoEscrituraFS.Lock()
		puerto, ok := puertosDispFSW[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaFSW(request.Dispositivo, puerto, datosFS)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoEscrituraFS.Unlock()
			return
		}
		dispositivoEscrituraFS.Unlock()
	case "FSREAD":
		fmt.Println("Entró en el case de write en kernel")
		var datosFS BodyFS
		datosFS.PID = request.PID
		datosFS.Tamaño = request.Tamaño
		datosFS.Direccion = request.Direccion
		datosFS.Archivo = request.Archivo
		datosFS.PtrArchivo = request.PtrArchivo

		dispositivoLecturaFS.Lock()
		puerto, ok := puertosDispFSR[request.Dispositivo]

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaFSR(request.Dispositivo, puerto, datosFS)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoLecturaFS.Unlock()
			return
		}
		dispositivoLecturaFS.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func agregarElemAListaGenericos(dispositivo string, puerto int, datosIO BodyIO) {
	<-semProcesoBloqueado
	listaEsperaGenericos[dispositivo] = append(listaEsperaGenericos[dispositivo], datosIO)
	if len(listaEsperaGenericos[dispositivo]) == 1 {
		go Sleep(dispositivo, puerto)
	}
}

func agregarElemAListaSTDIN(dispositivo string, puerto int, datosSTD BodySTD) {
	<-semProcesoBloqueado
	listaEsperaSTDIN[dispositivo] = append(listaEsperaSTDIN[dispositivo], datosSTD)
	if len(listaEsperaSTDIN[dispositivo]) == 1 {
		go Read(dispositivo, puerto)
	}
}

func agregarElemAListaSTDOUT(dispositivo string, puerto int, datosSTD BodySTD) {
	<-semProcesoBloqueado
	listaEsperaSTDOUT[dispositivo] = append(listaEsperaSTDOUT[dispositivo], datosSTD)
	if len(listaEsperaSTDOUT[dispositivo]) == 1 {
		go Write(dispositivo, puerto)
	}
}

func agregarElemAListaFSW(dispositivo string, puerto int, datosFS BodyFS) {
	<-semProcesoBloqueado
	listaEsperaFSW[dispositivo] = append(listaEsperaFSW[dispositivo], datosFS)
	if len(listaEsperaFSW[dispositivo]) == 1 {
		go WriteFS(dispositivo, puerto)
	}
}

func agregarElemAListaFSR(dispositivo string, puerto int, datosFS BodyFS) {
	<-semProcesoBloqueado
	listaEsperaFSR[dispositivo] = append(listaEsperaFSR[dispositivo], datosFS)
	if len(listaEsperaFSR[dispositivo]) == 1 {
		go ReadFS(dispositivo, puerto)
	}
}

func validarConexionIO(puerto int) bool {
	url := "http://localhost:" + strconv.Itoa(puerto) + "/validar"
	_, err := http.Get(url)
	return err == nil
}

func Sleep(nombreDispositivo string, puerto int) {
	dispositivoGenerico.Lock()
	for len(listaEsperaGenericos[nombreDispositivo]) > 0 {
		proceso := listaEsperaGenericos[nombreDispositivo][0]
		dispositivoGenerico.Unlock()

		mutexMapaEstados.Lock()
		_, ok := estadosProcesos[proceso.PID]
		mutexMapaEstados.Unlock()
		if !ok {
			dispositivoGenerico.Lock()
			listaEsperaGenericos[nombreDispositivo] = listaEsperaGenericos[nombreDispositivo][1:]
			continue
		}

		url := "http://localhost:" + strconv.Itoa(puerto) + "/sleep/" + strconv.Itoa(proceso.CantidadIO) + "/" + strconv.Itoa(proceso.PID)

		resp, err := http.Get(url)
		if err != nil {

			log.Printf("error enviando: %s", err.Error())
			dispositivoGenerico.Lock()

			for _, elemento := range listaEsperaGenericos[nombreDispositivo] {
				mutexColaBlocked.Lock()
				removerProcesoDeLista(&colaDeBlocked, elemento.PID, "LOST_CONNECTION_IO")
				mutexColaBlocked.Unlock()
			}

			delete(listaEsperaGenericos, nombreDispositivo)
			delete(puertosDispGenericos, nombreDispositivo)
			dispositivoGenerico.Unlock()
			agregarProcesosALaColaListos()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}

		dispositivoGenerico.Lock()
		listaEsperaGenericos[nombreDispositivo] = listaEsperaGenericos[nombreDispositivo][1:]
		dispositivoGenerico.Unlock()

		rehabilitarProcesoBlocked(proceso.PID)

		dispositivoGenerico.Lock()
	}
	dispositivoGenerico.Unlock()
}

func Read(nombreDispositivo string, puerto int) {
	dispositivoLectura.Lock()
	for len(listaEsperaSTDIN[nombreDispositivo]) > 0 {
		proceso := listaEsperaSTDIN[nombreDispositivo][0]
		dispositivoLectura.Unlock()

		mutexMapaEstados.Lock()
		_, ok := estadosProcesos[proceso.PID]
		mutexMapaEstados.Unlock()
		if !ok {
			dispositivoLectura.Lock()
			listaEsperaSTDIN[nombreDispositivo] = listaEsperaSTDIN[nombreDispositivo][1:]
			continue
		}

		url := "http://localhost:" + strconv.Itoa(puerto) + "/read/" + strconv.Itoa(proceso.PID) + "/" + strconv.Itoa(proceso.Tamaño) + "/" + strconv.Itoa(proceso.Direccion)

		resp, err := http.Get(url)
		if err != nil {

			log.Printf("error enviando: %s", err.Error())
			dispositivoLectura.Lock()

			for _, elemento := range listaEsperaSTDIN[nombreDispositivo] {
				mutexColaBlocked.Lock()
				removerProcesoDeLista(&colaDeBlocked, elemento.PID, "LOST_CONNECTION_IO")
				mutexColaBlocked.Unlock()
			}

			delete(listaEsperaSTDIN, nombreDispositivo)
			delete(puertosDispSTDIN, nombreDispositivo)
			dispositivoLectura.Unlock()
			agregarProcesosALaColaListos()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}

		dispositivoLectura.Lock()
		listaEsperaSTDIN[nombreDispositivo] = listaEsperaSTDIN[nombreDispositivo][1:]
		dispositivoLectura.Unlock()

		rehabilitarProcesoBlocked(proceso.PID)

		dispositivoLectura.Lock()
	}
	dispositivoLectura.Unlock()
}

func Write(nombreDispositivo string, puerto int) {
	dispositivoEscritura.Lock()
	for len(listaEsperaSTDOUT[nombreDispositivo]) > 0 {
		proceso := listaEsperaSTDOUT[nombreDispositivo][0]
		dispositivoEscritura.Unlock()

		mutexMapaEstados.Lock()
		_, ok := estadosProcesos[proceso.PID]
		mutexMapaEstados.Unlock()
		if !ok {
			dispositivoEscritura.Lock()
			listaEsperaSTDOUT[nombreDispositivo] = listaEsperaSTDIN[nombreDispositivo][1:]
			continue
		}

		url := "http://localhost:" + strconv.Itoa(puerto) + "/write/" + strconv.Itoa(proceso.PID) + "/" + strconv.Itoa(proceso.Tamaño) + "/" + strconv.Itoa(proceso.Direccion)

		resp, err := http.Get(url)
		if err != nil {

			log.Printf("error enviando: %s", err.Error())
			dispositivoEscritura.Lock()

			for _, elemento := range listaEsperaSTDOUT[nombreDispositivo] {
				mutexColaBlocked.Lock()
				removerProcesoDeLista(&colaDeBlocked, elemento.PID, "LOST_CONNECTION_IO")
				mutexColaBlocked.Unlock()
			}

			delete(listaEsperaSTDOUT, nombreDispositivo)
			delete(puertosDispSTDOUT, nombreDispositivo)
			dispositivoEscritura.Unlock()
			agregarProcesosALaColaListos()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}

		dispositivoEscritura.Lock()
		listaEsperaSTDOUT[nombreDispositivo] = listaEsperaSTDOUT[nombreDispositivo][1:]
		dispositivoEscritura.Unlock()

		rehabilitarProcesoBlocked(proceso.PID)

		dispositivoEscritura.Lock()
	}
	dispositivoEscritura.Unlock()
}

func WriteFS(nombreDispositivo string, puerto int) {
	log.Printf("\n Entro al write fs (kernel)")
	dispositivoEscrituraFS.Lock()
	for len(listaEsperaFSW[nombreDispositivo]) > 0 {
		proceso := listaEsperaFSW[nombreDispositivo][0]
		dispositivoEscrituraFS.Unlock()

		mutexMapaEstados.Lock()
		_, ok := estadosProcesos[proceso.PID]
		mutexMapaEstados.Unlock()
		if !ok {
			dispositivoEscritura.Lock()
			listaEsperaFSW[nombreDispositivo] = listaEsperaFSW[nombreDispositivo][1:]
			continue
		}

		request := BodyRequestFS{
			PID:        proceso.PID,
			Archivo:    proceso.Archivo,
			Tamaño:     proceso.Tamaño,
			Direccion:  proceso.Direccion,
			PtrArchivo: proceso.PtrArchivo,
		}

		requestBody, err := json.Marshal(request)
		if err != nil {
			log.Printf("error al codificar la solicitud: %s", err.Error())
		}

		url := fmt.Sprintf("http://localhost:%d/fs/write", globals.ClientConfig.PortIO)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))

		/*if err != nil {
			return fmt.Errorf("error al enviar la solicitud al módulo de Interfaz de I/O: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error del módulo de Interfaz de I/O: %s", resp.Status)
		}*/

		if err != nil {

			log.Printf("error enviando: %s", err.Error())
			dispositivoEscrituraFS.Lock()

			for _, elemento := range listaEsperaFSW[nombreDispositivo] {
				mutexColaBlocked.Lock()
				removerProcesoDeLista(&colaDeBlocked, elemento.PID, "LOST_CONNECTION_IO")
				mutexColaBlocked.Unlock()
			}

			delete(listaEsperaFSW, nombreDispositivo)
			delete(puertosDispFSW, nombreDispositivo)
			dispositivoEscrituraFS.Unlock()
			agregarProcesosALaColaListos()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}

		dispositivoEscrituraFS.Lock()
		listaEsperaFSW[nombreDispositivo] = listaEsperaFSW[nombreDispositivo][1:]
		dispositivoEscrituraFS.Unlock()

		rehabilitarProcesoBlocked(proceso.PID)

		dispositivoEscrituraFS.Lock()
	}
	dispositivoEscrituraFS.Unlock()
}

func ReadFS(nombreDispositivo string, puerto int) {
	log.Printf("\n Entro al read fs (kernel)")
	dispositivoEscrituraFS.Lock()
	for len(listaEsperaFSW[nombreDispositivo]) > 0 {
		proceso := listaEsperaFSW[nombreDispositivo][0]
		dispositivoEscrituraFS.Unlock()

		mutexMapaEstados.Lock()
		_, ok := estadosProcesos[proceso.PID]
		mutexMapaEstados.Unlock()
		if !ok {
			dispositivoEscritura.Lock()
			listaEsperaFSW[nombreDispositivo] = listaEsperaFSW[nombreDispositivo][1:]
			continue
		}

		request := BodyRequestFS{
			PID:        proceso.PID,
			Archivo:    proceso.Archivo,
			Tamaño:     proceso.Tamaño,
			Direccion:  proceso.Direccion,
			PtrArchivo: proceso.PtrArchivo,
		}

		requestBody, err := json.Marshal(request)
		if err != nil {
			log.Printf("error al codificar la solicitud: %s", err)
		}

		url := fmt.Sprintf("http://localhost:%d/fs/read", globals.ClientConfig.PortIO)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))

		/*if err != nil {
			return fmt.Errorf("error al enviar la solicitud al módulo de Interfaz de I/O: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error del módulo de Interfaz de I/O: %s", resp.Status)
		}*/

		if err != nil {

			log.Printf("error enviando: %s", err.Error())
			dispositivoEscrituraFS.Lock()

			for _, elemento := range listaEsperaFSW[nombreDispositivo] {
				mutexColaBlocked.Lock()
				removerProcesoDeLista(&colaDeBlocked, elemento.PID, "LOST_CONNECTION_IO")
				mutexColaBlocked.Unlock()
			}

			delete(listaEsperaFSW, nombreDispositivo)
			delete(puertosDispFSW, nombreDispositivo)
			dispositivoEscrituraFS.Unlock()
			agregarProcesosALaColaListos()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("error en la respuesta de la consulta: %s", resp.Status)
			return
		}

		dispositivoEscrituraFS.Lock()
		listaEsperaFSW[nombreDispositivo] = listaEsperaFSW[nombreDispositivo][1:]
		dispositivoEscrituraFS.Unlock()

		rehabilitarProcesoBlocked(proceso.PID)

		dispositivoEscrituraFS.Lock()
	}
	dispositivoEscrituraFS.Unlock()
}

func removerProcesoDeLista(lista *[]PCB, PID int, motivo string) {
	var contador int = 0
	for _, elemento := range *lista {
		if elemento.PID == PID {
			*lista = removerIndex(*lista, contador)
			liberarRecursosProceso(PID)
			log.Printf("Finaliza el proceso %d - Motivo: %v", PID, motivo)
			break
		} else {
			contador++
		}
	}
}

func removerIndex(s []PCB, index int) []PCB {
	ret := make([]PCB, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
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
	case "STDIN":
		puertosDispSTDIN[request.Nombre] = request.Puerto
	case "STDOUT":
		puertosDispSTDOUT[request.Nombre] = request.Puerto
	case "DIALFS":
		puertosDispSTDOUT[request.Nombre] = request.Puerto
	}
}

type BodyRRSS struct {
	PID     int    `json:"pid"`
	Recurso string `json:"recurso"`
}

func WAIT(pid int, recurso string) string {

	// Buscar el recurso
	for i, r := range globals.ClientConfig.Resources {
		if r == recurso {
			globals.ClientConfig.Resource_instances[i]--
			if globals.ClientConfig.Resource_instances[i] < 0 {
				// Agregar el proceso a la lista de espera de recursos
				if listaEsperaRecursos == nil {
					listaEsperaRecursos = make(map[string][]int)
				}
				listaEsperaRecursos[recurso] = append(listaEsperaRecursos[recurso], pid)
				return "BLOCKED"
			}
			return "OK"
		}
	}
	return "NOT_FOUND"
}

func SIGNAL(recurso string) string {
	// Buscar el recurso
	for i, r := range globals.ClientConfig.Resources {
		if r == recurso {
			globals.ClientConfig.Resource_instances[i]++
			if len(listaEsperaRecursos[recurso]) != 0 {
				rehabilitarProcesoBlocked(listaEsperaRecursos[recurso][0])
				listaEsperaRecursos[recurso] = listaEsperaRecursos[recurso][1:]
			}
			return "OK"
		}
	}
	return "NOT_FOUND"
}

type FileRequest struct {
	NombreArchivo string `json:"nombreArchivo"`
}

type FileResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HandleCreateFileRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("ENTRO AL HANDLE_CREATE_FILE_REQUEST")
	var request FileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Error al decodificar la solicitud: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if request.NombreArchivo == "" {
		log.Printf("Parámetros inválidos: nombreArchivo está vacío")
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	// Enviar la solicitud al módulo de Interfaz de I/O
	err := sendCreateFileRequestToIO(request.NombreArchivo)
	var response FileResponse
	if err != nil {
		response = FileResponse{
			Status:  "Error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response = FileResponse{
			Status:  "OK",
			Message: "Archivo creado correctamente",
		}
		w.WriteHeader(http.StatusOK)
	}

	// Responder a la CPU
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error al codificar la respuesta: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func sendCreateFileRequestToIO(nombreArchivo string) error {
	log.Printf("SEND_CREATE_FILE_REQUEST_TO_IO")
	// Crear la solicitud
	request := FileRequest{
		NombreArchivo: nombreArchivo,
	}

	// Codificar la solicitud a JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error al codificar la solicitud: %v", err)
		return fmt.Errorf("error al codificar la solicitud: %w", err)
	}

	// Enviar la solicitud al módulo de Interfaz de I/O
	url := fmt.Sprintf("http://localhost:%d/fs/create", globals.ClientConfig.PortIO)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Error al enviar la solicitud al módulo de Interfaz de I/O: %v", err)
		return fmt.Errorf("error al enviar la solicitud al módulo de Interfaz de I/O: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error del módulo de Interfaz de I/O: %s", resp.Status)
		return fmt.Errorf("error del módulo de Interfaz de I/O: %s", resp.Status)
	}

	return nil
}

func HandleDeleteFileRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("ENTRO AL HANDLE_DELETE_FILE_REQUEST")
	var request FileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Error al decodificar la solicitud: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if request.NombreArchivo == "" {
		log.Printf("Parámetros inválidos: nombreArchivo está vacío")
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	// Enviar la solicitud al módulo de Interfaz de I/O
	err := sendDeleteFileRequestToIO(request.NombreArchivo)
	var response FileResponse
	if err != nil {
		response = FileResponse{
			Status:  "Error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response = FileResponse{
			Status:  "OK",
			Message: "Archivo eliminado correctamente",
		}
		w.WriteHeader(http.StatusOK)
	}

	// Responder a la CPU
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error al codificar la respuesta: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func sendDeleteFileRequestToIO(nombreArchivo string) error {
	log.Printf("SEND_DELETE_FILE_REQUEST_TO_IO")
	// Crear la solicitud
	request := FileRequest{
		NombreArchivo: nombreArchivo,
	}

	// Codificar la solicitud a JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error al codificar la solicitud: %v", err)
		return fmt.Errorf("error al codificar la solicitud: %w", err)
	}

	// Enviar la solicitud al módulo de Interfaz de I/O
	url := fmt.Sprintf("http://localhost:%d/fs/delete", globals.ClientConfig.PortIO)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Error al enviar la solicitud al módulo de Interfaz de I/O: %v", err)
		return fmt.Errorf("error al enviar la solicitud al módulo de Interfaz de I/O: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error del módulo de Interfaz de I/O: %s", resp.Status)
		return fmt.Errorf("error del módulo de Interfaz de I/O: %s", resp.Status)
	}

	return nil
}
