package main

import (
	"cpu/globals"
	"cpu/utils"
	"net/http"
	"strconv"
)

func main() {
	//utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("config.json")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /probar", utils.ProbarCPU)
	mux.HandleFunc(" /PCB", utils.RecibirProceso)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), mux)
	if err != nil {
		panic(err)
	}
}
