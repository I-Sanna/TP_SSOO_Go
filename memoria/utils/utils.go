package utils

import (
	"encoding/json"
	"fmt"
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
	count, err := file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	strData := string(data)
	instrucciones := strings.Split(strings.TrimRight(strData, "\x00"), "\n")
	//for _, value := range instrucciones {
	//	fmt.Println(value)
	//}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])
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
	instr := readFile(request.Path)
	respuesta, err := json.Marshal(instr)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Print("se creo proceso exitosamente")

}
