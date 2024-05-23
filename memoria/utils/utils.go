package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"memoria/globals"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var memoria []byte

var tablaPaginas map[int]int

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
	memoria = make([]byte, globals.ClientConfig.MemorySize)

	tablaPaginas = make(map[int]int)
}

func readFile(fileName string) []string {
	file, err := os.Open(fileName) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	data := make([]byte, 150)
	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	strData := string(data)
	instrucciones := strings.Split(strings.TrimRight(strData, "\x00"), "\n")
	//for _, value := range instrucciones {
	//	fmt.Println(value)
	//}
	//fmt.Printf("read %d bytes: %q\n", count, data[:count])
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

/*
	 func allocateMemory(w http.ResponseWriter, r *http.Request) {
		address := len(memory)
		memory[address] = "DATA"

		fmt.Fprintf(w, "Memory allocated at address %d", address)
	}
*/

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	instrucciones := readFile(request.Path)
	instruccionesProcesos = append(instruccionesProcesos, instrucciones)

	log.Printf("%+v\n", instruccionesProcesos)
	w.WriteHeader(http.StatusOK)
}

func DevolverInstruccion(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	pc := r.PathValue("pc")
	log.Println(pc)
	log.Println(pid)
	indice, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}
	subindice, err := strconv.Atoi(pc)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}
	log.Println(indice)
	log.Println(subindice)
	log.Println(instruccionesProcesos[indice][subindice])
	respuesta, err := json.Marshal(instruccionesProcesos[indice][subindice])
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func EnviarPresudo(w http.ResponseWriter, r *http.Request) {

	filePath := "memoria/leer.txt"
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("error leyendo el archivo: %s", err.Error())
		http.Error(w, "Error leyendo el archivo", http.StatusInternalServerError)
		return
	}

	url := "http://localhost:8006/RecibirPseudo{pseudocodigo}"

	resp, err := http.Post(url, "text/plain", bytes.NewBuffer(fileContent)) // Cambiar a "text/plain" para enviar un archivo de texto
	if err != nil {
		log.Printf("error enviando txt: %s", err.Error())
		return
	}

	defer resp.Body.Close()
	log.Printf("respuesta del servidor: %s", resp.Status)
}
