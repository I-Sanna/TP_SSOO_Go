package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type IOInterface struct {
	Name string `json:"name"`
}

var ioInterfaces = make(map[string]IOInterface)

func main() {
	http.HandleFunc("GET /connect", leerDeConsola)
	//http.HandleFunc("/disconnect", disconnectInterface)

	fmt.Println("I/O Interfaces running on :8004")
	log.Fatal(http.ListenAndServe(":8004", nil))
}

func leerDeConsola(w http.ResponseWriter, r *http.Request) {
	respuesta, err := json.Marshal("se lee desde la consola")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Print("leer desde consola")

}
