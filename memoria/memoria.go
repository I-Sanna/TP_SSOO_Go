package main

import (
	"memoria/globals"
	"memoria/utils"
	"net/http"
	"strconv"
)

//var memory = make(map[int]string)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("config.json")

	mux := http.NewServeMux()

	//mux.HandleFunc("DELETE /process", finalizarProceso)
	mux.HandleFunc("PUT /process", utils.CrearProceso)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), mux)
	if err != nil {
		panic(err)
	}
}
