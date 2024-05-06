package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

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

/*
	 func allocateMemory(w http.ResponseWriter, r *http.Request) {
		address := len(memory)
		memory[address] = "DATA"

		fmt.Fprintf(w, "Memory allocated at address %d", address)
	}
*/
type BodyRequest struct {
	Path string `json:"path"`
}

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
