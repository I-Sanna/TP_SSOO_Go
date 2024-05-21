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
	mux.HandleFunc("GET /probar", utils.ProbarSET)
	mux.HandleFunc("GET /PCB", utils.RecibirProceso)
	mux.HandleFunc("PUT /IOKERNEL", utils.PeticionKernel)
	mux.HandleFunc("GET /RecibirPseudo{pseudocodigo", utils.LeerPseudo)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), mux)
	if err != nil {
		panic(err)
	}
}
