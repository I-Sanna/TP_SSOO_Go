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

	utils.InicializarVariables()
	utils.InicializarPlanificador()

	mux := http.NewServeMux()

	mux.HandleFunc("PUT /process", utils.IniciarProceso)
	mux.HandleFunc("DELETE /process/{pid}", utils.FinalizarProceso)
	mux.HandleFunc("GET /process/{pid}", utils.EstadoProceso)
	mux.HandleFunc("PUT /plani", utils.IniciarPlanificacion)
	mux.HandleFunc("DELETE /plani", utils.DetenerPlanificacion)
	mux.HandleFunc("GET /process", utils.ListarProcesos)
	mux.HandleFunc("POST /io", utils.PedirIO)
	mux.HandleFunc("POST /nuevoIO", utils.RegistrarIO)
	mux.HandleFunc("POST /fs/create", utils.HandleCreateFileRequest)
	mux.HandleFunc("POST /fs/delete", utils.HandleDeleteFileRequest)
	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
