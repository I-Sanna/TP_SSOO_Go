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
var mutexColaListos sync.Mutex
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
var colaDeBlocked []PCB
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

		ManejarInterrupcion(mensaje, proceso)

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

		go quantum(proceso.PID)
		mensaje := EnviarProcesoACPU(&proceso)

		if mensaje == "error" {
			log.Printf("Error ejecutando el proceso %d", proceso.PID)
		}

		// Manejar la interrupción y la actualización de la cola de listos
		planificadorCortoPlazo.Unlock()
		planificadorCortoPlazo.Lock()
		ManejarInterrupcion(mensaje, proceso)

		planificadorCortoPlazo.Unlock()
	}
}

/*
func planificarVRR() {
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
		start := time.Now()
		mensaje := EnviarProcesoACPU(&proceso)
		elapsed := time.Since(start)
		if mensaje != "error" {
			log.Printf("Proceso %d ejecutando con mensaje: %s", proceso.PID, mensaje)
		} else {
			log.Printf("Error ejecutando el proceso %d", proceso.PID)
		}

		proceso.Quantum = proceso.Quantum - int(elapsed) //Le restamos lo que tardo en ejecutar o le reseteamos el quantum si fue desalojado por ello
		// Simula la ejecución durante el quantum
		time.Sleep(time.Duration(globals.ClientConfig.Quantum) * time.Millisecond)

		if proceso.Estado != "EXIT" {
			log.Printf("\nfin de quantum\n")
		}

		// Manejar la interrupción y la actualización de la cola de listos
		planificadorCortoPlazo.Unlock()
		planificadorCortoPlazo.Lock()
		ManejarInterrupcion(mensaje, proceso)

		planificadorCortoPlazo.Unlock()
	}
}
*/

func quantum(PID int) {
	time.Sleep(time.Duration(globals.ClientConfig.Quantum) * time.Millisecond)
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
			colaDeListos = append(colaDeListos, proceso)
			semProcesosListos <- 0
			break
		} else {
			contador++
		}
	}

	planificadorLargoPlazo.Unlock()
	mutexColaBlocked.Unlock()
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

func ManejarInterrupcion(interrupcion string, proceso PCB) {
	motivo := strings.Split(strings.TrimRight(interrupcion, "\x00"), " ")

	mutexColaListos.Lock()
	colaDeListos = colaDeListos[1:]

	if killProcess {
		mutexColaListos.Unlock()

		mutexMapaEstados.Lock()
		delete(estadosProcesos, proceso.PID)
		mutexMapaEstados.Unlock()

		if motivo[0] == "BLOCKED" {
			semProcesoBloqueado <- 0
		}

		log.Printf("Finaliza el proceso %d - Motivo: %v", proceso.PID, "Se solicito finalizar el proceso")

		agregarProcesosALaColaListos()

		return
	}

	switch motivo[0] {
	case "EXIT":
		mutexColaListos.Unlock()

		mutexMapaEstados.Lock()
		delete(estadosProcesos, proceso.PID)
		mutexMapaEstados.Unlock()

		mensaje := ""
		if len(motivo) > 1 {
			mensaje = motivo[1]
		}

		log.Printf("Finaliza el proceso %d - Motivo: %v", proceso.PID, mensaje)

		agregarProcesosALaColaListos()
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
		mutexColaListos.Unlock()

		cambiarEstado(string(proceso.Estado), "BLOCKED", &proceso)

		mutexColaBlocked.Lock() // Lo devuelvo como estaba por la funcion Sleep que puede elminiar elementos de la lista
		colaDeBlocked = append(colaDeBlocked, proceso)
		mutexColaBlocked.Unlock()

		mensaje := ""
		if len(motivo) > 1 {
			mensaje = motivo[1]
		}

		log.Printf("PID: %d - Bloqueado por: %v", proceso.PID, mensaje)
		semProcesoBloqueado <- 0
	}
}

func cambiarEstado(estadoAnterior string, estadoNuevo string, proceso *PCB) {
	proceso.Estado = estadoNuevo

	mutexMapaEstados.Lock()
	estadosProcesos[proceso.PID] = estadoNuevo
	mutexMapaEstados.Unlock()

	log.Printf("PID: %d - Estado Anterior: %v - Estado Actual: %v", proceso.PID, estadoAnterior, estadoNuevo)
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
	planificadorLargoPlazo.Lock()
	mutexMapaEstados.Lock()
	estado, ok := estadosProcesos[pidInt]
	mutexMapaEstados.Unlock()
	if !ok {
		mutexColaListos.Unlock()
		mutexColaBlocked.Unlock()
		mutexColaNuevos.Unlock()
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
		mutexColaBlocked.Unlock()
		removerProcesoDeLista(&colaDeNuevos, pidInt, "Se solicito finalizar el proceso")
		mutexColaNuevos.Unlock()
	case "READY":
		mutexColaBlocked.Unlock()
		mutexColaNuevos.Unlock()
		removerProcesoDeLista(&colaDeListos, pidInt, "Se solicito finalizar el proceso")
		mutexColaListos.Unlock()
		agregarProcesosALaColaListos()
	case "BLOCKED":
		mutexColaListos.Unlock()
		mutexColaNuevos.Unlock()
		removerProcesoDeLista(&colaDeBlocked, pidInt, "Se solicito finalizar el proceso")
		mutexColaBlocked.Unlock()
		agregarProcesosALaColaListos()
	case "EXEC":
		mutexColaListos.Unlock()
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

		if ok && validarConexionIO(puerto) {
			go agregarElemAListaGenericos(request.Dispositivo, puerto, datosIO)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			dispositivoGenerico.Unlock()
			return
		}
		dispositivoGenerico.Unlock()
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

func removerProcesoDeLista(lista *[]PCB, PID int, motivo string) {
	var contador int = 0
	for _, elemento := range *lista {
		if elemento.PID == PID {
			*lista = removerIndex(*lista, contador)
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
	}
}
