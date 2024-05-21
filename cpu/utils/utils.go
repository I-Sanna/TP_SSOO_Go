package utils

import (
	"bytes"
	"cpu/globals"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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

var procesoActual PCB

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

func RecibirProceso(w http.ResponseWriter, r *http.Request) {
	var paquete PCB

	err := json.NewDecoder(r.Body).Decode(&paquete)
	if err != nil {
		log.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	procesoActual = paquete

	log.Println("me llegó un Proceso")
	log.Printf("%+v\n", paquete)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ProbarCPU(w http.ResponseWriter, r *http.Request) {
	IO_GEN_SLEEP("hola", 10)
}

func SET(nombreRegistro string, valor int) {
	if strlen(nombreRegistro) == 2 && strings.Contains(nombreRegistro, "X") {
		var registro *uint8 = ObtenerRegistro8Bits(nombreRegistro)
		*registro = uint8(valor)
	} else {
		var registro *uint32 = ObtenerRegistro32Bits(nombreRegistro)
		*registro = uint32(valor)
	}
	log.Printf("%+v\n", procesoActual)
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
	log.Printf("%+v\n", procesoActual)
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
	log.Printf("%+v\n", procesoActual)
}

func JNZ(nombreRegistro string, valor int) {
	if strlen(nombreRegistro) == 2 && strings.Contains(nombreRegistro, "X") {
		var registro *uint8 = ObtenerRegistro8Bits(nombreRegistro)
		if *registro != uint8(0) {
			procesoActual.RegistrosCPU.PC = uint32(valor)
		}
	} else {
		var registro *uint32 = ObtenerRegistro32Bits(nombreRegistro)
		if *registro != uint32(0) {
			procesoActual.RegistrosCPU.PC = uint32(valor)
		}
	}
	log.Printf("%+v\n", procesoActual)
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
	Dispositivo string
	CantidadIO  int
}

func IO_GEN_SLEEP(nombre string, tiempo int) {
	var sending BodyRequestTime

	sending.Dispositivo = nombre
	sending.CantidadIO = tiempo

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

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func LeerPseudo(w http.ResponseWriter, r *http.Request) {
	//var paquete PCB

	//err := json.NewDecoder(r.Body).Decode(&paquete)
}
