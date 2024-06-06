package main

import (
	"memoria/globals"
	"memoria/utils"
	"net/http"
	"strconv"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("config.json")

	utils.InicializarMemoriaYTablas()

	mux := http.NewServeMux()

	//mux.HandleFunc("DELETE /process", finalizarProceso)
	mux.HandleFunc("PUT /process", utils.CrearProceso)
	mux.HandleFunc("GET /instruccion/{pid}/{pc}", utils.DevolverInstruccion)
	mux.HandleFunc("GET /pagina/{pid}/{pagina}", utils.BuscarMarco)
	mux.HandleFunc("GET /memoria/{pid}/{tamaño}", utils.ReservarMemoria)
	mux.HandleFunc("DELETE /process/{pid}", utils.LiberarRecursos)
	mux.HandleFunc("POST /leer", utils.LeerMemoria)
	mux.HandleFunc("POST /escribir", utils.EscribirMemoria)

	err := http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), mux)
	if err != nil {
		panic(err)
	}
}
