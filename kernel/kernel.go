package main

import (
	"kernel/globals"
	"kernel/utils"
	"net/http"
	"strconv"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("config.json")

	mux := http.NewServeMux()

	mux.HandleFunc("PUT /process", utils.IniciarProceso)
	mux.HandleFunc("PUT /enviar", utils.PlanificadoCortoPlazo)
	mux.HandleFunc("DELETE /process/{pid}", utils.FinalizarProceso)
	mux.HandleFunc("GET /process/{pid}", utils.EstadoProceso)
	mux.HandleFunc("PUT /plani", utils.IniciarPlanificacion)
	mux.HandleFunc("DELETE /plani", utils.DetenerPlanificacion)
	mux.HandleFunc("GET /process", utils.ListarProcesos)
	mux.HandleFunc("POST /io", utils.PedirIO)
	print(globals.ClientConfig.PortKernel)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
