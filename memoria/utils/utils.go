package utils

import (
	"encoding/json"
	"io"
	"log"
	"memoria/globals"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var memoria []byte

var tablaPaginas map[int]int

var contadorPID int
var instruccionesPID []int
var instruccionesProcesos [][]string

type BodyRequest struct {
	Path string `json:"path"`
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
	logFile, err := os.OpenFile("logs/memoria.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func InicializarMemoriaYTablas() {
	contadorPID = 0
	memoria = make([]byte, globals.ClientConfig.MemorySize)
	tablaPaginas = make(map[int]int)
}

func readFile(fileName string) []string {
	file, err := os.Open(fileName) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	data := make([]byte, 300)
	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	strData := string(data)
	instrucciones := strings.Split(strings.TrimRight(strData, "\x00"), "\n")

	return instrucciones
}

func BuscarMarco(w http.ResponseWriter, r *http.Request) {
	pagina := r.PathValue("pagina")

	i, err := strconv.Atoi(pagina)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	var marco = tablaPaginas[i]

	respuesta, err := json.Marshal(marco)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	instrucciones := readFile(request.Path)
	instruccionesPID = append(instruccionesPID, contadorPID)
	instruccionesProcesos = append(instruccionesProcesos, instrucciones)

	contadorPID++

	w.WriteHeader(http.StatusOK)
}

func DevolverInstruccion(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	pc := r.PathValue("pc")

	var indice int

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	for index, valor := range instruccionesPID {
		if valor == pidInt {
			indice = index
			break
		}
	}

	subindice, err := strconv.Atoi(pc)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	respuesta, err := json.Marshal(instruccionesProcesos[indice][subindice])
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}
