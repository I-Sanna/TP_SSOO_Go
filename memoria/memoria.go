package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

//var memory = make(map[int]string)

func main() {
	readFile("../leer.txt") //con ../ busca el que esta en general, con ./ o sin aclarar busca el de memoria
	mux := http.NewServeMux()

	mux.HandleFunc("GET /memoria", crearProceso)

	err := http.ListenAndServe(":8002", mux)
	if err != nil {
		panic(err)
	}
}

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

/* func allocateMemory(w http.ResponseWriter, r *http.Request) {
	address := len(memory)
	memory[address] = "DATA"

	fmt.Fprintf(w, "Memory allocated at address %d", address)
}*/

func crearProceso(w http.ResponseWriter, r *http.Request) {

	respuesta, err := json.Marshal("se crea un nuevo proceso")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Print("se creo proceso exitosamente")

}
