package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func readFile(fileName string) {
	file, err := os.Open(fileName) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	data := make([]byte, 100)
	count, err := file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])
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

func crearProceso(w http.ResponseWriter, r *http.Request) {
	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	readFile(request.Path)
	respuesta, err := json.Marshal("se crea un nuevo proceso")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Print("se creo proceso exitosamente")

}
