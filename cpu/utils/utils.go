package utils

import (
	"bytes"
	"cpu/globals"
	"encoding/binary"
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
	Info      []byte `json:"info"`
	Tamaño    int    `json:"tamaño"`
	Direccion int    `json:"direccion"`
}

func LeerDeMemoria(pid int, direccion int, tamaño int) ([]byte, error) {
	request := BodyEscritura{
		PID:       pid,
		Direccion: direccion,
		Tamaño:    tamaño,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error al codificar solicitud: %v", err)
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/leer"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error al enviar solicitud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error en la respuesta de la consulta: %v", resp.Status)
	}

	var resultado []byte
	err = json.NewDecoder(resp.Body).Decode(&resultado)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar respuesta: %v", err)
	}
	if tamaño == 4 {
		valorUint32 := binary.LittleEndian.Uint32(resultado)
		log.Printf("PID: %d - Acción: LEER - Dirección Física: %d - Valor: %d", pid, direccion, valorUint32)
	} else if tamaño == 1 {
		valorUint8 := resultado[0]
		log.Printf("PID: %d - Acción: LEER - Dirección Física: %d - Valor: %d", pid, direccion, valorUint8)
	} else {
		return nil, fmt.Errorf("tamaño de datos no soportado: %d bytes", tamaño)
	}

	return resultado, nil
}

func EscribirEnMemoria(pid int, direccionFisica int, datos []byte, tamaño int) error {
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

	if tamaño == 4 {
		valorUint32 := binary.LittleEndian.Uint32(datos)
		log.Printf("PID: %d - Acción: ESCRIBIR - Dirección Física: %d - Valor: %d", pid, direccionFisica, valorUint32)
	} else if tamaño == 1 {
		// si el tamaño es de 1 byte interpreta el primer byte como uint8
		valorUint8 := datos[0]
		log.Printf("PID: %d - Acción: ESCRIBIR - Dirección Física: %d - Valor: %d", pid, direccionFisica, valorUint8)
	} else {
		return fmt.Errorf("tamaño de datos no soportado: %d bytes", tamaño)
	}

	return nil
}

func ObtenerValorRegistro(nombreRegistro string) int {
	if len(nombreRegistro) == 2 && strings.Contains(nombreRegistro, "X") {
		registro := ObtenerRegistro8Bits(nombreRegistro)
		return int(*registro)
	} else {
		registro := ObtenerRegistro32Bits(nombreRegistro)
		return int(*registro)
	}
}

// MOV_IN (Registro Datos, Registro Dirección)
func MOV_IN(registroDatos, registroDireccion string) {
	var bytesLeidos []byte

	if len(registroDatos) == 2 && strings.Contains(registroDatos, "X") {
		regDatos8 := ObtenerRegistro8Bits(registroDatos)
		if regDatos8 == nil {
			log.Printf("Error: registro de 8 bits inválido")
			return
		}

		// Leer 1 byte
		dirLogica := ObtenerValorRegistro(registroDireccion)

		direccionFisica, err := mmu(procesoActual.PID, dirLogica)
		if err != nil {
			log.Printf("Error al traducir dirección: %s", err.Error())
			return
		}

		bytesLeidos, err = LeerDeMemoria(procesoActual.PID, direccionFisica, 1)
		if err != nil {
			log.Printf("Error al leer de memoria: %s", err.Error())
			return
		}

		// Asignar valor leído a regDatos8
		*regDatos8 = bytesLeidos[0]
	} else {
		regDatos32 := ObtenerRegistro32Bits(registroDatos)
		if regDatos32 == nil {
			log.Printf("Error: registro de 32 bits inválido")
			return
		}

		// Leer 4 bytes
		dirLogica := ObtenerValorRegistro(registroDireccion)
		direccionFisica, err := mmu(procesoActual.PID, dirLogica)
		if err != nil {
			log.Printf("Error al traducir dirección: %s", err.Error())
			return
		}

		bytesLeidos, err = LeerDeMemoria(procesoActual.PID, direccionFisica, 4)
		if err != nil {
			log.Printf("Error al leer de memoria: %s", err.Error())
			return
		}

		// Convertir bytes a uint32
		if len(bytesLeidos) != 4 {
			log.Printf("Error: se esperaban 4 bytes, pero se recibieron %d", len(bytesLeidos))
			return
		}
		*regDatos32 = binary.LittleEndian.Uint32(bytesLeidos)

	}
}

// MOV_OUT (Registro Dirección, Registro Datos)
func MOV_OUT(registroDireccion, registroDatos string) {

	if len(registroDatos) == 2 && strings.Contains(registroDatos, "X") {

		regDatos8 := ObtenerRegistro8Bits(registroDatos)
		if regDatos8 == nil {
			log.Printf("Error: registro de 8 bits inválido")
			return
		}

		// Convertir el valor del registro de 8 bits a []byte
		datos := []byte{*regDatos8}

		// Escribir 1 byte en memoria
		dirLogica := ObtenerValorRegistro(registroDireccion)
		direccionFisica, err := mmu(procesoActual.PID, dirLogica)
		if err != nil {
			log.Printf("Error al traducir dirección: %s", err.Error())
			return
		}

		err = EscribirEnMemoria(procesoActual.PID, direccionFisica, datos, 1)
		if err != nil {
			log.Printf("Error al escribir memoria: %s", err.Error())
			return
		}

	} else {
		regDatos32 := ObtenerRegistro32Bits(registroDatos)
		if regDatos32 == nil {
			log.Printf("Error: registro de 32 bits inválido")
			return
		}

		// Convertir el valor del registro a []byte
		datos := make([]byte, 4)
		binary.LittleEndian.PutUint32(datos, *regDatos32)

		// Escribir 4 bytes en memoria
		dirLogica := ObtenerValorRegistro(registroDireccion)
		direccionFisica, err := mmu(procesoActual.PID, dirLogica)
		if err != nil {
			log.Printf("Error al traducir dirección: %s", err.Error())
			return
		}

		err = EscribirEnMemoria(procesoActual.PID, direccionFisica, datos, 4)
		if err != nil {
			log.Printf("Error al escribir memoria: %s", err.Error())
			return
		}
	}
}

type BodyRequestResize struct {
	PID int `json:"pid"`
	Tam int `json:"tamaño"`
}

// RESIZE (tamS)
func RESIZE(tamS string) {
	tam, err := strconv.Atoi(tamS)
	if err != nil {
		log.Printf("Error al convertir el tamaño a entero: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://localhost:%d/memoria/%d/%d", globals.ClientConfig.PortMemory, procesoActual.PID, tam)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error al enviar la solicitud: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en la respuesta del servidor: %s", resp.Status)
		if resp.StatusCode == http.StatusInsufficientStorage { // Out of Memory
			mutexMensaje.Lock()
			resultadoEjecucion.Mensaje = "EXIT OUT_OF_MEMORY"
			mutexMensaje.Unlock()
			interrupcion = true
		}
		return
	}
}

// COPY_STRING (Tamaño)
func COPY_STRING(tamS string) {
	tam, err := strconv.Atoi(tamS)
	if err != nil {
		log.Printf("Error al convertir el tamaño a entero: %s", err.Error())
		return
	}

	si := procesoActual.RegistrosCPU.SI
	di := procesoActual.RegistrosCPU.DI

	// Leer el contenido de la memoria desde la dirección apuntada por SI
	contenido, err := LeerDeMemoria(procesoActual.PID, int(si), tam)
	if err != nil {
		log.Printf("Error al leer de memoria: %s", err.Error())
		return
	}

	// Escribir este contenido en la memoria en la dirección apuntada por DI
	err = EscribirEnMemoria(procesoActual.PID, int(di), contenido, tam)
	if err != nil {
		log.Printf("Error al escribir en memoria: %s", err.Error())
		return
	}

	log.Printf("PID: %d - COPY_STRING ejecutado: %d bytes copiados desde %d a %d", procesoActual.PID, tam, si, di)
}

type BodyRRSS struct {
	PID     int    `json:"pid"`
	Recurso string `json:"recurso"`
}

// WAIT (Recurso)
func WAIT(recurso string) {
	mutexMensaje.Lock()
	resultadoEjecucion.Mensaje = "BLOCKED WAIT " + recurso
	mutexMensaje.Unlock()

	interrupcion = true
}

// SIGNAL (Recurso)
func SIGNAL(recurso string) {
	mutexMensaje.Lock()
	resultadoEjecucion.Mensaje = "BLOCKED SIGNAL " + recurso
	mutexMensaje.Unlock()

	interrupcion = true
}

func mmu(pid int, direccionLogica int) (int, error) {

	// Obtener el tamaño de página
	pageSize, err := ObtenerPageSize()
	if err != nil {
		return 0, fmt.Errorf("error al obtener el tamaño de página: %w", err)
	}

	numeroPagina := direccionLogica / pageSize
	desplazamiento := direccionLogica - numeroPagina*pageSize

	if TLBCPU == nil {
		log.Printf("Error: TLB no está inicializada")
		return 0, err
	}

	// Consultar TLB
	marcoTLB, err := buscarEnTLB(pid, numeroPagina)

	if err == nil {
		direccionFisica := marcoTLB*pageSize + desplazamiento
		log.Printf("PID: %d - TLB HIT - Pagina: %d", pid, numeroPagina)
		return direccionFisica, nil // TLB Hit
	}

	// Consultar tabla de páginas en la memoria principal
	marco, err := buscarEnMemoria(pid, numeroPagina)
	if err != nil {
		return 0, fmt.Errorf("error al buscar en memoria: %w", err)
	}

	// Actualizar TLB
	actualizarTLB(pid, numeroPagina, marco)
	// Calcular dirección física
	direccionFisica := marco*pageSize + desplazamiento

	return direccionFisica, nil
}

func InicializarTLB() {

	numEntradas := globals.ClientConfig.NumberFellingTbl

	TLBCPU = &TLB{Entradas: make([]TLBEntry, 0, numEntradas)}

	log.Printf("Inicializando TLB con %d entradas", numEntradas)

}

func buscarEnTLB(pid, numeroPagina int) (int, error) {

	for i, entry := range TLBCPU.Entradas {
		if entry.PID == pid && entry.Pagina == numeroPagina {
			if globals.ClientConfig.AlgorithmTbl == "LRU" {
				TLBCPU.Entradas = append(TLBCPU.Entradas[:i], TLBCPU.Entradas[i+1:]...) // elimina la entrada encontrada
				TLBCPU.Entradas = append(TLBCPU.Entradas, entry)                        // añade la entrada al final
			}
			return entry.Marco, nil // TLB Hit

		}
	}
	log.Printf("PID: %d - TLB MISS - Pagina: %d", pid, numeroPagina)
	return 0, fmt.Errorf("TLB Miss")
}

/*
	func imprimirTLB() {
		fmt.Println("Tabla de la TLB:")
		fmt.Println("-----------------------------------------")
		fmt.Printf("| %-5s | %-8s | %-5s | %-6s |\n", "Index", "PID", "Página", "Marco")
		fmt.Println("-----------------------------------------")

		for index, entry := range TLBCPU.Entradas {
			fmt.Printf("| %-5d | %-8d | %-5d | %-6d |\n", index, entry.PID, entry.Pagina, entry.Marco)
		}

		fmt.Println("-----------------------------------------")
	}
*/
func actualizarTLB(pid, numeroPagina, marco int) {

	if len(TLBCPU.Entradas) >= globals.ClientConfig.NumberFellingTbl {
		TLBCPU.Entradas = TLBCPU.Entradas[1:] // Elimina la entrada más antigua
	}

	nuevaEntrada := TLBEntry{
		PID:    pid,
		Pagina: numeroPagina,
		Marco:  marco,
	}

	TLBCPU.Entradas = append(TLBCPU.Entradas, nuevaEntrada)
	//imprimirTLB()

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

	log.Printf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, numeroPagina, marco)

	return marco, nil
}

func ObtenerPageSize() (int, error) {
	response, err := http.Get("http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/pageSize")
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
	Tamaño      int    `json:"tamaño"`
	Direccion   int    `json:"direccion"`
	Instruccion string `json:"instruccion"`
}

type BodyRequestSTD struct {
	Dispositivo string `json:"dispositivo"`
	PID         int    `json:"pid"`
	Tamaño      int    `json:"tamaño"`
	Direccion   int    `json:"direccion"`
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

func IO_STDIN_READ(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "READ"

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

func IO_STDOUT_WRITE(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "WRITE"

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

// Dial FS (modificar)
func IO_FS_CREATE(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "CREATE"

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
func IO_FS_DELETE(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "DELETE"

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
func IO_FS_TRUNCATE(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "TRUNCATE"

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
func IO_FS_WRITE(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "FSWRITE"

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
func IO_FS_READ(nombre string, tamaño int, direccion int) {
	var sending BodyRequestSTD

	sending.Dispositivo = nombre
	sending.PID = procesoActual.PID
	sending.Tamaño = tamaño
	sending.Direccion = direccion
	sending.Instruccion = "FSREAD"

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
	case "SIGNAL":
		log.Printf("PID: %d - Ejecutando: %v - %v", procesoActual.PID, instru[0], instru[1])
		SIGNAL(instru[1])
	case "WAIT":
		log.Printf("PID: %d - Ejecutando: %v - %v", procesoActual.PID, instru[0], instru[1])
		WAIT(instru[1])
	case "COPY_STRING":
		log.Printf("PID: %d - Ejecutando: %v - %v", procesoActual.PID, instru[0], instru[1])
		COPY_STRING(instru[1])
	case "RESIZE":
		log.Printf("PID: %d - Ejecutando: %v - %v", procesoActual.PID, instru[0], instru[1])
		RESIZE(instru[1])
	case "MOV_IN":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		MOV_IN(instru[1], instru[2])
	case "MOV_OUT":
		log.Printf("PID: %d - Ejecutando: %v - %v,%v", procesoActual.PID, instru[0], instru[1], instru[2])
		MOV_OUT(instru[1], instru[2])
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
	case "IO_STDIN_READ":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_STDIN_READ(instru[1], tamaño, direccion)
	case "IO_STDOUT_WRITE":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_STDOUT_WRITE(instru[1], tamaño, direccion)
	case "IO_FS_CREATE":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_FS_CREATE(instru[1], tamaño, direccion)
	case "IO_FS_DELETE":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_FS_DELETE(instru[1], tamaño, direccion)
	case "IO_FS_TRUNCATE":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_FS_TRUNCATE(instru[1], tamaño, direccion)
	case "IO_FS_WRITE":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_FS_WRITE(instru[1], tamaño, direccion)
	case "IO_FS_READ":
		log.Printf("PID: %d - Ejecutando: %v - %v , %v , %v", procesoActual.PID, instru[0], instru[1], instru[2], instru[3])
		tamaño, err := strconv.Atoi(instru[2])
		direccion, err := strconv.Atoi(instru[3])
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			return
		}
		IO_FS_READ(instru[1], tamaño, direccion)
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
