package utils

import (
	"bytes"
	"cpu/globals"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type TLBEntry struct {
	PID    int
	Pagina int
	Marco  int
}

type TLB struct {
	Entradas []TLBEntry
}

var TLBCPU *TLB

// InicializarTLB inicializa una TLB con el número de entradas especificado.
func InicializarTLB(numEntradas int) *TLB {
	return &TLB{
		Entradas: make([]TLBEntry, numEntradas),
	}
}

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

var mutexMensaje sync.Mutex
var mutex sync.Mutex
var tlbLock sync.Mutex

var procesoActual PCB
var interrupcion bool

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
	logFile, err := os.OpenFile("logs/cpu.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

type BodyReqExec struct {
	Pcb     PCB    `json:"pcb"`
	Mensaje string `json:"mensaje"`
}

var resultadoEjecucion BodyReqExec

func RecibirProceso(w http.ResponseWriter, r *http.Request) {
	var paquete PCB
	interrupcion = false
	resultadoEjecucion.Mensaje = ""

	err := json.NewDecoder(r.Body).Decode(&paquete)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		return
	}

	procesoActual = paquete

	//Ejecutar las instrucciones

	for !interrupcion {
		log.Printf("PID: %d - FETCH - Program Counter: %d", procesoActual.PID, procesoActual.ProgramCounter)
		instruccion := SolicitarInstruccion(procesoActual.PID, procesoActual.ProgramCounter)
		decoYExecInstru(instruccion)
		procesoActual.ProgramCounter++
	}

	resultadoEjecucion.Pcb = procesoActual

	respuesta, err := json.Marshal(resultadoEjecucion)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func SET(nombreRegistro string, valor int) {
	if strlen(nombreRegistro) == 2 && strings.Contains(nombreRegistro, "X") {
		var registro *uint8 = ObtenerRegistro8Bits(nombreRegistro)
		*registro = uint8(valor)
	} else {
		var registro *uint32 = ObtenerRegistro32Bits(nombreRegistro)
		*registro = uint32(valor)
	}
}

func SUM(nombreRegistroDestino string, nombreRegistroOrigen string) {
	if strlen(nombreRegistroDestino) == 2 && strlen(nombreRegistroOrigen) == 2 && strings.Contains(nombreRegistroDestino, "X") && strings.Contains(nombreRegistroOrigen, "X") {
		var registroDestino *uint8 = ObtenerRegistro8Bits(nombreRegistroDestino)
		var registroOrigen *uint8 = ObtenerRegistro8Bits(nombreRegistroOrigen)
		*registroDestino = *registroDestino + *registroOrigen
	} else {
		var registroDestino *uint32 = ObtenerRegistro32Bits(nombreRegistroDestino)
		var registroOrigen *uint32 = ObtenerRegistro32Bits(nombreRegistroOrigen)
		*registroDestino = *registroDestino + *registroOrigen
	}
}

func SUB(nombreRegistroDestino string, nombreRegistroOrigen string) {
	if strlen(nombreRegistroDestino) == 2 && strlen(nombreRegistroOrigen) == 2 && strings.Contains(nombreRegistroDestino, "X") && strings.Contains(nombreRegistroOrigen, "X") {
		var registroDestino *uint8 = ObtenerRegistro8Bits(nombreRegistroDestino)
		var registroOrigen *uint8 = ObtenerRegistro8Bits(nombreRegistroOrigen)
		*registroDestino = *registroDestino - *registroOrigen
	} else {
		var registroDestino *uint32 = ObtenerRegistro32Bits(nombreRegistroDestino)
		var registroOrigen *uint32 = ObtenerRegistro32Bits(nombreRegistroOrigen)
		*registroDestino = *registroDestino - *registroOrigen
	}
}

func JNZ(nombreRegistro string, valor int) {
	if strlen(nombreRegistro) == 2 && strings.Contains(nombreRegistro, "X") {
		var registro *uint8 = ObtenerRegistro8Bits(nombreRegistro)
		if *registro != uint8(0) {
			procesoActual.ProgramCounter = valor - 2
		}
	} else {
		var registro *uint32 = ObtenerRegistro32Bits(nombreRegistro)
		if *registro != uint32(0) {
			procesoActual.ProgramCounter = valor - 2
		}
	}
}

type BodyEscritura struct {
	PID       int    `json:"pid"`
	Info      string `json:"info"`
	Tamaño    int    `json:"tamaño"`
	Direccion int    `json:"direccion"`
}

func leerDeMemoria(pid int, direccion int, tamaño int) (string, error) {
	request := BodyEscritura{
		PID:       pid,
		Direccion: direccion,
		Tamaño:    tamaño,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error al codificar solicitud: %v", err)
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/leer"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("error al enviar solicitud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error en la respuesta de la consulta: %v", resp.Status)
	}

	var resultado string
	err = json.NewDecoder(resp.Body).Decode(&resultado)
	if err != nil {
		return "", fmt.Errorf("error al decodificar respuesta: %v", err)
	}

	return resultado, nil
}

func escribirEnMemoria(pid int, direccionFisica int, datos string, tamaño int) error {
	body := BodyEscritura{
		PID:       pid,
		Info:      datos,
		Tamaño:    tamaño,
		Direccion: direccionFisica,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error codificando la solicitud: %w", err)
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/escribir"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("error al enviar la solicitud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en la respuesta del servidor: %s", resp.Status)
	}

	return nil
}

// MOV_IN (Registro Datos, Registro Dirección)
func MOV_IN(registroDatos, registroDireccion string) {
	regDatos := ObtenerRegistro32Bits(registroDatos)
	regDireccion := ObtenerRegistro32Bits(registroDireccion)

	direccionesFisicas, err := mmu(procesoActual.PID, *regDireccion)
	if err != nil {
		log.Printf("Error al traducir dirección: %s", err.Error())
		return
	}

	// Leer de memoria considerando la posibilidad de direcciones físicas múltiples
	var valor string
	for _, direccionFisica := range direccionesFisicas {
		parte, err := leerDeMemoria(procesoActual.PID, direccionFisica, 4) // 4 porque el registro es de 32 bits (a chequear)
		if err != nil {
			log.Printf("Error al leer de memoria: %s", err.Error())
			return
		}
		valor += parte
	}

	// Convertir valor a uint32 antes de asignarlo a regDatos
	valorUint32, err := strconv.ParseUint(valor, 10, 32)
	if err != nil {
		log.Printf("Error al convertir valor a uint32: %s", err.Error())
		return
	}
	*regDatos = uint32(valorUint32)
}

// MOV_OUT (Registro Dirección, Registro Datos)
func MOV_OUT(registroDireccion, registroDatos string) {
	regDireccion := ObtenerRegistro32Bits(registroDireccion)
	regDatos := ObtenerRegistro32Bits(registroDatos)

	direccionesFisicas, err := mmu(procesoActual.PID, *regDireccion)
	if err != nil {
		log.Printf("Error al traducir dirección: %s", err.Error())
		return
	}

	datos := strconv.Itoa(int(*regDatos))
	// Escribir en memoria considerando la posibilidad de direcciones físicas múltiples
	for _, direccionFisica := range direccionesFisicas {
		err = escribirEnMemoria(procesoActual.PID, direccionFisica, datos, 4)
		if err != nil {
			log.Printf("Error al escribir en memoria: %s", err.Error())
			return
		}
	}
}

func mmu(pid int, direccionLogica uint32) ([]int, error) {

	// Obtener el tamaño de página
	pageSize, err := ObtenerPageSize()
	if err != nil {
		return nil, fmt.Errorf("error al obtener el tamaño de página: %w", err)
	}

	numeroPagina := int(direccionLogica / uint32(pageSize))
	desplazamiento := int(direccionLogica - uint32(numeroPagina)*uint32(pageSize))

	// Consultar TLB
	marcoTLB, err := buscarEnTLB(pid, numeroPagina, TLBCPU)
	if err == nil {
		direccionFisica := marcoTLB*pageSize + desplazamiento
		return []int{direccionFisica}, nil // TLB Hit
	}

	// Consultar tabla de páginas en la memoria principal
	marco, err := buscarEnMemoria(pid, numeroPagina)
	if err != nil {
		return nil, fmt.Errorf("error al buscar en memoria: %w", err)
	}

	// Actualizar TLB
	actualizarTLB(pid, int(direccionLogica), marco, TLBCPU)

	// Calcular dirección física
	direccionFisica := marco*pageSize + desplazamiento
	direccionesFisicas := []int{direccionFisica}

	// Verificar si el acceso cruza los límites de la página
	if desplazamiento+4 > pageSize { // 4 bytes (32 bits)
		siguientePagina := numeroPagina + 1
		siguienteMarco, err := buscarEnMemoria(pid, siguientePagina)
		if err != nil {
			return nil, fmt.Errorf("error al buscar en memoria para la siguiente página: %w", err)
		}
		actualizarTLB(pid, siguientePagina, siguienteMarco, TLBCPU)
		direccionesFisicas = append(direccionesFisicas, siguienteMarco*pageSize)
	}

	return direccionesFisicas, nil
}

func buscarEnTLB(pid, numeroPagina int, tlb *TLB) (int, error) {
	for _, entry := range tlb.Entradas {
		if entry.PID == pid && entry.Pagina == numeroPagina {
			return entry.Marco, nil // TLB Hit
		}
	}
	return 0, fmt.Errorf("TLB Miss")
}

func actualizarTLB(pid, direccionLogica, marco int, tlb *TLB) {
	// Reemplazar entrada usando FIFO o LRU según el algoritmo
	// FALTA IMPLEMENTAR LRU
	pageSize, err := ObtenerPageSize()
	if err != nil {
		return
	}

	nuevaEntrada := TLBEntry{
		PID:    pid,
		Pagina: direccionLogica / pageSize,
		Marco:  marco,
	}
	tlb.Entradas = append(tlb.Entradas[1:], nuevaEntrada)
}

func buscarEnMemoria(pid int, numeroPagina int) (int, error) {

	url := fmt.Sprintf("http://localhost:%d/pagina/%d/%d", globals.ClientConfig.PortMemory, pid, numeroPagina)

	response, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, err // Manejar el caso de respuesta no exitosa
	}

	var marco int
	err = json.NewDecoder(response.Body).Decode(&marco)
	if err != nil {
		return 0, err
	}

	return marco, nil
}

func ObtenerPageSize() (int, error) {
	response, err := http.Get("http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/page_size")
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, err
	}
	var pageSize int
	err = json.NewDecoder(response.Body).Decode(&pageSize)
	if err != nil {
		return 0, err
	}

	return pageSize, nil
}

func strlen(str string) int {
	return len([]rune(str))
}

func ObtenerRegistro8Bits(nombre string) *uint8 {
	switch nombre {
	case "AX":
		return &procesoActual.RegistrosCPU.AX
	case "BX":
		return &procesoActual.RegistrosCPU.BX
	case "CX":
		return &procesoActual.RegistrosCPU.CX
	case "DX":
		return &procesoActual.RegistrosCPU.DX
	}
	otherwise := uint8(0)
	return &otherwise
}

func ObtenerRegistro32Bits(nombre string) *uint32 {
	switch nombre {
	case "EAX":
		return &procesoActual.RegistrosCPU.EAX
	case "EBX":
		return &procesoActual.RegistrosCPU.EBX
	case "ECX":
		return &procesoActual.RegistrosCPU.ECX
	case "EDX":
		return &procesoActual.RegistrosCPU.EDX
	case "PC":
		return &procesoActual.RegistrosCPU.PC
	case "SI":
		return &procesoActual.RegistrosCPU.SI
	case "DI":
		return &procesoActual.RegistrosCPU.DI
	}
	otherwise := uint32(0)
	return &otherwise
}

type BodyRequestTime struct {
	Dispositivo string `json:"dispositivo"`
	CantidadIO  int    `json:"cantidad_io"`
	PID         int    `json:"pid"`
	Instruccion string `json:"instruccion"`
}

func IO_GEN_SLEEP(nombre string, tiempo int) {
	var sending BodyRequestTime

	sending.Dispositivo = nombre
	sending.CantidadIO = tiempo
	sending.PID = procesoActual.PID
	sending.Instruccion = "SLEEP"

	body, err := json.Marshal(sending)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortKernel) + "/io"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando: %s", err.Error())
		return
	}
	mutexMensaje.Lock()
	if resp.StatusCode != http.StatusOK {
		resultadoEjecucion.Mensaje = "EXIT INVALID_IO"
	} else {
		resultadoEjecucion.Mensaje = "BLOCKED " + sending.Dispositivo
	}
	mutexMensaje.Unlock()
	interrupcion = true
}

func decoYExecInstru(instrucciones string) {
	//"Decodificar" instruccion
	instru := strings.Split(strings.TrimRight(instrucciones, "\x00"), " ")
	//Ejecutar instruccion
	switch instru[0] {
	case "SET":
		valor, err := strconv.Atoi(instru[2])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		SET(instru[1], valor)
	case "SUM":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		SUM(instru[1], instru[2])
	case "SUB":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		SUB(instru[1], instru[2])
	case "JNZ":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		valor, err := strconv.Atoi(instru[2])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		JNZ(instru[1], valor)
	case "EXIT":
		log.Printf("PID: %d - Ejecutando: %v", procesoActual.PID, instru[0])
		mutexMensaje.Lock()
		resultadoEjecucion.Mensaje = "EXIT SUCCESS"
		mutexMensaje.Unlock()
		interrupcion = true
	case "IO_GEN_SLEEP":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		valor, err := strconv.Atoi(instru[2])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_GEN_SLEEP(instru[1], valor)
	}
}

func SolicitarInstruccion(pid int, pc int) string {

	url := fmt.Sprintf("http://localhost:%d/instruccion/%d/%d", globals.ClientConfig.PortMemory, pid, pc)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error al enviar la solicitud: %s", err.Error())
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en la respuesta del servidor: %s", resp.Status)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error al leer la respuesta: %s", err.Error())
		return ""
	}

	var instruccion string

	err = json.Unmarshal(body, &instruccion)
	if err != nil {
		log.Printf("Error al decodificar la respuesta JSON: %s", err.Error())
		return ""
	}

	return instruccion
}

func FinDeQuantum(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	if procesoActual.PID == pidInt {
		mutexMensaje.Lock()
		motivo := strings.Split(strings.TrimRight(resultadoEjecucion.Mensaje, "\x00"), " ")

		if motivo[0] == "BLOCKED" || motivo[0] == "EXIT" {
			mutexMensaje.Unlock()
			return
		} else {
			resultadoEjecucion.Mensaje = "READY QUANTUM"
		}
		mutexMensaje.Unlock()
		interrupcion = true
	}

}

func Desalojar(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de string a Int", 0)
		return
	}

	if procesoActual.PID == pidInt {
		interrupcion = true
	}

}
