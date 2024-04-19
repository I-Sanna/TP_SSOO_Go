package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /cpu", ejecutarInstruccion)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}

func ejecutarInstruccion(w http.ResponseWriter, r *http.Request) {

	//pid := obtenerPID(r)

	//delete(procesos, pid)

	//fmt.Printf("Finaliza el proceso %d - Motivo: SUCCESS\n", pid)

	respuesta, err := json.Marshal("Se solicito ejecutar instruccion")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Print("instruccion ejecutada correctamente")
}
