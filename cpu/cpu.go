package main

import (
	"cpu/globals"
	"cpu/utils"
	"net/http"
	"strconv"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("config.json")

	utils.InicializarTLB()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /PCB", utils.RecibirProceso)

	go muxInterrupciones(mux)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), mux)
	if err != nil {
		panic(err)
	}
}

func muxInterrupciones(mux *http.ServeMux) {
	mux.HandleFunc("GET /quantum/{pid}", utils.FinDeQuantum)
	mux.HandleFunc("GET /desalojar/{pid}", utils.Desalojar)
}
