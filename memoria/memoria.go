package main

import (
	"encoding/json"
	"log"
	"net/http"
)

//var memory = make(map[int]string)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /memoria", crearProceso)

	err := http.ListenAndServe(":8002", mux)
	if err != nil {
		panic(err)
	}
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
