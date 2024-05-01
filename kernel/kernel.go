package main

import (
	"net/http"
)

func main() {
	configurar()
	mux := http.NewServeMux()

	mux.HandleFunc("PUT /process", iniciarProceso)
	mux.HandleFunc("DELETE /process/{pid}", finalizarProceso)
	mux.HandleFunc("GET /process/{pid}", estadoProceso)
	mux.HandleFunc("PUT /plani", iniciarPlanificacion)
	mux.HandleFunc("DELETE /plani", detenerPlanificacion)
	mux.HandleFunc("GET /process", listarProcesos)

	err := http.ListenAndServe(":8001", mux)
	if err != nil {
		panic(err)
	}
}
